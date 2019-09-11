package cron

import (
	"time"

	"github.com/lancer-kit/uwe"
)

type Job struct {
	period time.Duration
	ticker *time.Ticker
	run    func() error
}

func NewJob(period time.Duration, run func() error) *Job {
	return &Job{
		period: period,
		run:    run,
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
			if err := j.run(); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}
