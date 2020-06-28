![CI Status](https://github.com/lancer-kit/uwe/workflows/Go/badge.svg)

# uwe

UWE (Ubiquitous Workers Engine) is a common toolset for building and organizing your Go application,  actor-like workers.  


## Table of Content

1. [Quick Start](#quick-start)
2. [Documentation](#documentation)
    2. [Chief](#chief)
    3. [Worker](#worker)
    4. [Presets](#presets)


## Quick Start

Get `uwe` using **go get**:

```shell script
go get github.com/lancer-kit/uwe/v2
```

Here is an example HelloWorld service with HTTP API and background worker:

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
	// fill configurations for the predefined worker that start an HTTP server
	apiCfg := api.Config{
		Host:              "0.0.0.0",
		Port:              8080,
		EnableCORS:        false,
		ApiRequestTimeout: 0,
	}

	// initialize new instance of Chief
	chief := uwe.NewChief()
	// will add workers into the pool
	chief.AddWorker("app-server", api.NewServer(apiCfg, getRouter()))
	chief.AddWorker("dummy", NewDummy())

	// will enable recover of internal panics
	chief.UseDefaultRecover()
	// pass handler for internal events like errors, panics, warning, etc.
	// you can log it with you favorite logger (ex Logrus, Zap, etc)
	chief.SetEventHandler(uwe.STDLogEventHandler())

	// init all registered workers and run it all
	chief.Run()
}

type dummy struct{}

// NewDummy initialize new instance of dummy Worker.
func NewDummy() uwe.Worker {
	// At this point in most cases there we are preparing some state of the worker,
	// like a logger, configuration, variable, and fields.
	 return &dummy{}
}

// Init is an interface method used to initialize some state of the worker
// that required interaction with outer context, for example, initialize some connectors.
func (d *dummy) Init() error { return nil }

// Run starts event loop of worker.
func (d *dummy) Run(ctx uwe.Context) error {
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

// getRouter is used to declare an API scheme, 
func getRouter() http.Handler {
	// instead default can be used any another compatible router
	mux := http.NewServeMux()
	mux.HandleFunc("/hello/uwe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "hello world")
	})

	log.Println("REST API router initialized")
	return mux
}
```

## Documentation

### Chief

**Chief** is a supervisor that can be placed at the top of the go application's execution stack, 
it is blocked until SIGTERM is intercepted and then it shutdown all workers gracefully.
Also, `Chief` can be used as a child supervisor inside the `Worker`, which is launched by `Chief` at the top-level.

### Worker

**Worker** is an interface for async workers which launches and manages by the **Chief**.

1. `Init()` - method used to initialize some state of the worker that required interaction with outer context,
 for example, initialize some connectors. In many cases this method is optional, so it can be implemented as empty:
  `func (*W) Init() error { return nil }`. 
2. `Run(ctx Context) error` - starts the `Worker` instance execution. The context will provide a signal
 when a worker must stop through the `ctx.Done()`.

Workers lifecycle:

```text
 (*) -> [New] -> [Initialized] -> [Run] -> [Stopped]
          |             |           |
          |             |           ↓
          |-------------|------> [Failed]
```

### Presets

This library provides some working presets to simplify the use of `Chief` in projects and reduce duplicate code.

#### HTTP Server

`api.Server` is worker by default for starting a standard HTTP server. Server requires configuration and initialized `http.Handler`. 

The HTTP server will work properly and will be correctly disconnected upon a signal from Supervisor (Chief).

> Warning: this Server does not process SSL/TLS certificates on its own.To start an HTTPS server, look for a specific worker.

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/lancer-kit/uwe/v2"
)

func main() {
	// fill configurations for the predefined worker that start an HTTP server
	apiCfg := Config{
		Host:              "0.0.0.0",
		Port:              8080,
		EnableCORS:        false,
		ApiRequestTimeout: 0,
	}

	// initialize new instance of Chief
	chief := uwe.NewChief()
	chief.UseDefaultRecover()
	chief.SetEventHandler(uwe.STDLogEventHandler())

	// instead default can be used any another compatible router
	mux := http.NewServeMux()
	mux.HandleFunc("/hello/uwe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "hello world")
	})

	chief.AddWorker("app-server", NewServer(apiCfg, mux))

	chief.Run()
}
```

#### Job

`presets.Job` is a primitive worker who performs an `action` callback with a given `period`.

```go
package main

import (
	"log"
	"time"

	"github.com/lancer-kit/uwe/v2"
	"github.com/lancer-kit/uwe/v2/presets"
)

func main() {
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
	chief.AddWorker("simple-job", presets.NewJob(time.Second, action))

	chief.Run()
}
```
#### WorkerFunc

`presets.WorkerFunc` is a type of worker that consist from one function. Allow to use the function as worker.

```go
package presets

import (
	"log"
	"time"

	"github.com/lancer-kit/uwe/v2"
	"github.com/lancer-kit/uwe/v2/presets"
)

func main() {
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

	// initialize new instance of Chief
	chief := uwe.NewChief()
	chief.UseDefaultRecover()
	chief.SetEventHandler(uwe.STDLogEventHandler())

	// will add workers into the pool
	chief.AddWorker("anon-func", WorkerFunc(anonFuncWorker))

	chief.Run()
}
```

## License

This library is distributed under the [Apache 2.0](LICENSE) license.

