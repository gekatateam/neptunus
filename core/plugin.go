package core

import (
	"io"
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/metrics"
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
	SetChannels(in <-chan *Event, out chan<- *Event)
	io.Closer
	Runner
	Initer
}

// output plugin produces events to outer world
type Output interface {
	SetChannels(in <-chan *Event)
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


type BaseInput struct {
	Alias    string
	Plugin   string
	Pipeline string
	
	Log *slog.Logger
	Obs metrics.ObserveFunc
	Out chan<- *Event
}

func (b *BaseInput) SetChannels(out chan<- *Event) {
	b.Out = out
}

func (b *BaseInput) Observe(status metrics.EventStatus, dur time.Duration) {
	b.Obs(b.Plugin, b.Alias, b.Pipeline, status, dur)
}

type BaseProcessor struct {
	Alias    string
	Plugin   string
	Pipeline string
	
	Log *slog.Logger
	Obs metrics.ObserveFunc
	In  <-chan *Event
	Out chan<- *Event
}

func (b *BaseProcessor) SetChannels(in <-chan *Event, out chan<- *Event) {
	b.In  = in
	b.Out = out
}

func (b *BaseProcessor) Observe(status metrics.EventStatus, dur time.Duration) {
	b.Obs(b.Plugin, b.Alias, b.Pipeline, status, dur)
}

type BaseOutput struct {
	Alias    string
	Plugin   string
	Pipeline string
	
	Log *slog.Logger
	Obs metrics.ObserveFunc
	In  <-chan *Event
}

func (b *BaseOutput) SetChannels(in <-chan *Event) {
	b.In = in
}

func (b *BaseOutput) Observe(status metrics.EventStatus, dur time.Duration) {
	b.Obs(b.Plugin, b.Alias, b.Pipeline, status, dur)
}

type BaseFilter struct {
	Alias    string
	Plugin   string
	Pipeline string
	
	Log *slog.Logger
	Obs metrics.ObserveFunc
	In  <-chan *Event
	Rej chan<- *Event
	Acc chan<- *Event
}

func (b *BaseFilter) SetChannels(in <-chan *Event, rejected chan<- *Event, accepted chan<- *Event) {
	b.In = in
	b.Rej = rejected
	b.Acc = accepted
}

func (b *BaseFilter) Observe(status metrics.EventStatus, dur time.Duration) {
	b.Obs(b.Plugin, b.Alias, b.Pipeline, status, dur)
}

type BaseParser struct {
	Alias    string
	Plugin   string
	Pipeline string
	
	Log *slog.Logger
	Obs metrics.ObserveFunc
}

func (b *BaseParser) Observe(status metrics.EventStatus, dur time.Duration) {
	b.Obs(b.Plugin, b.Alias, b.Pipeline, status, dur)
}

type BaseSerializer struct {
	Alias    string
	Plugin   string
	Pipeline string
	
	Log *slog.Logger
	Obs metrics.ObserveFunc
}

func (b *BaseSerializer) Observe(status metrics.EventStatus, dur time.Duration) {
	b.Obs(b.Plugin, b.Alias, b.Pipeline, status, dur)
}

type BaseCore struct {
	Alias    string
	Plugin   string
	Pipeline string
	
	Log *slog.Logger
	Obs metrics.ObserveFunc
}

func (b *BaseCore) Observe(status metrics.EventStatus, dur time.Duration) {
	b.Obs(b.Plugin, b.Alias, b.Pipeline, status, dur)
}
