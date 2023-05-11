package core

// soft (consistency) units does not guarantee the order of processing, but quickly handles events

import "sync"

type procSoftUnit struct {
	p   Processor
	f   map[Filter]chan<- *Event
	wg  *sync.WaitGroup
	in  <-chan *Event
	out chan<- *Event
}

func NewProcessorSoftUnit(p Processor, f []Filter, out chan<- *Event) (unit *procSoftUnit, unitInput chan<- *Event) {
	in := make(chan *Event, bufferSize) // unit input channel
	unitInput = in
	unit = &procSoftUnit{
		p:   p,
		f:   make(map[Filter]chan<- *Event, len(f)),
		wg:  &sync.WaitGroup{},
		in:  in,
		out: out, // also channel for rejected events
	}

	// initialize filters
	for _, filter := range f {
		acceptsChan := make(chan *Event, bufferSize)
		filter.Init(in, unit.out, acceptsChan)
		in = acceptsChan // connect current filter success output to next filter/plugin input
		unit.f[filter] = acceptsChan
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
	for filter, nextChan := range u.f {
		u.wg.Add(1)
		go func(f Filter, c chan<- *Event) {
			f.Filter() // blocking call, loop inside
			close(c)
			f.Close()
			u.wg.Done()
		}(filter, nextChan)
	}

	// run processor
	// processor will stop when its input channel closes
	// it will happen when all filters are stopped
	// or, if no filters are set, when the u.in channel is closed
	u.wg.Add(1)
	go func() {
		u.p.Process() // blocking call, loop inside
		u.p.Close()
		u.wg.Done()
	}()

	// then, we wait until all goruntins are finished
	// and close the outgoing channel - which is the incoming channel for the next processor or output
	u.wg.Wait()
	close(u.out)
}

type outSoftUnit struct {
	o   Output
	f   map[Filter]chan<- *Event
	wg  *sync.WaitGroup
	in  <-chan *Event
	rej chan *Event // rejected events doesn't processed anymore
}

func NewOutputSoftUnit(o Output, f []Filter) (unit *outSoftUnit, unitInput chan<- *Event) {
	in := make(chan *Event, bufferSize)
	unitInput = in
	unit = &outSoftUnit{
		o:   o,
		f:   make(map[Filter]chan<- *Event, len(f)),
		wg:  &sync.WaitGroup{},
		in:  in,
		rej: make(chan *Event, bufferSize),
	}

	// initialize filters
	for _, filter := range f {
		acceptsChan := make(chan *Event, bufferSize)
		filter.Init(in, unit.rej, acceptsChan)
		in = acceptsChan
		unit.f[filter] = acceptsChan
	}

	o.Init(in)
	return unit, unitInput
}

func (u *outSoftUnit) Run() {
	// run fliters
	for filter, nextChan := range u.f {
		u.wg.Add(1)
		go func(f Filter, c chan<- *Event) {
			f.Filter() // blocking call, loop inside
			close(c)
			f.Close()
			u.wg.Done()
		}(filter, nextChan)
	}

	// run output
	u.wg.Add(1)
	go func() {
		u.o.Listen() // blocking call, loop inside
		u.o.Close()
		u.wg.Done()
		close(u.rej) // close rejected events chan
	}()

	// consume rejected events to nothing
	u.wg.Add(1)
	go func() {
		for range u.rej {
		}
		u.wg.Done()
	}()

	u.wg.Wait()
}

type inSoftUnit struct {
	i      Input
	f      map[Filter]chan<- *Event
	wg     *sync.WaitGroup
	main   chan<- *Event // first channel in chain
	rej    chan *Event

	stopCh  chan struct{}
	iStopCh chan<- struct{}
	iDoneCh <-chan struct{}
}

func NewInputSoftUnit(i Input, f []Filter, out chan<- *Event) (unit *inSoftUnit, stop chan<- struct{}) {
	unit = &inSoftUnit{
		i:      i,
		wg:     &sync.WaitGroup{},
		f:      make(map[Filter]chan<- *Event, len(f)),
		stopCh: make(chan struct{}),
		rej:    make(chan *Event, bufferSize),
	}
	stop = unit.stopCh

	if len(f) > 0 {
		filtersIn := make(chan *Event, bufferSize)
		unit.iStopCh, unit.iDoneCh = i.Init(filtersIn)	
		unit.main = filtersIn
		for j := 0; j < len(f) - 1; j++ {
			acceptsChan := make(chan *Event, bufferSize)
			f[j].Init(filtersIn, unit.rej, acceptsChan)
			filtersIn = acceptsChan
			unit.f[f[j]] = acceptsChan
		}
		f[len(f) - 1].Init(filtersIn, unit.rej, out)
		unit.f[f[len(f) - 1]] = out
	} else {
		unit.iStopCh, unit.iDoneCh = i.Init(out)
		unit.main = out
	}

	return unit, stop
}

func (u *inSoftUnit) Run() {
	// wait for stop signal
	// then send stop signal to input
	// wait for done signal from input
	// and close main channel
	u.wg.Add(1)
	go func() {
		<- u.stopCh
		u.iStopCh <- struct{}{}
		<- u.iDoneCh
		close(u.main)
		u.wg.Done()
	}()

	// run input
	// it is possible that input is already writing messages 
	// to its own out channel, from which the filter(s) do not yet read
	u.wg.Add(1)
	go func() {
		u.i.Serve() // blocking call, loop inside
		u.i.Close()
		u.wg.Done()
	}()

	// run fliters
	for filter, nextChan := range u.f {
		u.wg.Add(1)
		go func(f Filter, c chan<- *Event) {
			f.Filter() // blocking call, loop inside
			close(c)
			f.Close()
			u.wg.Done()
		}(filter, nextChan)
	}

	// consume rejected events to nothing
	// TODO handle filters stopping and close unit.rej cahnnel
	// u.wg.Add(1)
	go func() {
		for range u.rej {
		}
		// u.wg.Done()
	}()

	u.wg.Wait()
}
