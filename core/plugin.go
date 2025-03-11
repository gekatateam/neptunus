package core

import (
	"io"
)

type Initer interface {
	Init() error
}

type Runner interface {
	Run()
}

// input plugin consumes events from outer world
type Input interface {
	SetChannels(out chan<- *Event)
	io.Closer
	Runner
	Initer
}

// filter plugin sorts events by conditions
type Filter interface {
	SetChannels(in <-chan *Event, rejected chan<- *Event, accepted chan<- *Event)
	io.Closer
	Runner
	Initer
}

// processor plugin transforms events
type Processor interface {
	SetChannels(in <-chan *Event, out chan<- *Event, drop chan<- *Event)
	io.Closer
	Runner
	Initer
}

// output plugin produces events to outer world
type Output interface {
	SetChannels(in <-chan *Event, done chan<- *Event)
	io.Closer
	Runner
	Initer
}

// parser plugin parses raw format data into events
type Parser interface {
	Parse(data []byte, routingKey string) ([]*Event, error)
	io.Closer
	Initer
}

// serializer plugin serializes events into configured format
type Serializer interface {
	Serialize(event ...*Event) ([]byte, error)
	io.Closer
	Initer
}

type Keykeeper interface {
	Get(key string) (any, error)
	io.Closer
	Initer
}

// plugins that need parsers must implement this interface
type SetParser interface {
	SetParser(p Parser)
}

// plugins that need serializers must implement this interface
type SetSerializer interface {
	SetSerializer(s Serializer)
}

// plugins that need unique id must implement this interface
// id is unique for each plugin, but it's same for one processor
// in multiple lines
// id is randomly generated at application startup
type SetId interface {
	SetId(id uint64)
}

// processor that need it's line number must implement this interface
type SetLine interface {
	SetLine(line int)
}

// core plugins
// used in core units only
type Fusion interface {
	SetChannels(ins []<-chan *Event, out chan<- *Event)
	Runner
}

type Broadcast interface {
	SetChannels(in <-chan *Event, outs []chan<- *Event)
	Runner
}
