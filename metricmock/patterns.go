package metricmock

import (
	"math"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

// applyDiurnalPattern applies business hour peaks and off-hour troughs
func applyDiurnalPattern(points []schema.MetricPoint, peakHour, troughHour int) []schema.MetricPoint {
	if len(points) == 0 {
		return points
	}

	result := make([]schema.MetricPoint, len(points))
	for i, pt := range points {
		hour := pt.Timestamp.Hour()

		// Calculate multiplier based on hour (peak during business hours 9-17)
		var multiplier float64
		if hour >= 9 && hour <= 17 {
			// Business hours: 1.0 to 1.4 multiplier
			multiplier = 1.0 + 0.4*math.Sin(float64(hour-9)*math.Pi/8.0)
		} else {
			// Off hours: 0.6 to 0.8 multiplier
			multiplier = 0.7 - 0.1*math.Cos(float64(hour)*math.Pi/12.0)
		}

		result[i] = schema.MetricPoint{
			Timestamp: pt.Timestamp,
			Value:     pt.Value * multiplier,
		}
	}
	return result
}

// applyWeeklyPattern applies weekend reduction
func applyWeeklyPattern(points []schema.MetricPoint) []schema.MetricPoint {
	if len(points) == 0 {
		return points
	}

	result := make([]schema.MetricPoint, len(points))
	for i, pt := range points {
		weekday := pt.Timestamp.Weekday()

		// Reduce activity on weekends
		var multiplier float64
		if weekday == time.Saturday || weekday == time.Sunday {
			multiplier = 0.65 // 35% reduction on weekends
		} else {
			multiplier = 1.0
		}

		result[i] = schema.MetricPoint{
			Timestamp: pt.Timestamp,
			Value:     pt.Value * multiplier,
		}
	}
	return result
}

// applyDegradationPattern applies gradual service degradation
func applyDegradationPattern(points []schema.MetricPoint, degradeStart time.Time) []schema.MetricPoint {
	if len(points) == 0 {
		return points
	}

	result := make([]schema.MetricPoint, len(points))
	for i, pt := range points {
		if pt.Timestamp.Before(degradeStart) {
			result[i] = pt
			continue
		}

		// Gradual increase after degradation starts
		elapsed := pt.Timestamp.Sub(degradeStart).Minutes()
		degradationFactor := 1.0 + (elapsed/60.0)*0.3 // 30% increase per hour
		if degradationFactor > 2.0 {
			degradationFactor = 2.0 // Cap at 2x
		}

		result[i] = schema.MetricPoint{
			Timestamp: pt.Timestamp,
			Value:     pt.Value * degradationFactor,
		}
	}
	return result
}

// applyRecoveryPattern applies gradual service recovery
func applyRecoveryPattern(points []schema.MetricPoint, recoveryStart time.Time) []schema.MetricPoint {
	if len(points) == 0 {
		return points
	}

	result := make([]schema.MetricPoint, len(points))

	// Find baseline (average of points before recovery)
	baseline := points[0].Value * 0.7 // Assume baseline is 70% of degraded value

	for i, pt := range points {
		if pt.Timestamp.Before(recoveryStart) {
			result[i] = pt
			continue
		}

		// Gradual recovery toward baseline
		elapsed := pt.Timestamp.Sub(recoveryStart).Minutes()
		recoveryProgress := math.Min(elapsed/30.0, 1.0) // Full recovery in 30 minutes

		// Linear interpolation from current value to baseline
		targetValue := pt.Value - (pt.Value-baseline)*recoveryProgress

		result[i] = schema.MetricPoint{
			Timestamp: pt.Timestamp,
			Value:     targetValue,
		}
	}
	return result
}

// addSpikes adds occasional latency spikes
func addSpikes(points []schema.MetricPoint, spikiness float64) []schema.MetricPoint {
	if len(points) == 0 || spikiness <= 0 {
		return points
	}

	result := make([]schema.MetricPoint, len(points))
	copy(result, points)

	// Add spikes at random intervals based on spikiness
	spikeInterval := int(1.0 / spikiness)
	if spikeInterval < 5 {
		spikeInterval = 5
	}

	for i := range result {
		if i%spikeInterval == 0 && i > 0 {
			// Create a spike (2-4x normal value)
			spikeMagnitude := 2.0 + float64(i%3)
			result[i].Value = result[i].Value * spikeMagnitude
		}
	}

	return result
}

// addNoise adds realistic random variation
func addNoise(points []schema.MetricPoint, noiseFactor float64) []schema.MetricPoint {
	if len(points) == 0 || noiseFactor <= 0 {
		return points
	}

	result := make([]schema.MetricPoint, len(points))
	for i, pt := range points {
		// Add random noise using simple pseudo-random based on index
		noise := math.Sin(float64(i)*2.718281828) * noiseFactor * pt.Value
		result[i] = schema.MetricPoint{
			Timestamp: pt.Timestamp,
			Value:     pt.Value + noise,
		}
	}
	return result
}
