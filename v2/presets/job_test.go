package presets

import (
	"log"
	"time"

	"github.com/lancer-kit/uwe/v2"
)

func ExampleJob_Run() {
	var action = func() error {
		// define all the processing code here
		// or move it to a method and make a call here
		log.Println("do something")
		return nil
	}

	// initialize new instance of Chief
	chief := uwe.NewChief()
	chief.UseDefaultRecover()
	chief.SetEventHandler(uwe.STDLogEventHandler())

	// will add workers into the pool
	chief.AddWorker("simple-job", NewJob(time.Second, action))

	chief.Run()
}
