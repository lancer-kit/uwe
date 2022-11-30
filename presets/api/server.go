package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/lancer-kit/uwe/v3"
)

const forceStopTimeout = 5 * time.Second

// Config is a parameters for `http.Server`.
// APIRequestTimeout and ReadHeaderTimeout time.Duration in Seconds.
type Config struct {
	Host       string `json:"host" yaml:"host" toml:"host"`
	Port       int    `json:"port" yaml:"port" toml:"port"`
	EnableCORS bool   `json:"enable_cors" yaml:"enable_cors" toml:"enable_cors"`
	// nolint:lll
	APIRequestTimeout int `json:"api_request_timeout" yaml:"api_request_timeout" toml:"api_request_timeout"`
	ReadHeaderTimeout int `json:"read_header_timeout" yaml:"read_header_timeout" toml:"read_header_timeout"`
}

// Validate - Validate config required fields
func (c Config) Validate() error {
	if c.Host == "" {
		return errors.New("host cannot be blank")
	}
	if c.Port == 0 {
		return errors.New("port cannot be blank")
	}
	return nil
}

// TCPAddr returns tcp address for server.
func (c *Config) TCPAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Server is worker by default for starting a standard HTTP server.
// Server requires configuration and filled `http.Handler`.
// The HTTP server will work properly and will be correctly disconnected upon a signal from Supervisor (Chief).
// Warning: this Server does not process SSL/TLS certificates on its own.
//
//	To start an HTTPS server, look for a specific worker.
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
	readHeaderTimeout := time.Minute
	if s.config.ReadHeaderTimeout > 0 {
		readHeaderTimeout = time.Duration(s.config.ReadHeaderTimeout) * time.Second
	}
	server := &http.Server{
		Addr:              s.config.TCPAddr(),
		Handler:           s.router,
		ReadHeaderTimeout: readHeaderTimeout,
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
			return fmt.Errorf("server shutdown failed: %s", err)
		}
		return nil
	case err := <-serverFailed:
		return fmt.Errorf("server failed: %s", err)
	}

}
