package sm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const nilStep = State("")

func TestNewStateMachine(t *testing.T) {
	sm := NewStateMachine()
	assert.Equal(t, nilStep, sm.current)
	assert.NotNil(t, sm.states)
	assert.NotNil(t, sm.hooks)
}

func TestStateMachine_State(t *testing.T) {
	sm := NewStateMachine()

	assert.Equal(t, nilStep, sm.current)
	assert.NotNil(t, sm.states)
	assert.NotNil(t, sm.hooks)

	sm.current = State("test")
	assert.Equal(t, sm.current, sm.State())
}

func TestStateMachine_getState(t *testing.T) {
	sm := NewStateMachine()
	name := State("test")

	state := sm.getState(name)
	assert.Equal(t, name, state.name)
	assert.Equal(t, nilStep, state.prev)
	assert.NotNil(t, state.from)
	assert.NotNil(t, state.to)
	assert.False(t, state.exist)
}

func TestStateMachine_SetState(t *testing.T) {
	sm := NewStateMachine()
	name := State("test")

	sm.SetState(name)
	assert.Equal(t, name, sm.State())

	state := sm.getState(name)
	assert.Equal(t, name, state.name)
	assert.NotNil(t, state.from)
	assert.NotNil(t, state.to)
	assert.True(t, state.exist)
}

func TestStateMachine_AddTransition(t *testing.T) {
	sm := NewStateMachine()
	from := State("from")
	to := State("to")

	err := sm.AddTransition(from, from)
	assert.Error(t, err)
	assert.Equal(t, invalidTransition(from, from).Error(), err.Error())

	err = sm.AddTransition(from, to)
	assert.NoError(t, err)

	state := sm.getState(from)
	assert.Equal(t, from, state.name)
	assert.True(t, state.exist)
	assert.NotNil(t, state.from)
	assert.NotNil(t, state.to)
	_, ok := state.to[to]
	assert.True(t, ok)

	state = sm.getState(to)
	assert.Equal(t, to, state.name)
	assert.True(t, state.exist)
	assert.NotNil(t, state.from)
	assert.NotNil(t, state.to)
	_, ok = state.from[from]
	assert.True(t, ok)
}

func TestStateMachine_AddTransitions(t *testing.T) {
	sm := NewStateMachine()
	from := State("from")
	tos := []State{"to_1", "to_2", "to_3"}
	err := sm.AddTransitions(from, tos...)
	assert.NoError(t, err)

	state := sm.getState(from)
	assert.Equal(t, from, state.name)
	assert.True(t, state.exist)
	assert.NotNil(t, state.from)
	assert.NotNil(t, state.to)

	for _, to := range tos {
		_, ok := state.to[to]
		assert.True(t, ok)

		stateTo := sm.getState(to)
		assert.Equal(t, to, stateTo.name)
		assert.True(t, stateTo.exist)
		assert.NotNil(t, stateTo.from)
		assert.NotNil(t, stateTo.to)
		_, ok = stateTo.from[from]
		assert.True(t, ok)
	}

}

func TestStateMachine_DoTransition(t *testing.T) {
	sm := NewStateMachine()
	from := State("from")
	tos := []State{"to_1", "to_2", "to_3"}
	err := sm.AddTransitions(from, tos...)
	assert.NoError(t, err)
	sm.SetState(from)

	err = sm.DoTransition(State("universe"))
	assert.Error(t, err)
	assert.Equal(t, stateNotFound(State("universe")).Error(), err.Error())
	assert.Equal(t, from, sm.State())

	err = sm.DoTransition(from)
	assert.NoError(t, err)
	assert.Equal(t, from, sm.State())

	clone := sm.Clone()
	{
		err = clone.DoTransition(tos[1])
		assert.NoError(t, err)
		assert.Equal(t, tos[1], clone.State())
	}

	err = sm.AddTransitions(tos[0], tos[2])
	assert.NoError(t, err)
	err = sm.AddTransitions(tos[1], tos[0], tos[2])
	assert.NoError(t, err)

	{
		clone = sm.Clone()
		err = clone.DoTransition(tos[0])
		assert.NoError(t, err)
		assert.Equal(t, tos[0], clone.State())

		err = clone.DoTransition(tos[2])
		assert.NoError(t, err)
		assert.Equal(t, tos[2], clone.State())
	}

	err = sm.DoTransition(tos[1])
	assert.NoError(t, err)
	assert.Equal(t, tos[1], sm.State())

	clone = sm.Clone()
	err = clone.DoTransition(tos[0])
	assert.NoError(t, err)
	assert.Equal(t, tos[0], clone.State())

	clone = sm.Clone()
	err = clone.DoTransition(tos[2])
	assert.NoError(t, err)
	assert.Equal(t, tos[2], clone.State())
}

func TestStateMachine_GoBack(t *testing.T) {
	sm := NewStateMachine()
	from := State("from")
	tos := []State{"to_1", "to_2", "to_3"}

	err := sm.AddTransitions(from, tos[0])
	assert.NoError(t, err)

	err = sm.AddTransitions(tos[0], tos[1])
	assert.NoError(t, err)

	err = sm.GoBack()
	assert.Error(t, err)
	assert.Equal(t, stateNotFound(State("")).Error(), err.Error())

	sm.SetState(from)

	err = sm.GoBack()
	assert.Error(t, err)
	assert.Equal(t, invalidTransition(from, State("")).Error(), err.Error())

	err = sm.DoTransition(tos[0])
	assert.NoError(t, err)
	assert.Equal(t, tos[0], sm.State())

	err = sm.DoTransition(tos[1])
	assert.NoError(t, err)
	assert.Equal(t, tos[1], sm.State())

	err = sm.GoBack()
	assert.NoError(t, err)
	assert.Equal(t, tos[0], sm.State())

	err = sm.GoBack()
	assert.NoError(t, err)
	assert.Equal(t, from, sm.State())
}
