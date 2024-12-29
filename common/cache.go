package common

import "time"

// CacheRepository defines a minimal interface for a key/value cache.
// The values are stored as raw []byte, which you can marshal/unmarshal
// from JSON or other formats as needed.
//
// For example, you could back this with:
//   - an in-memory map
//   - Redis
//   - Memcached
//   - or any other caching system
type CacheRepository interface {
	Get(key string) (value []byte, found bool)
	Set(key string, value []byte, expiration time.Duration)
	Delete(key string)
}
