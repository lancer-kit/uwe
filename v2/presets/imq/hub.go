package imq

import (
	"context"

	"github.com/lancer-kit/uwe/v2"
)

const (
	TargetBroadcast = "*"
	TargetSelfInit  = "self-init"
)

type Broker struct {
	ctx        context.Context
	cancelFunc context.CancelFunc

	defaultChanLen  int
	workersHub      map[uwe.WorkerName]chan<- *Message
	workersMessages chan *Message
}

func NewBroker(defaultChanLen int) *Broker {
	if defaultChanLen < 1 {
		defaultChanLen = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Broker{
		ctx:             ctx,
		cancelFunc:      cancel,
		workersHub:      map[uwe.WorkerName]chan<- *Message{},
		workersMessages: make(chan *Message, defaultChanLen)}
}

func (hub *Broker) DefaultBus() SenderBus {
	return NewSenderBus(hub.ctx, hub.workersMessages)
}

func (hub *Broker) AddWorker(name uwe.WorkerName) EventBus {
	workerDirectChan := make(chan *Message, hub.defaultChanLen)
	hub.workersHub[name] = workerDirectChan
	return NewBus(hub.ctx, name, workerDirectChan, hub.workersMessages)
}

func (hub *Broker) Init() error {
	return nil
}

func (hub *Broker) Run(ctx uwe.Context) error {
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
			hub.cancelFunc()
			return nil
		}
	}
}

func sendMsg(bus chan<- *Message, msg Message) {
	bus <- &msg
}
