package uwe

import (
	"log"
	"time"
)

func Example() {
	// initialize new instance of Chief
	chief := NewChief()
	// will add workers into the pool
	chief.AddWorker("dummy", NewDummy())

	// will enable recover of internal panics
	chief.UseDefaultRecover()
	// pass handler for internal events like errors, panics, warning, etc.
	// you can log it with you favorite logger (ex Logrus, Zap, etc)
	chief.SetEventHandler(STDLogEventHandler())

	// init all registered workers and run it all
	chief.Run()
}

type dummy struct{}

// NewDummy initialize new instance of dummy Worker.
func NewDummy() Worker {
	// At this point in most cases there we are preparing some state of the worker,
	// like a logger, configuration, variable, and fields.
	return &dummy{}
}

// Init is an interface method used to initialize some state of the worker
// that required interaction with outer context, for example, initialize some connectors.
func (d *dummy) Init() error { return nil }

// Run starts event loop of worker.
func (d *dummy) Run(ctx Context) error {
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
