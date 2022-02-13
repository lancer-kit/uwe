package presets

import (
	"github.com/lancer-kit/uwe/v3"
)

// WorkerFunc is a type of worker that consist from one function.
// Allow to use the function as worker.
type WorkerFunc func(ctx uwe.Context) error

// Init is a method to satisfy `uwe.Worker` interface.
func (WorkerFunc) Init() error { return nil }

// Run executes function as worker.
func (f WorkerFunc) Run(ctx uwe.Context) error { return f(ctx) }
