package lookup

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/gekatateam/mappath"
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
)

var _ core.Lookup = (*Lookup)(nil)

type LazyLookup interface {
	Update() (any, error)
	core.Initer
	io.Closer
}

type Lookup struct {
	*core.BaseLookup `mapstructure:"-"`
	LazyLookup       `mapstructure:",squash"`
	Interval         time.Duration `mapstructure:"interval"`

	data       any
	fetchCtx   context.Context
	cancelFunc context.CancelFunc
	mu         *sync.RWMutex
}

func (l *Lookup) Init() error {
	baseField := reflect.ValueOf(l.LazyLookup).Elem().FieldByName(core.KindLookup)
	if baseField.IsValid() && baseField.CanSet() {
		baseField.Set(reflect.ValueOf(l.BaseLookup))
	} else {
		return errors.New("embedded lookup plugin does not contains BaseLookup")
	}

	if err := l.LazyLookup.Init(); err != nil {
		return err
	}

	data, err := l.Update()
	if err != nil {
		return err
	}

	l.data = data
	l.mu = &sync.RWMutex{}
	l.fetchCtx, l.cancelFunc = context.WithCancel(context.Background())
	return nil
}

func (l *Lookup) Close() error {
	return l.LazyLookup.Close()
}

func (l *Lookup) Stop() {
	l.cancelFunc()
}

func (l *Lookup) Run() {
	if l.Interval <= 0 {
		l.Log.Info("lookup update disabled")
		<-l.fetchCtx.Done()
	} else {
		ticker := time.NewTicker(l.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-l.fetchCtx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				if data, err := l.Update(); err != nil {
					l.Log.Error("failed to update lookup data",
						"error", err,
					)
					l.Observe(metrics.EventFailed, time.Since(now))
				} else {
					l.mu.Lock()
					l.data = data
					l.mu.Unlock()
					l.Log.Debug("lookup updated")
					l.Observe(metrics.EventAccepted, time.Since(now))
				}
			}
		}
	}
}

func (l *Lookup) Get(key string) (any, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	data, err := mappath.Get(l.data, key)
	if err != nil {
		return nil, err
	}

	return mappath.Clone(data), nil
}
