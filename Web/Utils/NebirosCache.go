package Utils

import (
	"github.com/patrickmn/go-cache"
	"time"
)

type NebirosCache struct {
	Cache    *cache.Cache
	Interval time.Duration
	stop     chan bool
}

func NewNebirosCache(defaultExpiration time.Duration, interval time.Duration, items map[string]cache.Item) *NebirosCache {
	var c *cache.Cache

	if items != nil {
		c = cache.NewFrom(defaultExpiration, interval, items)
	} else {
		c = cache.New(defaultExpiration, interval)
	}

	nc := &NebirosCache{
		Cache:    c,
		Interval: interval,
		stop:     make(chan bool),
	}

	return nc
}
