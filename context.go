package uwe

import (
	"context"
	"fmt"
)

type WorkerName string

type Message struct {
	UID    int64
	Target WorkerName
	Sender WorkerName
	Data   interface{}
}

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

func NewContext(name WorkerName, ctx context.Context, in, out chan *Message) WContext {
	return &wContext{
		Context: ctx,
		name:    name,
		in:      in,
		out:     out,
	}

}

func (wc *wContext) SendMessage(target WorkerName, data interface{}) error {
	wc.out <- &Message{
		UID:    0,
		Target: target,
		Sender: wc.name,
		Data:   data,
	}
	return nil
}

func (wc *wContext) MessageBus() <-chan *Message {
	return wc.in
}

type execStatus string

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

func (s *workerSignal) Error() string {
	return fmt.Sprintf("%s(%v); %s", s.name, s.sig, s.msg)
}
