package elasticsearch

import (
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
)

type indexersPool struct {
	writers map[string]*indexer
	new     func(topic string) *indexer
	wg      *sync.WaitGroup
}

func (p *indexersPool) Get(pipeline string) *indexer

func (p *indexersPool) Pipelines() []string

type indexer struct {
	lastWrite time.Time

	client *elasticsearch.Client
	*batcher.Batcher[*core.Event]
}


