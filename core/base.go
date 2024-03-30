package core

import (
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/metrics"
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

	Log *slog.Logger
	Obs metrics.ObserveFunc
	In  <-chan *Event
	Out chan<- *Event
}

func (b *BaseProcessor) SetChannels(in <-chan *Event, out chan<- *Event) {
	b.In = in
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

type BaseKeykeeper struct {
	Alias    string
	Plugin   string
	Pipeline string

	Log *slog.Logger
}
