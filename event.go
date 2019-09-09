package uwe

import "github.com/pkg/errors"

type EventLevel string

const (
	LvlError EventLevel = "error"
	LvlInfo  EventLevel = "info"
)

type Event struct {
	Level   EventLevel
	Worker  WorkerName
	Fields  map[string]interface{}
	Message string
}

func (e Event) IsError() bool {
	return e.Level == LvlError
}

func (e Event) ToError() error {
	if !e.IsError() {
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
