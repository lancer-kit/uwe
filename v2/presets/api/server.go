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

const ForceStopTimeout = 5 * time.Second

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

// Server
type Server struct {
	config Config
	router http.Handler
}

func NewServer(config Config, router http.Handler) *Server {
	return &Server{
		config: config,
		router: router,
	}
}

func (s *Server) Init() error {
	return nil
}

func (s *Server) Run(ctx uwe.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	server := &http.Server{
		Addr:    addr,
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
		serverCtx, cancel := context.WithTimeout(context.Background(), ForceStopTimeout)
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
