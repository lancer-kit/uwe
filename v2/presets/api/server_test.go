package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/lancer-kit/uwe/v2"
)

func Example() {
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

	// will add workers into the pool
	chief.AddWorker("app-server", NewServer(apiCfg, getRouter()))

	// init all registered workers and run it all
	chief.Run()
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
