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
	"github.com/lancer-kit/uwe/v2/socket"
	"github.com/pkg/errors"
)

// Chief is a supervisor that can be placed at the top of the go application's execution stack,
// it is blocked until SIGTERM is intercepted and then it shutdown all workers gracefully.
// Also, `Chief` can be used as a child supervisor inside the `Worker`, which is launched by `Chief` at the top-level.
type Chief interface {
	// AddWorker registers the worker in the pool.
	AddWorker(WorkerName, Worker)
	// AddWorkers registers the list of workers in the pool.
	AddWorkers(map[WorkerName]Worker)
	// GetWorkersStates returns the current state of all registered workers.
	GetWorkersStates() map[WorkerName]sam.State
	// EnableServiceSocket initializes `net.Socket` server for internal management purposes.
	// By default includes two actions:
	// 	- "status" is a command useful for health-checks, because it returns status of all workers;
	// 	- "ping" is a simple command that returns the "pong" message.
	// The user can provide his own list of actions with handler closures.
	EnableServiceSocket(app AppInfo, actions ...socket.Action)
	// Event returns the channel with internal Events.
	// ATTENTION: `Event() <-chan Event` and `SetEventHandler(EventHandler)` is mutually exclusive,
	// but one of them must be used!
	Event() <-chan Event
	// SetEventHandler adds a callback that processes the `Chief`
	// internal events and can log them or do something else.
	// ATTENTION: `Event() <-chan Event` and `SetEventHandler(EventHandler)` is mutually exclusive,
	// but one of them must be used!
	SetEventHandler(EventHandler)
	// SetContext replaces the default context with the provided one.
	// It can be used to deliver some values inside `(Worker) .Run (ctx Context)`.
	SetContext(context.Context)
	// SetLocker sets a custom `Locker`, if it is not set,
	// the default `Locker` will be used, which expects SIGTERM or SIGINT system signals.
	SetLocker(Locker)
	// SetRecover sets a custom `recover` that catches panic.
	SetRecover(Recover)
	// SetShutdown sets `Shutdown` callback.
	SetShutdown(Shutdown)
	// UseDefaultRecover sets a standard handler as a `recover`
	// that catches panic and sends a fatal event to the event channel.
	UseDefaultRecover()
	// Run is the main entry point into the `Chief` run loop.
	// This method initializes all added workers, the server `net.Socket`,
	// if enabled, starts the workers in separate routines
	// and waits for the end of lock produced by the locker function.
	Run()
	// Shutdown sends stop signal to all child goroutines by triggering of the `context.CancelFunc()`
	// and executes `Shutdown` callback.
	Shutdown()
}

type (
	// Locker is a function whose completion of a call is a signal to stop `Chief` and all workers.
	Locker func()
	// Recover is a function that will be used as a `defer call` to handle each worker's panic.
	Recover func(name WorkerName)
	// // Shutdown is a callback function that will be executed after the Chief and workers are stopped.
	// Its main purpose is to close, complete, or retain some global states or shared resources.
	Shutdown func()
	// EventHandler callback that processes the `Chief` internal events, can log them or do something else.
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

	sw *socket.Server
}

// NewChief returns new instance of standard `Chief` implementation.
func NewChief() Chief {
	c := &chief{
		eventChan:        make(chan Event),
		forceStopTimeout: DefaultForceStopTimeout,
		wPool: &WorkerPool{
			mutex:   new(sync.RWMutex),
			workers: make(map[WorkerName]*workerRO),
		},
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())
	return c
}

// EnableServiceSocket initializes `net.Socket` server for internal management purposes.
// By default includes two actions:
// 	- "status" is a command useful for health-checks, because it returns status of all workers;
// 	- "ping" is a simple command that returns the "pong" message.
// The user can provide his own list of actions with handler closures.
func (c *chief) EnableServiceSocket(app AppInfo, actions ...socket.Action) {
	statusAction := socket.Action{Name: StatusAction,
		Handler: func(_ socket.Request) socket.Response {
			return socket.NewResponse(socket.StatusOk,
				StateInfo{App: app, Workers: c.wPool.GetWorkersStates()}, "")
		},
	}

	pingAction := socket.Action{Name: PingAction,
		Handler: func(_ socket.Request) socket.Response {
			return socket.NewResponse(socket.StatusOk, "pong", "")
		},
	}

	actions = append(actions, statusAction, pingAction)
	c.sw = socket.NewServer(app.SocketName(), actions...)
}

// AddWorker registers the worker in the pool.
func (c *chief) AddWorker(name WorkerName, worker Worker) {
	if err := c.wPool.SetWorker(name, worker); err != nil {
		c.eventChan <- ErrorEvent(err.Error()).SetWorker(name)
	}
}

// AddWorkers registers the list of workers in the pool.
func (c *chief) AddWorkers(workers map[WorkerName]Worker) {
	for name, worker := range workers {
		if err := c.wPool.SetWorker(name, worker); err != nil {
			c.eventChan <- ErrorEvent(err.Error()).SetWorker(name)
		}
	}
}

// GetWorkersStates returns the current state of all registered workers.
func (c *chief) GetWorkersStates() map[WorkerName]sam.State {
	return c.wPool.GetWorkersStates()
}

// SetEventHandler adds a callback that processes the `Chief`
// internal events and can log them or do something else.
func (c *chief) SetEventHandler(handler EventHandler) {
	c.eventHandler = handler
}

// SetContext replaces the default context with the provided one.
// It can be used to deliver some values inside `(Worker) .Run (ctx Context)`.
func (c *chief) SetContext(ctx context.Context) {
	c.ctx = ctx
}

// SetLocker sets a custom `Locker`, if it is not set,
// the default `Locker` will be used, which expects SIGTERM or SIGINT system signals.
func (c *chief) SetLocker(locker Locker) {
	c.locker = locker
}

// SetRecover sets a custom `recover` that catches panic.
func (c *chief) SetRecover(recover Recover) {
	c.recover = recover
}

// UseDefaultRecover sets a standard handler as a `recover`
// that catches panic and sends a fatal event to the event channel.
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

// SetShutdown sets `Shutdown` callback.
func (c *chief) SetShutdown(shutdown Shutdown) {
	c.shutdown = shutdown
}

// SetForceStopTimeout replaces the `DefaultForceStopTimeout`.
func (c *chief) SetForceStopTimeout(forceStopTimeout time.Duration) {
	c.forceStopTimeout = forceStopTimeout
}

// Event returns the channel with internal Events.
func (c *chief) Event() <-chan Event {
	if c.eventHandler != nil {
		return nil
	}

	c.eventMutex.Lock()
	return c.eventChan
}

// Run is the main entry point into the `Chief` run loop.
// This method initializes all added workers, the server `net.Socket`,
// if enabled, starts the workers in separate goroutines
// and waits for the end of lock produced by the locker function.
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

// Shutdown sends stop signal to all child goroutines by triggering of the `context.CancelFunc()`
// and executes `Shutdown` callback.
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

	if c.sw != nil {
		wg.Add(1)
		go func() {
			if err := c.sw.Serve(ctx); err != nil {
				c.eventChan <- ErrorEvent(
					errors.Wrap(err, "failed to run socket listener").Error()).
					SetWorker("internal_socket_listener")
			}
			wg.Done()
		}()
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
