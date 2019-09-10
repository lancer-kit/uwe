package uwe

import "context"

type Context interface {
	context.Context
}

type ctx struct {
	context.Context
}

func NewContext() Context {
	return ctx{
		Context: context.Background(),
	}
}
