package dynamicgrpc

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"

	"google.golang.org/grpc/peer"

	"github.com/gekatateam/neptunus/core"
)

type subscriber struct {
	id     uint64
	events chan *core.Event
	result chan error

	rpc  string
	peer *peer.Peer
}

type Router struct {
	*core.BaseOutput

	mu          *sync.RWMutex
	behavior    string
	waitForSubs bool
	subs        map[string][]subscriber
}

func (r *Router) subscribe(rpc string, peer *peer.Peer) subscriber {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Log.Info(fmt.Sprintf("new stream accepted from %v", peer.String()),
		"procedure", rpc,
	)

	sub := subscriber{
		id:     rand.Uint64(),
		events: make(chan *core.Event),
		result: make(chan error),
		rpc:    rpc,
		peer:   peer,
	}
	r.subs[rpc] = append(r.subs[rpc], sub)
	return sub
}

func (r *Router) unsubscribe(sub subscriber) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Log.Info(fmt.Sprintf("%v is going away", sub.peer.String()),
		"procedure", sub.rpc,
	)

	slices.DeleteFunc(r.subs[sub.rpc], func(s subscriber) bool {
		return s.id == sub.id
	})
}
