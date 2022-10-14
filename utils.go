package uwe

import (
	"errors"
	"log"
	"reflect"
)

// WorkerExistRule is a custom validation rule for the validation libs.
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
	return &WorkerExistRule{message: message}
}

// STDLogEventHandler returns a callback that handles internal `Chief` events and logs its.
func STDLogEventHandler() EventHandler {
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

		log.Printf("%s: %s %s\n", level, event.Message, event.FormatFields())
	}
}
