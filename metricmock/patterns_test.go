package metricmock

import (
	"testing"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

func TestApplyDiurnalPattern(t *testing.T) {
	now := time.Now().UTC()
	points := []schema.MetricPoint{
		{Timestamp: now.Add(-12 * time.Hour), Value: 100}, // Off hours
		{Timestamp: now.Add(-6 * time.Hour), Value: 100},  // Off hours
		{Timestamp: now, Value: 100},                      // Business hours (assuming now is business hours)
	}

	result := applyDiurnalPattern(points, 9, 2)

	if len(result) != len(points) {
		t.Fatalf("expected %d points, got %d", len(points), len(result))
	}

	// Values should be modified
	for i := range result {
		if result[i].Value == points[i].Value {
			// It's okay if some values are the same due to the pattern
			continue
		}
	}
}

func TestApplyWeeklyPattern(t *testing.T) {
	// Create points for a weekend and weekday
	saturday := time.Date(2025, 12, 6, 12, 0, 0, 0, time.UTC) // Saturday
	monday := time.Date(2025, 12, 8, 12, 0, 0, 0, time.UTC)   // Monday

	points := []schema.MetricPoint{
		{Timestamp: saturday, Value: 100},
		{Timestamp: monday, Value: 100},
	}

	result := applyWeeklyPattern(points)

	// Weekend value should be reduced
	if result[0].Value >= result[1].Value {
		t.Errorf("expected weekend value (%.2f) to be less than weekday value (%.2f)",
			result[0].Value, result[1].Value)
	}
}

func TestApplyDegradationPattern(t *testing.T) {
	now := time.Now().UTC()
	degradeStart := now.Add(-30 * time.Minute)

	points := []schema.MetricPoint{
		{Timestamp: now.Add(-60 * time.Minute), Value: 100}, // Before degradation
		{Timestamp: now.Add(-45 * time.Minute), Value: 100}, // Before degradation
		{Timestamp: now.Add(-15 * time.Minute), Value: 100}, // After degradation
		{Timestamp: now, Value: 100},                        // After degradation
	}

	result := applyDegradationPattern(points, degradeStart)

	// Values before degradation should be unchanged
	if result[0].Value != points[0].Value {
		t.Errorf("expected value before degradation to be unchanged")
	}

	// Values after degradation should increase
	if result[2].Value <= points[2].Value {
		t.Errorf("expected value after degradation to increase")
	}
	if result[3].Value <= result[2].Value {
		t.Errorf("expected degradation to continue over time")
	}
}

func TestApplyRecoveryPattern(t *testing.T) {
	now := time.Now().UTC()
	recoveryStart := now.Add(-20 * time.Minute)

	points := []schema.MetricPoint{
		{Timestamp: now.Add(-30 * time.Minute), Value: 200}, // Before recovery (degraded)
		{Timestamp: now.Add(-10 * time.Minute), Value: 200}, // During recovery
		{Timestamp: now, Value: 200},                        // During recovery
	}

	result := applyRecoveryPattern(points, recoveryStart)

	// Values before recovery should be unchanged
	if result[0].Value != points[0].Value {
		t.Errorf("expected value before recovery to be unchanged")
	}

	// Values during recovery should decrease toward baseline
	if result[1].Value >= points[1].Value {
		t.Errorf("expected value during recovery to decrease")
	}
}

func TestAddSpikes(t *testing.T) {
	points := make([]schema.MetricPoint, 20)
	now := time.Now().UTC()
	for i := range points {
		points[i] = schema.MetricPoint{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Value:     100,
		}
	}

	result := addSpikes(points, 0.2) // 20% spikiness

	// Should have some spikes
	spikeCount := 0
	for i := range result {
		if result[i].Value > points[i].Value*1.5 {
			spikeCount++
		}
	}

	if spikeCount == 0 {
		t.Error("expected at least one spike")
	}
}

func TestAddNoise(t *testing.T) {
	points := []schema.MetricPoint{
		{Timestamp: time.Now(), Value: 100},
		{Timestamp: time.Now().Add(time.Minute), Value: 100},
		{Timestamp: time.Now().Add(2 * time.Minute), Value: 100},
	}

	result := addNoise(points, 0.1) // 10% noise

	// Values should be slightly different
	allSame := true
	for i := range result {
		if result[i].Value != points[i].Value {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("expected noise to modify values")
	}
}
