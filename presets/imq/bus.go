package imq

import (
	"context"

	"github.com/lancer-kit/uwe/v3"
)

type MessageKind int

type Message struct {
	Target uwe.WorkerName
	Sender uwe.WorkerName
	Kind   MessageKind
	Data   interface{}
}

type SenderBus interface {
	Send(target uwe.WorkerName, data interface{})
	SendWithKind(target uwe.WorkerName, kind MessageKind, data interface{})
	SendToMany(kind MessageKind, data interface{}, targets ...uwe.WorkerName)
	SelfInit(name uwe.WorkerName) EventBus
}

type ReaderBus interface {
	Messages() <-chan *Message
}

type EventBus interface {
	context.Context
	SenderBus
	ReaderBus
}

type eventBus struct {
	context.Context

	name      uwe.WorkerName
	readOnly  bool
	writeOnly bool

	// in is channel for incoming messages for a worker
	in chan *Message
	// out is channel for outgoing messages from a worker
	out chan<- *Message
}

func NewSenderBus(ctx context.Context, fromWorker chan<- *Message) SenderBus {
	in := make(chan *Message, 2)
	return &eventBus{
		Context:   ctx,
		writeOnly: true,
		name:      "",
		in:        in,
		out:       fromWorker,
	}
}

func NewReaderBus(ctx context.Context, name uwe.WorkerName, toWorker chan *Message) ReaderBus {
	return &eventBus{
		Context:  ctx,
		readOnly: true,
		name:     name,
		in:       toWorker,
	}
}

func NewBus(ctx context.Context, name uwe.WorkerName, toWorker, fromWorker chan *Message) EventBus {
	return &eventBus{
		Context: ctx,
		name:    name,
		in:      toWorker,
		out:     fromWorker,
	}

}

func (wc *eventBus) SendWithKind(target uwe.WorkerName, kind MessageKind, data interface{}) {
	if wc.readOnly {
		return
	}

	wc.out <- &Message{
		Target: target,
		Sender: wc.name,
		Kind:   kind,
		Data:   data,
	}
}

func (wc *eventBus) SendToMany(kind MessageKind, data interface{}, targets ...uwe.WorkerName) {
	if wc.readOnly {
		return
	}

	for _, target := range targets {
		wc.out <- &Message{
			Target: target,
			Sender: wc.name,
			Kind:   kind,
			Data:   data,
		}
	}

}

func (wc *eventBus) Send(target uwe.WorkerName, data interface{}) {
	if wc.readOnly {
		return
	}

	wc.out <- &Message{
		Target: target,
		Sender: wc.name,
		Data:   data,
	}
}

func (wc *eventBus) SelfInit(name uwe.WorkerName) EventBus {
	wc.name = name
	wc.writeOnly = false

	wc.out <- &Message{
		Target: TargetSelfInit,
		Sender: wc.name,
		Data:   wc.in,
	}

	return wc
}

func (wc *eventBus) Messages() <-chan *Message {
	return wc.in
}
