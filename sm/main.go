package sm

import "fmt"

type State string

type stateObj struct {
	name  State
	prev  State
	to    map[State]struct{}
	from  map[State]struct{}
	exist bool
}

var invalidTransition = func(from, to State) error {
	return fmt.Errorf("invalid transition: %v --> %v", from, to)
}

var stateNotFound = func(name State) error {
	return fmt.Errorf("state not found: %v", name)
}

type Hook func(from, to State) (bool, error)

type HookList struct {
	before      []Hook
	after       []Hook
	beforeState map[State]Hook
	afterState  map[State]Hook
}

func (HookList) New() HookList {
	return HookList{
		before:      []Hook{},
		after:       []Hook{},
		beforeState: map[State]Hook{},
		afterState:  map[State]Hook{},
	}

}

type StateMachine struct {
	current State
	states  map[State]stateObj
	hooks   HookList
}

func NewStateMachine() StateMachine {
	return StateMachine{}.New()
}

func (StateMachine) New() StateMachine {
	return StateMachine{
		current: "",
		states:  map[State]stateObj{},
		hooks:   HookList.New(HookList{}),
	}
}

func (sm StateMachine) Clone() StateMachine {
	return sm
}

func (sm *StateMachine) State() State {
	return sm.current
}

func (sm *StateMachine) SetState(state State) {
	stateObj := sm.getState(state)
	stateObj.exist = true
	stateObj.prev = sm.current
	sm.states[state] = stateObj

	sm.current = stateObj.name
}

func (sm *StateMachine) getState(name State) stateObj {
	state, ok := sm.states[name]
	if !ok {
		state = stateObj{
			name:  name,
			exist: false,
			to:    map[State]struct{}{},
			from:  map[State]struct{}{},
		}
	}
	return state
}

func (sm *StateMachine) AddTransitions(from State, to ...State) error {
	for _, name := range to {
		err := sm.AddTransition(from, name)
		if err != nil {
			return err
		}

	}
	return nil
}

func (sm *StateMachine) AddTransition(from, to State) error {
	if from == to {
		return invalidTransition(from, to)
	}

	fromState := sm.getState(from)
	fromState.exist = true
	fromState.to[to] = struct{}{}
	sm.states[from] = fromState

	toState := sm.getState(to)
	toState.exist = true
	toState.from[from] = struct{}{}
	sm.states[to] = toState

	return nil
}

func (sm *StateMachine) DoTransition(name State) error {
	newState, ok := sm.states[name]
	if !ok {
		return stateNotFound(name)
	}
	if sm.current == name {
		return nil
	}

	state := sm.states[sm.current]
	_, ok = state.to[name]
	if !ok {
		return invalidTransition(sm.current, name)
	}

	newState.prev = sm.current
	sm.states[name] = newState
	sm.current = name
	return nil
}

func (sm *StateMachine) GoBack() error {
	current := sm.getState(sm.current)
	if !current.exist {
		return stateNotFound(sm.current)
	}

	_, ok := current.from[current.prev]
	if !ok {
		return invalidTransition(sm.current, current.prev)
	}

	prev := sm.getState(current.prev)
	if !prev.exist {
		return stateNotFound(current.prev)
	}

	sm.current = prev.name
	return nil
}
