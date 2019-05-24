package main

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/lancer-kit/uwe"
)

type dummy struct {
	ctx context.Context
}

func (dummy) Init(ctx context.Context) uwe.Worker {
	return &dummy{ctx: ctx}
}

func (dummy) RestartOnFail() bool {
	return true
}

func (d dummy) Run() uwe.ExitCode {
	logger := logrus.New().WithField("worker", "dummy")
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			logger.Info("Perform my task")
		case <-d.ctx.Done():
			ticker.Stop()
			logger.Info("Receive exit code, stop working")
			return uwe.ExitCodeOk
		}
	}

}

func main() {
	chief := uwe.NewChief(
		"chief",
		true,
		true,
		logrus.New().WithField("env", "example"),
	)
	chief.AddWorker("dummy", &dummy{})

	err := chief.RunAll("test", "dummy")
	if err != nil {
		logrus.Fatal(err)
	}
}
