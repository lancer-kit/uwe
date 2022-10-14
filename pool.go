package uwe

import (
	"errors"
	"fmt"
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
	p.check()
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
			p.workers[name].restartMode = o
		}
	}

	return nil
}

// replaceWorker replaces the worker with `name` by new `worker`.
func (p *workerPool) replaceWorker(name WorkerName, worker Worker) {
	p.check()

	p.mutex.Lock()
	p.workers[name].worker = worker
	p.mutex.Unlock()
}

func (p *workerPool) workersList() []WorkerName {
	list := make([]WorkerName, 0, len(p.workers))
	for name := range p.workers {
		list = append(list, name)
	}

	return list
}

// runWorkerExec adds worker into pool.
func (p *workerPool) runWorkerExec(ctx Context, name WorkerName) (err error) {
	w := p.getWorker(name)

InitPoint:
	if e := p.setState(name, WStateInitialized); e != nil {
		return err
	}

	if e := w.worker.Init(); e != nil {
		return e
	}

RunPoint:
	if err = p.startWorker(name); err != nil {
		return err
	}
	var runClosure = func() (panicked bool, e error) {
		defer func() {
			r := recover()
			if r != nil {
				panicked = true
				e = fmt.Errorf("%v", e)
			}
		}()

		e = w.worker.Run(ctx)
		return
	}

	panicked, err := runClosure()
	if panicked || err != nil {
		if e := p.setState(name, WStateFailed); e != nil {
			return e
		}
	}

	switch {
	case (panicked && !w.restartMode.Is(RestartOnFail)) ||
		(!panicked && !w.restartMode.Is(RestartOnError)):
		return err

	case (panicked && w.restartMode.Is(RestartOnFail)) ||
		(!panicked && w.restartMode.Is(RestartOnError) && err != nil):

		if stopIntiated(ctx) {
			return err
		}

		if w.restartMode.Is(RestartWithReinit) {
			goto InitPoint
		} else {
			goto RunPoint
		}
	}

	return nil
}

func stopIntiated(ctx Context) bool {
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

// getState returns current state for workers with the specified `name`.
func (p *workerPool) getState(name WorkerName) sam.State {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if wk, ok := p.workers[name]; ok {
		return wk.State()
	}

	return WStateNotExists
}

// isRun checks is active worker with passed `name`.
func (p *workerPool) isRun(name WorkerName) bool {
	state := p.getState(name)
	return state == WStateRun
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
	p.check()

	p.mutex.Lock()
	_, ok := p.workers[name]
	if !ok {
		return errors.New(string(name) + ": not exist")
	}

	err := p.workers[name].GoTo(state)
	p.mutex.Unlock()
	return fmt.Errorf("%s: %s", string(name), err)
}

func (p *workerPool) check() {
	p.mutex.Lock()

	if p.workers == nil {
		p.workers = make(map[WorkerName]*workerRO)
	}

	p.mutex.Unlock()
}
