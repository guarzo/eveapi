package common

import (
	"time"
)

const (
	DefaultExpiration = 30 * time.Minute
	cleanupInterval   = 32 * time.Minute
)

type CacheRepository interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, expiration time.Duration)
	Delete(key string)
}

var _ CacheRepository = (*cacheStore)(nil)

type cacheStore struct {
	cache *cache.Cache
}

func NewCacheStore() CacheRepository {
	return &cacheStore{
		cache: cache.New(DefaultExpiration, cleanupInterval),
	}
}

func (c *cacheStore) Get(key string) ([]byte, bool) {
	value, found := c.cache.Get(key)
	if found {
		return value.([]byte), true
	}
	return nil, false
}

func (c *cacheStore) Delete(key string) {
	c.cache.Delete(key)
}

func (c *cacheStore) Set(key string, value []byte, expiration time.Duration) {
	c.cache.Set(key, value, expiration)
}
