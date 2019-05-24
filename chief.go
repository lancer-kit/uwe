package uwe

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

type execStatus string

const (
	signalOk             execStatus = "ok"
	signalInterrupted    execStatus = "interrupted"
	signalFailure        execStatus = "failure"
	signalStop           execStatus = "stop"
	signalUnexpectedStop execStatus = "unexpected_stop"
)

type workerSignal struct {
	name string
	sig  execStatus
	msg  string
}

func (s *workerSignal) Error() string {
	return fmt.Sprintf("%s(%v); %s", s.name, s.sig, s.msg)
}

// Chief is a head of workers, it must be used to register, initialize
// and correctly start and stop asynchronous executors of the type `Worker`.
type Chief struct {
	//ctx    context.Context
	//cancel context.CancelFunc
	logger *logrus.Entry
	ctx    context.Context
	wPool  WorkerPool

	// active indicates that the `Chief` has been started.
	active bool
	// initialized indicates that the workers have been initialized.
	initialized bool

	// systemEvents
	workersSignals chan workerSignal

	// EnableByDefault sets all the working `Enabled`
	// if none of the workers is passed on to enable.
	EnableByDefault bool
	// AppName main app identifier of instance for logger and etc.
	AppName string
	// RestartAllOnFail sets the force restart of all workers
	// if one of them failed.
	RestartAllOnFail bool
}

func NewChief(name string, enableByDefault bool, restartAllOnFail bool, logger *logrus.Entry) *Chief {
	chief := Chief{
		AppName:          name,
		EnableByDefault:  enableByDefault,
		RestartAllOnFail: restartAllOnFail}
	return chief.Init(logger)

}

func (chief *Chief) Init(logger *logrus.Entry) *Chief {
	chief.logger = logger.WithFields(logrus.Fields{
		"app":     chief.AppName,
		"service": "worker-chief",
	})

	chief.ctx = context.Background()
	chief.ctx = context.WithValue(chief.ctx, CtxKeyLog, chief.logger)
	chief.workersSignals = make(chan workerSignal, 4)
	chief.initialized = true

	return chief
}

// AddWorker register a new `Worker` to the `Chief` worker pool.
func (chief *Chief) AddWorker(name string, worker Worker) {
	chief.wPool.SetWorker(name, worker)
}

// EnableWorkers enables all worker from the `names` list.
// By default, all added workers are enabled. After the first call
// of this method, only directly enabled workers will be active
func (chief *Chief) EnableWorkers(names ...string) {
	for _, name := range names {
		chief.wPool.EnableWorker(name)
	}

	if len(names) == 0 && chief.EnableByDefault {
		for name := range chief.wPool.workers {
			chief.wPool.EnableWorker(name)
		}
	}
}

// EnableWorker enables the worker with the specified `name`.
// By default, all added workers are enabled. After the first call
// of this method, only directly enabled workers will be active
func (chief *Chief) EnableWorker(name string) {
	chief.wPool.EnableWorker(name)
}

// IsEnabled checks is enable worker with passed `name`.
func (chief *Chief) IsEnabled(name string) bool {
	return chief.wPool.IsEnabled(name)
}

func (chief *Chief) GetWorkersStates() map[string]WorkerState {
	return chief.wPool.GetWorkersStates()
}

func (chief *Chief) GetContext() context.Context {
	return chief.ctx
}

func (chief *Chief) AddValueToContext(key, value interface{}) {
	chief.ctx = context.WithValue(chief.ctx, key, value)
}

// RunAll start worker pool and lock context
// until it intercepts `syscall.SIGTERM`, `syscall.SIGINT`.
// NOTE: Use this method ONLY as a top-level action.
func (chief *Chief) RunAll(workers ...string) error {
	waitForSignal := func() {
		var gracefulStop = make(chan os.Signal, 1)
		signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

		exitSignal := <-gracefulStop
		chief.logger.WithField("signal", exitSignal).
			Info("Received signal. Terminating service...")
	}

	return chief.RunWithLocker(waitForSignal, workers...)
}

func (chief *Chief) RunWithContext(ctx context.Context, workers ...string) error {
	waitForSignal := func() {
		<-ctx.Done()
	}

	return chief.RunWithLocker(waitForSignal, workers...)
}

func (chief *Chief) RunWithLocker(locker func(), workers ...string) (err error) {
	poolStopped := make(chan struct{})
	executionUnlocked := make(chan struct{}, 2)
	startPoolCtx, startPollCancel := context.WithCancel(context.Background())

	chief.EnableWorkers(workers...)

	ctxLocker := NewContextLocker(locker)
	ctxLockerCancel := ctxLocker.CancelFunc()

dawn:

	go func() {
		//locker function should block
		// the execution context and
		// wait for some signal to stop.
		ctxLocker.Lock()
		startPollCancel()
		executionUnlocked <- struct{}{}
	}()
	var restartAll bool

	go func() {
		exitCode := chief.StartPool(startPoolCtx)
		if exitCode == workerPoolStartFailed {
			err = errors.New("worker pool starting failed")
		}

		executionUnlocked <- struct{}{}
		poolStopped <- struct{}{}
	}()

	go func() {
		for s := range chief.workersSignals {
			if s.sig == signalUnexpectedStop && chief.RestartAllOnFail {
				ctxLockerCancel()
				restartAll = true
			}
		}
	}()

	<-executionUnlocked

	select {
	case <-poolStopped:
		if restartAll {
			goto dawn
		}
		chief.logger.Info("Graceful exit.")
		return nil
	case <-time.NewTimer(ForceStopTimeout).C:
		chief.logger.Warn("Graceful exit timeout exceeded. Force exit.")
		return nil
	}
}

const (
	workerPoolStartFailed            = -1
	workerPoolStoppedProperly        = 0
	workerPoolStoppedUnintentionally = 1
)

// StartPool runs all registered workers, locks until the `parentCtx` closes,
// and then gracefully stops all workers.
// Returns result code:
// 	-1 — start failed
// 	 0 — stopped properly
// 	 1 — stopped unintentionally
func (chief *Chief) StartPool(parentCtx context.Context) int {
	if !chief.initialized {
		logrus.Error("Workers is not initialized! Unable to start.")
		return workerPoolStartFailed
	}

	chief.active = true
	wg := sync.WaitGroup{}
	chief.logger.Info(chief.AppName + " started")
	ctx, cancel := context.WithCancel(chief.ctx)

	for name, worker := range chief.wPool.workers {
		if !chief.wPool.IsEnabled(name) {
			chief.logger.WithField("worker", name).
				Debug("Worker disabled")
			continue
		}
		chief.wPool.InitWorker(name, ctx)

		wg.Add(1)
		go chief.runWorker(name, worker, wg.Done)
	}

	<-parentCtx.Done()

	chief.logger.Info("Begin graceful shutdown of workers")

	chief.active = false
	cancel()

	wg.Wait()
	chief.logger.Info("Workers stopped")

	return workerPoolStoppedProperly
}

func (chief *Chief) runWorker(name string, worker Worker, doneCall func()) {
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

	chief.logger.WithField("worker", name).Info("Starting worker")

startWorker:
	err := chief.wPool.RunWorkerExec(name)
	if err != nil {
		chief.logger.WithError(err).
			WithField("worker", name).
			Error("Worker failed")

		if chief.RestartAllOnFail {
			chief.wPool.StopWorker(name)
			chief.workersSignals <- workerSignal{name: name, sig: signalUnexpectedStop}
			return
		}

		if worker.RestartOnFail() && chief.active {
			time.Sleep(time.Second)
			chief.logger.WithField("worker", name).Warn("Do worker restart...")
			goto startWorker
		}
	}

	chief.wPool.StopWorker(name)
	chief.workersSignals <- workerSignal{name: name, sig: signalStop}
}
