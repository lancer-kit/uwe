package uwe

const (
	TargetBroadcast = "*"
	TargetSelfInit  = "self-init"
)

type (
	Mailbox interface {
		SenderBus
		ReaderBus
	}

	SenderBus interface {
		Send(target WorkerName, data interface{})
		SendWithKind(target WorkerName, kind MessageKind, data interface{})
		SendToMany(kind MessageKind, data interface{}, targets ...WorkerName)
		SelfInit(name WorkerName) Mailbox
	}

	ReaderBus interface {
		Messages() <-chan *Message
	}

	MessageKind int

	Message struct {
		Target WorkerName
		Sender WorkerName
		Kind   MessageKind
		Data   interface{}
	}
)

type eventBus struct {
	name      WorkerName
	readOnly  bool
	writeOnly bool

	// in is channel for incoming messages for a worker
	in chan *Message
	// out is channel for outgoing messages from a worker
	out chan<- *Message
}

func NewSenderBus(fromWorker chan<- *Message) SenderBus {
	in := make(chan *Message, 2)
	return &eventBus{
		writeOnly: true,
		name:      "",
		in:        in,
		out:       fromWorker,
	}
}

func NewReaderBus(name WorkerName, toWorker chan *Message) ReaderBus {
	return &eventBus{
		readOnly: true,
		name:     name,
		in:       toWorker,
	}
}

func NewBus(name WorkerName, toWorker, fromWorker chan *Message) Mailbox {
	return &eventBus{
		name: name,
		in:   toWorker,
		out:  fromWorker,
	}

}

func (wc *eventBus) SendWithKind(target WorkerName, kind MessageKind, data interface{}) {
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

func (wc *eventBus) SendToMany(kind MessageKind, data interface{}, targets ...WorkerName) {
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

func (wc *eventBus) Send(target WorkerName, data interface{}) {
	if wc.readOnly {
		return
	}

	wc.out <- &Message{
		Target: target,
		Sender: wc.name,
		Data:   data,
	}
}

func (wc *eventBus) SelfInit(name WorkerName) Mailbox {
	wc.name = name
	wc.writeOnly = false

	wc.out <- &Message{
		Target: TargetSelfInit,
		Sender: wc.name,
		Data:   wc.in,
	}

	return wc
}

func (wc *eventBus) Messages() <-chan *Message { return wc.in }

// NopMailbox is an empty Mailbox
type NopMailbox struct{}

func (*NopMailbox) Send(WorkerName, interface{})                       {}
func (*NopMailbox) SendWithKind(WorkerName, MessageKind, interface{})  {}
func (*NopMailbox) SendToMany(MessageKind, interface{}, ...WorkerName) {}
func (m *NopMailbox) SelfInit(WorkerName) Mailbox                      { return m }
func (*NopMailbox) Messages() <-chan *Message {
	c := make(chan *Message)

	// TODO: Requires a decision whether to return an empty channel or return a closed channel.
	// An empty channel results in blocking the reader forever.

	// close(c)
	return c
}
