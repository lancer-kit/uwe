package uwe

import "github.com/pkg/errors"

// EventLevel ...
type EventLevel string

const (
	LvlFatal EventLevel = "fatal"
	LvlError EventLevel = "error"
	LvlInfo  EventLevel = "info"
)

// Event is a message object that is used to signalize
// about Chief's internal events and processed by `EventHandlers`.
type Event struct {
	Level   EventLevel
	Worker  WorkerName
	Fields  map[string]interface{}
	Message string
}

// IsFatal returns `true` if event level is `Fatal`
func (e Event) IsFatal() bool {
	return e.Level == LvlFatal
}

// IsError returns `true` if event level is `Error`
func (e Event) IsError() bool {
	return e.Level == LvlError
}

// ToError validates event level and cast to builtin `error`.
func (e Event) ToError() error {
	if !e.IsError() && !e.IsFatal() {
		return nil
	}
	return errors.New(e.Message)
}

// SetField add to event some Key/Value.
func (e Event) SetField(key string, value interface{}) Event {
	e.Fields[key] = value
	return e
}

// SetWorker sets the provided `worker` as the event source.
func (e Event) SetWorker(name WorkerName) Event {
	e.Worker = name
	return e
}

// ErrorEvent returns new Event with `LvlError` and provided message.
func ErrorEvent(msg string) Event {
	return Event{
		Level:   LvlError,
		Message: msg,
	}
}
