package core

// soft (consistency) units does not guarantee the order of processing, but (planned to be) fast

import "sync"

// fToCh stores a pair of a filter and it's acceptsChan output
// which is a next filter/processor/output input
// because Go does not guarantee maps ordering
type fToCh struct {
	f Filter
	c chan<- *Event
}

// processor unit consumes events from input channel
// if filters are set, each event passes through them
// rejected events are going to unit output
// accepted events are going to next filter or processor
//	┌────────────────┐
//	|┌───┐           |
// ─┼┤ f ├┬─────────┐|
//	|└─┬┬┴┴─┐ ┌────┐||
//	|  └┤ f ├─┤proc├┴┼─
//	|   └───┘ └────┘ |
//	└────────────────┘
type procSoftUnit struct {
	p   Processor
	f   []fToCh
	wg  *sync.WaitGroup
	in  <-chan *Event
	out chan<- *Event
}

func NewProcessorSoftUnit(p Processor, f []Filter, out chan<- *Event) (unit *procSoftUnit, unitInput chan<- *Event) {
	in := make(chan *Event, bufferSize) // unit input channel
	unitInput = in
	unit = &procSoftUnit{
		p:   p,
		f:   make([]fToCh, len(f)),
		wg:  &sync.WaitGroup{},
		in:  in,
		out: out, // also channel for rejected events
	}

	// initialize filters
	for _, filter := range f {
		acceptsChan := make(chan *Event, bufferSize)
		filter.Init(in, unit.out, acceptsChan)
		in = acceptsChan // connect current filter success output to next filter/plugin input
		unit.f = append(unit.f, fToCh{filter, acceptsChan})
	}

	p.Init(in, unit.out)
	return unit, unitInput
}

// starts events processing
// this function returns when the input channel is closed and read out to the end
// in the end, it closes the outgoing channel, which is incoming for a next consumer in a pipeline
func (u *procSoftUnit) Run() {
	// run fliters
	// first filter will stop when its input channel - u.in - closes
	// the input channel for the next filter is closed at the exit of the current filter goroutine
	for _, v := range u.f {
		u.wg.Add(1)
		go func(f Filter, c chan<- *Event) {
			f.Filter() // blocking call, loop inside
			close(c)
			f.Close()
			u.wg.Done()
		}(v.f, v.c)
	}

	// run processor
	// processor will stop when its input channel closes
	// it will happen when all filters are stopped
	// or, if no filters are set, when the u.in channel is closed
	u.p.Process() // blocking call, loop inside
	u.p.Close()
	u.wg.Done()

	// then, we wait until all goruntins are finished
	// and close the outgoing channel - which is the incoming channel for the next processor or output
	u.wg.Wait()
	close(u.out)
}

// output unit consumes events from input channel
// if filters are set, each event passes through them
// rejected events are not going to next filter or output
//	┌────────────────┐
//	|┌───┐           |
// ─┼┤ f ├┬────────Θ |
//	|└─┬┬┴┴─┐ ┌────┐ |
//	|  └┤ f ├─┤out>| |
//	|   └───┘ └────┘ |
//	└────────────────┘
type outSoftUnit struct {
	o   Output
	f   []fToCh
	wg  *sync.WaitGroup
	in  <-chan *Event
	rej chan *Event // rejected events doesn't processed anymore
}

func NewOutputSoftUnit(o Output, f []Filter) (unit *outSoftUnit, unitInput chan<- *Event) {
	in := make(chan *Event, bufferSize)
	unitInput = in
	unit = &outSoftUnit{
		o:   o,
		f:   make([]fToCh, len(f)),
		wg:  &sync.WaitGroup{},
		in:  in,
		rej: make(chan *Event, bufferSize),
	}

	// initialize filters
	for _, filter := range f {
		acceptsChan := make(chan *Event, bufferSize)
		filter.Init(in, unit.rej, acceptsChan)
		in = acceptsChan
		unit.f = append(unit.f, fToCh{filter, acceptsChan})
	}

	o.Init(in)
	return unit, unitInput
}

func (u *outSoftUnit) Run() {
	// run fliters
	for _, v := range u.f {
		u.wg.Add(1)
		go func(f Filter, c chan<- *Event) {
			f.Filter() // blocking call, loop inside
			close(c)
			f.Close()
			u.wg.Done()
		}(v.f, v.c)
	}

	// consume rejected events to nothing
	u.wg.Add(1)
	go func() {
		for range u.rej {
		}
		u.wg.Done()
	}()

	// run output
	u.o.Listen() // blocking call, loop inside
	u.o.Close()
	close(u.rej) // close rejected events chan

	u.wg.Wait()
}

// input unit sends consumed events to output channel
// input units are not like the others:
// - they do not close their output channels (to avoid goroutines hell)
// - they do not use filters
// - they wait for the closing signal through a dedicated channel
// ┌───────┐
// | ┌───┐ |
// | |>in├─┼─
// | └───┘ |
// └───────┘
type inSoftUnit struct {
	i    Input
	wg   *sync.WaitGroup
	out  chan<- *Event // first channel in chain
	stop chan struct{}
}

func NewInputSoftUnit(i Input, out chan<- *Event) (unit *inSoftUnit, stop chan<- struct{}) {
	unit = &inSoftUnit{
		i:    i,
		wg:   &sync.WaitGroup{},
		out:  out,
		stop: make(chan struct{}),
	}
	i.Init(out)

	return unit, unit.stop
}

func (u *inSoftUnit) Run() {
	// wait for stop signal
	// then close the input
	u.wg.Add(1)
	go func() {
		<-u.stop
		u.i.Close()
		u.wg.Done()
	}()

	// run input
	u.i.Serve() // blocking call, loop inside

	u.wg.Wait()
}

// broadcast unit consumes events from input
// and sends clones of each event to all outputs
//	┌────────┐
//	|   ┌────┼─
// ─┼───█────┼─
//	|   └────┼─
//	└────────┘
type bcastSoftUnit struct {
	in   <-chan *Event
	outs []chan<- *Event
}

func NewBroadcastSoftUnit(outs ...chan<- *Event) (unit *bcastSoftUnit, unitInput chan<- *Event) {
	in := make(chan *Event, bufferSize)
	unit = &bcastSoftUnit{
		in:   in,
		outs: outs,
	}

	return unit, in
}

func (u *bcastSoftUnit) Run() {
	// starts consumer which will broadcast each event
	// to all outputs
	// this loop breaks when the input channel closes
	for e := range u.in {
		for _, out := range u.outs {
			out <- e.Clone()
		}
	}
	// close all outputs
	for _, out := range u.outs {
		close(out)
	}
}
