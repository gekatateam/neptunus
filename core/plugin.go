package core

type Filter interface {
	Init(in <-chan *Event, rejected chan<- *Event, accepted chan<- *Event)
	Filter()
	Close() error
}

type Processor interface {
	Init(in <-chan *Event, out chan<- *Event)
	Process()
	Close() error
}

type Input interface {
	Init(out chan<- *Event)
	Serve()
	Close() error
}

type Output interface {
	Init(in <-chan *Event)
	Listen()
	Close() error
}
