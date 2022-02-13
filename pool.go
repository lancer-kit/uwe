package uwe

import (
	"sync"

	"github.com/lancer-kit/sam"
	"github.com/pkg/errors"
)

// WorkerPool provides a mechanism to combine many workers into the one pool, manage them, and run.
type WorkerPool struct {
	mutex   *sync.RWMutex
	workers map[WorkerName]*workerRO
}

// InitWorker initializes all present workers.
func (p *WorkerPool) InitWorker(name WorkerName) error {
	if err := p.SetState(name, wStateInitialized); err != nil {
		return err
	}

	w := p.getWorker(name)
	return w.worker.Init()
}

// SetWorker adds worker into pool.
func (p *WorkerPool) SetWorker(name WorkerName, worker Worker) error {
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

// ReplaceWorker replaces the worker with `name` by new `worker`.
func (p *WorkerPool) ReplaceWorker(name WorkerName, worker Worker) {
	p.check()

	p.mutex.Lock()
	p.workers[name].worker = worker
	p.mutex.Unlock()
}

// RunWorkerExec adds worker into pool.
func (p *WorkerPool) RunWorkerExec(ctx Context, name WorkerName) (err error) {
	if err = p.StartWorker(name); err != nil {
		return err
	}

	w := p.getWorker(name)
	if err = w.worker.Run(ctx); err != nil {
		return err
	}

	return nil
}

// getWorker - get WorkerRO by name
func (p *WorkerPool) getWorker(name WorkerName) *workerRO {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if wk, ok := p.workers[name]; ok {
		return wk
	}
	return nil
}

// ============ Methods related to workers status management ============

// GetWorkersStates returns current state of all workers.
func (p *WorkerPool) GetWorkersStates() map[WorkerName]sam.State {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	r := map[WorkerName]sam.State{}
	for name, worker := range p.workers {
		r[name] = worker.State()
	}
	return r
}

// GetState returns current state for workers with the specified `name`.
func (p *WorkerPool) GetState(name WorkerName) sam.State {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if wk, ok := p.workers[name]; ok {
		return wk.State()
	}

	return wStateNotExists
}

// IsRun checks is active worker with passed `name`.
func (p *WorkerPool) IsRun(name WorkerName) bool {
	state := p.GetState(name)
	return state == wStateRun
}

// StartWorker sets state `WorkerEnabled` for workers with the specified `name`.
func (p *WorkerPool) StartWorker(name WorkerName) error {
	return p.SetState(name, wStateRun)
}

// StopWorker sets state `WorkerStopped` for workers with the specified `name`.
func (p *WorkerPool) StopWorker(name WorkerName) error {
	return p.SetState(name, wStateStopped)
}

// FailWorker sets state `WorkerFailed` for workers with the specified `name`.
func (p *WorkerPool) FailWorker(name WorkerName) error {
	return p.SetState(name, wStateFailed)
}

// SetState updates state of specified worker.
func (p *WorkerPool) SetState(name WorkerName, state sam.State) error {
	p.check()

	p.mutex.Lock()
	_, ok := p.workers[name]
	if !ok {
		return errors.New(string(name) + ": not exist")
	}

	err := p.workers[name].GoTo(state)
	p.mutex.Unlock()
	return errors.Wrap(err, string(name))
}

func (p *WorkerPool) check() {
	p.mutex.Lock()

	if p.workers == nil {
		p.workers = make(map[WorkerName]*workerRO)
	}

	p.mutex.Unlock()
}
