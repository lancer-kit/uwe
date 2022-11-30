package uwe

import "context"

// Context is a wrapper over the standard `context.Context`.
// The main purpose of this is to extend in the future.
type Context interface {
	context.Context
	Mailbox
}

type ctx struct {
	context.Context
	Mailbox
}

// NewContext returns new context.
func NewContext(c context.Context, m Mailbox) Context {
	return ctx{Context: c, Mailbox: m}
}
