package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/lancer-kit/uwe"
	"github.com/sirupsen/logrus"
)

// ForceStopTimeout ...
var ForceStopTimeout = 5 * time.Second // nolint:gochecknoglobals

// Config is a parameters for `http.Server`.
type Config struct {
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`

	ApiRequestTimeout int  `json:"api_request_timeout" yaml:"api_request_timeout"` // nolint:golint,stylecheck
	DevMod            bool `json:"dev_mod" yaml:"dev_mod"`
	EnableCORS        bool `json:"enable_cors" yaml:"enable_cors"`

	RestartOnFail bool `json:"restart_on_fail" yaml:"restart_on_fail"`
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

// Server worker object
type Server struct {
	Name      string
	Config    Config
	GetConfig func() Config
	GetRouter func(*logrus.Entry, Config) http.Handler

	logger *logrus.Entry
	ctx    uwe.WContext
}

// NewServer constructor
func NewServer(name string, config Config, rGetter func(*logrus.Entry, Config) http.Handler) Server {
	return Server{
		Name:      name,
		Config:    config,
		GetRouter: rGetter,
	}
}

// Init worker implementation
func (s *Server) Init(parentCtx context.Context) uwe.Worker {
	var ok bool
	s.logger, ok = parentCtx.Value(uwe.CtxKeyLog).(*logrus.Entry)
	if !ok {
		s.logger = logrus.New().WithField("service", s.Name)
	}

	if s.Name == "" {
		s.Name = "http-server"
	}

	s.logger = s.logger.WithField("service", s.Name)
	return s
}

// RestartOnFail property for Chief
func (s *Server) RestartOnFail() bool {
	if s.GetConfig != nil {
		return s.GetConfig().RestartOnFail
	}
	return s.Config.RestartOnFail
}

// Run api-server
func (s *Server) Run(ctx uwe.WContext) uwe.ExitCode {
	if s.GetConfig != nil {
		s.Config = s.GetConfig()
		s.ctx = ctx
	}

	addr := fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: s.GetRouter(s.logger, s.Config),
	}

	serverFailed := make(chan struct{})
	go func() {
		s.logger.Info("Starting API Server at: ", addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Error("server failed")
			close(serverFailed)
		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down the API Server...")
		serverCtx, cancel := context.WithTimeout(context.Background(), ForceStopTimeout)
		err := server.Shutdown(serverCtx)
		if err != nil {
			s.logger.Info("Api Server gracefully stopped")
		}

		cancel()
		s.logger.Info("Api Server gracefully stopped")
		return uwe.ExitCodeOk
	case <-serverFailed:
		return uwe.ExitCodeFailed
	}

}

// Context return application context
func (s *Server) Context() uwe.WContext {
	return s.ctx
}
