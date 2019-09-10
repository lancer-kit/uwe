package main

import (
	"encoding/json"
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

	chief.SetRecover(func() {
		r := recover()
		switch r.(type) {
		case error:
			fmt.Printf("oh, no! there is an error: %v\n", r)
		case string:
			fmt.Printf("oh, no! there is no error, just string: %v\n", r)
		}
	})
	chief.SetEventHandler(func(event uwe.Event) {
		msg, err := json.Marshal(event)
		if err != nil {
			fmt.Println("err: " + err.Error())
		}
		fmt.Println(string(msg))
	})

	chief.AddWorker("panicDummy-1", &panicDummy{})
	chief.AddWorker("panicDummy-2", &panicDummy{})

	chief.Run()
}
