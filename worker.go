package uwe

import (
	"context"

	"github.com/lancer-kit/sam"
	"github.com/sirupsen/logrus"
)

// workerRO worker runtime object, hold worker instance, state and communication chanel
type workerRO struct {
	sam.StateMachine
	worker   Worker
	canceler context.CancelFunc
	eventBus chan Message
	exitCode *ExitCode
}

// Const for worker state
const (
	WStateDisabled    sam.State = "Disabled"
	WStateEnabled     sam.State = "Enabled"
	WStateInitialized sam.State = "Initialized"
	WStateRun         sam.State = "Run"
	WStateStopped     sam.State = "Stopped"
	WStateFailed      sam.State = "Failed"
)

// WorkersStates list of valid workers states
var WorkersStates = map[sam.State]struct{}{
	WStateDisabled:    {},
	WStateEnabled:     {},
	WStateInitialized: {},
	WStateRun:         {},
	WStateStopped:     {},
	WStateFailed:      {},
}

// newWorkerSM returns filled state machine of the worker lifecycle
//
// (*) -> [Disabled] -> [Enabled] -> [Initialized] -> [Run] <-> [Stopped]
//          ↑ ↑____________|  |          |  |  ↑         |
//          |_________________|__________|  |  |------|  ↓
//                            |-------------|-----> [Failed]
func newWorkerSM() sam.StateMachine {
	sm := sam.NewStateMachine()
	workerSM, err := sm.
		AddTransitions(WStateDisabled, WStateEnabled).
		AddTransitions(WStateEnabled, WStateInitialized, WStateFailed, WStateDisabled).
		AddTransitions(WStateInitialized, WStateRun, WStateFailed, WStateDisabled).
		AddTransitions(WStateRun, WStateStopped, WStateFailed).
		AddTransitions(WStateStopped, WStateRun).
		AddTransitions(WStateFailed, WStateInitialized, WStateDisabled).
		Finalize(WStateDisabled)
	if err != nil || workerSM == nil {
		logrus.Fatal("worker state machine init failed: ", err)
	}

	return workerSM.Clone()
}
