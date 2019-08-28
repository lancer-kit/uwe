package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lancer-kit/uwe"
	"github.com/sirupsen/logrus"
)

type dummy struct {
	name   string
	logger *logrus.Entry
}

func (d dummy) Init(ctx context.Context) uwe.Worker {
	logger, ok := ctx.Value(uwe.CtxKeyLog).(*logrus.Entry)
	if !ok {
		logger = logrus.NewEntry(logrus.New())
	}
	return &dummy{
		name:   d.name,
		logger: logger.WithField("worker", d.name),
	}
}

func (dummy) RestartOnFail() bool {
	return true
}

func (d dummy) Run(wCtx uwe.WContext) uwe.ExitCode {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ticker.C:
			d.logger.Info("Perform my task")
			switch d.name {
			case "dummy-1":
				_ = wCtx.SendMessage("dummy-2", "Hi, Johnny")
				_ = wCtx.SendMessage("dummy-3", "Hi, Johnny")
			case "dummy-2":
				_ = wCtx.SendMessage("dummy-1", "Hi, Johnny")
				_ = wCtx.SendMessage("dummy-3", "Hi, Johnny")
			case "dummy-3":
				_ = wCtx.SendMessage("dummy-1", "Hi, Johnny")
				_ = wCtx.SendMessage("dummy-2", "Hi, Johnny")
			}

		case m := <-wCtx.MessageBus():
			d.logger.
				WithField("Sender", m.Sender).
				WithField("Target", m.Target).
				WithField("data", fmt.Sprintf("%+v", m.Data)).
				Info("got new message")
		case <-wCtx.Done():
			ticker.Stop()
			d.logger.Info("Receive exit code, stop working")
			return uwe.ExitCodeOk
		}
	}

}

func main() {
	chief := uwe.NewChief(
		"chief",
		true,

		logrus.New().WithField("env", "example"),
	)
	chief.EnableAdminAPI = true
	chief.AdminConfig = uwe.AdminConfig{
		PPROF:        true,
		AllowControl: true,
		Host:         "0.0.0.0",
		Port:         8080,
	}

	chief.AddWorker("dummy-1", &dummy{name: "dummy-1"})
	chief.AddWorker("dummy-2", &dummy{name: "dummy-2"})
	chief.AddWorker("dummy-3", &dummy{name: "dummy-3"})

	err := chief.Run("dummy-1", "dummy-2", "dummy-3")
	if err != nil {
		logrus.Fatal(err)
	}
}
