package unit

// soft (consistency) units does not guarantee the order of processing,
// because of async filtering, but (planned to be) fast

import (
	"strconv"
	"sync"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

// fToCh stores a pair of a filter and it's acceptsChan output
// which is a next filter/processor/output input
// because Go does not guarantee maps ordering
type fToCh struct {
	f core.Filter
	c chan<- *core.Event
}

type procSoftUnit struct {
	p    core.Processor
	f    []fToCh
	wg   *sync.WaitGroup
	in   <-chan *core.Event
	out  chan<- *core.Event
	drop chan *core.Event
}

func newProcessorSoftUnit(p core.Processor, f []core.Filter, in <-chan *core.Event, bufferSize int) (unit *procSoftUnit, unitOut <-chan *core.Event, chansStats []metrics.ChanStatsFunc) {
	out := make(chan *core.Event, bufferSize)
	drop := make(chan *core.Event, bufferSize)
	unit = &procSoftUnit{
		p:    p,
		f:    make([]fToCh, 0, len(f)),
		wg:   &sync.WaitGroup{},
		in:   in,
		out:  out,
		drop: drop,
	}

	for _, filter := range f {
		chansStats = append(chansStats, registerChan(in, filter, metrics.ChanIn, core.KindFilter))
		acceptsChan := make(chan *core.Event, bufferSize)
		filter.SetChannels(in, unit.out, acceptsChan)
		in = acceptsChan // connect current filter success output to next filter/plugin input
		unit.f = append(unit.f, fToCh{filter, acceptsChan})
	}

	chansStats = append(chansStats, registerChan(in, p, metrics.ChanIn, core.KindProcessor))
	chansStats = append(chansStats, registerChan(drop, p, metrics.ChanDrop, core.KindProcessor))

	p.SetChannels(in, unit.out, unit.drop)
	return unit, out, chansStats
}

// starts events processing
// this function returns when the input channel is closed and read out to the end
// in the end, it closes the outgoing channel, which is incoming for a next consumer in a pipeline
func (u *procSoftUnit) Run() {
	// run filters
	// first filter will stop when its input channel - u.in - closes
	// the input channel for the next filter is closed at the exit of the current filter goroutine
	for _, v := range u.f {
		u.wg.Add(1)
		go func(f core.Filter, c chan<- *core.Event) {
			f.Run() // blocking call, loop inside
			close(c)
			f.Close()
			u.wg.Done()
		}(v.f, v.c)
	}

	// consume dropped events
	u.wg.Add(1)
	go func() {
		for e := range u.drop {
			e.Done()
		}
		u.wg.Done()
	}()

	// run processor
	// processor will stop when its input channel closes
	// it will happen when all filters are stopped
	// or, if no filters are set, when the u.in channel is closed
	u.p.Run() // blocking call, loop inside
	u.p.Close()
	close(u.drop) // close drop events channel

	// then, we wait until all goroutines are finished
	// and close the outgoing channel - which is the incoming channel for the next processor or output
	u.wg.Wait()
	close(u.out)
}

type outSoftUnit struct {
	o    core.Output
	f    []fToCh
	wg   *sync.WaitGroup
	in   <-chan *core.Event
	rej  chan *core.Event // rejected events doesn't processed anymore
	done chan *core.Event // done events are completely processed
}

func newOutputSoftUnit(o core.Output, f []core.Filter, in <-chan *core.Event, bufferSize int) (unit *outSoftUnit, chansStats []metrics.ChanStatsFunc) {
	unit = &outSoftUnit{
		o:    o,
		f:    make([]fToCh, 0, len(f)),
		wg:   &sync.WaitGroup{},
		in:   in,
		rej:  make(chan *core.Event, bufferSize),
		done: make(chan *core.Event, bufferSize),
	}

	for _, filter := range f {
		chansStats = append(chansStats, registerChan(in, filter, metrics.ChanIn, core.KindFilter))
		acceptsChan := make(chan *core.Event, bufferSize)
		filter.SetChannels(in, unit.rej, acceptsChan)
		in = acceptsChan
		unit.f = append(unit.f, fToCh{filter, acceptsChan})
	}

	chansStats = append(chansStats, registerChan(in, o, metrics.ChanIn, core.KindOutput))
	chansStats = append(chansStats, registerChan(unit.rej, o, metrics.ChanDrop, core.KindOutput))
	chansStats = append(chansStats, registerChan(unit.done, o, metrics.ChanDone, core.KindOutput))

	o.SetChannels(in, unit.done)
	return unit, chansStats
}

func (u *outSoftUnit) Run() {
	// run filters
	for _, v := range u.f {
		u.wg.Add(1)
		go func(f core.Filter, c chan<- *core.Event) {
			f.Run() // blocking call, loop inside
			close(c)
			f.Close()
			u.wg.Done()
		}(v.f, v.c)
	}

	// consume rejected events
	u.wg.Add(1)
	go func() {
		for e := range u.rej {
			e.Done()
		}
		u.wg.Done()
	}()

	// consume processed events
	u.wg.Add(1)
	go func() {
		for e := range u.done {
			e.Done()
		}
		u.wg.Done()
	}()

	// run output
	u.o.Run() // blocking call, loop inside
	u.o.Close()
	close(u.rej)  // close rejected events chan
	close(u.done) // close done events chan

	u.wg.Wait()
}

type inSoftUnit struct {
	i    core.Input
	f    []fToCh
	wg   *sync.WaitGroup
	out  chan<- *core.Event // first channel in chain
	rej  chan *core.Event
	stop <-chan struct{}
}

func newInputSoftUnit(i core.Input, f []core.Filter, stop <-chan struct{}, bufferSize int) (unit *inSoftUnit, unitOut <-chan *core.Event, chansStats []metrics.ChanStatsFunc) {
	out := make(chan *core.Event, bufferSize)
	unit = &inSoftUnit{
		i:    i,
		wg:   &sync.WaitGroup{},
		out:  out,
		rej:  make(chan *core.Event, bufferSize),
		stop: stop,
	}
	i.SetChannels(out)

	for _, filter := range f {
		chansStats = append(chansStats, registerChan(out, filter, metrics.ChanIn, core.KindFilter))
		acceptsChan := make(chan *core.Event, bufferSize)
		filter.SetChannels(out, unit.rej, acceptsChan)
		out = acceptsChan
		unit.f = append(unit.f, fToCh{filter, acceptsChan})
	}

	chansStats = append(chansStats, registerChan(unit.rej, i, metrics.ChanDrop, core.KindInput))

	return unit, out, chansStats
}

func (u *inSoftUnit) Run() {
	// run input
	u.wg.Add(1)
	go func() {
		u.i.Run()    // blocking call, loop inside
		close(u.out) // close first channel in unit chain (trigger filters to close)
		u.wg.Done()
	}()

	// run filters
	for _, v := range u.f {
		u.wg.Add(1)
		go func(f core.Filter, c chan<- *core.Event) {
			f.Run() // blocking call, loop inside
			close(c)
			f.Close()
			u.wg.Done()
		}(v.f, v.c)
	}

	rejWg := &sync.WaitGroup{}
	rejWg.Add(1)
	go func() {
		for e := range u.rej {
			e.Done()
		}
		rejWg.Done()
	}()

	<-u.stop     // wait for stop signal
	u.i.Close()  // then close the input
	u.wg.Wait()  // wait for all goroutines stopped
	close(u.rej) // rejected chan can be closed only when all filters stopped
	rejWg.Wait() // and wait for rejection goroutine
}

type bcastSoftUnit struct {
	c    core.Broadcast
	in   <-chan *core.Event
	outs []chan<- *core.Event
}

func newBroadcastSoftUnit(c core.Broadcast, in <-chan *core.Event, outsCount, bufferSize int) (unit *bcastSoftUnit, unitOuts []<-chan *core.Event, chansStats []metrics.ChanStatsFunc) {
	outs := make([]<-chan *core.Event, 0, outsCount)
	unit = &bcastSoftUnit{
		c:    c,
		in:   in,
		outs: make([]chan<- *core.Event, 0, outsCount),
	}

	for i := 0; i < outsCount; i++ {
		outCh := make(chan *core.Event, bufferSize)
		outs = append(outs, outCh)
		unit.outs = append(unit.outs, outCh)
	}
	c.SetChannels(in, unit.outs)

	chansStats = append(chansStats, registerChan(in, c, metrics.ChanIn, core.KindCore))

	return unit, outs, chansStats
}

func (u *bcastSoftUnit) Run() {
	// starts consumer which will broadcast each event
	// to all outputs
	// this loop breaks when the input channel closes
	u.c.Run()
	// close all outputs
	for _, out := range u.outs {
		close(out)
	}
}

type fusionSoftUnit struct {
	c   core.Fusion
	ins []<-chan *core.Event
	out chan<- *core.Event
}

func newFusionSoftUnit(c core.Fusion, ins []<-chan *core.Event, bufferSize int) (unit *fusionSoftUnit, unitOut <-chan *core.Event, chansStats []metrics.ChanStatsFunc) {
	out := make(chan *core.Event, bufferSize)
	unit = &fusionSoftUnit{
		c:   c,
		ins: ins,
		out: out,
	}
	c.SetChannels(ins, out)

	for i, in := range ins {
		chansStats = append(chansStats, registerChan(in, c, metrics.ChanIn, core.KindCore, strconv.Itoa(i)))
	}

	return unit, out, chansStats
}

func (u *fusionSoftUnit) Run() {
	u.c.Run()
	close(u.out)
}

type mixerSoftUnit struct {
	c   core.Mixer
	in  <-chan *core.Event
	out chan<- *core.Event
}

func newMixerSoftUnit(c core.Mixer, in <-chan *core.Event, bufferSize int) (unit *mixerSoftUnit, unitOut <-chan *core.Event, chansStats []metrics.ChanStatsFunc) {
	out := c.OutChan()
	if out == nil {
		out = make(chan *core.Event, bufferSize)
	}

	unit = &mixerSoftUnit{
		c:   c,
		in:  in,
		out: out,
	}
	c.AppendChannels(in, out)

	chansStats = append(chansStats, registerChan(in, c, metrics.ChanIn, core.KindProcessor, strconv.Itoa(c.IncrIndex())))

	return unit, out, chansStats
}

func (u *mixerSoftUnit) Run() {
	u.c.Run()
}
