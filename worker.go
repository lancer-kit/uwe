package uwe

import (
	"context"

	"github.com/lancer-kit/uwe/sm"
)

// wro worker runtime object, hold worker instance, state and communication chanel
type wro struct {
	worker   Worker
	state    sm.StateMachine
	canceler context.CancelFunc
	eventBus chan WorkerEvent
	exitCode *ExitCode
}

type WorkerEvent struct {
	UID  int64
	Data interface{}
}

func (pool *WorkerPool) getWRO(name string) (*wro, bool) {
	workerRO, ok := pool.pool[name]
	if !ok {
		return nil, ok
	}

	return workerRO, ok
}

const (
	WStateDisabled    sm.State = "Disabled"
	WStateEnabled     sm.State = "Enabled"
	WStateInitialized sm.State = "Initialized"
	WStateRun         sm.State = "Run"
	WStateStopped     sm.State = "Stopped"
	WStateFailed      sm.State = "Failed"
)

// newWorkerSM returns filled state machine of worker lifecycle
//
// (*) -> [Disabled] -> [Enabled] -> [Initialized] -> [Run] <-> [Stopped]
//          ↑ ↑____________|  |          |  |  ↑         |
//          |_________________|__________|  |  |------|  ↓
//                            |-------------|-----> [Failed]

func newWorkerSM() sm.StateMachine {
	workerSM := sm.NewStateMachine()
	_ = workerSM.AddTransitions(WStateDisabled, WStateEnabled)
	_ = workerSM.AddTransitions(WStateEnabled, WStateInitialized, WStateFailed, WStateDisabled)
	_ = workerSM.AddTransitions(WStateInitialized, WStateRun, WStateFailed, WStateDisabled)
	_ = workerSM.AddTransitions(WStateRun, WStateStopped, WStateFailed)
	_ = workerSM.AddTransitions(WStateStopped, WStateRun)
	_ = workerSM.AddTransitions(WStateFailed, WStateInitialized, WStateDisabled)
	workerSM.SetState(WStateDisabled)
	return workerSM
}
