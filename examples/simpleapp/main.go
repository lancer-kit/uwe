package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/lancer-kit/uwe/v3"
	"github.com/lancer-kit/uwe/v3/presets/api"
)

func main() {
	cfg := api.Config{
		Host: "0.0.0.0",
		Port: 2490,
	}

	const (
		workerAPI  = "api-server"
		workerBack = "background_worker"
	)
	server := api.NewServer(cfg, router())

	chief := uwe.NewChief()
	chief.SetEventHandler(uwe.STDLogEventHandler())
	//chief.UseNopIMQBroker()
	chief.AddWorker(workerAPI, server, uwe.RestartOnFail)

	//chief.AddWorker("restart-on-err", &daemon{name: "restart-on-err"}, uwe.RestartOnError)
	//chief.AddWorker("restart-on-fail", &daemon{name: "restart-on-fail"}, uwe.RestartOnFail)
	//chief.AddWorker("restart-on-err-with-re-init", &daemon{name: "restart-on-err-with-re-init"},
	//  uwe.RestartOnError, uwe.RestartWithReInit)
	//chief.AddWorker("restart-on-fail-with-re-init", &daemon{name: "restart-on-fail-with-re-init"},
	//  uwe.RestartOnFail, uwe.RestartWithReInit)

	chief.AddWorker("no-restart", &daemon{name: "no-restart"}, uwe.NoRestart)

	chief.AddWorker("restart", &daemon{name: "restart"}, uwe.Restart)
	chief.AddWorker("restart-full", &daemon{name: "restart-full"}, uwe.RestartAndReInit)

	chief.AddWorker("show-stopper", &daemon{name: "show-stopper"}, uwe.StopAppOnFail)

	chief.Run()
}

func router() http.Handler {
	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Get("/hello-world", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "plain/text; charset=utf-8")
			_, err := w.Write([]byte("Hello, World"))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		})
	})
	return r
}

type daemon struct {
	name    string
	counter int

	mode uwe.RestartOption
}

func (d *daemon) Init() error {
	d.counter = 0
	fmt.Printf("-> [INIT] daemon=%s i=%d \n", d.name, d.counter)
	return nil
}

func (d *daemon) Run(ctx uwe.Context) error {
	fmt.Printf("-> [RUN] daemon=%s i=%d \n", d.name, d.counter)

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ctx.Messages():
			fmt.Printf("-> [GOT MESSAGE] daemon=%s sender(%s) kind(%d) data(%v)\n",
				d.name, msg.Sender, msg.Kind, msg.Data)

		case <-ticker.C:
			d.counter++
			fmt.Printf("-> [JOB TICK] daemon=%s i=%d \n", d.name, d.counter)

			if d.counter%10 == 0 {
				panic("time to panic")
			}

			if d.counter%5 == 0 {
				return errors.New("time to error")
			}
		}
	}
}
