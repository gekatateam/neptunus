package dynamic_template

import (
	"sync"
	"text/template"
	"time"

	"github.com/gekatateam/neptunus/plugins/common/refreshmap"
)

type storedTemplate struct {
	t *template.Template
	d time.Time
}

var cache = templateCache{
	u:  0,
	m:  make(map[string]storedTemplate),
	mu: &sync.Mutex{},
}

type templateCache struct {
	u  int
	m  map[string]storedTemplate
	mu *sync.Mutex
}

func (c *templateCache) Size() int {
	return len(c.m)
}

func (c *templateCache) Reg() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.u++
}

func (c *templateCache) Get(key string) (*template.Template, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	t, ok := c.m[key]
	if ok {
		t.d = time.Now()
		c.m[key] = t
	}

	return t.t, ok
}

func (c *templateCache) Put(key string, t *template.Template) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.m[key] = storedTemplate{
		t: t,
		d: time.Now(),
	}
}

func (c *templateCache) DropOlderThan(olderThan time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prev := len(c.m)

	for k, v := range c.m {
		if time.Since(v.d) > olderThan {
			delete(c.m, k)
		}
	}

	c.m = refreshmap.RefreshIfNeeded(c.m, prev)
}

func (c *templateCache) Leave() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.u--
	if c.u <= 0 {
		clear(c.m)
		c.u = 0
	}
}
