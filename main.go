package uwe

import (
	"context"
)

// Worker is an interface for async workers
// which launches and manages by the `Chief`.
type Worker interface {
	// Init initializes new instance of the `Worker` implementation.
	Init(context.Context) Worker
	// RestartOnFail determines the need to restart the worker, if it stopped.
	RestartOnFail() bool
	// Run starts the `Worker` instance execution.
	Run() ExitCode
}

type ExitCode int

const (
	// ExitCodeOk means that the worker is stopped.
	ExitCodeOk ExitCode = iota
	// ExitCodeInterrupted means that the work cycle has been interrupted and can be restarted.
	ExitCodeInterrupted
	// ExitCodeFailed means that the worker fails.
	ExitCodeFailed
)
