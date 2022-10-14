package uwe

import (
	"context"
	"fmt"

	"github.com/sheb-gregor/sam"
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

	worker      Worker
	restartMode RestartOption
	canceler    context.CancelFunc
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
//
//	|             |           |
//	|             |           ↓
//	|--------------------> [Failed]
//							(from [Failed] state can get back
//							 to [Initialized] or to [Run])
func newWorkerSM() (sam.StateMachine, error) {
	workerSM, err := sam.NewStateMachine().
		AddTransitions(WStateNew, WStateInitialized, WStateFailed).
		AddTransitions(WStateInitialized, WStateRun, WStateFailed).
		AddTransitions(WStateRun, WStateStopped, WStateFailed).
		AddTransitions(WStateFailed, WStateInitialized, WStateRun).
		Finalize(WStateStopped)
	if err != nil || workerSM == nil {
		return sam.StateMachine{}, fmt.Errorf("worker state machine init failed: %s", err)
	}

	if err = workerSM.SetState(WStateNew); err != nil {
		return sam.StateMachine{}, fmt.Errorf("failed to set state new: %s", err)
	}

	return workerSM.Clone(), nil
}
