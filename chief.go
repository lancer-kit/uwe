package uwe

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

type Chief interface {
	Init()
	Run()
	Shutdown()

	AddWorker(WorkerName, Worker)
	AddWorkers(map[WorkerName]Worker)

	SetEventHandler(EventHandler)
	SetContext(context.Context)
	SetLocker(Locker)

	Event() <-chan Event
}

type (
	Locker       func()
	Recover      func()
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
	return new(chief)
}

func (c *chief) Init() {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.eventChan = make(chan Event)
	c.forceStopTimeout = DefaultForceStopTimeout
	c.wPool = &WorkerPool{
		mutex:   new(sync.RWMutex),
		workers: make(map[WorkerName]*workerRO),
	}
}

func (c *chief) AddWorker(name WorkerName, worker Worker) {
	c.wPool.SetWorker(name, worker)
}

func (c *chief) AddWorkers(workers map[WorkerName]Worker) {
	for name, worker := range workers {
		c.wPool.SetWorker(name, worker)
	}
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

func (c *chief) SetRecover(recover func()) {
	c.recover = recover
}

func (c *chief) SetShutdown(shutdown func()) {
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
	defer c.Shutdown()
	if c.locker == nil {
		c.locker = waitForSignal
	}
	if c.eventHandler != nil {
		go c.handleEvents()
	}

	c.run()
}

func (c *chief) Shutdown() {
	c.cancel()
	c.shutdown()
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

func (c *chief) handleEvents() {
	c.eventMutex.Lock()
	defer c.eventMutex.Unlock()

	for event := range c.eventChan {
		c.eventHandler(event)
	}
}

func (c *chief) runPool() error {
	wg := new(sync.WaitGroup)

	var runCount int
	ctx, cancel := context.WithCancel(c.ctx)

	for name := range c.wPool.workers {
		if err := c.wPool.InitWorker(ctx, name); err != nil {
			c.eventChan <- ErrorEvent("failed to init worker").SetWorker(name)
			continue
		}

		runCount++
		wg.Add(1)

		go c.runWorker(name, wg.Done)
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

func (c *chief) runWorker(name WorkerName, doneCall func()) {
	defer doneCall()
	if c.recover != nil {
		defer c.recover()
	}

	err := c.wPool.RunWorkerExec(name)
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
