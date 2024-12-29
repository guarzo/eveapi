package common_test

import (
	"testing"
	"time"
)

type inMemCache struct {
	store map[string][]byte
}

func (c *inMemCache) Get(key string) ([]byte, bool) {
	val, ok := c.store[key]
	return val, ok
}
func (c *inMemCache) Set(key string, val []byte, _ time.Duration) {
	c.store[key] = val
}
func (c *inMemCache) Delete(key string) {
	delete(c.store, key)
}

func TestCacheRepository(t *testing.T) {
	cache := &inMemCache{store: make(map[string][]byte)}

	// 1) Set + Get
	cache.Set("foo", []byte("bar"), time.Hour)
	val, found := cache.Get("foo")
	if !found {
		t.Error("expected 'foo' to be in cache, not found")
	}
	if string(val) != "bar" {
		t.Errorf("expected 'bar', got %s", string(val))
	}

	// 2) Delete
	cache.Delete("foo")
	_, found = cache.Get("foo")
	if found {
		t.Error("expected 'foo' to be deleted, but still found")
	}
}
