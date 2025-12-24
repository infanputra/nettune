package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name     string
		rate     int
		burst    int
		interval time.Duration
	}{
		{"basic limiter", 10, 5, time.Second},
		{"burst defaults to rate", 10, 0, time.Second},
		{"high rate", 1000, 100, time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewRateLimiter(tt.rate, tt.burst, tt.interval)
			if limiter == nil {
				t.Error("NewRateLimiter returned nil")
			}
			if limiter.rate != tt.rate {
				t.Errorf("rate = %d, want %d", limiter.rate, tt.rate)
			}
			expectedBurst := tt.burst
			if expectedBurst <= 0 {
				expectedBurst = tt.rate
			}
			if limiter.burst != expectedBurst {
				t.Errorf("burst = %d, want %d", limiter.burst, expectedBurst)
			}
		})
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(5, 5, time.Second)

	// First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		if !limiter.Allow("test-client") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if limiter.Allow("test-client") {
		t.Error("6th request should be denied")
	}
}

func TestRateLimiter_DifferentClients(t *testing.T) {
	limiter := NewRateLimiter(2, 2, time.Second)

	// Each client gets their own bucket
	for i := 0; i < 2; i++ {
		if !limiter.Allow("client-a") {
			t.Error("client-a request should be allowed")
		}
	}

	// client-b should still have quota
	if !limiter.Allow("client-b") {
		t.Error("client-b request should be allowed")
	}

	// client-a should be exhausted
	if limiter.Allow("client-a") {
		t.Error("client-a should be rate limited")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	limiter := NewRateLimiter(10, 2, 100*time.Millisecond)

	// Exhaust the burst
	limiter.Allow("test")
	limiter.Allow("test")

	if limiter.Allow("test") {
		t.Error("Should be rate limited after burst")
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow("test") {
		t.Error("Should be allowed after refill")
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	limiter := NewRateLimiter(100, 100, time.Second)

	var wg sync.WaitGroup
	allowed := make(chan bool, 200)

	// 200 concurrent requests
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- limiter.Allow("concurrent-client")
		}()
	}

	wg.Wait()
	close(allowed)

	// Count allowed requests
	count := 0
	for a := range allowed {
		if a {
			count++
		}
	}

	// Initial burst should allow 100 requests
	if count != 100 {
		t.Errorf("Expected 100 allowed requests, got %d", count)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	router := gin.New()
	limiter := NewRateLimiter(2, 2, time.Second)
	router.Use(RateLimit(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", w.Code)
	}
}

func TestRequestSizeLimit(t *testing.T) {
	router := gin.New()
	router.Use(RequestSizeLimit(100)) // 100 bytes limit
	router.POST("/test", func(c *gin.Context) {
		// Just try to bind/read - success means content was within limit
		c.JSON(200, gin.H{"message": "ok"})
	})

	// Small request should succeed
	t.Run("small request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

func TestLimitedReader(t *testing.T) {
	t.Run("read within limit", func(t *testing.T) {
		data := []byte("hello world")
		reader := newLimitedReader(&mockReader{data: data}, 100)

		buf := make([]byte, 20)
		n, err := reader.Read(buf)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected %d bytes, got %d", len(data), n)
		}
	})

	t.Run("read exceeds limit", func(t *testing.T) {
		data := []byte("hello world this is a long message")
		reader := newLimitedReader(&mockReader{data: data}, 10)

		buf := make([]byte, 5)
		// First read
		n1, _ := reader.Read(buf)

		// Second read
		n2, _ := reader.Read(buf)

		// Third read should fail
		_, err := reader.Read(buf)

		if err == nil {
			t.Error("Expected error when exceeding limit")
		}

		total := n1 + n2
		if total > 10 {
			t.Errorf("Read more than limit: %d", total)
		}
	})

	t.Run("close with closer", func(t *testing.T) {
		closer := &mockReadCloser{data: []byte("test")}
		reader := newLimitedReader(closer, 100)

		err := reader.Close()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !closer.closed {
			t.Error("Close was not called on underlying reader")
		}
	})

	t.Run("close without closer", func(t *testing.T) {
		reader := newLimitedReader(&mockReader{data: []byte("test")}, 100)

		err := reader.Close()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

type mockReader struct {
	data []byte
	pos  int
}

func (m *mockReader) Read(p []byte) (int, error) {
	if m.pos >= len(m.data) {
		return 0, nil
	}
	n := copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

type mockReadCloser struct {
	data   []byte
	pos    int
	closed bool
}

func (m *mockReadCloser) Read(p []byte) (int, error) {
	if m.pos >= len(m.data) {
		return 0, nil
	}
	n := copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockReadCloser) Close() error {
	m.closed = true
	return nil
}

func TestRequestTooLargeError(t *testing.T) {
	err := newRequestTooLargeError()
	if err.Error() != "request body too large" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}
