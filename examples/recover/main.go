package main

import (
	"fmt"

	"github.com/lancer-kit/uwe"
	"github.com/pkg/errors"
)

type panicDummy struct {
}

func (d panicDummy) Init() error {
	return nil
}

func (d panicDummy) Run(ctx uwe.Context) error {
	panic(errors.New("implement me"))
}

func main() {
	chief := uwe.NewChief()

	chief.SetDefaultRecover()
	chief.SetEventHandler(func(event uwe.Event) {
		if event.IsFatal() || event.IsError() {
			fmt.Println(event.ToError())
			return
		}
		fmt.Printf("level: %s\nmsg: %s", event.Level, event.Message)
	})

	chief.AddWorker("panicDummy-1", &panicDummy{})

	chief.Run()
}
