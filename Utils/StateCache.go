package Utils

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"github.com/patrickmn/go-cache"
	"log"
	"os"
	"time"
)

type StateCache struct {
	memCache          *cache.Cache
	defaultExpiration time.Duration
	interval          time.Duration
	CacheFileName     string
	ticker            *time.Ticker
	lastTickTime      time.Time
}

func NewStateCache(defaultExpiration, interval time.Duration, path string) *StateCache {
	if path == "" {
		path = "state-cache"
	}

	s := &StateCache{
		defaultExpiration: defaultExpiration,
		interval:          interval,
		memCache:          cache.New(defaultExpiration, interval),
		CacheFileName:     path,
		ticker:            time.NewTicker(interval),
		lastTickTime:      time.Now(),
	}

	gob.Register(map[string]interface{}{})

	return s
}

func (s *StateCache) Get(key string) (interface{}, bool) {
	return s.memCache.Get(key)
}

func (s *StateCache) Add(key string, value interface{}, expiration time.Duration) {
	s.memCache.Set(key, value, expiration)

	// write the cache file when there is more than 30 seconds to next tick
	if s.durationToNextTicket() > time.Second*30 {
		s.SaveCache()
	}
}

func (s *StateCache) Items() map[string]cache.Item {
	return s.memCache.Items()
}

func (s *StateCache) AutoSaver() {
	for {
		select {
		case <-s.ticker.C:
			s.lastTickTime = time.Now()
			s.SaveCache()
		default:
		}
	}
}

func (s *StateCache) SaveCache() {
	buffer := new(bytes.Buffer)
	e := gob.NewEncoder(buffer)

	err := e.Encode(s.memCache.Items())
	if err != nil {
		log.Printf("error encoding cache: %s", err)
		return
	}

	f, err := os.Create(s.CacheFileName)
	if err != nil {
		log.Printf("error creating file: %s", err)
		return
	}
	defer f.Close()

	_, err = f.Write(buffer.Bytes())
	if err != nil {
		log.Printf("error writing to file: %s", err)
	}

	err = f.Sync()
	if err != nil {
		log.Printf("error syncing to file: %s", err)
	}
}

func (s *StateCache) LoadCache() {
	var cacheMap map[string]cache.Item

	f, err := os.Open(s.CacheFileName)
	if err != nil {
		log.Printf("error opening file: %s", err)
	}

	reader := bufio.NewReader(f)

	d := gob.NewDecoder(reader)

	err = d.Decode(&cacheMap)
	if err != nil {
		log.Printf("error decoding file: %s", err)
	}

	s.memCache = cache.NewFrom(s.defaultExpiration, s.interval, cacheMap)
}

func (s *StateCache) durationToNextTicket() time.Duration {
	return s.lastTickTime.Add(s.interval).Sub(time.Now())
}
