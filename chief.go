package uwe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/lancer-kit/uwe/v3/socket"
	"github.com/sheb-gregor/sam"
)

// Chief is a supervisor that can be placed at the top  of the go app's execution stack,
// it is blocked until SIGTERM is intercepted, and then it shut down all workers gracefully.
// Also, `Chief` can be used as a child supervisor inside the `Worker`,
// which is launched by `Chief` at the top-level.
type Chief interface {
	// AddWorker registers the worker in the pool.
	AddWorker(WorkerName, Worker, ...WorkerOpts) Chief
	// GetWorkersStates returns the current state of all registered workers.
	GetWorkersStates() map[WorkerName]sam.State
	// EnableServiceSocket initializes `net.Socket` server for internal management purposes.
	// By default, includes two actions:
	// 	- "status" is a healthcheck-like, because it returns status of all workers;
	// 	- "ping" is a simple command that returns the "pong" message.
	// The user can provide his own list of actions with handler closures.
	EnableServiceSocket(app AppInfo, actions ...socket.Action) Chief
	// Event returns the channel with internal Events.
	// > ATTENTION:
	//   `Event() <-chan Event` and `SetEventHandler(EventHandler)`
	// are mutually exclusive, but one of them must be used!
	Event() <-chan Event
	// SetEventHandler adds a callback that processes the `Chief`
	// internal events and can log them or do something else.
	// > ATTENTION:
	//   `Event() <-chan Event` and `SetEventHandler(EventHandler)`
	// are mutually exclusive, but one of them must be used!
	SetEventHandler(EventHandler) Chief
	// SetContext replaces the default context with the provided one.
	// It can be used to deliver some values inside `(Worker) .Run (ctx Context)`.
	SetContext(context.Context) Chief
	// SetLocker sets a custom `Locker`, if it is not set,
	// the default `Locker` will be used, which expects SIGTERM or SIGINT system signals.
	SetLocker(Locker) Chief
	// SetShutdown sets `Shutdown` callback.
	SetShutdown(Shutdown) Chief
	// SetForceStopTimeout replaces the `DefaultForceStopTimeout`.
	// ForceStopTimeout is the duration before
	// the worker will be killed if it wouldn't finish Run after the stop signal.
	SetForceStopTimeout(time.Duration) Chief
	// UseCustomIMQBroker sets non-standard implementation
	// of the IMQBroker to replace default one.
	UseCustomIMQBroker(IMQBroker) Chief
	// UseNopIMQBroker replaces default IMQ Broker by empty stub.
	// NOP stands for no-operations.
	UseNopIMQBroker() Chief
	// Run is the main entry point into the `Chief` run loop.
	// This method initializes all added workers, the server `net.Socket`,
	// if enabled, starts the workers in separate routines
	// and waits for the end of lock produced by the locker function.
	Run()
	// Shutdown sends stop signal to all child goroutines
	// by triggering of the `context.CancelFunc()` and
	// executes `Shutdown` callback.
	Shutdown()
}

type (
	// Locker is a function whose completion of
	// a call is a signal to stop `Chief` and all workers.
	Locker func()
	// Recover is a function that will be used as a
	// `defer call` to handle each worker's panic.
	Recover func(name WorkerName)
	// Shutdown is a callback function that will be executed after the Chief
	// and workers are stopped. Its main purpose is to close, complete,
	// or retain some global states or shared resources.
	Shutdown func()
	// EventHandler callback that processes the `Chief` internal events,
	// can log them or do something else.
	EventHandler func(Event)
)

// DefaultForceStopTimeout is a timeout for killing all workers.
const DefaultForceStopTimeout = 45 * time.Second

type chief struct {
	ctx    context.Context
	cancel context.CancelFunc

	forceStopTimeout time.Duration
	locker           Locker
	shutdown         Shutdown
	wPool            *workerPool

	eventMutexLocked bool
	eventMutex       sync.Mutex
	eventChan        chan Event
	eventHandler     EventHandler

	broker IMQBroker
	sw     *socket.Server
}

// NewChief returns new instance of standard `Chief` implementation.
func NewChief() Chief {
	c := &chief{
		eventChan:        make(chan Event),
		forceStopTimeout: DefaultForceStopTimeout,
		wPool: &workerPool{
			workers: make(map[WorkerName]*workerRO),
		},
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())
	return c
}

// EnableServiceSocket initializes `net.Socket` server for internal management purposes.
// By default, includes two actions:
//   - "status" is a command useful for health-checks, because it returns status of all workers;
//   - "ping" is a simple command that returns the "pong" message.
//
// The user can provide his own list of actions with handler closures.
func (c *chief) EnableServiceSocket(app AppInfo, actions ...socket.Action) Chief {
	statusAction := socket.Action{Name: StatusAction,
		Handler: func(_ socket.Request) socket.Response {
			return socket.NewResponse(socket.StatusOk,
				StateInfo{App: app, Workers: c.wPool.getWorkersStates()}, "")
		},
	}

	pingAction := socket.Action{Name: PingAction,
		Handler: func(_ socket.Request) socket.Response {
			return socket.NewResponse(socket.StatusOk, "pong", "")
		},
	}

	actions = append(actions, statusAction, pingAction)
	c.sw = socket.NewServer(app.SocketName(), actions...)
	return c
}

// AddWorker registers the worker in the pool.
func (c *chief) AddWorker(name WorkerName, worker Worker, opts ...WorkerOpts) Chief {
	if err := c.wPool.setWorker(name, worker, opts); err != nil {
		c.eventChan <- ErrorEvent(err.Error()).SetWorker(name)
	}
	return c
}

// GetWorkersStates returns the current state of all registered workers.
func (c *chief) GetWorkersStates() map[WorkerName]sam.State {
	return c.wPool.getWorkersStates()
}

// SetEventHandler adds a callback that processes the `Chief`
// internal events and can log them or do something else.
func (c *chief) SetEventHandler(handler EventHandler) Chief {
	c.eventHandler = handler
	return c
}

// SetContext replaces the default context with the provided one.
// It can be used to deliver some values inside `(Worker).Run(ctx Context)`.
func (c *chief) SetContext(ctx context.Context) Chief {
	c.ctx = ctx
	return c
}

// SetLocker sets a custom `Locker`, if it is not set,
// the default `Locker` will be used, which expects SIGTERM or SIGINT system signals.
func (c *chief) SetLocker(locker Locker) Chief {
	c.locker = locker
	return c
}

func (c *chief) UseCustomIMQBroker(broker IMQBroker) Chief {
	c.broker = broker
	return c
}

func (c *chief) UseNopIMQBroker() Chief {
	c.broker = &NopBroker{}
	return c
}

// SetShutdown sets `Shutdown` callback.
func (c *chief) SetShutdown(shutdown Shutdown) Chief {
	c.shutdown = shutdown
	return c
}

// SetForceStopTimeout replaces the `DefaultForceStopTimeout`.
func (c *chief) SetForceStopTimeout(forceStopTimeout time.Duration) Chief {
	c.forceStopTimeout = forceStopTimeout
	return c
}

// Event returns the channel with internal Events.
func (c *chief) Event() <-chan Event {
	if c.eventHandler != nil {
		return nil
	}
	c.eventMutexLocked = true
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

// Shutdown sends stop signal to all child goroutines
// by triggering of the `context.CancelFunc()`
// and executes `Shutdown` callback.
func (c *chief) Shutdown() {
	c.cancel()
	if c.eventMutexLocked {
		c.eventMutex.Unlock()
	}

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
	c.eventMutexLocked = true
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

	if c.broker == nil {
		c.broker = NewBroker(len(c.wPool.workers) * 4)
	}
	if err := c.broker.Init(); err != nil {
		cancel()
		return fmt.Errorf("unable to init imq broker: %w", err)
	}

	for _, name := range c.wPool.workersList() {
		runCount++
		wg.Add(1)

		mailbox := c.broker.AddWorker(name)
		go c.runWorker(NewContext(ctx, mailbox), name, wg.Done)
	}

	if runCount == 0 {
		cancel()
		return errors.New("unable to start: there is no initialized workers")
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.broker.Serve(ctx)
	}()

	if c.sw != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.sw.Serve(ctx); err != nil {
				c.eventChan <- ErrorEvent(
					fmt.Sprintf("failed to run listener: %s", err)).
					SetWorker("internal_socket_listener")
			}

		}()
	}

	<-c.ctx.Done()

	cancel()
	wg.Wait()

	return nil
}

func (c *chief) runWorker(ctx Context, name WorkerName, doneCall func()) {
	defer doneCall()

	err := c.wPool.runWorkerExec(ctx, c.eventChan, name)
	if err != nil {
		c.eventChan <- ErrorEvent(err.Error()).SetWorker(name)
	}
}

func waitForSignal() {
	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)
	<-gracefulStop
}
