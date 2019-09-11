package uwe

import "github.com/pkg/errors"

type EventLevel string

const (
	LvlFatal EventLevel = "fatal"
	LvlError EventLevel = "error"
	LvlInfo  EventLevel = "info"
)

type Event struct {
	Level   EventLevel
	Worker  WorkerName
	Fields  map[string]interface{}
	Message string
}

func (e Event) IsFatal() bool {
	return e.Level == LvlFatal
}

func (e Event) IsError() bool {
	return e.Level == LvlError
}

func (e Event) ToError() error {
	if !e.IsError() && !e.IsFatal() {
		return nil
	}
	return errors.New(e.Message)
}

func (e Event) SetField(key string, value interface{}) Event {
	e.Fields[key] = value
	return e
}

func (e Event) SetWorker(name WorkerName) Event {
	e.Worker = name
	return e
}

func ErrorEvent(msg string) Event {
	return Event{
		Level:   LvlError,
		Message: msg,
	}
}
