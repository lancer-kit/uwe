package uwe

import "errors"

// WorkerExistRule implements Rule interface for validation
type WorkerExistRule struct {
	message          string
	AvailableWorkers map[WorkerName]struct{}
}

// Validate checks that service exist on the system
func (r *WorkerExistRule) Validate(value interface{}) error {
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
