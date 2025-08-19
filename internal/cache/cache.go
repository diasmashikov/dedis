package cache

import "sync"

type Cache struct {
	sync.RWMutex
	m map[string]string
}

func New() *Cache {
	return &Cache{m: make(map[string]string)}
}

func (c *Cache) Set(k, v string) {
	c.Lock()
	c.m[k] = v
	c.Unlock()
}

func (c *Cache) Get(k string) (string, bool) {
	c.RLock()
	v, ok := c.m[k]
	c.RUnlock()
	return v, ok
}
