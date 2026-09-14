package middleware

import (
	"sync"

	"github.com/juju/ratelimit"
)

type RateLimiter struct {
	buckets  map[string]*ratelimit.Bucket
	mu       sync.RWMutex
	rate     float64
	capacity int64
}

func NewRateLimiter(rate float64, capacity int64) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*ratelimit.Bucket),
		rate:     rate,
		capacity: capacity,
	}
}

func (rl *RateLimiter) GetBucket(clientID string) *ratelimit.Bucket {
	rl.mu.RLock()
	bucket, exists := rl.buckets[clientID]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if bucket, exists := rl.buckets[clientID]; exists {
		return bucket
	}

	bucket = ratelimit.NewBucketWithRate(rl.rate, rl.capacity)
	rl.buckets[clientID] = bucket
	return bucket
}
