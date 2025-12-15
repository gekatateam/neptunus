package unit

import (
	"errors"
	"log/slog"
	"reflect"
	"strings"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

const (
	ConsistencySoft = "soft"
	ConsistencyHard = "hard"
)

// processor unit consumes events from input channel
// if filters are set, each event passes through them
// rejected events are going to unit output
// accepted events are going to next filter or processor
//
// ┌────────────────┐
// |┌───┐           |
// ┼┤ f ├┬─────────┐|
// |└─┬┬┴┴─┐ ┌────┐||
// |  └┤ f ├─┤proc├┴┼─
// |   └───┘ └────┘ |
// └────────────────┘
func NewProcessor(c *config.PipeSettings, l *slog.Logger, p core.Processor, f []core.Filter, in <-chan *core.Event) (unit core.Runner, unitOut <-chan *core.Event, chansStats []metrics.ChanStatsFunc) {
	switch c.Consistency {
	case ConsistencyHard:
		panic(errors.ErrUnsupported)
	default:
		return newProcessorSoftUnit(p, f, in, c.Buffer)
	}
}

// output unit consumes events from input channel
// if filters are set, each event passes through them
// rejected events are not going to next filter or output
//
// ┌────────────────┐
// |┌───┐           |
// ┼┤ f ├┬────────Θ |
// |└─┬┬┴┴─┐ ┌────┐ |
// |  └┤ f ├─┤out>| |
// |   └───┘ └────┘ |
// └────────────────┘
func NewOutput(c *config.PipeSettings, l *slog.Logger, o core.Output, f []core.Filter, in <-chan *core.Event) (unit core.Runner, chansStats []metrics.ChanStatsFunc) {
	switch c.Consistency {
	case ConsistencyHard:
		panic(errors.ErrUnsupported)
	default:
		return newOutputSoftUnit(o, f, in, c.Buffer)
	}
}

// input unit sends consumed events to output channel
// input unit wait for the closing signal through a dedicated channel
// if filters are set, each event passes through them
// rejected events are not going to next filter or processor
//
// ┌────────────────┐
// |┌───┐ ┌───┐     |
// ||>in├─┤ f ├┬──Θ |
// |└───┘ └─┬┬┴┴─┐  |
// |        └┤ f ├──┼─
// |         └───┘  |
// └────────────────┘
func NewInput(c *config.PipeSettings, l *slog.Logger, i core.Input, f []core.Filter, stop <-chan struct{}) (unit core.Runner, unitOut <-chan *core.Event, chansStats []metrics.ChanStatsFunc) {
	switch c.Consistency {
	case ConsistencyHard:
		panic(errors.ErrUnsupported)
	default:
		return newInputSoftUnit(i, f, stop, c.Buffer)
	}
}

// fan-out unit consumes events from input
// and sends clones of each event to all outputs
// this unit uses plugin for avoid concrete metrics writing in core
//
// ┌────────┐
// |   ┌────┼─
// ┼───█────┼─
// |   └────┼─
// └────────┘
func NewFanOut(c *config.PipeSettings, l *slog.Logger, b core.FanOut, in <-chan *core.Event, outsCount int) (unit core.Runner, unitOuts []<-chan *core.Event, chansStats []metrics.ChanStatsFunc) {
	switch c.Consistency {
	case ConsistencyHard:
		panic(errors.ErrUnsupported)
	default:
		return newFanOutSoftUnit(b, in, outsCount, c.Buffer)
	}
}

// fan-in unit consumes events from multiple inputs
// and sends them to one output channel
// this unit uses plugin for avoid concrete metrics writing in core
//
// ┌────────┐
// ┼───┐    |
// ┼───█────┼─
// ┼───┘    |
// └────────┘
func NewFanIn(c *config.PipeSettings, l *slog.Logger, f core.FanIn, ins []<-chan *core.Event) (unit core.Runner, unitOut <-chan *core.Event, chansStats []metrics.ChanStatsFunc) {
	switch c.Consistency {
	case ConsistencyHard:
		panic(errors.ErrUnsupported)
	default:
		return newFanInSoftUnit(f, ins, c.Buffer)
	}
}

// register plugin input channel in metrics system
func registerChan(ch <-chan *core.Event, p any, desc metrics.ChanDesc, kind string, extra ...string) metrics.ChanStatsFunc {
	var e string
	if len(extra) > 0 {
		e = "::" + strings.Join(extra, "::")
	}

	return func() metrics.ChanStats {
		return metrics.ChanStats{
			Capacity:   cap(ch),
			Length:     len(ch),
			Plugin:     pluginPlugin(p, kind),
			Name:       pluginAlias(p, kind) + e,
			Descriptor: desc,
		}
	}
}

// easy way to get plugin alias and type without boilerplate
// there is no affection on performance, because it's used only on startup
func pluginAlias(p any, kind string) string {
	return reflect.ValueOf(p).
		Elem().
		FieldByName(kind).
		Elem().
		FieldByName("Alias").
		String()
}

func pluginPlugin(p any, kind string) string {
	return reflect.ValueOf(p).
		Elem().
		FieldByName(kind).
		Elem().
		FieldByName("Plugin").
		String()
}
