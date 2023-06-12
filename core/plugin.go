package core

type Aliaser interface {
	Alias() string
}

// input plugin consumes events from outer world
type Input interface {
	Init(out chan<- *Event)
	Serve()
	Close() error
	Aliaser
}

// filter plugin sorts events by conditions
type Filter interface {
	Init(in <-chan *Event, rejected chan<- *Event, accepted chan<- *Event)
	Filter()
	Close() error
	Aliaser
}

// processor plugin transforms events
type Processor interface {
	Init(in <-chan *Event, out chan<- *Event)
	Process()
	Close() error
	Aliaser
}

// output plugin produces events to outer world
type Output interface {
	Init(in <-chan *Event)
	Listen()
	Close() error
	Aliaser
}

// parser plugin parses raw format data into events
type Parser interface {
	Parse(data []byte, routingKey string) ([]*Event, error)
	Aliaser
}

// serializer plugin serializes events into configured format
type Serializer interface {
	Serialize(event *Event) ([]byte, error)
	Aliaser
}

// core plugins
// used in core units only
type Fusion interface {
	Init(ins []<-chan *Event, out chan<- *Event)
	Run()
}

type Broadcast interface {
	Init(in <-chan *Event, outs []chan<- *Event)
	Run()
}
