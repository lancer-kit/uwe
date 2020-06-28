package uwe

import "context"

// Context is a wrapper over the standard `context.Context`.
// The main purpose of this is to extend in the future.
type Context interface {
	context.Context
}

type ctx struct {
	context.Context
}

// NewContext returns new context.
func NewContext() Context {
	return ctx{
		Context: context.Background(),
	}
}
