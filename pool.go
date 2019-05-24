package uwe

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

var (
	ErrWorkerNotInitialized = errors.New("worker not initialized")
)

// WorkerPool is
type WorkerPool struct {
	//workers map[string]Worker
	//states  map[string]WorkerState

	rw   sync.RWMutex
	pool map[string]*Worker
}

//GetWorker -  get Worker interface by name
func (pool *WorkerPool) GetWorker(name string) Worker {
	pool.rw.RLock()
	defer pool.rw.RUnlock()

	if wk, ok := pool.workers[name]; ok {
		return wk
	}
	return nil
}

// GetState returns current state for workers with the specified `name`.
func (pool *WorkerPool) GetState(name string) WorkerState {
	pool.rw.RLock()
	defer pool.rw.RUnlock()
	if wk, ok := pool.states[name]; ok {
		return wk
	}
	return WorkerDisabled
}

// GetWorkersStates returns current state of all workers.
func (pool *WorkerPool) GetWorkersStates() map[string]WorkerState {
	return pool.states
}

// IsEnabled checks is enable worker with passed `name`.
func (pool *WorkerPool) IsEnabled(name string) bool {
	if pool.states == nil {
		return false
	}

	state := pool.GetState(name)
	return state >= WorkerEnabled
}

// IsAlive checks is active worker with passed `name`.
func (pool *WorkerPool) IsAlive(name string) bool {
	state := pool.GetState(name)
	return state == WorkerRun
}

// DisableWorker sets state `WorkerDisabled` for workers with the specified `name`.
func (pool *WorkerPool) DisableWorker(name string) {
	pool.SetState(name, WorkerDisabled)
}

// EnableWorker sets state `WorkerEnabled` for workers with the specified `name`.
func (pool *WorkerPool) EnableWorker(name string) {
	if s := pool.states[name]; s != WorkerPresent {
		pool.SetState(name, WorkerWrongStateChange)
		return
	}
	pool.SetState(name, WorkerEnabled)
}

// InitWorker initializes all present workers.
func (pool *WorkerPool) InitWorker(name string, ctx context.Context) {
	if s := pool.GetState(name); s < WorkerEnabled {
		return
	}
	w := pool.GetWorker(name)
	w.Init(ctx)
	pool.ReplaceWorker(name, w)
	pool.SetState(name, WorkerRun)
}

// StartWorker sets state `WorkerEnabled` for workers with the specified `name`.
func (pool *WorkerPool) StartWorker(name string) {
	if s := pool.GetState(name); s != WorkerStopped && s != WorkerInitialized && s != WorkerFailed {
		pool.SetState(name, WorkerWrongStateChange)
		return
	}
	pool.SetState(name, WorkerRun)
}

// StopWorker sets state `WorkerStopped` for workers with the specified `name`.
func (pool *WorkerPool) StopWorker(name string) {
	if s := pool.GetState(name); s != WorkerRun && s != WorkerFailed {
		pool.SetState(name, WorkerWrongStateChange)
		return
	}
	pool.SetState(name, WorkerStopped)
}

// FailWorker sets state `WorkerFailed` for workers with the specified `name`.
func (pool *WorkerPool) FailWorker(name string) {
	pool.SetState(name, WorkerFailed)
}

// SetState updates state of specified worker.
func (pool *WorkerPool) SetState(name string, state WorkerState) {
	pool.check()

	pool.rw.Lock()
	pool.states[name] = state
	pool.rw.Unlock()
}

// SetWorker adds worker into pool.
func (pool *WorkerPool) SetWorker(name string, worker Worker) {
	pool.check()

	pool.rw.Lock()
	pool.workers[name] = worker
	pool.states[name] = WorkerPresent
	pool.rw.Unlock()
}

func (pool *WorkerPool) ReplaceWorker(name string, worker Worker) {
	pool.check()

	pool.rw.Lock()
	pool.workers[name] = worker
	pool.rw.Unlock()
}

// RunWorkerExec adds worker into pool.
func (pool *WorkerPool) RunWorkerExec(name string) (err error) {
	defer func() {
		rec := recover()
		if rec == nil {
			return
		}
		pool.FailWorker(name)

		e, ok := rec.(error)
		if !ok {
			e = fmt.Errorf("%v", rec)
		}
		err = errors.WithStack(e)
	}()

	if s := pool.GetState(name); s != WorkerInitialized {
		return ErrWorkerNotInitialized
	}

	pool.StartWorker(name)
	extCode := pool.workers[name].Run()
	switch extCode {
	case ExitCodeOk:
		pool.StopWorker(name)
	case ExitCodeFailed:
		pool.FailWorker(name)
	case ExitCodeInterrupted:
		pool.FailWorker(name)
	}

	return
}

func (pool *WorkerPool) check() {
	pool.rw.Lock()

	if pool.states == nil {
		pool.states = make(map[string]WorkerState)
	}
	if pool.workers == nil {
		pool.workers = make(map[string]Worker)
	}

	pool.rw.Unlock()
}
