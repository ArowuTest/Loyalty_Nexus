package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// bucket holds per-key sliding-window state.
type bucket struct {
	mu        sync.Mutex
	hits      []time.Time
	blockedUntil time.Time
}

// RateLimiter is a simple in-process token-bucket rate limiter.
// For production, replace with Redis INCR + EXPIRE (already in redis_queue.go).
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	window  time.Duration
	max     int
	// Hard-ban duration after exceeding max
	banDuration time.Duration
}

func NewRateLimiter(window time.Duration, maxRequests int, banDuration time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets:     make(map[string]*bucket),
		window:      window,
		max:         maxRequests,
		banDuration: banDuration,
	}
	// Background cleanup
	go func() {
		for range time.Tick(5 * time.Minute) {
			rl.gc()
		}
	}()
	return rl
}

func (rl *RateLimiter) getBucket(key string) *bucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{}
		rl.buckets[key] = b
	}
	return b
}

// Allow returns true if the request should proceed.
func (rl *RateLimiter) Allow(key string) bool {
	b := rl.getBucket(key)
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if now.Before(b.blockedUntil) {
		return false
	}

	cutoff := now.Add(-rl.window)
	filtered := b.hits[:0]
	for _, t := range b.hits {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	b.hits = filtered

	if len(b.hits) >= rl.max {
		b.blockedUntil = now.Add(rl.banDuration)
		return false
	}
	b.hits = append(b.hits, now)
	return true
}

func (rl *RateLimiter) gc() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-rl.window * 2)
	for k, b := range rl.buckets {
		b.mu.Lock()
		empty := len(b.hits) == 0 && b.blockedUntil.Before(cutoff)
		b.mu.Unlock()
		if empty {
			delete(rl.buckets, k)
		}
	}
}

// Middleware returns an http.Handler that enforces the rate limit.
// keyFn extracts the rate-limit key from the request (e.g. IP or phone).
func (rl *RateLimiter) Middleware(keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if !rl.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", rl.banDuration.Seconds()))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"too many requests — please slow down"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IPKey extracts the client IP for use as a rate-limit key.
func IPKey(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return "ip:" + ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return "ip:" + ip
	}
	return "ip:" + r.RemoteAddr
}
