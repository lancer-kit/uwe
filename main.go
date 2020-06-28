package uwe

import "context"

// Worker is an interface for async workers
// which launches and manages by the `Chief`.
type Worker interface {
	// Init initializes new instance of the `Worker` implementation,
	// this context should be used only as Key/Value transmitter,
	// DO NOT use it for `<- ctx.Done()`
	Init(ctx context.Context) Worker
	// RestartOnFail determines the need to restart the worker, if it stopped.
	RestartOnFail() bool
	// Run starts the `Worker` instance execution.
	Run(ctx WContext) ExitCode
}

// ExitCode custom type
type ExitCode int

const (
	// ExitCodeOk means that the worker is stopped.
	ExitCodeOk ExitCode = iota
	// ExitCodeInterrupted means that the work cycle has been interrupted and can be restarted.
	ExitCodeInterrupted
	// ExitCodeFailed means that the worker fails.
	ExitCodeFailed
	// ExitReinitReq means that the worker can't do job and requires reinitialization.
	ExitReinitReq
)
