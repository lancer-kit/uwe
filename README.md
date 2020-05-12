# uwe

UWE (Ubiquitous Workers Engine) is a common toolset for building and organizing your Go application, life cycles of actor-like workers.  

`uwe.Chief` is a supervisor that can be placed at the top of the go application's execution stack, it is blocked until SIGTERM is intercepted and then it shutdown all workers gracefully.

## Installation

Standard `go get`:

```shell script
go get github.com/lancer-kit/uwe/v2
```

## Usage & Example
 

An example of service with some API and background worker:
 
```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/lancer-kit/uwe/v2"
	"github.com/lancer-kit/uwe/v2/presets/api"
)


func main()  {
	apiCfg := api.Config{
		Host:              "0.0.0.0",
		Port:              8080,
		EnableCORS:        false,
		ApiRequestTimeout: 0,
	}

	chief := uwe.NewChief()
	// will add worker into the pool
	chief.AddWorker("app-server", api.NewServer(apiCfg, getRouter()))

	// will enable recover of internal panics
	chief.UseDefaultRecover()
	// pass handler for internal events like errors, panics, warning, etc.
	// you can log it with you favorite logger (ex Logrus, Zap, etc)
	chief.SetEventHandler(chiefEventHandler())
    // init all registered workers and run it all
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


func getRouter() http.Handler {
    // instead default can be used any another compatible router
    mux := http.NewServeMux()
    mux.HandleFunc("/hello/uwe", func(w http.ResponseWriter, r *http.Request) {
        _, _ = fmt.Fprintln(w, "hello world")
    })
	log.Println("REST API router initialized")
	return mux
}


func chiefEventHandler() func(event uwe.Event) {
	return func(event uwe.Event) {
		var level string
		switch event.Level {
		case uwe.LvlFatal, uwe.LvlError:
			level = "ERROR"
		case uwe.LvlInfo:
			level = "INFO"
		default:
			level = "WARN"
		}
		log.Println(fmt.Sprintf("%s: %s %+v", level, event.Message, event.Fields))
	}
}
```

