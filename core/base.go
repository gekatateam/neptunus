package core

import (
	"log/slog"
	"reflect"
	"time"

	"github.com/gekatateam/neptunus/metrics"
)

var (
	KindCore       = reflect.ValueOf(BaseCore{}).Type().Name()
	KindInput      = reflect.ValueOf(BaseInput{}).Type().Name()
	KindProcessor  = reflect.ValueOf(BaseProcessor{}).Type().Name()
	KindOutput     = reflect.ValueOf(BaseOutput{}).Type().Name()
	KindFilter     = reflect.ValueOf(BaseFilter{}).Type().Name()
	KindParser     = reflect.ValueOf(BaseParser{}).Type().Name()
	KindSerializer = reflect.ValueOf(BaseSerializer{}).Type().Name()
	KindKeykeeper  = reflect.ValueOf(BaseKeykeeper{}).Type().Name()
)

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

	Log  *slog.Logger
	Obs  metrics.ObserveFunc
	In   <-chan *Event
	Out  chan<- *Event
	Drop chan<- *Event
}

func (b *BaseProcessor) SetChannels(in <-chan *Event, out chan<- *Event, drop chan<- *Event) {
	b.In = in
	b.Out = out
	b.Drop = drop
}

func (b *BaseProcessor) Observe(status metrics.EventStatus, dur time.Duration) {
	b.Obs(b.Plugin, b.Alias, b.Pipeline, status, dur)
}

type BaseOutput struct {
	Alias    string
	Plugin   string
	Pipeline string

	Log  *slog.Logger
	Obs  metrics.ObserveFunc
	In   <-chan *Event
	Done chan<- *Event
}

func (b *BaseOutput) SetChannels(in <-chan *Event, done chan<- *Event) {
	b.In = in
	b.Done = done
}

func (b *BaseOutput) Observe(status metrics.EventStatus, dur time.Duration) {
	b.Obs(b.Plugin, b.Alias, b.Pipeline, status, dur)
}

type BaseFilter struct {
	Alias    string
	Plugin   string
	Pipeline string

	Reverse bool

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

	if b.Reverse {
		b.Rej = accepted
		b.Acc = rejected
	}
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

type BaseKeykeeper struct {
	Alias    string
	Plugin   string
	Pipeline string

	Log *slog.Logger
}
