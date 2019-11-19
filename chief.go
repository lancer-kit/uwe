package uwe

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/lancer-kit/sam"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CtxKey is the type of context keys for the values placed by`Chief`.
type CtxKey string

const (
	// CtxKeyLog is a context key for a `*logrus.Entry` value.
	CtxKeyLog CtxKey = "chief-log"
)

// ForceStopTimeout is a timeout for killing all workers.
var ForceStopTimeout = 45 * time.Second

// Chief is a head of workers, it must be used to register, initialize
// and correctly start and stop asynchronous executors of the type `Worker`.
type Chief struct {
	logger *logrus.Entry
	ctx    context.Context
	//cancel context.CancelFunc

	wPool WorkerPool

	// active indicates that the `Chief` has been started.
	active bool
	// initialized indicates that the workers have been initialized.
	initialized bool

	// systemEvents
	workersSignals chan workerSignal

	workersEventHub map[WorkerName]chan<- *Message
	eventHub        <-chan *Message

	// EnableByDefault sets all the working `Enabled`
	// if none of the workers is passed on to enable.
	EnableByDefault bool
	// AppName main app identifier of instance for logger and etc.
	AppName string
}

// NewChief creates and initialize new instance of `Chief`
func NewChief(name string, enableByDefault bool, logger *logrus.Entry) *Chief {
	chief := Chief{
		AppName:         name,
		EnableByDefault: enableByDefault}
	return chief.Init(logger)
}

// Init initializes all internal states properly.
func (chief *Chief) Init(logger *logrus.Entry) *Chief {
	chief.logger = logger.WithFields(logrus.Fields{
		"app":     chief.AppName,
		"service": "worker-chief",
	})

	chief.ctx = context.WithValue(context.Background(), CtxKeyLog, chief.logger)
	chief.initialized = true

	chief.workersSignals = make(chan workerSignal, 4)
	chief.workersEventHub = make(map[WorkerName]chan<- *Message)

	return chief
}

// AddWorker register a new `Worker` to the `Chief` worker pool.
func (chief *Chief) AddWorker(name WorkerName, worker Worker) {
	chief.wPool.SetWorker(name, worker)
}

// EnableWorkers enables all worker from the `names` list.
// By default, all added workers are enabled. After the first call
// of this method, only directly enabled workers will be active
func (chief *Chief) EnableWorkers(names ...WorkerName) (err error) {
	for _, name := range names {
		err = chief.wPool.EnableWorker(name)
		if err != nil {
			return
		}
	}

	if len(names) == 0 && chief.EnableByDefault {
		for name := range chief.wPool.workers {
			err = chief.wPool.EnableWorker(name)
			if err != nil {
				return
			}
		}
	}

	return nil
}

// EnableWorker enables the worker with the specified `name`.
// By default, all added workers are enabled. After the first call
// of this method, only directly enabled workers will be active
func (chief *Chief) EnableWorker(name WorkerName) error {
	return chief.wPool.EnableWorker(name)
}

// IsEnabled checks is enable worker with passed `name`.
func (chief *Chief) IsEnabled(name WorkerName) bool {
	return chief.wPool.IsEnabled(name)
}

func (chief *Chief) GetWorkersStates() map[WorkerName]sam.State {
	return chief.wPool.GetWorkersStates()
}

func (chief *Chief) GetContext() context.Context {
	return chief.ctx
}

func (chief *Chief) AddValueToContext(key, value interface{}) {
	chief.ctx = context.WithValue(chief.ctx, key, value)
}

// Run enables passed workers, starts worker pool and lock context
// until it intercepts `syscall.SIGTERM`, `syscall.SIGINT`.
// NOTE: Use this method ONLY as a top-level action.
func (chief *Chief) Run(workers ...WorkerName) error {
	waitForSignal := func() {
		var gracefulStop = make(chan os.Signal, 1)
		signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

		exitSignal := <-gracefulStop
		chief.logger.WithField("signal", exitSignal).
			Info("Received signal. Terminating service...")
	}

	return chief.RunWithLocker(waitForSignal, workers...)
}

func (chief *Chief) RunWithContext(ctx context.Context, workers ...WorkerName) error {
	waitForSignal := func() {
		<-ctx.Done()
	}

	return chief.RunWithLocker(waitForSignal, workers...)
}

// RunWithLocker
// `locker` function should block the execution context and wait for some signal to stop.
func (chief *Chief) RunWithLocker(locker func(), workers ...WorkerName) (err error) {
	err = chief.EnableWorkers(workers...)
	if err != nil {
		return
	}

	lockerDone := make(chan struct{})
	poolCtx, poolCanceler := context.WithCancel(context.Background())
	go func() {
		locker()
		poolCanceler()
		lockerDone <- struct{}{}
	}()

	poolStopped := make(chan struct{})
	go func() {
		exitCode := chief.StartPool(poolCtx)
		if exitCode == workerPoolStartFailed {
			err = errors.New("worker pool starting failed")
			lockerDone <- struct{}{}
		}

		poolStopped <- struct{}{}
	}()

	<-lockerDone

	select {
	case <-poolStopped:
		chief.logger.Info("Graceful exit.")
		return
	case <-time.NewTimer(ForceStopTimeout).C:
		chief.logger.Warn("Graceful exit timeout exceeded. Force exit.")
		return
	}
}

const (
	workerPoolStartFailed     = -1
	workerPoolStoppedProperly = 0
)

// StartPool runs all registered workers, locks until the `parentCtx` closes,
// and then gracefully stops all workers.
// Returns result code:
// 	-1 — start failed
// 	 0 — stopped properly
func (chief *Chief) StartPool(parentCtx context.Context) int {
	if !chief.initialized {
		logrus.Error("Workers is not initialized! Unable to start.")
		return workerPoolStartFailed
	}

	chief.active = true
	wg := sync.WaitGroup{}
	chief.logger.Info(chief.AppName + " started")

	var runCount int
	ctx, cancel := context.WithCancel(chief.ctx)
	workersEventBus := make(chan *Message, len(chief.wPool.workers)*10)

	chief.eventHub = workersEventBus

	for name := range chief.wPool.workers {
		if chief.wPool.IsDisabled(name) {
			chief.logger.WithField("worker", name).
				Debug("Worker disabled")
			continue
		}

		if err := chief.wPool.InitWorker(name, ctx); err != nil {
			chief.logger.WithField("worker", name).
				Debug("Worker disabled")
			continue
		}

		runCount++
		wg.Add(1)

		workersDirectBus := make(chan *Message, len(chief.wPool.workers)*10)
		chief.workersEventHub[name] = workersDirectBus
		wCtx := NewContext(name, ctx, workersDirectBus, workersEventBus)

		go chief.runWorker(name, wCtx, wg.Done)
	}

	if runCount == 0 {
		chief.logger.Warn("No worker was running")
		return workerPoolStartFailed
	}

	go chief.runEventMux(chief.ctx)

	<-parentCtx.Done()
	chief.logger.Info("Begin graceful shutdown of workers")

	chief.active = false
	cancel()
	wg.Wait()

	chief.logger.Info("Workers pool stopped")
	return workerPoolStoppedProperly
}

func (chief *Chief) runWorker(name WorkerName, wCtx WContext, doneCall func()) {
	defer doneCall()

	defer func() {
		rec := recover()
		if rec == nil {
			return
		}
		e, ok := rec.(error)
		if !ok {
			e = fmt.Errorf("%v", rec)
		}
		chief.workersSignals <- workerSignal{name: name, sig: signalFailure, msg: e.Error()}
	}()

	logger := chief.logger.WithField("worker", name)
	logger.Info("Starting worker")

startWorker:
	err := chief.wPool.RunWorkerExec(name, wCtx)
	if err != nil {
		logger.WithError(err).
			Error("Worker failed")

		if chief.wPool.getWorker(name).worker.RestartOnFail() && chief.active {
			time.Sleep(time.Second)
			logger.Warn("Do worker restart...")
			goto startWorker
		}
	}

	err = chief.wPool.StopWorker(name)
	if err != nil {
		logger.WithError(err).
			Error("Worker state change failed")
	}
	chief.workersSignals <- workerSignal{name: name, sig: signalStop}
}

func (chief *Chief) runEventMux(ctx context.Context) {
	for {
		select {
		case m := <-chief.eventHub:
			if m == nil {
				continue
			}

			switch m.Target {
			case "*", "broadcast":
				for to := range chief.workersEventHub {
					chief.workersEventHub[to] <- m
				}
			default:
				if _, ok := chief.workersEventHub[m.Target]; ok {
					chief.workersEventHub[m.Target] <- m
				}
			}

		case <-ctx.Done():
			return
		}
	}
}
