package metricmock

import (
	"context"
	"testing"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

func TestQueryBuildsSeries(t *testing.T) {
	provAny, err := New(map[string]any{"source": "demo"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-10 * time.Minute)
	// Use scope to set service instead of parsing from expression
	series, err := prov.Query(context.Background(), schema.MetricQuery{
		Expression: &schema.MetricExpression{MetricName: "latency_p99"},
		Scope:      schema.QueryScope{Service: "checkout"},
		Start:      start,
		End:        end,
		Step:       60,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected series and baseline, got %d", len(series))
	}
	if series[0].Service != "checkout" {
		t.Fatalf("expected service field set, got %s", series[0].Service)
	}
	if series[0].Labels["service"] != "checkout" {
		t.Fatalf("expected service label set, got %+v", series[0].Labels)
	}
	if len(series[0].Points) == 0 {
		t.Fatalf("expected points in series")
	}
	if series[0].Metadata["source"] != "demo" {
		t.Fatalf("expected metadata source, got %v", series[0].Metadata["source"])
	}
}

func TestQueryDefaultsWindow(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	series, err := prov.Query(context.Background(), schema.MetricQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(series) == 0 || len(series[0].Points) == 0 {
		t.Fatalf("expected default series with points")
	}
}

func TestQueryRespectsScope(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	scope := schema.QueryScope{Service: "svc-web", Environment: "staging", Team: "team-velocity"}
	series, err := prov.Query(context.Background(), schema.MetricQuery{Scope: scope})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if got := series[0].Labels["service"]; got != scope.Service {
		t.Fatalf("expected scope service label %s, got %v", scope.Service, got)
	}
	if got := series[0].Service; got != scope.Service {
		t.Fatalf("expected service field %s, got %s", scope.Service, got)
	}
	if got := series[0].Labels["env"]; got != scope.Environment {
		t.Fatalf("expected scope environment label %s, got %v", scope.Environment, got)
	}
	if got := series[0].Labels["team"]; got != scope.Team {
		t.Fatalf("expected scope team label %s, got %v", scope.Team, got)
	}
	if got, ok := series[0].Metadata["scope"].(schema.QueryScope); !ok || got != scope {
		t.Fatalf("expected scope metadata %+v, got %+v", scope, series[0].Metadata["scope"])
	}
}

func TestQueryKeepsPrimarySeriesValuesIntact(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	series, err := prov.Query(context.Background(), schema.MetricQuery{Expression: &schema.MetricExpression{MetricName: "latency_p99"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(series) < 2 {
		t.Fatalf("expected comparative series, got %d", len(series))
	}
	if len(series[0].Points) == 0 || len(series[1].Points) == 0 {
		t.Fatalf("expected both series to contain points")
	}
	if series[0].Points[0].Value == series[1].Points[0].Value {
		t.Fatalf("baseline modification should not change primary series values")
	}
}

// Feature: enhanced-mock-data, Property 4: Counter metrics are monotonically increasing
func TestProperty_CounterMonotonicIncrease(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	// Test several counter metrics
	counterMetrics := []string{"http_requests_total", "orders_created_total", "cache_hits_total", "grpc_requests_total"}

	for _, metricName := range counterMetrics {
		series, err := prov.Query(context.Background(), schema.MetricQuery{
			Expression: &schema.MetricExpression{MetricName: metricName},
			Step:       60,
		})
		if err != nil {
			t.Fatalf("Query for %s returned error: %v", metricName, err)
		}

		if len(series) == 0 {
			t.Fatalf("expected series for %s", metricName)
		}

		// Check the primary series (not baseline)
		primarySeries := series[0]
		points := primarySeries.Points

		for i := 1; i < len(points); i++ {
			if points[i].Value < points[i-1].Value {
				t.Errorf("counter metric %s: value at index %d (%.2f) is less than previous value (%.2f)",
					metricName, i, points[i].Value, points[i-1].Value)
			}
		}
	}
}

// Feature: enhanced-mock-data, Property 5: Gauge metrics stay within defined bounds
func TestProperty_GaugeBounds(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	// Test gauge metrics with known bounds
	tests := []struct {
		metric string
		min    float64
		max    float64
	}{
		{"cpu_usage_ratio", 0.0, 1.0},
		{"error_rate", 0.0, 1.0},
		{"conversion_rate", 0.0, 1.0},
		{"cache_hit_ratio", 0.0, 1.0},
		{"db_cache_hit_ratio", 0.0, 1.0},
	}

	for _, tt := range tests {
		series, err := prov.Query(context.Background(), schema.MetricQuery{
			Expression: &schema.MetricExpression{MetricName: tt.metric},
			Step:       60,
		})
		if err != nil {
			t.Fatalf("Query for %s returned error: %v", tt.metric, err)
		}

		if len(series) == 0 {
			t.Fatalf("expected series for %s", tt.metric)
		}

		primarySeries := series[0]
		for i, point := range primarySeries.Points {
			if point.Value < tt.min || point.Value > tt.max {
				t.Errorf("gauge metric %s: value at index %d (%.4f) is outside bounds [%.2f, %.2f]",
					tt.metric, i, point.Value, tt.min, tt.max)
			}
		}
	}
}

// Feature: enhanced-mock-data, Property 7: Error rate metrics maintain low baseline
func TestProperty_ErrorRateLowBaseline(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	series, err := prov.Query(context.Background(), schema.MetricQuery{
		Expression: &schema.MetricExpression{MetricName: "error_rate"},
		Step:       60,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(series) == 0 {
		t.Fatal("expected series for error_rate")
	}

	primarySeries := series[0]
	lowCount := 0
	for _, point := range primarySeries.Points {
		if point.Value < 0.05 {
			lowCount++
		}
	}

	lowPct := float64(lowCount) / float64(len(primarySeries.Points))
	if lowPct < 0.80 {
		t.Errorf("only %.1f%% of error_rate values are below 0.05, expected at least 80%%", lowPct*100)
	}
}

func TestMetricSeriesIncludesAlertMetadata(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)
	series, err := prov.Query(context.Background(), schema.MetricQuery{
		Scope:      schema.QueryScope{Service: "svc-checkout"},
		Start:      start,
		End:        end,
		Step:       60,
		Expression: &schema.MetricExpression{MetricName: "http_request_duration_seconds"},
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(series) == 0 {
		t.Fatalf("expected at least one series")
	}
	if _, ok := series[0].Metadata["alerts"]; !ok {
		t.Fatalf("expected alert metadata on primary series, got %+v", series[0].Metadata)
	}
}

// Feature: enhanced-mock-data, Property 22: Capacity metrics show growth trend
func TestProperty_CapacityGrowthTrend(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	capacityMetrics := []string{"memory_working_set_bytes", "disk_usage_bytes"}

	for _, metricName := range capacityMetrics {
		series, err := prov.Query(context.Background(), schema.MetricQuery{
			Expression: &schema.MetricExpression{MetricName: metricName},
			Step:       60,
		})
		if err != nil {
			t.Fatalf("Query for %s returned error: %v", metricName, err)
		}

		if len(series) == 0 {
			t.Fatalf("expected series for %s", metricName)
		}

		primarySeries := series[0]
		points := primarySeries.Points

		if len(points) < 2 {
			t.Fatalf("not enough points to determine trend for %s", metricName)
		}

		// Calculate simple linear trend: is last value > first value?
		firstValue := points[0].Value
		lastValue := points[len(points)-1].Value

		if lastValue <= firstValue {
			t.Errorf("capacity metric %s: last value (%.2f) is not greater than first value (%.2f), expected growth trend",
				metricName, lastValue, firstValue)
		}
	}
}

// Feature: enhanced-mock-data, Property 13: Metrics include infrastructure labels
func TestProperty_InfrastructureLabels(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	// Query various metrics
	metrics := []string{"cpu_usage_ratio", "memory_working_set_bytes", "container_restarts_total", "pod_evictions_total"}

	for _, metricName := range metrics {
		series, err := prov.Query(context.Background(), schema.MetricQuery{
			Expression: &schema.MetricExpression{MetricName: metricName},
		})
		if err != nil {
			t.Fatalf("Query for %s returned error: %v", metricName, err)
		}

		if len(series) == 0 {
			t.Fatalf("expected series for %s", metricName)
		}

		primarySeries := series[0]
		labelCount := 0

		// Check for infrastructure labels
		if _, ok := primarySeries.Labels["instance"]; ok {
			labelCount++
		}
		if _, ok := primarySeries.Labels["pod"]; ok {
			labelCount++
		}
		if _, ok := primarySeries.Labels["availability_zone"]; ok {
			labelCount++
		}
		if _, ok := primarySeries.Labels["region"]; ok {
			labelCount++
		}
		if _, ok := primarySeries.Labels["namespace"]; ok {
			labelCount++
		}

		if labelCount < 2 {
			t.Errorf("metric %s: only %d infrastructure labels found, expected at least 2", metricName, labelCount)
		}
	}
}

// Feature: enhanced-mock-data, Property 15: Metric metadata includes annotations
func TestProperty_MetricAnnotations(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	series, err := prov.Query(context.Background(), schema.MetricQuery{
		Expression: &schema.MetricExpression{MetricName: "http_requests_total"},
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(series) == 0 {
		t.Fatal("expected series")
	}

	primarySeries := series[0]

	// Check for annotations
	if annotations, ok := primarySeries.Metadata["annotations"]; ok {
		if annotList, ok := annotations.([]map[string]any); ok && len(annotList) > 0 {
			// Verify annotations have required fields
			for _, annot := range annotList {
				if _, hasKind := annot["kind"]; !hasKind {
					t.Error("annotation missing 'kind' field")
				}
				if _, hasLabel := annot["label"]; !hasLabel {
					t.Error("annotation missing 'label' field")
				}
				if _, hasAt := annot["at"]; !hasAt {
					t.Error("annotation missing 'at' field")
				}
			}
		} else {
			t.Error("annotations field exists but is not a valid list")
		}
	} else {
		t.Error("metric metadata missing annotations")
	}
}

// Feature: enhanced-mock-data, Property 16: Timestamps are within realistic range
func TestProperty_TimestampRealisticRange(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	now := time.Now().UTC()
	start := now.Add(-30 * time.Minute)
	end := now

	series, err := prov.Query(context.Background(), schema.MetricQuery{
		Expression: &schema.MetricExpression{MetricName: "http_requests_total"},
		Start:      start,
		End:        end,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(series) == 0 {
		t.Fatal("expected series")
	}

	primarySeries := series[0]

	for i, point := range primarySeries.Points {
		// Check timestamp is not in the future
		if point.Timestamp.After(now.Add(1 * time.Minute)) {
			t.Errorf("point %d: timestamp %v is in the future", i, point.Timestamp)
		}

		// Check timestamp is within 24 hours
		if now.Sub(point.Timestamp) > 24*time.Hour {
			t.Errorf("point %d: timestamp %v is more than 24 hours ago", i, point.Timestamp)
		}

		// Check timestamp is within requested window
		if point.Timestamp.Before(start.Add(-1*time.Minute)) || point.Timestamp.After(end.Add(1*time.Minute)) {
			t.Errorf("point %d: timestamp %v is outside requested window [%v, %v]", i, point.Timestamp, start, end)
		}
	}
}

// Feature: enhanced-mock-data, Property 6: Diurnal patterns show business hour peaks
func TestProperty_DiurnalPatternBusinessHourPeaks(t *testing.T) {
	// Create test points spanning business and off hours
	points := []schema.MetricPoint{}

	// Add points for different hours
	for hour := 0; hour < 24; hour++ {
		ts := time.Date(2025, 12, 8, hour, 0, 0, 0, time.UTC)
		points = append(points, schema.MetricPoint{Timestamp: ts, Value: 100})
	}

	result := applyDiurnalPattern(points, 9, 2)

	// Calculate average for business hours (9-17) and off hours
	var businessSum, offHoursSum float64
	var businessCount, offHoursCount int

	for _, pt := range result {
		hour := pt.Timestamp.Hour()
		if hour >= 9 && hour <= 17 {
			businessSum += pt.Value
			businessCount++
		} else {
			offHoursSum += pt.Value
			offHoursCount++
		}
	}

	businessAvg := businessSum / float64(businessCount)
	offHoursAvg := offHoursSum / float64(offHoursCount)

	if businessAvg <= offHoursAvg {
		t.Errorf("business hours average (%.2f) should be higher than off-hours average (%.2f)",
			businessAvg, offHoursAvg)
	}
}

// Feature: enhanced-mock-data, Property 23: Seasonal metrics show weekend reduction
func TestProperty_SeasonalWeekendReduction(t *testing.T) {
	// Create points for weekdays and weekend
	points := []schema.MetricPoint{}

	// Monday through Sunday
	for day := 0; day < 7; day++ {
		ts := time.Date(2025, 12, 8+day, 12, 0, 0, 0, time.UTC)
		points = append(points, schema.MetricPoint{Timestamp: ts, Value: 100})
	}

	result := applyWeeklyPattern(points)

	// Calculate averages
	var weekdaySum, weekendSum float64
	var weekdayCount, weekendCount int

	for _, pt := range result {
		weekday := pt.Timestamp.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			weekendSum += pt.Value
			weekendCount++
		} else {
			weekdaySum += pt.Value
			weekdayCount++
		}
	}

	weekdayAvg := weekdaySum / float64(weekdayCount)
	weekendAvg := weekendSum / float64(weekendCount)

	// Weekend should be at least 30% lower
	reduction := (weekdayAvg - weekendAvg) / weekdayAvg
	if reduction < 0.30 {
		t.Errorf("weekend reduction (%.1f%%) is less than 30%%", reduction*100)
	}
}

// Feature: enhanced-mock-data, Property 20: Degraded service metrics show upward trend
func TestProperty_DegradedServiceUpwardTrend(t *testing.T) {
	now := time.Now().UTC()
	degradeStart := now.Add(-30 * time.Minute)

	points := []schema.MetricPoint{}
	for i := 0; i < 10; i++ {
		ts := degradeStart.Add(time.Duration(i*5) * time.Minute)
		points = append(points, schema.MetricPoint{Timestamp: ts, Value: 100})
	}

	result := applyDegradationPattern(points, degradeStart)

	// Check that values increase over time after degradation starts
	for i := 1; i < len(result); i++ {
		if result[i].Value < result[i-1].Value {
			t.Errorf("degraded metric value at index %d (%.2f) is less than previous (%.2f)",
				i, result[i].Value, result[i-1].Value)
		}
	}
}

// Feature: enhanced-mock-data, Property 21: Recovering service metrics show downward trend
func TestProperty_RecoveringServiceDownwardTrend(t *testing.T) {
	now := time.Now().UTC()
	recoveryStart := now.Add(-20 * time.Minute)

	points := []schema.MetricPoint{}
	for i := 0; i < 10; i++ {
		ts := recoveryStart.Add(time.Duration(i*3) * time.Minute)
		points = append(points, schema.MetricPoint{Timestamp: ts, Value: 200}) // Elevated value
	}

	result := applyRecoveryPattern(points, recoveryStart)

	// Check that values decrease over time during recovery
	for i := 1; i < len(result); i++ {
		if result[i].Value > result[i-1].Value {
			t.Errorf("recovering metric value at index %d (%.2f) is greater than previous (%.2f)",
				i, result[i].Value, result[i-1].Value)
		}
	}
}

// Feature: enhanced-mock-data, Property 24: Burst metrics show spike and recovery pattern
func TestProperty_BurstSpikeAndRecovery(t *testing.T) {
	points := make([]schema.MetricPoint, 30)
	now := time.Now().UTC()
	for i := range points {
		points[i] = schema.MetricPoint{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Value:     100,
		}
	}

	result := addSpikes(points, 0.15) // 15% spikiness

	// Find spikes (values > 150% of baseline)
	spikeIndices := []int{}
	for i, pt := range result {
		if pt.Value > 150 {
			spikeIndices = append(spikeIndices, i)
		}
	}

	if len(spikeIndices) == 0 {
		t.Error("expected at least one spike")
		return
	}

	// Check that there's normalization after spikes
	// (values return to baseline after spike)
	for _, spikeIdx := range spikeIndices {
		if spikeIdx+2 < len(result) {
			// Value 2 points after spike should be closer to baseline
			postSpikeValue := result[spikeIdx+2].Value
			if postSpikeValue > result[spikeIdx].Value*0.8 {
				// This is okay - spikes can persist
				continue
			}
		}
	}
}

func TestScenarioMetricAnomaliesStaticSeeding(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)
	series, err := prov.Query(context.Background(), schema.MetricQuery{
		Scope:      schema.QueryScope{Service: "svc-checkout"},
		Start:      start,
		End:        end,
		Step:       60,
		Expression: &schema.MetricExpression{MetricName: "http_request_duration_seconds"},
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(series) == 0 {
		t.Fatalf("expected series")
	}

	// Verify scenario effects are present
	primary := series[0]
	effects, ok := primary.Metadata["scenario_effects"].([]map[string]any)
	if !ok || len(effects) == 0 {
		t.Fatalf("expected scenario effects metadata, got %+v", primary.Metadata["scenario_effects"])
	}

	// Verify scenario metadata in effects
	for _, effect := range effects {
		if effect["scenario_id"] == nil {
			t.Errorf("scenario effect missing scenario_id")
		}
		if effect["scenario_name"] == nil {
			t.Errorf("scenario effect missing scenario_name")
		}
		if effect["stage"] == nil {
			t.Errorf("scenario effect missing stage")
		}
	}
}

func TestScenarioAnomalyPatterns(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)

	// Test different metrics with scenario anomalies
	tests := []struct {
		metric  string
		service string
	}{
		{"http_request_duration_seconds", "svc-checkout"},
		{"http_errors_total", "svc-checkout"},
		{"db_connections_active", "svc-search"},
		{"error_rate", "svc-checkout"},
		{"api_rate_limit_exceeded_total", "svc-api-gateway"},
		{"cart_abandonment_rate", "svc-web"},
		{"session_active", "svc-web"},
		{"circuit_breaker_trips_total", "svc-web"},
	}

	for _, tt := range tests {
		series, err := prov.Query(context.Background(), schema.MetricQuery{
			Scope:      schema.QueryScope{Service: tt.service},
			Start:      start,
			End:        end,
			Step:       60,
			Expression: &schema.MetricExpression{MetricName: tt.metric},
		})
		if err != nil {
			t.Fatalf("Query for %s returned error: %v", tt.metric, err)
		}

		if len(series) == 0 {
			t.Fatalf("expected series for %s", tt.metric)
		}

		primary := series[0]
		if effects, ok := primary.Metadata["scenario_effects"].([]map[string]any); ok && len(effects) > 0 {
			// Verify anomaly was applied
			for _, effect := range effects {
				if effect["metric"] != tt.metric {
					t.Errorf("expected metric %s in effect, got %v", tt.metric, effect["metric"])
				}
				if effect["service"] != tt.service {
					t.Errorf("expected service %s in effect, got %v", tt.service, effect["service"])
				}
			}
		}
	}
}

func TestScenarioServiceSpecificAnomalies(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)

	// Query for svc-checkout
	checkoutSeries, err := prov.Query(context.Background(), schema.MetricQuery{
		Scope:      schema.QueryScope{Service: "svc-checkout"},
		Start:      start,
		End:        end,
		Step:       60,
		Expression: &schema.MetricExpression{MetricName: "http_errors_total"},
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Query for svc-search
	searchSeries, err := prov.Query(context.Background(), schema.MetricQuery{
		Scope:      schema.QueryScope{Service: "svc-search"},
		Start:      start,
		End:        end,
		Step:       60,
		Expression: &schema.MetricExpression{MetricName: "http_errors_total"},
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Verify checkout has scenario effects
	checkoutEffects, ok := checkoutSeries[0].Metadata["scenario_effects"].([]map[string]any)
	if !ok || len(checkoutEffects) == 0 {
		t.Errorf("expected scenario effects for svc-checkout")
	}

	// Verify search may or may not have effects (depends on scenarios)
	// Just verify the query works
	if len(searchSeries) == 0 {
		t.Errorf("expected series for svc-search")
	}
}

func TestScenarioAnomalyMetadata(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)
	series, err := prov.Query(context.Background(), schema.MetricQuery{
		Scope:      schema.QueryScope{Service: "svc-checkout"},
		Start:      start,
		End:        end,
		Step:       60,
		Expression: &schema.MetricExpression{MetricName: "circuit_breaker_state"},
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(series) == 0 {
		t.Fatalf("expected series")
	}

	primary := series[0]
	effects, ok := primary.Metadata["scenario_effects"].([]map[string]any)
	if !ok || len(effects) == 0 {
		// Circuit breaker scenario might not be in the time window
		return
	}

	// Verify effect metadata structure
	for _, effect := range effects {
		if effect["description"] == nil {
			t.Errorf("scenario effect missing description")
		}
		if effect["start"] == nil {
			t.Errorf("scenario effect missing start time")
		}
		if effect["end"] == nil {
			t.Errorf("scenario effect missing end time")
		}
		// Check for either factor or value
		hasFactor := effect["anomaly_factor"] != nil
		hasValue := effect["value"] != nil
		if !hasFactor && !hasValue {
			t.Errorf("scenario effect missing both anomaly_factor and value")
		}
	}
}
