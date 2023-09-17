package core

import (
	"sync/atomic"
)

type hookFunc func(payload any)

type tracker struct {
	duty    int32
	hook    hookFunc
	payload any
}

func newTracker(hook hookFunc, payload any) *tracker {
	return &tracker{
		duty:    1,
		hook:    hook,
		payload: payload,
	}
}

func (d *tracker) SetHook(hook hookFunc, payload any) {
	d.hook = hook
	d.payload = payload
}

func (d *tracker) Copy() *tracker {
	atomic.AddInt32(&d.duty, 1)
	return d
}

func (d *tracker) Increace() {
	atomic.AddInt32(&d.duty, 1)
}

func (d *tracker) Decreace() {
	n := atomic.AddInt32(&d.duty, -1)

	if n < 0 {
		panic("duty counter less than zero")
	}

	if n == 0 {
		d.hook(d.payload)
	}
}
