package presets

import (
	"log"
	"time"

	"github.com/lancer-kit/uwe/v3"
)

type dummy struct {
	tickDuration time.Duration
}

func (d *dummy) doSomething(ctx uwe.Context) error {
	// initialize all required stuffs for the execution flow
	ticker := time.NewTicker(d.tickDuration)

	for {
		select {
		case <-ticker.C:
			// define all the processing code here
			// or move it to a method and make a call here
			log.Println("do something")
		case <-ctx.Done():
			// close all connections, channels and finalise state if needed
			log.Println("good bye")
			return nil
		}
	}
}

func ExampleWorkerFunc_Run() {
	var anonFuncWorker = func(ctx uwe.Context) error {
		// initialize all required stuffs for the execution flow
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-ticker.C:
				// define all the processing code here
				// or move it to a method and make a call here
				log.Println("do something")
			case <-ctx.Done():
				// close all connections, channels and finalise state if needed
				log.Println("good bye")
				return nil
			}
		}
	}
	var dummyWorker = dummy{tickDuration: time.Second}

	// initialize new instance of Chief
	chief := uwe.NewChief()
	chief.UseDefaultRecover()
	chief.SetEventHandler(uwe.STDLogEventHandler())

	// will add workers into the pool
	chief.AddWorker("anon-func", WorkerFunc(anonFuncWorker))
	chief.AddWorker("method-as-worker", WorkerFunc(dummyWorker.doSomething))

	chief.Run()
}
