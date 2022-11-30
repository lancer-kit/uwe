package presets

import (
	"time"

	"github.com/lancer-kit/uwe/v3"
)

// Job is a primitive worker who performs an `action` callback with a given period.
type Job struct {
	period time.Duration
	ticker *time.Ticker
	action func() error
}

// NewJob create new job with given `period`.
func NewJob(period time.Duration, action func() error) *Job {
	return &Job{
		period: period,
		action: action,
	}
}

// Init is a method to satisfy `uwe.Worker` interface.
func (j *Job) Init() error { return nil }

// Run executes the `action` callback with the specified `period` until a stop signal is received.
func (j *Job) Run(ctx uwe.Context) error {
	j.ticker = time.NewTicker(j.period)
	for {
		select {
		case <-j.ticker.C:
			if err := j.action(); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}
