package dynamicgrpc

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"google.golang.org/grpc/peer"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/gekatateam/protomap"
	"github.com/gekatateam/protomap/interceptors"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type subscription struct {
	id     uint64
	msgs   chan proto.Message
	result chan error

	rpc  string
	peer *peer.Peer
}

type router struct {
	mu   *sync.Mutex
	wg   *sync.WaitGroup
	pubs map[string]*publisher

	closed           bool
	newPublisherFunc func(rpc string) *publisher
}

func (r *router) _get(rpc string) *publisher {
	pub, ok := r.pubs[rpc]
	if !ok {
		pub = r.newPublisherFunc(rpc)
		r.pubs[rpc] = pub
		r.wg.Go(pub.run)
	}

	return pub
}

func (r *router) push(e *core.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		panic("pushing to closed router")
	}

	r._get(e.RoutingKey).events <- e
}

func (r *router) close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true

	for _, pub := range r.pubs {
		close(pub.events)
	}
}

func (r *router) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, pub := range r.pubs {
		close(pub.stop)
	}
}

// Freakin dirty hack inside - close sub input chan as an indication of router shutdown.
// If any new subscriber arrives after the router is closed, it will receive a closed input chan
// and can return from stream immediately without waiting for new events that will never arrive.
func (r *router) subscribe(rpc string, peer *peer.Peer) subscription {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub := subscription{
		id:     rand.Uint64(),
		msgs:   make(chan proto.Message),
		result: make(chan error),
		rpc:    rpc,
		peer:   peer,
	}

	if r.closed {
		close(sub.msgs)
		return sub
	}

	pub := r._get(rpc)
	pub.subscription <- sub

	return sub
}

func (r *router) unsubscribe(sub subscription) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pub, ok := r.pubs[sub.rpc]
	if !ok {
		panic("unsubscribing from non existing publisher: " + sub.rpc)
	}

	pub.unsubscription <- sub
}

type publisher struct {
	*core.BaseOutput

	behavior    string
	rpc         string
	waitForSubs bool
	subs        []subscription
	respMsg     protoreflect.MessageDescriptor

	stop           chan struct{}
	events         chan *core.Event
	subscription   chan subscription
	unsubscription chan subscription
}

func (p *publisher) run() {
PUBLISHER_MAIN_LOOP:
	for {
		select {
		case sub := <-p.subscription:
			p._subscribe(sub)
		case sub := <-p.unsubscription:
			p._unsubscribe(sub)
		default:
		}

		select {
		case sub := <-p.subscription:
			p._subscribe(sub)
		case sub := <-p.unsubscription:
			p._unsubscribe(sub)
		case <-p.stop:
			if len(p.subs) != 0 {
				p.Log.Info(fmt.Sprintf("waiting for %v subscribers to leave", len(p.subs)),
					"procedure", p.rpc,
				)
				time.Sleep(time.Second)
				continue PUBLISHER_MAIN_LOOP
			}

			return
		case event, ok := <-p.events:
			if !ok {
				for _, sub := range p.subs {
					close(sub.msgs)
				}
				p.events = nil
				continue PUBLISHER_MAIN_LOOP
			}

			if len(p.subs) == 0 {
				p.Log.Debug("no subscribers, event dropped",
					"procedure", p.rpc,
					elog.EventGroup(event),
				)
				p.Done <- event
				p.Observe(metrics.EventAccepted, 0)
				continue PUBLISHER_MAIN_LOOP
			}

			switch p.behavior {
			case behaviourBroadcast:
				p._publishBroadcast(event)
			case behaviourRandom:
				p._publishRandom(event)
			}
		}
	}
}

func (p *publisher) _subscribe(sub subscription) {
	p.Log.Info("new subscriber arrived",
		"procedure", p.rpc,
		"peer", sub.peer.String(),
	)
	p.subs = append(p.subs, sub)
}

func (p *publisher) _unsubscribe(sub subscription) {
	p.Log.Info("subscriber left",
		"procedure", p.rpc,
		"peer", sub.peer.String(),
	)
	p.subs = slices.DeleteFunc(p.subs, func(s subscription) bool {
		return s.id == sub.id
	})
}

func (p *publisher) _publishBroadcast(event *core.Event) {
	now := time.Now()

	m := dynamicpb.NewMessage(p.respMsg)
	if err := protomap.AnyToMessage(event.Data, m, interceptors.DurationEncoder, interceptors.TimeEncoder); err != nil {
		p.Log.Error("message encoding failed, event skipped",
			"error", err,
			"procedure", p.rpc,
			elog.EventGroup(event),
		)
		p.Done <- event
		p.Observe(metrics.EventFailed, time.Since(now))
		return
	}

	p.Log.Debug("message encoded",
		"procedure", p.rpc,
		elog.EventGroup(event),
	)

	hasError := false
	for _, sub := range p.subs {
		select {
		case sub.msgs <- m:
		case _, ok := <-sub.result:
			if !ok { // subscriber already gone and will be removed in the next iterations
				continue
			}
		}

		if err := <-sub.result; err != nil {
			hasError = true
			p.Log.Warn("message sending failed",
				"error", err,
				"peer", sub.peer.String(),
				elog.EventGroup(event),
			)
			continue
		}

		p.Log.Debug("message sent",
			"procedure", p.rpc,
			"peer", sub.peer.String(),
			elog.EventGroup(event),
		)
	}

	if hasError {
		p.Log.Error("event delivery to some subscribers failed",
			"procedure", p.rpc,
			elog.EventGroup(event),
		)
	}

	p.Done <- event
	p.Observe(metrics.EventAccepted, time.Since(now))
}

func (p *publisher) _publishRandom(event *core.Event) {
	now := time.Now()

	m := dynamicpb.NewMessage(p.respMsg)
	if err := protomap.AnyToMessage(event.Data, m, interceptors.DurationEncoder, interceptors.TimeEncoder); err != nil {
		p.Log.Error("message encoding failed, event skipped",
			"error", err,
			"procedure", p.rpc,
			elog.EventGroup(event),
		)
		p.Done <- event
		p.Observe(metrics.EventFailed, time.Since(now))
		return
	}

	p.Log.Debug("message encoded",
		"procedure", p.rpc,
		elog.EventGroup(event),
	)

	subs := slices.Clone(p.subs)
	rand.Shuffle(len(subs), func(i, j int) {
		subs[i], subs[j] = subs[j], subs[i]
	})

	for _, sub := range subs {
		select {
		case sub.msgs <- m:
		case _, ok := <-sub.result:
			if !ok { // subscriber already gone and will be removed in the next iterations
				continue
			}
		}

		err := <-sub.result
		if err == nil {
			p.Log.Debug("message sent",
				"procedure", p.rpc,
				"peer", sub.peer.String(),
				elog.EventGroup(event),
			)
			p.Done <- event
			p.Observe(metrics.EventAccepted, time.Since(now))
			return
		}

		p.Log.Warn("message sending failed",
			"error", err,
			"peer", sub.peer.String(),
			elog.EventGroup(event),
		)
	}

	p.Log.Error("event delivery to all subscribers failed",
		"procedure", p.rpc,
		elog.EventGroup(event),
	)
	p.Done <- event
	p.Observe(metrics.EventFailed, time.Since(now))
}
