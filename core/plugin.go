package core

type Aliaser interface {
	Alias() string
}

type Input interface {
	Init(out chan<- *Event)
	Serve()
	Close() error
	Aliaser
}

type Filter interface {
	Init(in <-chan *Event, rejected chan<- *Event, accepted chan<- *Event)
	Filter()
	Close() error
	Aliaser
}

type Processor interface {
	Init(in <-chan *Event, out chan<- *Event)
	Process()
	Close() error
	Aliaser
}

type Output interface {
	Init(in <-chan *Event)
	Listen()
	Close() error
	Aliaser
}

type Parser interface {
	Parse(data []byte, routingKey string) ([]*Event, error)
	Aliaser
}

type Serializer interface {
	Serialize(events ...*Event) ([]byte, error)
	Aliaser
}

// core plugins
type Fusion interface {
	Init(ins []<-chan *Event, out chan<- *Event)
	Run()
}

type Broadcast interface {
	Init(in <-chan *Event, outs []chan<- *Event)
	Run()
}
