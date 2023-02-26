package uwe

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/sheb-gregor/sam"
)

// workerPool provides a mechanism to combine many workers into the one pool, manage them, and run.
type workerPool struct {
	mutex   sync.RWMutex
	workers map[WorkerName]*workerRO
}

// setWorker adds worker into pool.
func (p *workerPool) setWorker(name WorkerName, worker Worker, opts []WorkerOpts) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	sm, err := newWorkerSM()
	if err != nil {
		return err
	}

	p.workers[name] = &workerRO{
		StateMachine: sm,
		worker:       worker,
		canceler:     nil,
	}

	for _, opt := range opts {
		// nolint: gocritic
		switch o := opt.(type) {
		case RestartOption:
			p.workers[name].restartMode |= o
		}
	}

	if p.workers[name].restartMode == 0 {
		p.workers[name].restartMode = NoRestart
	}

	return nil
}

func (p *workerPool) workersList() []WorkerName {
	list := make([]WorkerName, 0, len(p.workers))
	for name := range p.workers {
		list = append(list, name)
	}

	return list
}

// runWorkerExec adds worker into pool.
func (p *workerPool) runWorkerExec(ctx Context, eventChan chan<- Event, name WorkerName) error {
	w := p.getWorker(name)

InitPoint:
	if err := p.setState(name, WStateInitialized); err != nil {
		return err
	}

	worker, ok := w.worker.(WorkerWithInit)
	if ok {
		if err := worker.Init(); err != nil {
			eventChan <- Event{
				Level: LvlFatal, Worker: name,
				Message: "Worker can not be initialized due to an error",
				Fields:  map[string]interface{}{"error": err.Error()},
			}

			if w.restartMode == StopAppOnFail {
				msg := fmt.Sprintf("execution cannot be continued due to a failed worker(%s)", name)
				eventChan <- Event{Level: LvlFatal, Worker: name, Message: msg}
				panic(msg) // TODO: tbd, probably it is better to replace panic by os.Exit(1)
			}

			return err
		}
		eventChan <- Event{
			Level: LvlInfo, Worker: name,
			Message: "Worker is initialized",
		}
	}

RunPoint:
	if err := p.startWorker(name); err != nil {
		return err
	}
	eventChan <- Event{
		Level: LvlInfo, Worker: name,
		Message: "Starting worker",
	}

	var runClosure = func() (panicked bool, e error) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}

			var ok bool

			panicked = true
			e, ok = r.(error)
			if !ok {
				e = fmt.Errorf("%v", r)
			}

			eventChan <- Event{
				Level: LvlError, Worker: name,
				Message: "Worker failed with panic",
				Fields: map[string]interface{}{
					"error": e.Error(),
					"stack": string(debug.Stack()),
				},
			}
		}()

		e = w.worker.Run(ctx)
		if e != nil {
			eventChan <- Event{
				Level: LvlError, Worker: name,
				Message: "Worker ended execution with error",
				Fields:  map[string]interface{}{"error": e.Error()},
			}
		}
		return
	}

	eventChan <- Event{
		Level: LvlInfo, Worker: name,
		Message: "Run worker",
	}

	panicked, err := runClosure()
	if !panicked && err == nil {
		eventChan <- Event{
			Level: LvlInfo, Worker: name,
			Message: "Worker ended execution",
		}
		return p.stopWorker(name)
	}

	if e := p.failWorker(name); e != nil {
		return e
	}

	switch {
	case w.restartMode == StopAppOnFail:
		msg := fmt.Sprintf("execution cannot be continued due to a failed worker(%s)", name)
		eventChan <- Event{Level: LvlFatal, Worker: name, Message: msg}
		panic(msg) // TODO: tbd, probably it is better to replace panic by os.Exit(1)

	case (panicked && !w.restartMode.Is(RestartOnFail)) ||
		(!panicked && !w.restartMode.Is(RestartOnError)):
		return err

	case (panicked && w.restartMode.Is(RestartOnFail)) ||
		(!panicked && w.restartMode.Is(RestartOnError) && err != nil):

		if stopInitiated(ctx) {
			return err
		}

		if w.restartMode.Is(RestartWithReInit) {
			// todo: log -- init worker again
			eventChan <- Event{
				Level: LvlInfo, Worker: name,
				Message: "Worker will be re-initialized and restarted",
			}
			goto InitPoint
		} else {
			eventChan <- Event{
				Level: LvlInfo, Worker: name,
				Message: "Worker will be restarted",
			}
			goto RunPoint
		}
	}

	return nil
}

func stopInitiated(ctx Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// getWorker - get WorkerRO by name
func (p *workerPool) getWorker(name WorkerName) *workerRO {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if wk, ok := p.workers[name]; ok {
		return wk
	}
	return nil
}

// ============ Methods related to workers status management ============

// getWorkersStates returns current state of all workers.
func (p *workerPool) getWorkersStates() map[WorkerName]sam.State {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	r := map[WorkerName]sam.State{}
	for name, worker := range p.workers {
		r[name] = worker.State()
	}
	return r
}

// startWorker sets state `WorkerEnabled` for workers with the specified `name`.
func (p *workerPool) startWorker(name WorkerName) error {
	return p.setState(name, WStateRun)
}

// stopWorker sets state `WorkerStopped` for workers with the specified `name`.
func (p *workerPool) stopWorker(name WorkerName) error {
	return p.setState(name, WStateStopped)
}

// failWorker sets state `WorkerFailed` for workers with the specified `name`.
func (p *workerPool) failWorker(name WorkerName) error {
	return p.setState(name, WStateFailed)
}

// setState updates state of specified worker.
func (p *workerPool) setState(name WorkerName, state sam.State) error {
	p.mutex.Lock()
	_, ok := p.workers[name]
	if !ok {
		return errors.New(string(name) + ": not exist")
	}

	err := p.workers[name].GoTo(state)
	p.mutex.Unlock()
	if err != nil {
		return fmt.Errorf("%s: %w", string(name), err)
	}
	return nil
}
