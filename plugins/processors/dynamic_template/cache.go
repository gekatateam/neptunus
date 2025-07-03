package dynamic_template

import (
	"sync"
	"text/template"
	"time"
)

var cache = templateCache{
	u:  0,
	m:  make(map[string]*template.Template),
	d:  make(map[string]time.Time),
	mu: &sync.Mutex{},
}

type templateCache struct {
	u  int
	m  map[string]*template.Template
	d  map[string]time.Time
	mu *sync.Mutex
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
		c.d[key] = time.Now()
	}

	return t, ok
}

func (c *templateCache) Put(key string, t *template.Template) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.m[key] = t
	c.d[key] = time.Now()
}

func (c *templateCache) DropOlderThan(olderThan time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range c.d {
		if time.Since(v) > olderThan {
			delete(c.d, k)
			delete(c.m, k)
		}
	}
}

func (c *templateCache) Leave() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.u--
	if c.u == 0 {
		clear(c.m)
		clear(c.d)
	}
}
