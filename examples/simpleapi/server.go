package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/lancer-kit/uwe/v2"
	"github.com/lancer-kit/uwe/v2/presets/api"
)

func main() {
	cfg := api.Config{
		Host: "0.0.0.0",
		Port: 2490,
	}

	server := api.NewServer(cfg, router())

	chief := uwe.NewChief()
	chief.AddWorker("api-server", server)
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
