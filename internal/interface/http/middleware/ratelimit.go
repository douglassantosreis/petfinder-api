package middleware

import (
	"net/http"
	"sync"
	"time"
)

const bucketIdleTTL = 5 * time.Minute

type tokenBucket struct {
	tokens     int
	lastRefill time.Time
	lastSeen   time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	capacity int
	refill   time.Duration
}

func NewRateLimiter(capacity int, refill time.Duration) *RateLimiter {
	r := &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		capacity: capacity,
		refill:   refill,
	}
	go r.cleanupLoop()
	return r
}

func (r *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		key := req.RemoteAddr
		if !r.allow(key) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func (r *RateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	bucket, ok := r.buckets[key]
	if !ok {
		r.buckets[key] = &tokenBucket{tokens: r.capacity - 1, lastRefill: now, lastSeen: now}
		return true
	}
	bucket.lastSeen = now
	if now.Sub(bucket.lastRefill) >= r.refill {
		bucket.tokens = r.capacity
		bucket.lastRefill = now
	}
	if bucket.tokens <= 0 {
		return false
	}
	bucket.tokens--
	return true
}

// cleanupLoop removes buckets that have not been seen for bucketIdleTTL.
// Runs every refill period to avoid unbounded memory growth.
func (r *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(r.refill)
	defer ticker.Stop()
	for range ticker.C {
		r.mu.Lock()
		cutoff := time.Now().UTC().Add(-bucketIdleTTL)
		for key, b := range r.buckets {
			if b.lastSeen.Before(cutoff) {
				delete(r.buckets, key)
			}
		}
		r.mu.Unlock()
	}
}
