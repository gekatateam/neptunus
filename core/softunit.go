package core

import "sync"

// soft unit does not guarantee the order of processing, but quickly handles events
type softProcessorUnit struct {
	p   Processor
	f   map[Filter]chan *Event
	wg  *sync.WaitGroup
	in  <-chan *Event
	out chan<- *Event
}

func NewSoftProcessorUnit(p Processor, f []Filter, in <-chan *Event) (*softProcessorUnit, chan *Event) {
	outputChan := make(chan *Event, bufferSize) // also channel for rejected events

	unit := &softProcessorUnit{
		p: p,
		f: make(map[Filter]chan *Event, len(f)),
		wg: &sync.WaitGroup{},
		in: in,
		out: outputChan,
	}

	// initialize filters
	for _, filter := range f {
		acceptsChan := make(chan *Event, bufferSize)
		filter.Init(in, unit.out, acceptsChan)
		in = acceptsChan
		unit.f[filter] = acceptsChan
	}

	p.Init(in, unit.out)
	return unit, outputChan
}

// starts events processing
// this function returns when the input channel is closed and read out to the end
// in the end, it closes the outgoing channel, which is incoming for a next consumer in a pipeline
func (u *softProcessorUnit) Run() {
	// run fliters
	// first filter will stop when its input channel - u.in - closes
	// the input channel for the next filter is closed at the exit of the current filter goroutine
	for filter, nextChan := range u.f {
		u.wg.Add(1)
		go func(f Filter, c chan *Event) {
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
