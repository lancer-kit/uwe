package uwe

import (
	"context"
	"fmt"
	"sync"

	"github.com/lancer-kit/sam"
	"github.com/pkg/errors"
)

var (
	//ErrWorkerNotExist custom error for not-existing worker
	ErrWorkerNotExist = func(name WorkerName) error {
		return fmt.Errorf("%s: not exist", name)
	}
)

// WorkerPool is
type WorkerPool struct {
	rw      sync.RWMutex
	workers map[WorkerName]*workerRO
}

// getWorker - get WorkerRO by name
func (pool *WorkerPool) getWorker(name WorkerName) *workerRO {
	pool.rw.RLock()
	defer pool.rw.RUnlock()
	if wk, ok := pool.workers[name]; ok {
		return wk
	}
	return nil
}

// InitWorker initializes all present workers.
func (pool *WorkerPool) InitWorker(ctx context.Context, name WorkerName) error {
	if err := pool.SetState(name, WStateInitialized); err != nil {
		return err
	}

	w := pool.getWorker(name)
	initObj := w.worker.Init(ctx)
	pool.ReplaceWorker(name, initObj)
	return nil
}

// SetWorker adds worker into pool.
func (pool *WorkerPool) SetWorker(name WorkerName, worker Worker) {
	pool.check()

	pool.rw.Lock()
	pool.workers[name] = &workerRO{
		StateMachine: newWorkerSM(),
		worker:       worker,
		canceler:     nil,
		eventBus:     nil,
		exitCode:     nil,
	}
	pool.rw.Unlock()
}

// ReplaceWorker replace worker in the pool
func (pool *WorkerPool) ReplaceWorker(name WorkerName, worker Worker) {
	pool.check()

	pool.rw.Lock()
	pool.workers[name].worker = worker
	pool.rw.Unlock()
}

// RunWorkerExec adds worker into pool.
func (pool *WorkerPool) RunWorkerExec(name WorkerName, ctx WContext) (err error) {
	defer func() {
		rec := recover()
		if rec == nil {
			return
		}

		e, ok := rec.(error)
		if !ok {
			e = fmt.Errorf("%v", rec)
		}
		err = errors.WithStack(e)

		if er := pool.FailWorker(name); er != nil {
			err = errors.Wrap(err, er.Error())
		}
	}()

	if err = pool.StartWorker(name); err != nil {
		return err
	}

	// todo:
	w := pool.getWorker(name)
	extCode := w.worker.Run(ctx)
	switch extCode {
	case ExitCodeOk:
		return pool.StopWorker(name)
	case ExitCodeFailed:
		return pool.FailWorker(name)
	case ExitCodeInterrupted:
		return pool.FailWorker(name)
	case ExitReinitReq:
		// todo
	}

	return
}

// ============ Methods relating to the workers states ============

// GetWorkersStates returns current state of all workers.
func (pool *WorkerPool) GetWorkersStates() map[WorkerName]sam.State {
	pool.rw.RLock()
	defer pool.rw.RUnlock()
	r := map[WorkerName]sam.State{}
	for name, worker := range pool.workers {
		r[name] = worker.State()
	}
	return r
}

// GetState returns current state for workers with the specified `name`.
func (pool *WorkerPool) GetState(name WorkerName) sam.State {
	pool.rw.RLock()
	defer pool.rw.RUnlock()
	if wk, ok := pool.workers[name]; ok {
		return wk.State()
	}

	return WStateDisabled
}

// IsEnabled checks is enabled worker with passed `name`.
func (pool *WorkerPool) IsEnabled(name WorkerName) bool {
	if pool.workers == nil {
		return false
	}

	return pool.GetState(name) != WStateDisabled
}

// IsDisabled checks is disabled worker with passed `name`.
func (pool *WorkerPool) IsDisabled(name WorkerName) bool {
	if pool.workers == nil {
		return false
	}

	return pool.GetState(name) == WStateDisabled
}

// IsRun checks is active worker with passed `name`.
func (pool *WorkerPool) IsRun(name WorkerName) bool {
	state := pool.GetState(name)
	return state == WStateRun
}

// DisableWorker sets state `WorkerDisabled` for workers with the specified `name`.
func (pool *WorkerPool) DisableWorker(name WorkerName) error {
	return pool.SetState(name, WStateDisabled)
}

// EnableWorker sets state `WorkerEnabled` for workers with the specified `name`.
func (pool *WorkerPool) EnableWorker(name WorkerName) error {
	return pool.SetState(name, WStateEnabled)
}

// StartWorker sets state `WorkerEnabled` for workers with the specified `name`.
func (pool *WorkerPool) StartWorker(name WorkerName) error {
	return pool.SetState(name, WStateRun)
}

// StopWorker sets state `WorkerStopped` for workers with the specified `name`.
func (pool *WorkerPool) StopWorker(name WorkerName) error {
	return pool.SetState(name, WStateStopped)
}

// FailWorker sets state `WorkerFailed` for workers with the specified `name`.
func (pool *WorkerPool) FailWorker(name WorkerName) error {
	return pool.SetState(name, WStateFailed)
}

// SetState updates state of specified worker.
func (pool *WorkerPool) SetState(name WorkerName, state sam.State) error {
	pool.check()

	pool.rw.Lock()
	_, ok := pool.workers[name]
	if !ok {
		return ErrWorkerNotExist(name)
	}

	err := pool.workers[name].GoTo(state)
	pool.rw.Unlock()
	return errors.Wrap(err, string(name))
}

func (pool *WorkerPool) check() {
	pool.rw.Lock()

	if pool.workers == nil {
		pool.workers = make(map[WorkerName]*workerRO)
	}

	pool.rw.Unlock()
}
