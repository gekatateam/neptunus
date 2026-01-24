package file

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/gekatateam/mappath"
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type File struct {
	*core.BaseLookup `mapstructure:"-"`
	File             string        `mapstructure:"file"`
	Interval         time.Duration `mapstructure:"interval"`

	data       any
	parser     core.Parser
	fetchCtx   context.Context
	cancelFunc context.CancelFunc
	mu         *sync.RWMutex
}

func (l *File) Init() error {
	l.mu = &sync.RWMutex{}

	if err := l.update(); err != nil {
		return err
	}

	l.fetchCtx, l.cancelFunc = context.WithCancel(context.Background())
	return nil
}

func (l *File) Close() error {
	return nil
}

func (l *File) SetParser(p core.Parser) {
	l.parser = p
}

func (l *File) Stop() {
	l.cancelFunc()
}

func (l *File) Run() {
	if l.Interval <= 0 {
		l.Log.Info("file lookup plugin running in one-shot mode")
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
				if err := l.update(); err != nil {
					l.Log.Error("failed to update lookup data",
						"error", err,
					)
					l.Observe(metrics.EventFailed, time.Since(now))
				} else {
					l.Log.Debug("lookup updated")
					l.Observe(metrics.EventAccepted, time.Since(now))
				}
			}
		}
	}
}

func (l *File) Get(key string) (any, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return mappath.Get(l.data, key)
}

func (l *File) update() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	content, err := os.ReadFile(l.File)
	if err != nil {
		return err
	}

	event, err := l.parser.Parse(content, "")
	if err != nil {
		return err
	}

	if len(event) == 0 {
		err := errors.New("parser returns zero events, nothing to store")
		return err
	}

	if len(event) > 1 {
		l.Log.Warn("parser returns more than one event, only first event will be used for lookup data")
	}

	l.data = event[0].Data
	return nil
}

func init() {
	plugins.AddLookup("file", func() core.Lookup {
		return &File{
			Interval: 30 * time.Second,
		}
	})
}
