package uwe

import (
	"context"

	"github.com/lancer-kit/sam"
	"github.com/pkg/errors"
)

type WorkerName string

// Worker is an interface for async workers
// which launches and manages by the `Chief`.
type Worker interface {
	// Init initializes some state of the worker that required interaction with outer context,
	// for example, initialize some connectors. In many cases this method is optional,
	// so it can be implemented as empty: `func (*W) Init() error { return nil }`.
	Init() error

	// Run  starts the `Worker` instance execution. The context will provide a signal
	// when a worker must stop through the `ctx.Done()`.
	Run(ctx Context) error
}

// workerRO worker runtime object, hold worker instance, state and communication chanel
type workerRO struct {
	sam.StateMachine
	worker   Worker
	canceler context.CancelFunc
}

const (
	WStateNotExists   sam.State = "NotExists"
	WStateNew         sam.State = "New"
	WStateInitialized sam.State = "Initialized"
	WStateRun         sam.State = "Run"
	WStateStopped     sam.State = "Stopped"
	WStateFailed      sam.State = "Failed"
)

// newWorkerSM returns filled state machine of the worker lifecycle
//
// (*) -> [New] -> [Initialized] -> [Run] -> [Stopped]
//          |             |           |
//          |             |           ↓
//          |-------------|------> [Failed]
func newWorkerSM() (sam.StateMachine, error) {
	sm := sam.NewStateMachine()
	s := &sm

	workerSM, err := s.
		AddTransitions(WStateNew, WStateInitialized, WStateFailed).
		AddTransitions(WStateInitialized, WStateRun, WStateFailed).
		AddTransitions(WStateRun, WStateStopped, WStateFailed).
		Finalize(WStateStopped)
	if err != nil || workerSM == nil {
		return sm, errors.Wrap(err, "worker state machine init failed: ")
	}

	if err = workerSM.SetState(WStateNew); err != nil {
		return sm, errors.Wrap(err, "failed to set state new")
	}

	return workerSM.Clone(), nil
}
