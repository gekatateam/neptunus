package core

import "log/slog"

type Aliaser interface {
	Alias() string
}

type Initer interface {
	Init(config map[string]any, alias, pipeline string, log *slog.Logger) error
}

// input plugin consumes events from outer world
type Input interface {
	Prepare(out chan<- *Event)
	Run()
	Close() error
	Initer
	Aliaser
}

// filter plugin sorts events by conditions
type Filter interface {
	Prepare(in <-chan *Event, rejected chan<- *Event, accepted chan<- *Event)
	Run()
	Close() error
	Initer
	Aliaser
}

// processor plugin transforms events
type Processor interface {
	Prepare(in <-chan *Event, out chan<- *Event)
	Run()
	Close() error
	Initer
	Aliaser
}

// output plugin produces events to outer world
type Output interface {
	Prepare(in <-chan *Event)
	Run()
	Close() error
	Initer
	Aliaser
}

// parser plugin parses raw format data into events
type Parser interface {
	Parse(data []byte, routingKey string) ([]*Event, error)
	Close() error
	Initer
	Aliaser
}

// serializer plugin serializes events into configured format
type Serializer interface {
	Serialize(event ...*Event) ([]byte, error)
	Close() error
	Initer
	Aliaser
}

// plugins that need parsers must implement this interface
type ParserNeedy interface {
	SetParser(p Parser)
}

// plugins that need serializers must implement this interface
type SerializerNeedy interface {
	SetSerializer(s Serializer)
}

// plugins that need unique id must implement this interface
// id is unique for each plugin, but it's same for one processor
// in multiple lines
// id is randomly generated at application startup
type IdNeedy interface {
	SetId(id uint64)
}

// core plugins
// used in core units only
type Fusion interface {
	Prepare(ins []<-chan *Event, out chan<- *Event)
	Run()
}

type Broadcast interface {
	Prepare(in <-chan *Event, outs []chan<- *Event)
	Run()
}
