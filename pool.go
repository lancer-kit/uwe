package uwe

import (
	"errors"
	"fmt"
	"sync"

	"github.com/lancer-kit/sam"
)

// workerPool provides a mechanism to combine many workers into the one pool, manage them, and run.
type workerPool struct {
	mutex   *sync.RWMutex
	workers map[WorkerName]*workerRO
}

// initWorker initializes all present workers.
func (p *workerPool) initWorker(name WorkerName) error {
	if err := p.setState(name, WStateInitialized); err != nil {
		return err
	}

	w := p.getWorker(name)
	return w.worker.Init()
}

// setWorker adds worker into pool.
func (p *workerPool) setWorker(name WorkerName, worker Worker) error {
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

	return nil
}

// replaceWorker replaces the worker with `name` by new `worker`.
func (p *workerPool) replaceWorker(name WorkerName, worker Worker) {
	p.check()

	p.mutex.Lock()
	p.workers[name].worker = worker
	p.mutex.Unlock()
}

// runWorkerExec adds worker into pool.
func (p *workerPool) runWorkerExec(ctx Context, name WorkerName) (err error) {
	if err = p.startWorker(name); err != nil {
		return err
	}

	w := p.getWorker(name)
	if err = w.worker.Run(ctx); err != nil {
		return err
	}

	return nil
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
