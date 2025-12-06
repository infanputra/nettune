package probe

import (
	"context"
	"sync"
	"time"

	"github.com/jtsang4/nettune/internal/client/http"
	"github.com/jtsang4/nettune/internal/shared/types"
)

// LatencyLoadTester measures latency under load
type LatencyLoadTester struct {
	client *http.Client
}

// NewLatencyLoadTester creates a new latency under load tester
func NewLatencyLoadTester(client *http.Client) *LatencyLoadTester {
	return &LatencyLoadTester{client: client}
}

// TestLatencyUnderLoad measures latency while generating network load
func (t *LatencyLoadTester) TestLatencyUnderLoad(
	durationSec int,
	loadParallel int,
	echoIntervalMs int,
) (*types.LatencyUnderLoadResult, error) {
	if durationSec <= 0 {
		durationSec = 10
	}
	if loadParallel <= 0 {
		loadParallel = 4
	}
	if echoIntervalMs <= 0 {
		echoIntervalMs = 100
	}

	// First measure baseline RTT (without load)
	rttTester := NewRTTTester(t.client)
	baselineResult, err := rttTester.TestRTT(20, 1)
	if err != nil {
		return nil, err
	}

	// Now measure RTT under load
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(durationSec)*time.Second)
	defer cancel()

	// Start load generators
	var loadWg sync.WaitGroup
	var totalLoadBytes int64
	var loadMu sync.Mutex

	for i := 0; i < loadParallel; i++ {
		loadWg.Add(1)
		go func() {
			defer loadWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Download chunks to generate load
					received, _, _ := t.client.ProbeDownload(10 * 1024 * 1024) // 10MB chunks
					loadMu.Lock()
					totalLoadBytes += received
					loadMu.Unlock()
				}
			}
		}()
	}

	// Collect RTT samples while load is running
	var rtts []float64
	var rttMu sync.Mutex
	echoInterval := time.Duration(echoIntervalMs) * time.Millisecond

	var echoWg sync.WaitGroup
	echoWg.Add(1)
	go func() {
		defer echoWg.Done()
		ticker := time.NewTicker(echoInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				start := time.Now()
				_, err := t.client.ProbeEcho()
				if err == nil {
					rttMu.Lock()
					rtts = append(rtts, float64(time.Since(start).Milliseconds()))
					rttMu.Unlock()
				}
			}
		}
	}()

	// Wait for load to complete
	<-ctx.Done()
	loadWg.Wait()
	echoWg.Wait()

	// Calculate results
	result := &types.LatencyUnderLoadResult{
		Baseline:       baselineResult.RTT,
		LoadDurationMs: int64(durationSec * 1000),
	}

	if len(rtts) > 0 {
		result.UnderLoad = calculateStats(rtts)
	}

	// Calculate load throughput
	result.LoadMbps = float64(totalLoadBytes*8) / float64(durationSec) / 1000000

	// Calculate inflation factors
	if result.Baseline != nil && result.UnderLoad != nil {
		if result.Baseline.P50 > 0 {
			result.InflationP50 = result.UnderLoad.P50 / result.Baseline.P50
		}
		if result.Baseline.P99 > 0 {
			result.InflationP99 = result.UnderLoad.P99 / result.Baseline.P99
		}
	}

	return result, nil
}
