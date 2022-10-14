package uwe

import (
	"context"
)

type IMQBroker interface {
	DefaultBus() SenderBus
	AddWorker(name WorkerName) Mailbox
	Init() error
	Serve(ctx context.Context)
}

type Broker struct {
	defaultChanLen  int
	workersHub      map[WorkerName]chan<- *Message
	workersMessages chan *Message
}

func NewBroker(defaultChanLen int) *Broker {
	if defaultChanLen < 1 {
		defaultChanLen = 1
	}

	return &Broker{
		workersHub:      map[WorkerName]chan<- *Message{},
		workersMessages: make(chan *Message, defaultChanLen)}
}

func (hub *Broker) DefaultBus() SenderBus {
	return NewSenderBus(hub.workersMessages)
}

func (hub *Broker) AddWorker(name WorkerName) Mailbox {
	workerDirectChan := make(chan *Message, hub.defaultChanLen)
	hub.workersHub[name] = workerDirectChan
	return NewBus(name, workerDirectChan, hub.workersMessages)
}

func (hub *Broker) Init() error { return nil }

func (hub *Broker) Serve(ctx context.Context) {
	for {
		select {
		case msg := <-hub.workersMessages:
			if msg == nil {
				continue
			}

			switch msg.Target {
			case TargetSelfInit:
				_, ok := hub.workersHub[msg.Sender]
				if ok {
					continue
				}

				bus, ok := msg.Data.(chan *Message)
				if ok {
					hub.workersHub[msg.Sender] = bus
				}

			case TargetBroadcast:
				for to := range hub.workersHub {
					if to == msg.Sender {
						continue
					}
					b := hub.workersHub[to]
					go sendMsg(b, *msg)

				}
			default:
				if b, ok := hub.workersHub[msg.Target]; ok {
					go sendMsg(b, *msg)
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func sendMsg(bus chan<- *Message, msg Message) { bus <- &msg }

// NopBroker is an empty IMQBroker
type NopBroker struct{}

func (*NopBroker) DefaultBus() SenderBus             { return &NopMailbox{} }
func (*NopBroker) AddWorker(name WorkerName) Mailbox { return &NopMailbox{} }
func (*NopBroker) Init() error                       { return nil }
func (*NopBroker) Serve(ctx context.Context)         {}
