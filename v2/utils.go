package uwe

import (
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/sirupsen/logrus"
)

// WorkerExistRule is a custom validation rule for the `ozzo-validation` package.
type WorkerExistRule struct {
	message          string
	AvailableWorkers map[WorkerName]struct{}
}

// Validate checks that service exist on the system
func (r *WorkerExistRule) Validate(value interface{}) error {
	if value == nil || reflect.ValueOf(value).IsNil() {
		return nil
	}
	arr, ok := value.([]string)
	if !ok {
		return errors.New("can't convert list of workers to []string")
	}
	for _, v := range arr {
		if _, ok := r.AvailableWorkers[WorkerName(v)]; !ok {
			return errors.New("invalid service name " + v)
		}
	}
	return nil
}

// Error sets the error message for the rule.
func (r *WorkerExistRule) Error(message string) *WorkerExistRule {
	return &WorkerExistRule{
		message: message,
	}
}

// LogrusEventHandler returns default `EventHandler` that can be used for `Chief.SetEventHandler(...)`.
func LogrusEventHandler(entry *logrus.Entry) EventHandler {
	return func(event Event) {
		var level logrus.Level
		switch event.Level {
		case LvlFatal, LvlError:
			level = logrus.ErrorLevel
		case LvlInfo:
			level = logrus.InfoLevel
		default:
			level = logrus.WarnLevel
		}

		entry.WithFields(event.Fields).Log(level, event.Message)
	}
}

// STDLogEventHandler returns a callback that handles internal `Chief` events and logs its.
func STDLogEventHandler() func(event Event) {
	return func(event Event) {
		var level string
		switch event.Level {
		case LvlFatal, LvlError:
			level = "ERROR"
		case LvlInfo:
			level = "INFO"
		default:
			level = "WARN"
		}

		log.Println(fmt.Sprintf("%s: %s %+v", level, event.Message, event.Fields))
	}
}
