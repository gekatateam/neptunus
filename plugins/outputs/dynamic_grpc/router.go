package dynamicgrpc

import (
	"errors"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"google.golang.org/grpc/peer"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type subscription struct {
	id     uint64
	events chan *core.Event
	result chan error

	rpc  string
	peer *peer.Peer
}

type Router struct {
	mu          *sync.RWMutex
	behavior    string
	waitForSubs bool
	subs        map[string][]subscription
}

func (r *Router) subscribe(rpc string, peer *peer.Peer) subscription {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub := subscription{
		id:     rand.Uint64(),
		events: make(chan *core.Event),
		result: make(chan error),
		rpc:    rpc,
		peer:   peer,
	}
	r.subs[rpc] = append(r.subs[rpc], sub)
	return sub
}

func (r *Router) unsubscribe(sub subscription) {
	r.mu.Lock()
	defer r.mu.Unlock()

	slices.DeleteFunc(r.subs[sub.rpc], func(s subscription) bool {
		return s.id == sub.id
	})
}

type publisher struct {
	*core.BaseOutput

	behavior    string
	rpc         string
	waitForSubs bool
	subs        []subscription

	stop           chan struct{}
	events         chan *core.Event
	subscription   chan subscription
	unsubscription chan subscription
}

func (p *publisher) run() {
PUBLISHER_MAIN_LOOP:
	for {
		select {
		case <-p.stop:
			p.Log.Info("publisher stopped",
				"procedure", p.rpc,
			)
			return
		case sub := <-p.subscription:
			p.Log.Info("new subscriber arrived",
				"procedure", p.rpc,
				"peer", sub.peer.String(),
			)
			p.subs = append(p.subs, sub)
		case sub := <-p.unsubscription:
			p.Log.Info("subscriber left",
				"procedure", p.rpc,
				"peer", sub.peer.String(),
			)
			p.subs = slices.DeleteFunc(p.subs, func(s subscription) bool {
				return s.id == sub.id
			})
		default:
			select {
			case <-p.stop:
				p.Log.Info("publisher stopped",
					"procedure", p.rpc,
				)
				return
			case sub := <-p.subscription:
				p.Log.Info("new subscriber arrived",
					"procedure", p.rpc,
					"peer", sub.peer.String(),
				)
				p.subs = append(p.subs, sub)
			case sub := <-p.unsubscription:
				p.Log.Info("subscriber left",
					"procedure", p.rpc,
					"peer", sub.peer.String(),
				)
				p.subs = slices.DeleteFunc(p.subs, func(s subscription) bool {
					return s.id == sub.id
				})
			case event := <-p.events:
				if len(p.subs) == 0 {
					p.Log.Debug("no subscribers, event dropped",
						"procedure", p.rpc,
						elog.EventGroup(event),
					)
					p.Done <- event
					p.Observe(metrics.EventAccepted, 0)
					continue PUBLISHER_MAIN_LOOP
				}

				p.publish(event)
			}
		}
	}
}

func (p *publisher) publish(event *core.Event) {
	now := time.Now()

	switch p.behavior {
	case behaviourBroadcast:
		hasError := false
		for _, sub := range p.subs {
			select {
			case sub.events <- event:
			case _, ok := <-sub.result:
				if !ok { // subscriber already gone and will be removed in the next iterations
					continue
				}
			}

			err := <-sub.result
			if errors.Is(err, ErrEncodingFailed) {
				p.Log.Error("message encoding failed, event skipped",
					"error", err,
					"procedure", p.rpc,
					elog.EventGroup(event),
				)
				p.Done <- event
				p.Observe(metrics.EventFailed, time.Since(now))
				return
			}

			if err != nil {
				hasError = true
			}
		}

		if hasError {
			p.Log.Warn("event delivery to some subscribers failed",
				"procedure", p.rpc,
				elog.EventGroup(event),
			)
		}

		p.Done <- event
		p.Observe(metrics.EventAccepted, time.Since(now))
	case behaviourRandom:
		subs := slices.Clone(p.subs)
		rand.Shuffle(len(subs), func(i, j int) {
			subs[i], subs[j] = subs[j], subs[i]
		})

		hasSuccess := false
		for _, sub := range subs {
			select {
			case sub.events <- event:
			case _, ok := <-sub.result:
				if !ok { // subscriber already gone and will be removed in the next iterations
					continue
				}
			}

			err := <-sub.result
			if errors.Is(err, ErrEncodingFailed) {
				p.Log.Error("message encoding failed, event skipped",
					"error", err,
					"procedure", p.rpc,
					elog.EventGroup(event),
				)
				p.Done <- event
				p.Observe(metrics.EventFailed, time.Since(now))
				return
			}

			if err == nil {
				hasSuccess = true
				break
			}
		}

		if !hasSuccess {
			p.Log.Error("event delivery to all subscribers failed",
				"procedure", p.rpc,
				elog.EventGroup(event),
			)
			p.Done <- event
			p.Observe(metrics.EventFailed, time.Since(now))
		} else {
			p.Done <- event
			p.Observe(metrics.EventAccepted, time.Since(now))
		}
	}
}

// type routersPool struct {
// 	*core.BaseOutput
// 	*pool.Pool[*core.Event, string]
// 	mu *sync.RWMutex
// }

// func (p *routersPool) Get(key string) pool.Runner[*core.Event] {
// 	p.mu.Lock()
// 	defer p.mu.Unlock()

// 	return p.Pool.Get(key)
// }

// func (p *routersPool) subscribe(rpc string, peer *peer.Peer) subscription {
// 	p.mu.Lock()
// 	defer p.mu.Unlock()

// 	p.Log.Info(fmt.Sprintf("new stream accepted from %v", peer.String()),
// 		"procedure", rpc,
// 	)

// 	sub := subscription{
// 		id:     rand.Uint64(),
// 		events: make(chan *core.Event),
// 		result: make(chan error),
// 		rpc:    rpc,
// 		peer:   peer,
// 	}

// 	router := p.Pool.Get(rpc).(*router)
// 	router.subscribe(sub)
// 	return sub
// }

// type router struct {
// 	*core.BaseOutput

// 	behavior    string
// 	waitForSubs bool
// 	subs        []subscription
// }

// func (r *router) subscribe(sub subscription) {
// 	r.subs = append(r.subs, sub)
// }

// func (r *router) unsubscribe(sub subscription) {
// 	slices.DeleteFunc(r.subs, func(s subscription) bool {
// 		return s.id == sub.id
// 	})
// }
