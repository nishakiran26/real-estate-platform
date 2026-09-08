package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/cache"
	"github.com/rera/property-platform/backend/internal/dto"
)

// RateLimiter throttles requests per client key. It uses Redis so limits hold
// across instances, and falls back to an in-process counter when Redis is down
// rather than leaving endpoints unprotected.
type RateLimiter struct {
	cache    *cache.Client
	fallback *memoryLimiter
}

func NewRateLimiter(c *cache.Client) *RateLimiter {
	return &RateLimiter{cache: c, fallback: newMemoryLimiter()}
}

// Limit allows at most n requests per window for each caller. The caller key is
// the authenticated user ID when present, otherwise the client IP.
func (r *RateLimiter) Limit(name string, n int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := fmt.Sprintf("ratelimit:%s:%s", name, callerKey(c))
			count, err := r.cache.Incr(c.Request().Context(), key, window)
			if err != nil {
				count = r.fallback.incr(key, window)
			}
			remaining := n - int(count)
			if remaining < 0 {
				remaining = 0
			}
			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(n))
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			if int(count) > n {
				c.Response().Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				return c.JSON(http.StatusTooManyRequests,
					dto.NewErrorResponse("Too many requests. Please try again later.", nil))
			}
			return next(c)
		}
	}
}

func callerKey(c echo.Context) string {
	if id := UserID(c); id != 0 {
		return "u" + strconv.FormatUint(uint64(id), 10)
	}
	return "ip" + c.RealIP()
}

// memoryLimiter is a bounded in-process fallback used only while Redis is down.
type memoryLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	count     int64
	expiresAt time.Time
}

func newMemoryLimiter() *memoryLimiter {
	m := &memoryLimiter{buckets: make(map[string]*bucket)}
	go m.reap()
	return m
}

func (m *memoryLimiter) incr(key string, window time.Duration) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.buckets[key]
	if !ok || time.Now().After(b.expiresAt) {
		b = &bucket{expiresAt: time.Now().Add(window)}
		m.buckets[key] = b
	}
	b.count++
	return b.count
}

// reap drops expired buckets so the fallback map cannot grow without bound.
func (m *memoryLimiter) reap() {
	for range time.Tick(time.Minute) {
		m.mu.Lock()
		now := time.Now()
		for k, b := range m.buckets {
			if now.After(b.expiresAt) {
				delete(m.buckets, k)
			}
		}
		m.mu.Unlock()
	}
}
