package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/lancer-kit/uwe/v2"
	"github.com/pkg/errors"
)

const forceStopTimeout = 5 * time.Second

// Config is a parameters for `http.Server`.
type Config struct {
	Host              string `json:"host" yaml:"host"`
	Port              int    `json:"port" yaml:"port"`
	EnableCORS        bool   `json:"enable_cors" yaml:"enable_cors"`
	ApiRequestTimeout int    `json:"api_request_timeout" yaml:"api_request_timeout"`
}

// Validate - Validate config required fields
func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Host, validation.Required),
		validation.Field(&c.Port, validation.Required),
	)
}

// TCPAddr returns tcp address for server.
func (c *Config) TCPAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Server is worker by default for starting a standard HTTP server.
// Server requires configuration and filled `http.Handler`.
// The HTTP server will work properly and will be correctly disconnected upon a signal from Supervisor (Chief).
// Warning: this Server does not process SSL/TLS certificates on its own.
// 		To start an HTTPS server, look for a specific worker.
type Server struct {
	config Config
	router http.Handler
}

// NewServer returns a new instance of `Server` with the passed configuration and HTTP router.
func NewServer(config Config, router http.Handler) *Server {
	return &Server{
		config: config,
		router: router,
	}
}

// Init is a method to satisfy `uwe.Worker` interface.
func (s *Server) Init() error { return nil }

// Run starts serving the passed `http.Handler` with HTTP server.
func (s *Server) Run(ctx uwe.Context) error {
	server := &http.Server{
		Addr:    s.config.TCPAddr(),
		Handler: s.router,
	}

	serverFailed := make(chan error)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverFailed <- err
			close(serverFailed)
		}
	}()

	select {
	case <-ctx.Done():
		serverCtx, cancel := context.WithTimeout(context.Background(), forceStopTimeout)
		defer cancel()

		err := server.Shutdown(serverCtx)
		if err != nil {
			return errors.Wrap(err, "server shutdown failed")
		}
		return nil
	case err := <-serverFailed:
		return errors.Wrap(err, "server failed")
	}

}
