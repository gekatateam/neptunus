package core

import (
	"sync/atomic"
)

type hookFunc func()

type tracker struct {
	duty    int32
	hook    hookFunc
}

func newTracker(hook hookFunc) *tracker {
	return &tracker{
		duty:    1,
		hook:    hook,
	}
}

func (d *tracker) SetHook(hook hookFunc) {
	d.hook = hook
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
		d.hook()
	}
}
