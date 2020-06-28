package uwe

import (
	"context"
	"fmt"
)

// WorkerName custom type
type WorkerName string

// Message type declaration
type Message struct {
	UID    int64
	Target WorkerName
	Sender WorkerName
	Data   interface{}
}

// WContext declare interface for workers with context
type WContext interface {
	context.Context
	SendMessage(target WorkerName, data interface{}) error
	MessageBus() <-chan *Message
}

type wContext struct {
	context.Context

	name WorkerName

	// in is channel for incoming messages for a worker
	in chan *Message
	// out is channel for outgoing messages from a worker
	out chan<- *Message
}

// NewContext constructor for worker with context
func NewContext(ctx context.Context, name WorkerName, in, out chan *Message) WContext {
	return &wContext{
		Context: ctx,
		name:    name,
		in:      in,
		out:     out,
	}

}

// SendMessage implements WContext.SendMessage. Send message to worker by name.
func (wc *wContext) SendMessage(target WorkerName, data interface{}) error {
	wc.out <- &Message{
		UID:    0,
		Target: target,
		Sender: wc.name,
		Data:   data,
	}
	return nil
}

// MessageBus implements WContext.MessageBus receiver for messages
func (wc *wContext) MessageBus() <-chan *Message {
	return wc.in
}

type execStatus string

// nolint:varcheck,deadcode
const (
	signalOk             execStatus = "ok"
	signalInterrupted    execStatus = "interrupted"
	signalFailure        execStatus = "failure"
	signalStop           execStatus = "stop"
	signalUnexpectedStop execStatus = "unexpected_stop"
)

type workerSignal struct {
	name WorkerName
	sig  execStatus
	msg  string
}

// Error implements error message stringer
func (s *workerSignal) Error() string {
	return fmt.Sprintf("%s(%v); %s", s.name, s.sig, s.msg)
}
