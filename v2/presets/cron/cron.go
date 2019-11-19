package cron

import (
	"time"

	"github.com/lancer-kit/uwe/v2"
)

type Job struct {
	period    time.Duration
	ticker    *time.Ticker
	runAction func() error
}

func NewJob(period time.Duration, runAction func() error) *Job {
	return &Job{
		period:    period,
		runAction: runAction,
	}
}

func (j *Job) Init() error {
	j.ticker = time.NewTicker(j.period)
	return nil
}

func (j *Job) Run(ctx uwe.Context) error {
	for {
		select {
		case <-j.ticker.C:
			if err := j.runAction(); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}
