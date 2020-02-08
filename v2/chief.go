package uwe

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/lancer-kit/sam"
	"github.com/pkg/errors"
)

type Chief interface {
	Run()
	Shutdown()

	AddWorker(WorkerName, Worker)
	AddWorkers(map[WorkerName]Worker)
	GetWorkersStates() map[WorkerName]sam.State

	SetEventHandler(EventHandler)
	SetContext(context.Context)
	SetLocker(Locker)
	SetRecover(Recover)
	UseDefaultRecover()
	SetShutdown(Shutdown)

	Event() <-chan Event
}

type (
	Locker       func()
	Recover      func(name WorkerName)
	Shutdown     func()
	EventHandler func(Event)
)

// DefaultForceStopTimeout is a timeout for killing all workers.
const DefaultForceStopTimeout = 45 * time.Second

type chief struct {
	ctx    context.Context
	cancel context.CancelFunc

	forceStopTimeout time.Duration

	locker   Locker
	recover  Recover
	shutdown Shutdown
	wPool    *WorkerPool

	eventMutex   sync.Mutex
	eventChan    chan Event
	eventHandler EventHandler
}

func NewChief() Chief {
	c := new(chief)
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.eventChan = make(chan Event)
	c.forceStopTimeout = DefaultForceStopTimeout
	c.wPool = &WorkerPool{
		mutex:   new(sync.RWMutex),
		workers: make(map[WorkerName]*workerRO),
	}
	return c
}

func (c *chief) RunInfoHandler() {

}

func (c *chief) AddWorker(name WorkerName, worker Worker) {
	if err := c.wPool.SetWorker(name, worker); err != nil {
		c.eventChan <- ErrorEvent(err.Error()).SetWorker(name)
	}
}

func (c *chief) AddWorkers(workers map[WorkerName]Worker) {
	for name, worker := range workers {
		if err := c.wPool.SetWorker(name, worker); err != nil {
			c.eventChan <- ErrorEvent(err.Error()).SetWorker(name)
		}
	}
}

func (c *chief) GetWorkersStates() map[WorkerName]sam.State {
	return c.wPool.GetWorkersStates()
}

func (c *chief) SetEventHandler(handler EventHandler) {
	c.eventHandler = handler
}

func (c *chief) SetContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *chief) SetLocker(locker Locker) {
	c.locker = locker
}

func (c *chief) SetRecover(recover Recover) {
	c.recover = recover
}

func (c *chief) UseDefaultRecover() {
	c.recover = func(name WorkerName) {
		r := recover()
		if r == nil {
			return
		}

		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}

		c.eventChan <- Event{
			Level:   LvlFatal,
			Worker:  name,
			Message: "caught panic",
			Fields: map[string]interface{}{
				"worker": name,
				"error":  err.Error(),
				"stack":  string(debug.Stack()),
			},
		}
	}
}

func (c *chief) SetShutdown(shutdown Shutdown) {
	c.shutdown = shutdown
}

func (c *chief) SetForceStopTimeout(forceStopTimeout time.Duration) {
	c.forceStopTimeout = forceStopTimeout
}

func (c *chief) Event() <-chan Event {
	if c.eventHandler != nil {
		return nil
	}

	c.eventMutex.Lock()
	return c.eventChan
}

func (c *chief) Run() {
	if c.locker == nil {
		c.locker = waitForSignal
	}
	if c.eventHandler != nil {
		stop := make(chan struct{})
		defer func() {
			stop <- struct{}{}
		}()
		go c.handleEvents(stop)
	}

	c.run()
}

func (c *chief) Shutdown() {
	c.cancel()

	c.eventMutex.Unlock()
	if c.shutdown != nil {
		c.shutdown()
	}
}

func (c *chief) run() {
	lockerDone := make(chan struct{})
	go func() {
		c.locker()
		c.Shutdown()
		lockerDone <- struct{}{}
	}()

	poolStopped := make(chan struct{})
	go func() {
		err := c.runPool()
		if err != nil {
			c.eventChan <- ErrorEvent(err.Error())
			lockerDone <- struct{}{}
		}
		poolStopped <- struct{}{}
	}()

	<-lockerDone

	select {
	case <-poolStopped:
		return
	case <-time.NewTimer(c.forceStopTimeout).C:
		c.eventChan <- ErrorEvent("graceful shutdown failed")
		return
	}
}

func (c *chief) handleEvents(stop <-chan struct{}) {
	c.eventMutex.Lock()

	for {
		select {
		case event := <-c.eventChan:
			c.eventHandler(event)
		case <-stop:
			return
		}
	}
}

func (c *chief) runPool() error {
	wg := new(sync.WaitGroup)

	var runCount int
	ctx, cancel := context.WithCancel(c.ctx)

	for name := range c.wPool.workers {
		if err := c.wPool.InitWorker(name); err != nil {
			c.eventChan <- ErrorEvent(errors.Wrap(err, "failed to init worker").Error()).SetWorker(name)
			continue
		}

		runCount++
		wg.Add(1)

		go c.runWorker(ctx, name, wg.Done)
	}

	if runCount == 0 {
		cancel()
		return errors.New("unable to start: there is no initialized workers")
	}

	<-c.ctx.Done()

	cancel()
	wg.Wait()

	return nil
}

func (c *chief) runWorker(ctx Context, name WorkerName, doneCall func()) {
	defer doneCall()
	if c.recover != nil {
		defer c.recover(name)
	}

	err := c.wPool.RunWorkerExec(ctx, name)
	if err != nil {
		c.eventChan <- ErrorEvent(err.Error()).SetWorker(name)
	}

	err = c.wPool.StopWorker(name)
	if err != nil {
		c.eventChan <- ErrorEvent(err.Error()).SetWorker(name)
	}
}

func waitForSignal() {
	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)
	<-gracefulStop
}
