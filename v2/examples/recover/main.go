package main

import (
	"fmt"
	"log"
	"time"

	"github.com/lancer-kit/uwe/v2"
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

	chief.UseDefaultRecover()
	chief.SetEventHandler(func(event uwe.Event) {
		if event.IsFatal() || event.IsError() {
			fmt.Println(event.ToError())
			return
		}
		fmt.Printf("level: %s\nmsg: %s", event.Level, event.Message)
	})

	chief.AddWorker("panicDummy-1", &panicDummy{})
	chief.AddWorker("dummy-1", &dummy{})

	chief.Run()
}

type dummy struct{}

func (d dummy) Init() error { return nil }

func (d dummy) Run(ctx uwe.Context) error {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			log.Println("good bye")
		case <-ticker.C:
			log.Println("do something")
		}
	}

}
