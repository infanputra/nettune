package probe

import (
	"testing"
)

func TestCalculateStats(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50}

	stats := calculateStats(values)

	if stats.Min != 10 {
		t.Errorf("Min = %f, want %f", stats.Min, 10.0)
	}

	if stats.Max != 50 {
		t.Errorf("Max = %f, want %f", stats.Max, 50.0)
	}

	if stats.Mean != 30 {
		t.Errorf("Mean = %f, want %f", stats.Mean, 30.0)
	}

	if stats.P50 != 30 {
		t.Errorf("P50 = %f, want %f", stats.P50, 30.0)
	}
}

func TestCalculateStatsEmpty(t *testing.T) {
	stats := calculateStats(nil)

	if stats != nil {
		t.Error("Stats should be nil for empty input")
	}
}

func TestCalculateStatsSingle(t *testing.T) {
	values := []float64{42}

	stats := calculateStats(values)

	if stats.Min != 42 {
		t.Errorf("Min = %f, want %f", stats.Min, 42.0)
	}

	if stats.Max != 42 {
		t.Errorf("Max = %f, want %f", stats.Max, 42.0)
	}

	if stats.Mean != 42 {
		t.Errorf("Mean = %f, want %f", stats.Mean, 42.0)
	}
}

func TestCalculateJitter(t *testing.T) {
	// All same values should have 0 jitter
	values := []float64{10, 10, 10, 10}
	jitter := calculateJitter(values)

	if jitter != 0 {
		t.Errorf("Jitter = %f, want %f for identical values", jitter, 0.0)
	}
}

func TestCalculateJitterWithVariance(t *testing.T) {
	// Values with variance
	values := []float64{0, 10, 20, 30}
	jitter := calculateJitter(values)

	// Mean is 15, so deviations are 15, 5, 5, 15 = 40 total, 10 average
	if jitter != 10 {
		t.Errorf("Jitter = %f, want %f", jitter, 10.0)
	}
}

func TestMean(t *testing.T) {
	tests := []struct {
		values   []float64
		expected float64
	}{
		{[]float64{1, 2, 3, 4, 5}, 3},
		{[]float64{10}, 10},
		{[]float64{}, 0},
		{[]float64{-5, 5}, 0},
	}

	for _, tt := range tests {
		result := mean(tt.values)
		if result != tt.expected {
			t.Errorf("mean(%v) = %f, want %f", tt.values, result, tt.expected)
		}
	}
}

func TestPercentile(t *testing.T) {
	sorted := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// P50 should be around 5
	p50 := percentile(sorted, 50)
	if p50 < 4 || p50 > 6 {
		t.Errorf("P50 = %f, expected around 5", p50)
	}

	// P90 should be around 9
	p90 := percentile(sorted, 90)
	if p90 < 8 || p90 > 10 {
		t.Errorf("P90 = %f, expected around 9", p90)
	}
}

func TestPercentileEmpty(t *testing.T) {
	result := percentile(nil, 50)
	if result != 0 {
		t.Errorf("Percentile of empty slice should be 0, got %f", result)
	}
}
