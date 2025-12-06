package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	requests map[string]*bucket
	mu       sync.Mutex
	rate     int // requests per interval
	interval time.Duration
}

type bucket struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]*bucket),
		rate:     rate,
		interval: interval,
	}
}

// Allow checks if a request from the given key is allowed
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	b, exists := r.requests[key]
	if !exists {
		r.requests[key] = &bucket{
			tokens:    r.rate - 1,
			lastReset: now,
		}
		return true
	}

	// Reset tokens if interval has passed
	if now.Sub(b.lastReset) >= r.interval {
		b.tokens = r.rate - 1
		b.lastReset = now
		return true
	}

	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// RateLimit creates a rate limiting middleware
func RateLimit(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.Allow(key) {
			c.JSON(429, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "too many requests",
				},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequestSizeLimit creates a middleware that limits request body size
func RequestSizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = newLimitedReader(c.Request.Body, maxBytes)
		c.Next()
	}
}

// limitedReader wraps an io.Reader and limits the number of bytes read
type limitedReader struct {
	r         interface{ Read([]byte) (int, error) }
	remaining int64
}

func newLimitedReader(r interface{ Read([]byte) (int, error) }, limit int64) *limitedReader {
	return &limitedReader{r: r, remaining: limit}
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.remaining <= 0 {
		return 0, newRequestTooLargeError()
	}
	if int64(len(p)) > l.remaining {
		p = p[:l.remaining]
	}
	n, err := l.r.Read(p)
	l.remaining -= int64(n)
	return n, err
}

func (l *limitedReader) Close() error {
	if closer, ok := l.r.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

type requestTooLargeError struct{}

func (requestTooLargeError) Error() string {
	return "request body too large"
}

func newRequestTooLargeError() error {
	return requestTooLargeError{}
}
