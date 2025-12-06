package probe

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/jtsang4/nettune/internal/client/http"
	"github.com/jtsang4/nettune/internal/shared/types"
)

// ThroughputTester performs throughput measurements
type ThroughputTester struct {
	client *http.Client
}

// NewThroughputTester creates a new throughput tester
func NewThroughputTester(client *http.Client) *ThroughputTester {
	return &ThroughputTester{client: client}
}

// TestDownload performs download throughput test
func (t *ThroughputTester) TestDownload(bytes int64, parallel int) (*types.ThroughputResult, error) {
	if bytes <= 0 {
		bytes = 100 * 1024 * 1024 // 100MB default
	}
	if parallel <= 0 {
		parallel = 1
	}

	bytesPerConnection := bytes / int64(parallel)
	var wg sync.WaitGroup

	type connResult struct {
		bytes    int64
		duration time.Duration
		err      error
	}
	results := make(chan connResult, parallel)

	start := time.Now()

	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			received, duration, err := t.client.ProbeDownload(bytesPerConnection)
			results <- connResult{bytes: received, duration: duration, err: err}
		}()
	}

	wg.Wait()
	totalDuration := time.Since(start)
	close(results)

	var totalBytes int64
	var errs []string
	for r := range results {
		totalBytes += r.bytes
		if r.err != nil {
			errs = append(errs, r.err.Error())
		}
	}

	throughputMbps := float64(totalBytes*8) / float64(totalDuration.Milliseconds()) / 1000

	return &types.ThroughputResult{
		Direction:      "download",
		Bytes:          totalBytes,
		DurationMs:     totalDuration.Milliseconds(),
		ThroughputMbps: throughputMbps,
		Parallel:       parallel,
		Errors:         errs,
	}, nil
}

// TestUpload performs upload throughput test
func (t *ThroughputTester) TestUpload(bytes int64, parallel int) (*types.ThroughputResult, error) {
	if bytes <= 0 {
		bytes = 100 * 1024 * 1024 // 100MB default
	}
	if parallel <= 0 {
		parallel = 1
	}

	bytesPerConnection := bytes / int64(parallel)
	var wg sync.WaitGroup

	type connResult struct {
		bytes    int64
		duration time.Duration
		err      error
	}
	results := make(chan connResult, parallel)

	// Generate random data
	data := make([]byte, bytesPerConnection)
	rand.Read(data)

	start := time.Now()

	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			uploadStart := time.Now()
			resp, err := t.client.ProbeUpload(data)
			duration := time.Since(uploadStart)

			if err != nil {
				results <- connResult{err: err, duration: duration}
				return
			}
			results <- connResult{bytes: resp.ReceivedBytes, duration: duration}
		}()
	}

	wg.Wait()
	totalDuration := time.Since(start)
	close(results)

	var totalBytes int64
	var errs []string
	for r := range results {
		totalBytes += r.bytes
		if r.err != nil {
			errs = append(errs, r.err.Error())
		}
	}

	throughputMbps := float64(totalBytes*8) / float64(totalDuration.Milliseconds()) / 1000

	return &types.ThroughputResult{
		Direction:      "upload",
		Bytes:          totalBytes,
		DurationMs:     totalDuration.Milliseconds(),
		ThroughputMbps: throughputMbps,
		Parallel:       parallel,
		Errors:         errs,
	}, nil
}
