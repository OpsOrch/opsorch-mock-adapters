package metricmock

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/opsorch/opsorch-core/metric"
	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/internal/mockutil"
)

// ProviderName can be referenced via OPSORCH_METRIC_PROVIDER.
const ProviderName = "mock"

// Config tunes metric generation.
type Config struct {
	Source string
}

// Provider generates deterministic demo time-series data.
type Provider struct {
	cfg Config
}

type metricDefinition struct {
	Name           string
	Type           string
	Unit           string
	Description    string
	Labels         []string
	DefaultService string
	Profile        seriesProfile
	ExtraLabels    map[string]string
}

var metricCatalog = []metricDefinition{
	// APPLICATION METRICS (15 types)
	{Name: "http_requests_total", Type: "counter", Unit: "requests", Description: "Total number of HTTP requests", Labels: []string{"service", "method", "status"}, DefaultService: "svc-checkout", Profile: seriesProfile{baseline: 240, amplitude: 45, trend: 1.2}, ExtraLabels: map[string]string{"method": "GET", "status": "200"}},
	{Name: "http_request_duration_seconds", Type: "histogram", Unit: "seconds", Description: "HTTP request latency", Labels: []string{"service", "method", "status"}, DefaultService: "svc-checkout", Profile: seriesProfile{baseline: 0.24, amplitude: 0.06, trend: 0.002}, ExtraLabels: map[string]string{"method": "GET"}},
	{Name: "http_errors_total", Type: "counter", Unit: "errors", Description: "Total HTTP errors", Labels: []string{"service", "status"}, DefaultService: "svc-search", Profile: seriesProfile{baseline: 12, amplitude: 8, trend: 0.3}, ExtraLabels: map[string]string{"status": "500"}},
	{Name: "grpc_requests_total", Type: "counter", Unit: "requests", Description: "Total gRPC requests", Labels: []string{"service", "method"}, DefaultService: "svc-realtime", Profile: seriesProfile{baseline: 3200, amplitude: 380, trend: 25}, ExtraLabels: map[string]string{"method": "Connect"}},
	{Name: "grpc_request_duration_seconds", Type: "histogram", Unit: "seconds", Description: "gRPC request latency", Labels: []string{"service", "method"}, DefaultService: "svc-realtime", Profile: seriesProfile{baseline: 0.085, amplitude: 0.025, trend: 0.001}, ExtraLabels: map[string]string{"method": "Connect"}},
	{Name: "api_rate_limit_remaining", Type: "gauge", Unit: "requests", Description: "Remaining API rate limit", Labels: []string{"service", "client"}, DefaultService: "svc-api-gateway", Profile: seriesProfile{baseline: 850, amplitude: 200, trend: -5}},
	{Name: "api_rate_limit_exceeded_total", Type: "counter", Unit: "events", Description: "Rate limit exceeded events", Labels: []string{"service", "client"}, DefaultService: "svc-api-gateway", Profile: seriesProfile{baseline: 45, amplitude: 15, trend: 0.8}},
	{Name: "cache_hits_total", Type: "counter", Unit: "hits", Description: "Cache hits", Labels: []string{"service", "cache"}, DefaultService: "svc-catalog", Profile: seriesProfile{baseline: 8500, amplitude: 1200, trend: 45}, ExtraLabels: map[string]string{"cache": "redis"}},
	{Name: "cache_misses_total", Type: "counter", Unit: "misses", Description: "Cache misses", Labels: []string{"service", "cache"}, DefaultService: "svc-catalog", Profile: seriesProfile{baseline: 650, amplitude: 180, trend: 12}, ExtraLabels: map[string]string{"cache": "redis"}},
	{Name: "cache_hit_ratio", Type: "gauge", Unit: "ratio", Description: "Cache hit ratio", Labels: []string{"service", "cache"}, DefaultService: "svc-catalog", Profile: seriesProfile{baseline: 0.92, amplitude: 0.05, trend: 0}, ExtraLabels: map[string]string{"cache": "redis"}},
	{Name: "circuit_breaker_state", Type: "gauge", Unit: "state", Description: "Circuit breaker state (0=closed, 1=open)", Labels: []string{"service", "downstream"}, DefaultService: "svc-web", Profile: seriesProfile{baseline: 0, amplitude: 0.3, trend: 0}},
	{Name: "circuit_breaker_trips_total", Type: "counter", Unit: "trips", Description: "Circuit breaker trips", Labels: []string{"service", "downstream"}, DefaultService: "svc-web", Profile: seriesProfile{baseline: 3, amplitude: 2, trend: 0.1}},
	{Name: "websocket_connections_active", Type: "gauge", Unit: "connections", Description: "Active WebSocket connections", Labels: []string{"service", "env"}, DefaultService: "svc-realtime", Profile: seriesProfile{baseline: 4200, amplitude: 850, trend: 35}},
	{Name: "websocket_messages_sent_total", Type: "counter", Unit: "messages", Description: "WebSocket messages sent", Labels: []string{"service", "type"}, DefaultService: "svc-realtime", Profile: seriesProfile{baseline: 12500, amplitude: 2400, trend: 85}},
	{Name: "background_jobs_queued", Type: "gauge", Unit: "jobs", Description: "Queued background jobs", Labels: []string{"service", "queue"}, DefaultService: "svc-workers", Profile: seriesProfile{baseline: 1200, amplitude: 350, trend: 18}},
	{Name: "background_jobs_processing_duration_seconds", Type: "histogram", Unit: "seconds", Description: "Background job processing time", Labels: []string{"service", "job_type"}, DefaultService: "svc-workers", Profile: seriesProfile{baseline: 2.4, amplitude: 0.8, trend: 0.05}},

	// INFRASTRUCTURE METRICS (12 types)
	{Name: "cpu_usage_ratio", Type: "gauge", Unit: "ratio", Description: "CPU usage ratio", Labels: []string{"service", "instance"}, DefaultService: "svc-warehouse", Profile: seriesProfile{baseline: 0.42, amplitude: 0.12, trend: 0.01}},
	{Name: "cpu_throttling_seconds_total", Type: "counter", Unit: "seconds", Description: "CPU throttling time", Labels: []string{"service", "instance"}, DefaultService: "svc-warehouse", Profile: seriesProfile{baseline: 45, amplitude: 18, trend: 1.2}},
	{Name: "memory_working_set_bytes", Type: "gauge", Unit: "bytes", Description: "Working set memory", Labels: []string{"service", "instance"}, DefaultService: "svc-warehouse", Profile: seriesProfile{baseline: 780 * 1024 * 1024, amplitude: 64 * 1024 * 1024, trend: 4 * 1024 * 1024}},
	{Name: "memory_available_bytes", Type: "gauge", Unit: "bytes", Description: "Available memory", Labels: []string{"service", "instance"}, DefaultService: "svc-warehouse", Profile: seriesProfile{baseline: 2.2 * 1024 * 1024 * 1024, amplitude: 512 * 1024 * 1024, trend: -8 * 1024 * 1024}},
	{Name: "disk_usage_bytes", Type: "gauge", Unit: "bytes", Description: "Disk space used", Labels: []string{"service", "instance", "mount"}, DefaultService: "svc-logging", Profile: seriesProfile{baseline: 650 * 1024 * 1024 * 1024, amplitude: 45 * 1024 * 1024 * 1024, trend: 2.1 * 1024 * 1024 * 1024}},
	{Name: "disk_io_operations_total", Type: "counter", Unit: "operations", Description: "Disk I/O operations", Labels: []string{"service", "instance", "operation"}, DefaultService: "svc-database", Profile: seriesProfile{baseline: 8500, amplitude: 1800, trend: 95}},
	{Name: "network_transmit_bytes_total", Type: "counter", Unit: "bytes", Description: "Network bytes transmitted", Labels: []string{"service", "instance"}, DefaultService: "svc-api-gateway", Profile: seriesProfile{baseline: 450 * 1024 * 1024, amplitude: 85 * 1024 * 1024, trend: 12 * 1024 * 1024}},
	{Name: "network_receive_bytes_total", Type: "counter", Unit: "bytes", Description: "Network bytes received", Labels: []string{"service", "instance"}, DefaultService: "svc-api-gateway", Profile: seriesProfile{baseline: 380 * 1024 * 1024, amplitude: 72 * 1024 * 1024, trend: 9 * 1024 * 1024}},
	{Name: "container_restarts_total", Type: "counter", Unit: "restarts", Description: "Container restart count", Labels: []string{"service", "pod"}, DefaultService: "svc-checkout", Profile: seriesProfile{baseline: 2, amplitude: 1.5, trend: 0.05}},
	{Name: "pod_evictions_total", Type: "counter", Unit: "evictions", Description: "Pod eviction count", Labels: []string{"service", "namespace"}, DefaultService: "svc-checkout", Profile: seriesProfile{baseline: 0.5, amplitude: 0.8, trend: 0.02}},
	{Name: "node_ready_status", Type: "gauge", Unit: "status", Description: "Node ready status (1=ready, 0=not ready)", Labels: []string{"node", "cluster"}, DefaultService: "svc-platform", Profile: seriesProfile{baseline: 1, amplitude: 0.05, trend: 0}},
	{Name: "load_average_1m", Type: "gauge", Unit: "load", Description: "1-minute load average", Labels: []string{"service", "instance"}, DefaultService: "svc-warehouse", Profile: seriesProfile{baseline: 2.8, amplitude: 0.9, trend: 0.05}},

	// DATABASE METRICS (8 types)
	{Name: "db_connections_active", Type: "gauge", Unit: "connections", Description: "Active database connections", Labels: []string{"service", "database"}, DefaultService: "svc-catalog", Profile: seriesProfile{baseline: 45, amplitude: 18, trend: 2}},
	{Name: "db_connections_max", Type: "gauge", Unit: "connections", Description: "Maximum database connections", Labels: []string{"service", "database"}, DefaultService: "svc-catalog", Profile: seriesProfile{baseline: 100, amplitude: 0, trend: 0}},
	{Name: "db_query_duration_seconds", Type: "histogram", Unit: "seconds", Description: "Database query duration", Labels: []string{"service", "database", "query_type"}, DefaultService: "svc-catalog", Profile: seriesProfile{baseline: 0.045, amplitude: 0.025, trend: 0.002}},
	{Name: "db_slow_queries_total", Type: "counter", Unit: "queries", Description: "Slow query count", Labels: []string{"service", "database"}, DefaultService: "svc-catalog", Profile: seriesProfile{baseline: 8, amplitude: 5, trend: 0.3}},
	{Name: "db_replication_lag_seconds", Type: "gauge", Unit: "seconds", Description: "Database replication lag", Labels: []string{"service", "replica"}, DefaultService: "svc-database", Profile: seriesProfile{baseline: 0.8, amplitude: 0.4, trend: 0.02}},
	{Name: "db_deadlocks_total", Type: "counter", Unit: "deadlocks", Description: "Database deadlock count", Labels: []string{"service", "database"}, DefaultService: "svc-catalog", Profile: seriesProfile{baseline: 1.2, amplitude: 0.8, trend: 0.05}},
	{Name: "db_transaction_rollbacks_total", Type: "counter", Unit: "rollbacks", Description: "Transaction rollback count", Labels: []string{"service", "database"}, DefaultService: "svc-catalog", Profile: seriesProfile{baseline: 15, amplitude: 8, trend: 0.4}},
	{Name: "db_cache_hit_ratio", Type: "gauge", Unit: "ratio", Description: "Database cache hit ratio", Labels: []string{"service", "database"}, DefaultService: "svc-catalog", Profile: seriesProfile{baseline: 0.96, amplitude: 0.03, trend: 0}},

	// BUSINESS METRICS (5 types)
	{Name: "orders_created_total", Type: "counter", Unit: "orders", Description: "Orders created", Labels: []string{"service", "region"}, DefaultService: "svc-order", Profile: seriesProfile{baseline: 1800, amplitude: 260, trend: 22}},
	{Name: "revenue_total", Type: "counter", Unit: "dollars", Description: "Total revenue", Labels: []string{"service", "currency"}, DefaultService: "svc-order", Profile: seriesProfile{baseline: 125000, amplitude: 28000, trend: 1850}},
	{Name: "conversion_rate", Type: "gauge", Unit: "ratio", Description: "Checkout conversion rate", Labels: []string{"service", "env"}, DefaultService: "svc-web", Profile: seriesProfile{baseline: 0.27, amplitude: 0.03, trend: 0}},
	{Name: "cart_abandonment_rate", Type: "gauge", Unit: "ratio", Description: "Shopping cart abandonment rate", Labels: []string{"service", "env"}, DefaultService: "svc-web", Profile: seriesProfile{baseline: 0.68, amplitude: 0.08, trend: 0}},
	{Name: "active_users_total", Type: "gauge", Unit: "users", Description: "Active users", Labels: []string{"service", "env"}, DefaultService: "svc-web", Profile: seriesProfile{baseline: 8900, amplitude: 1400, trend: 45}},

	// LEGACY METRICS (kept for backward compatibility)
	{Name: "latency_p99", Type: "gauge", Unit: "milliseconds", Description: "p99 latency across handlers", Labels: []string{"service", "env"}, DefaultService: "svc-web", Profile: seriesProfile{baseline: 310, amplitude: 45, trend: 1}},
	{Name: "error_rate", Type: "gauge", Unit: "ratio", Description: "Request error ratio", Labels: []string{"service", "env"}, DefaultService: "svc-checkout", Profile: seriesProfile{baseline: 0.01, amplitude: 0.002, trend: 0}},
	{Name: "auth_tokens_issued_total", Type: "counter", Unit: "tokens", Description: "Number of auth tokens issued", Labels: []string{"service", "region"}, DefaultService: "svc-identity", Profile: seriesProfile{baseline: 5200, amplitude: 320, trend: 12}},
	{Name: "session_active", Type: "gauge", Unit: "sessions", Description: "Active user sessions", Labels: []string{"service", "env"}, DefaultService: "svc-web", Profile: seriesProfile{baseline: 8900, amplitude: 1400, trend: 45}},
	{Name: "kafka_consumer_lag", Type: "gauge", Unit: "messages", Description: "Kafka consumer lag", Labels: []string{"service", "partition"}, DefaultService: "svc-notifications", Profile: seriesProfile{baseline: 4200, amplitude: 600, trend: -45}, ExtraLabels: map[string]string{"partition": "0"}},
	{Name: "queue_depth", Type: "gauge", Unit: "messages", Description: "Queue backlog depth", Labels: []string{"service", "queue"}, DefaultService: "svc-notifications", Profile: seriesProfile{baseline: 1800, amplitude: 420, trend: 35}, ExtraLabels: map[string]string{"queue": "promo-delivery"}},
	{Name: "feature_flag_override_total", Type: "counter", Unit: "overrides", Description: "Feature flag overrides applied", Labels: []string{"service", "flag"}, DefaultService: "svc-web", Profile: seriesProfile{baseline: 90, amplitude: 18, trend: 1.2}, ExtraLabels: map[string]string{"flag": "reco-rollout"}},
}

var metricCatalogIndex map[string]metricDefinition

// New constructs the mock metric provider.
func New(cfg map[string]any) (metric.Provider, error) {
	parsed := parseConfig(cfg)
	return &Provider{cfg: parsed}, nil
}

func init() {
	metricCatalogIndex = make(map[string]metricDefinition, len(metricCatalog))
	for _, def := range metricCatalog {
		metricCatalogIndex[def.Name] = def
	}
	_ = metric.RegisterProvider(ProviderName, New)
}

// Query returns a single synthetic series derived from the expression and window.
func (p *Provider) Query(ctx context.Context, query schema.MetricQuery) ([]schema.MetricSeries, error) {
	_ = ctx

	start := query.Start
	end := query.End
	if end.IsZero() {
		end = time.Now().UTC()
	}
	if start.IsZero() {
		start = end.Add(-30 * time.Minute)
	}
	if start.After(end) {
		start, end = end, start
	}
	step := time.Duration(query.Step) * time.Second
	if step <= 0 {
		step = 60 * time.Second
	}

	metricName := ""
	if query.Expression != nil {
		metricName = query.Expression.MetricName
	}

	requested := requestedMetricNames(metricName)
	defs := definitionsForRequest(metricName, requested)
	series := make([]schema.MetricSeries, 0, len(defs)*2)
	alertSnapshot := mockutil.SnapshotAlerts()
	scenarioAnomalies := getScenarioMetricAnomalies(end)
	// Filter alerts for time window
	for _, def := range defs {
		labels := scopedLabelsForDefinition(def, query)
		service := labelString(labels, "service")
		// Filter alerts for this service and time window
		serviceAlerts := make([]schema.Alert, 0)
		for _, alert := range alertSnapshot {
			if (service == "" || alert.Service == service) &&
				alert.CreatedAt.Before(end) && alert.UpdatedAt.After(start) {
				serviceAlerts = append(serviceAlerts, alert)
			}
		}
		points := generateSeriesPoints(start, end, step, def, service, serviceAlerts)
		var scenarioEffects []map[string]any
		if len(scenarioAnomalies) > 0 {
			scenarioEffects = applyScenarioMetricAnomalies(points, scenarioAnomalies, def.Name, service, start, end)
		}
		metadata := buildSeriesMetadata(def, query, labels, start, end, step, p.cfg.Source, service, points)
		if len(serviceAlerts) > 0 {
			metadata["alerts"] = mockutil.SummarizeAlerts(serviceAlerts)
		}
		if len(scenarioEffects) > 0 {
			metadata["scenario_effects"] = scenarioEffects
		}
		metadata["variant"] = "active"
		active := schema.MetricSeries{
			Name:     def.Name,
			Service:  service,
			Labels:   labels,
			Points:   points,
			Metadata: metadata,
		}
		series = append(series, active)

		baseline := active
		baseline.Name = def.Name + ".baseline"
		baseline.Labels = mockutil.CloneMap(active.Labels)
		baseline.Labels["variant"] = "baseline"
		baseline.Metadata = mockutil.CloneMap(active.Metadata)
		baseline.Metadata["variant"] = "baseline"
		baseline.Points = buildBaselinePoints(active.Points)
		series = append(series, baseline)
	}

	return series, nil
}

// Describe lists available metrics.
func (p *Provider) Describe(ctx context.Context, scope schema.QueryScope) ([]schema.MetricDescriptor, error) {
	descriptors := make([]schema.MetricDescriptor, 0, len(metricCatalog))
	for _, def := range metricCatalog {
		descriptors = append(descriptors, schema.MetricDescriptor{
			Name:        def.Name,
			Type:        def.Type,
			Description: def.Description,
			Labels:      def.Labels,
			Unit:        def.Unit,
		})
	}
	return descriptors, nil
}

func parseConfig(cfg map[string]any) Config {
	out := Config{Source: "mock-metric"}
	if v, ok := cfg["source"].(string); ok && v != "" {
		out.Source = v
	}
	return out
}

func inferService(expr string) string {
	lower := strings.ToLower(expr)
	for _, candidate := range []string{"checkout", "search", "web"} {
		if strings.Contains(lower, candidate) {
			return candidate
		}
	}
	return ""
}

func generatePoints(start, end time.Time, step time.Duration, profile seriesProfile, metricType string) []schema.MetricPoint {
	points := []schema.MetricPoint{}

	count := int(end.Sub(start) / step)
	if count < 3 {
		count = 3
	}

	// For counters, we want a running total.
	runningTotal := profile.baseline

	for i := 0; i <= count; i++ {
		ts := start.Add(time.Duration(i) * step)
		if ts.After(end) {
			break
		}

		var val float64
		if metricType == "counter" {
			// Monotonically increasing
			// Add a random increment based on "trend" (rate) + some noise
			increment := profile.trend + (math.Sin(float64(i)/3.5)+1.0)*profile.amplitude*0.1
			if increment < 0 {
				increment = 0
			}
			runningTotal += increment
			val = runningTotal
		} else {
			// Gauge / Histogram (latency view) - fluctuating
			wave := math.Sin(float64(i)/3.5) * profile.amplitude
			trend := profile.trend * float64(i) // slight trend up/down
			noise := float64((i%4)-1) * profile.amplitude * 0.5
			val = profile.baseline + wave + trend + noise
			if val < 0 {
				val = 0
			}
		}

		points = append(points, schema.MetricPoint{Timestamp: ts, Value: math.Round(val*100) / 100})
	}
	return points
}

func inferType(expr string) string {
	lower := strings.ToLower(expr)
	if strings.HasSuffix(lower, "_total") || strings.HasSuffix(lower, "_count") || strings.HasSuffix(lower, "_sum") {
		return "counter"
	}
	if strings.Contains(lower, "requests") && !strings.Contains(lower, "duration") {
		return "counter"
	}
	return "gauge"
}

type seriesProfile struct {
	baseline  float64
	amplitude float64
	trend     float64
}

func profileForExpression(expr string) seriesProfile {
	lower := strings.ToLower(expr)
	switch {
	case strings.Contains(lower, "latency") || strings.Contains(lower, "duration"):
		return seriesProfile{baseline: 0.250, amplitude: 0.050, trend: 0.001} // seconds
	case strings.Contains(lower, "error"):
		return seriesProfile{baseline: 0.01, amplitude: 0.005, trend: 0.0} // ratio
	case strings.Contains(lower, "rps") || strings.Contains(lower, "throughput"):
		return seriesProfile{baseline: 150, amplitude: 30, trend: 0.5}
	case strings.Contains(lower, "active") || strings.Contains(lower, "connections"):
		return seriesProfile{baseline: 320, amplitude: 55, trend: 1.1}
	case strings.Contains(lower, "cpu"):
		return seriesProfile{baseline: 0.45, amplitude: 0.15, trend: 0.05} // seconds (counter) or usage (gauge)? usually cpu_seconds_total is counter
	case strings.Contains(lower, "memory") || strings.Contains(lower, "bytes"):
		return seriesProfile{baseline: 512 * 1024 * 1024, amplitude: 64 * 1024 * 1024, trend: 1024 * 1024}
	default:
		seed := float64(len(expr)%7 + 4)
		return seriesProfile{baseline: seed * 10, amplitude: seed, trend: seed * 0.1}
	}
}

func inferUnit(expr string) string {
	lower := strings.ToLower(expr)
	switch {
	case strings.Contains(lower, "latency"):
		return "milliseconds"
	case strings.Contains(lower, "error"):
		return "ratio"
	case strings.Contains(lower, "rps") || strings.Contains(lower, "throughput"):
		return "requests_per_second"
	case strings.Contains(lower, "active"):
		return "count"
	case strings.Contains(lower, "cpu"):
		return "seconds"
	case strings.Contains(lower, "memory") || strings.Contains(lower, "bytes"):
		return "bytes"
	default:
		return "value"
	}
}

func fallback(val, def string) string {
	if strings.TrimSpace(val) != "" {
		return val
	}
	return def
}

func generateSeriesPoints(start, end time.Time, step time.Duration, def metricDefinition, service string, alerts []schema.Alert) []schema.MetricPoint {
	profile := def.Profile
	if profile == (seriesProfile{}) {
		profile = profileForExpression(def.Name)
	}
	typ := def.Type
	if typ == "" {
		typ = inferType(def.Name)
	}
	points := generatePoints(start, end, step, profile, typ)
	applyAlertAnomalies(points, typ, service, alerts)

	// Apply bounds for ratio metrics
	if def.Unit == "ratio" || strings.Contains(strings.ToLower(def.Name), "ratio") || strings.HasSuffix(strings.ToLower(def.Name), "_rate") {
		for i := range points {
			if points[i].Value < 0 {
				points[i].Value = 0
			}
			if points[i].Value > 1 {
				points[i].Value = 1
			}
		}
	}

	return points
}

func applyAlertAnomalies(points []schema.MetricPoint, metricType, service string, alerts []schema.Alert) {
	if len(points) == 0 || len(alerts) == 0 || service == "" {
		return
	}
	cumulativeExtra := 0.0
	for i := range points {
		factor, _ := mockutil.StrongestAlertFactor(service, points[i].Timestamp, alerts)
		if factor <= 1.01 {
			continue
		}
		switch metricType {
		case "counter":
			base := points[i].Value + cumulativeExtra
			surge := base * (factor - 1)
			if surge < 5 {
				surge = 5
			}
			cumulativeExtra += surge
			points[i].Value = base + surge
		default:
			points[i].Value = math.Round(points[i].Value*factor*100) / 100
		}
	}
	if metricType == "counter" {
		for i := 1; i < len(points); i++ {
			if points[i].Value < points[i-1].Value {
				points[i].Value = points[i-1].Value + 1
			}
		}
	}
}

func buildBaselinePoints(points []schema.MetricPoint) []schema.MetricPoint {
	if len(points) == 0 {
		return nil
	}
	out := make([]schema.MetricPoint, len(points))
	for i, pt := range points {
		factor := 0.90 + float64(i%5)*0.008
		out[i] = schema.MetricPoint{Timestamp: pt.Timestamp, Value: math.Round(pt.Value*factor*100) / 100}
	}
	return out
}

func applyScenarioMetricAnomalies(points []schema.MetricPoint, anomalies []ScenarioMetricAnomaly, metricName, service string, queryStart, queryEnd time.Time) []map[string]any {
	if len(points) == 0 || len(anomalies) == 0 {
		return nil
	}
	effects := make([]map[string]any, 0, len(anomalies))
	for _, anomaly := range anomalies {
		if anomaly.MetricName != "" && anomaly.MetricName != metricName {
			continue
		}
		if anomaly.Service != "" && service != "" && anomaly.Service != service {
			continue
		}
		if anomaly.Value == nil && anomaly.Factor <= 0 {
			continue
		}
		windowStart, windowEnd, ok := clampAnomalyWindow(anomaly, queryStart, queryEnd)
		if !ok {
			continue
		}
		applied := false
		for i := range points {
			ts := points[i].Timestamp
			if ts.Before(windowStart) || ts.After(windowEnd) {
				continue
			}
			if anomaly.Value != nil {
				points[i].Value = *anomaly.Value
			} else if anomaly.Factor > 0 {
				points[i].Value = math.Round(points[i].Value*anomaly.Factor*1000) / 1000
			}
			applied = true
		}
		if !applied {
			continue
		}
		effect := map[string]any{
			"scenario_id":   anomaly.ScenarioID,
			"scenario_name": anomaly.ScenarioName,
			"stage":         anomaly.StageName,
			"metric":        metricName,
			"start":         windowStart,
			"end":           windowEnd,
		}
		if service != "" {
			effect["service"] = service
		} else if anomaly.Service != "" {
			effect["service"] = anomaly.Service
		}
		if anomaly.Description != "" {
			effect["description"] = anomaly.Description
		}
		if anomaly.Factor > 0 {
			effect["anomaly_factor"] = anomaly.Factor
		}
		if anomaly.Value != nil {
			effect["value"] = *anomaly.Value
		}
		if len(anomaly.Labels) > 0 {
			effect["labels"] = anomaly.Labels
		}
		if len(anomaly.Metadata) > 0 {
			effect["data"] = anomaly.Metadata
		}
		effects = append(effects, effect)
	}
	if len(effects) == 0 {
		return nil
	}
	return effects
}

func clampAnomalyWindow(anomaly ScenarioMetricAnomaly, queryStart, queryEnd time.Time) (time.Time, time.Time, bool) {
	start := queryStart
	if !anomaly.Start.IsZero() && anomaly.Start.After(start) {
		start = anomaly.Start
	}
	end := queryEnd
	if !anomaly.End.IsZero() && anomaly.End.Before(end) {
		end = anomaly.End
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func requestedMetricNames(expr string) []string {
	if strings.TrimSpace(expr) == "" {
		return nil
	}
	lower := strings.ToLower(expr)
	builder := strings.Builder{}
	flush := func(results *[]string) {
		if builder.Len() == 0 {
			return
		}
		*results = append(*results, builder.String())
		builder.Reset()
	}
	tokens := []string{}
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
			continue
		}
		flush(&tokens)
	}
	flush(&tokens)
	seen := map[string]bool{}
	requested := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if _, ok := metricCatalogIndex[token]; ok && !seen[token] {
			requested = append(requested, token)
			seen[token] = true
		}
	}
	return requested
}

func definitionsForRequest(expr string, names []string) []metricDefinition {
	if len(names) > 0 {
		defs := make([]metricDefinition, 0, len(names))
		seen := map[string]bool{}
		for _, name := range names {
			if seen[name] {
				continue
			}
			if def, ok := metricCatalogIndex[name]; ok {
				defs = append(defs, def)
				seen[name] = true
			}
		}
		if len(defs) > 0 {
			return defs
		}
	}

	// If no specific metrics requested and expression is empty, return all catalog metrics
	if strings.TrimSpace(expr) == "" {
		return metricCatalog
	}

	return []metricDefinition{adHocDefinition(sanitizeMetricName(expr))}
}

func sanitizeMetricName(expr string) string {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return "demo.series"
	}
	for _, stop := range []string{"{", "(", "["} {
		if idx := strings.Index(trimmed, stop); idx >= 0 {
			trimmed = trimmed[:idx]
		}
	}
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return "demo.series"
	}
	return trimmed
}

func adHocDefinition(name string) metricDefinition {
	if name == "" {
		name = "demo.series"
	}
	profile := profileForExpression(name)
	unit := inferUnit(name)
	typ := inferType(name)
	return metricDefinition{
		Name:           name,
		Type:           typ,
		Unit:           unit,
		Description:    fmt.Sprintf("Synthetic metric for %s", name),
		Labels:         []string{"service", "env"},
		DefaultService: inferService(name),
		Profile:        profile,
	}
}

func scopedLabelsForDefinition(def metricDefinition, query schema.MetricQuery) map[string]any {
	labels := map[string]any{"env": envForScope(query.Scope)}
	service := def.DefaultService
	if query.Scope.Service != "" {
		service = query.Scope.Service
	}
	if service != "" {
		labels["service"] = service
	}
	if query.Scope.Team != "" {
		labels["team"] = query.Scope.Team
	} else if team := defaultTeamForMetric(service); team != "" {
		labels["team"] = team
	}
	if region := regionForMetricService(service); region != "" {
		labels["region"] = region
	}
	if query.Scope.Environment != "" {
		labels["env"] = query.Scope.Environment
	}
	for k, v := range def.ExtraLabels {
		labels[k] = v
	}
	if agg := aggregationFromMetric(def.Name); agg != "" {
		labels["aggregation"] = agg
	}

	// Add infrastructure labels based on metric type
	metricLower := strings.ToLower(def.Name)
	if strings.Contains(metricLower, "pod") || strings.Contains(metricLower, "container") {
		labels["pod"] = generatePodName(service)
		labels["namespace"] = "production"
	}
	// Add instance labels for infrastructure metrics
	if strings.Contains(metricLower, "cpu") || strings.Contains(metricLower, "memory") ||
		strings.Contains(metricLower, "disk") || strings.Contains(metricLower, "network") ||
		strings.Contains(metricLower, "node") || strings.Contains(metricLower, "load") {
		labels["instance"] = generateInstanceID(service)
		regionStr := labelString(labels, "region")
		if regionStr != "" {
			labels["availability_zone"] = generateAZ(regionStr)
		}
	}

	return labels
}

func generatePodName(service string) string {
	svcKey := strings.TrimPrefix(service, "svc-")
	return fmt.Sprintf("%s-7d4f9c8b-xk2m", svcKey)
}

func generateInstanceID(service string) string {
	svcKey := strings.TrimPrefix(service, "svc-")
	return fmt.Sprintf("%s-instance-01", svcKey)
}

func generateAZ(region string) string {
	if region == "" {
		return "us-east-1a"
	}
	return region + "a"
}

func labelString(labels map[string]any, key string) string {
	if labels == nil {
		return ""
	}
	if v, ok := labels[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func buildSeriesMetadata(def metricDefinition, query schema.MetricQuery, labels map[string]any, start, end time.Time, step time.Duration, source, service string, points []schema.MetricPoint) map[string]any {
	unit := fallback(def.Unit, inferUnit(def.Name))
	typ := fallback(def.Type, inferType(def.Name))
	metadata := map[string]any{
		"source":      source,
		"step":        step.String(),
		"unit":        unit,
		"metricType":  typ,
		"description": def.Description,
		"window":      map[string]string{"start": start.Format(time.RFC3339), "end": end.Format(time.RFC3339)},
	}
	if query.Scope != (schema.QueryScope{}) {
		metadata["scope"] = query.Scope
	}
	if service != "" {
		metadata["service"] = service
		// Add deployment version
		metadata["version"] = generateVersion(service)
	}
	if region := labelString(labels, "region"); region != "" {
		metadata["region"] = region
	}
	if annotations := annotationsForSeries(service, points); len(annotations) > 0 {
		metadata["annotations"] = annotations
	}
	return metadata
}

func generateVersion(service string) string {
	svcKey := strings.TrimPrefix(service, "svc-")
	return fmt.Sprintf("%s-v2.14.3", svcKey)
}

func envForScope(scope schema.QueryScope) string {
	if scope.Environment != "" {
		return scope.Environment
	}
	return "prod"
}

func annotationsForSeries(service string, points []schema.MetricPoint) []map[string]any {
	if len(points) < 3 {
		return nil
	}
	first := points[len(points)/3]
	second := points[(len(points)*2)/3]
	label := fallback(strings.TrimSpace(service), "service")
	return []map[string]any{
		{"kind": "deploy", "label": fmt.Sprintf("%s release", label), "at": first.Timestamp},
		{"kind": "anomaly", "label": fmt.Sprintf("%s spike", label), "at": second.Timestamp},
	}
}

func regionForMetricService(service string) string {
	switch metricServiceKey(service) {
	case "checkout", "order":
		return "use1"
	case "search":
		return "usw2"
	case "web":
		return "global"
	case "identity":
		return "use1"
	case "analytics":
		return "apse1"
	case "warehouse":
		return "usw2"
	default:
		return "use1"
	}
}

func defaultTeamForMetric(service string) string {
	switch metricServiceKey(service) {
	case "checkout", "order", "web":
		return "team-velocity"
	case "search":
		return "team-aurora"
	case "analytics":
		return "team-lumen"
	case "identity":
		return "team-guardian"
	case "warehouse":
		return "team-foundry"
	default:
		return ""
	}
}

func aggregationFromMetric(metric string) string {
	lower := strings.ToLower(metric)
	switch {
	case strings.Contains(lower, "p99"):
		return "p99"
	case strings.Contains(lower, "p95"):
		return "p95"
	case strings.Contains(lower, "sum"):
		return "sum"
	case strings.Contains(lower, "count"):
		return "count"
	case strings.Contains(lower, "avg"):
		return "avg"
	default:
		return "mean"
	}
}

func metricServiceKey(service string) string {
	if service == "" {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(service), "svc-")
}

// ScenarioMetricAnomaly describes how a scenario should influence metric output.
type ScenarioMetricAnomaly struct {
	ScenarioID   string
	ScenarioName string
	StageName    string
	MetricName   string
	Service      string
	Labels       map[string]string
	Value        *float64
	Factor       float64
	Start        time.Time
	End          time.Time
	Description  string
	Metadata     map[string]any
}

// getScenarioMetricAnomalies returns static scenario-themed metric anomalies
func getScenarioMetricAnomalies(now time.Time) []ScenarioMetricAnomaly {
	return []ScenarioMetricAnomaly{
		{
			ScenarioID:   "scenario-001",
			ScenarioName: "SLO Budget Exhaustion",
			StageName:    "active",
			MetricName:   "http_request_duration_seconds",
			Service:      "svc-checkout",
			Factor:       2.5,
			Start:        now.Add(-25 * time.Minute),
			End:          now.Add(-10 * time.Minute),
			Description:  "Elevated latency during SLO budget exhaustion",
			Metadata: map[string]any{
				"anomaly_type": "latency_spike",
				"severity":     "critical",
			},
		},
		{
			ScenarioID:   "scenario-001",
			ScenarioName: "SLO Budget Exhaustion",
			StageName:    "active",
			MetricName:   "http_errors_total",
			Service:      "svc-checkout",
			Factor:       3.2,
			Start:        now.Add(-25 * time.Minute),
			End:          now.Add(-10 * time.Minute),
			Description:  "Increased error rate during SLO budget exhaustion",
			Metadata: map[string]any{
				"anomaly_type": "error_spike",
				"severity":     "critical",
			},
		},
		{
			ScenarioID:   "scenario-001",
			ScenarioName: "SLO Budget Exhaustion",
			StageName:    "guardrail",
			MetricName:   "cart_abandonment_rate",
			Service:      "svc-web",
			Factor:       1.25,
			Start:        now.Add(-30 * time.Minute),
			End:          now.Add(-12 * time.Minute),
			Description:  "Cart abandonment rises while latency violates the SLO",
			Metadata: map[string]any{
				"anomaly_type":      "business_impact",
				"throttling_policy": "adaptive",
			},
		},
		{
			ScenarioID:   "scenario-001",
			ScenarioName: "SLO Budget Exhaustion",
			StageName:    "customer-impact",
			MetricName:   "conversion_rate",
			Service:      "svc-web",
			Factor:       0.72,
			Start:        now.Add(-22 * time.Minute),
			End:          now.Add(-9 * time.Minute),
			Description:  "Checkout conversion drops while latency violates the SLO",
			Metadata: map[string]any{
				"anomaly_type":          "business_impact",
				"affected_segments":     []string{"web", "mobile"},
				"estimated_lost_orders": 420,
			},
		},
		{
			ScenarioID:   "scenario-002",
			ScenarioName: "Cascading Database Failure",
			StageName:    "escalating",
			MetricName:   "db_connections_active",
			Service:      "svc-search",
			Factor:       0,
			Value:        floatPtr(100.0),
			Start:        now.Add(-20 * time.Minute),
			End:          now.Add(-5 * time.Minute),
			Description:  "Database connection pool exhausted",
			Metadata: map[string]any{
				"anomaly_type": "resource_exhaustion",
				"severity":     "critical",
				"pool_size":    100,
			},
		},
		{
			ScenarioID:   "scenario-002",
			ScenarioName: "Cascading Database Failure",
			StageName:    "escalating",
			MetricName:   "db_query_duration_seconds",
			Service:      "svc-search",
			Factor:       4.5,
			Start:        now.Add(-20 * time.Minute),
			End:          now.Add(-5 * time.Minute),
			Description:  "Slow queries due to connection pool exhaustion",
			Metadata: map[string]any{
				"anomaly_type": "latency_spike",
				"severity":     "critical",
			},
		},
		{
			ScenarioID:   "scenario-002",
			ScenarioName: "Cascading Database Failure",
			StageName:    "failover",
			MetricName:   "db_replication_lag_seconds",
			Service:      "svc-database",
			Factor:       4.2,
			Start:        now.Add(-22 * time.Minute),
			End:          now.Add(-4 * time.Minute),
			Description:  "Replica lag grows while traffic shifts between primaries",
			Labels: map[string]string{
				"replica": "read-replica-1",
			},
			Metadata: map[string]any{
				"anomaly_type":    "replication_lag",
				"severity":        "critical",
				"failover_active": true,
			},
		},
		{
			ScenarioID:   "scenario-002",
			ScenarioName: "Cascading Database Failure",
			StageName:    "failover",
			MetricName:   "db_deadlocks_total",
			Service:      "svc-database",
			Factor:       3.5,
			Start:        now.Add(-18 * time.Minute),
			End:          now.Add(-6 * time.Minute),
			Description:  "Deadlocks spike as retries pile up during failover",
			Metadata: map[string]any{
				"anomaly_type": "lock_contention",
				"severity":     "error",
			},
		},
		{
			ScenarioID:   "scenario-003",
			ScenarioName: "Deployment Rollback",
			StageName:    "mitigating",
			MetricName:   "error_rate",
			Service:      "svc-checkout",
			Factor:       5.0,
			Start:        now.Add(-12 * time.Minute),
			End:          now.Add(-9 * time.Minute),
			Description:  "Error rate spike triggering deployment rollback",
			Metadata: map[string]any{
				"anomaly_type":    "error_spike",
				"severity":        "critical",
				"deployment_id":   "deploy-2024-12-07-003",
				"rollback_reason": "error_rate_threshold_exceeded",
			},
		},
		{
			ScenarioID:   "scenario-003",
			ScenarioName: "Deployment Rollback",
			StageName:    "instability",
			MetricName:   "container_restarts_total",
			Service:      "svc-payments",
			Factor:       2.8,
			Start:        now.Add(-18 * time.Minute),
			End:          now.Add(-6 * time.Minute),
			Description:  "Pods restart repeatedly while the rollback proceeds",
			Metadata: map[string]any{
				"anomaly_type":      "deployment_instability",
				"failing_version":   "payments-v2.8.3",
				"stabilizing_build": "v2.8.2",
			},
		},
		{
			ScenarioID:   "scenario-003",
			ScenarioName: "Deployment Rollback",
			StageName:    "business-impact",
			MetricName:   "session_active",
			Service:      "svc-web",
			Factor:       0.78,
			Start:        now.Add(-20 * time.Minute),
			End:          now.Add(-8 * time.Minute),
			Description:  "Active sessions drop while checkout errors persist",
			Metadata: map[string]any{
				"anomaly_type":            "user_friction",
				"estimated_lost_orders":   280,
				"lost_revenue_per_minute": 5200,
			},
		},
		{
			ScenarioID:   "scenario-004",
			ScenarioName: "External Dependency Failure - Stripe",
			StageName:    "active",
			MetricName:   "http_request_duration_seconds",
			Service:      "svc-checkout",
			Factor:       3.8,
			Start:        now.Add(-12 * time.Minute),
			End:          now.Add(-3 * time.Minute),
			Description:  "Increased latency due to Stripe API rate limiting",
			Metadata: map[string]any{
				"anomaly_type":     "latency_spike",
				"severity":         "error",
				"external_service": "stripe",
				"external_error":   "rate_limit_exceeded",
			},
		},
		{
			ScenarioID:   "scenario-004",
			ScenarioName: "External Dependency Failure - Stripe",
			StageName:    "active",
			MetricName:   "http_errors_total",
			Service:      "svc-checkout",
			Factor:       4.2,
			Start:        now.Add(-12 * time.Minute),
			End:          now.Add(-3 * time.Minute),
			Description:  "Increased errors due to Stripe API failures",
			Metadata: map[string]any{
				"anomaly_type":     "error_spike",
				"severity":         "error",
				"external_service": "stripe",
			},
		},
		{
			ScenarioID:   "scenario-004",
			ScenarioName: "External Dependency Failure - Stripe",
			StageName:    "rate-limited",
			MetricName:   "api_rate_limit_exceeded_total",
			Service:      "svc-api-gateway",
			Factor:       4.5,
			Start:        now.Add(-13 * time.Minute),
			End:          now.Add(-3 * time.Minute),
			Description:  "API gateway floods Stripe until rate limits hit",
			Metadata: map[string]any{
				"anomaly_type": "external_dependency",
				"provider":     "stripe",
				"limit_bucket": "write",
			},
		},
		{
			ScenarioID:   "scenario-004",
			ScenarioName: "External Dependency Failure - Stripe",
			StageName:    "business-impact",
			MetricName:   "revenue_total",
			Service:      "svc-order",
			Factor:       0.78,
			Start:        now.Add(-15 * time.Minute),
			End:          now.Add(-4 * time.Minute),
			Description:  "Revenue drops while payment authorizations fail",
			Metadata: map[string]any{
				"anomaly_type":              "business_impact",
				"estimated_loss_per_minute": 8400,
			},
		},
		{
			ScenarioID:   "scenario-005",
			ScenarioName: "Autoscaling Lag",
			StageName:    "active",
			MetricName:   "cpu_usage_ratio",
			Service:      "svc-search",
			Factor:       1.8,
			Start:        now.Add(-8 * time.Minute),
			End:          now.Add(-2 * time.Minute),
			Description:  "Elevated CPU usage during autoscaling lag",
			Metadata: map[string]any{
				"anomaly_type":       "resource_pressure",
				"severity":           "warning",
				"autoscaling_status": "scaling_up",
				"current_instances":  3,
				"target_instances":   8,
			},
		},
		{
			ScenarioID:   "scenario-005",
			ScenarioName: "Autoscaling Lag",
			StageName:    "active",
			MetricName:   "http_request_duration_seconds",
			Service:      "svc-search",
			Factor:       2.1,
			Start:        now.Add(-8 * time.Minute),
			End:          now.Add(-2 * time.Minute),
			Description:  "Increased latency during autoscaling lag",
			Metadata: map[string]any{
				"anomaly_type": "latency_spike",
				"severity":     "warning",
			},
		},
		{
			ScenarioID:   "scenario-005",
			ScenarioName: "Autoscaling Lag",
			StageName:    "capacity",
			MetricName:   "cpu_throttling_seconds_total",
			Service:      "svc-search",
			Factor:       3.8,
			Start:        now.Add(-9 * time.Minute),
			End:          now.Add(-2 * time.Minute),
			Description:  "CPU throttling surges until new pods are ready",
			Metadata: map[string]any{
				"anomaly_type":       "resource_pressure",
				"autoscaling_status": "lagging",
			},
		},
		{
			ScenarioID:   "scenario-005",
			ScenarioName: "Autoscaling Lag",
			StageName:    "queueing",
			MetricName:   "background_jobs_queued",
			Service:      "svc-search",
			Factor:       1.9,
			Start:        now.Add(-7 * time.Minute),
			End:          now.Add(-1 * time.Minute),
			Description:  "Search job backlog builds while capacity catches up",
			Metadata: map[string]any{
				"anomaly_type": "queue_backlog",
				"queue":        "search-indexer",
			},
		},
		{
			ScenarioID:   "scenario-006",
			ScenarioName: "Circuit Breaker Cascade",
			StageName:    "escalating",
			MetricName:   "circuit_breaker_state",
			Service:      "svc-checkout",
			Factor:       0,
			Value:        floatPtr(1.0),
			Start:        now.Add(-5 * time.Minute),
			End:          now.Add(-1 * time.Minute),
			Description:  "Circuit breaker opened due to cascading failures",
			Metadata: map[string]any{
				"anomaly_type":       "circuit_open",
				"severity":           "critical",
				"downstream_service": "payment-gateway",
			},
		},
		{
			ScenarioID:   "scenario-006",
			ScenarioName: "Circuit Breaker Cascade",
			StageName:    "escalating",
			MetricName:   "http_errors_total",
			Service:      "svc-checkout",
			Factor:       6.5,
			Start:        now.Add(-5 * time.Minute),
			End:          now.Add(-1 * time.Minute),
			Description:  "Elevated errors during circuit breaker cascade",
			Metadata: map[string]any{
				"anomaly_type": "error_spike",
				"severity":     "critical",
			},
		},
		{
			ScenarioID:   "scenario-006",
			ScenarioName: "Circuit Breaker Cascade",
			StageName:    "escalating",
			MetricName:   "circuit_breaker_trips_total",
			Service:      "svc-web",
			Factor:       4.0,
			Start:        now.Add(-6 * time.Minute),
			End:          now.Add(-1 * time.Minute),
			Description:  "Circuit breaker trips pile up as dependencies flap",
			Metadata: map[string]any{
				"anomaly_type":       "circuit_instability",
				"downstream_service": "payment-gateway",
			},
		},
		{
			ScenarioID:   "scenario-006",
			ScenarioName: "Circuit Breaker Cascade",
			StageName:    "customer-impact",
			MetricName:   "active_users_total",
			Service:      "svc-web",
			Factor:       0.6,
			Start:        now.Add(-6 * time.Minute),
			End:          now.Add(-2 * time.Minute),
			Description:  "Active sessions drop as users hit cascading failures",
			Metadata: map[string]any{
				"anomaly_type": "user_impact",
				"channels":     []string{"web", "mobile"},
			},
		},
	}
}

func floatPtr(f float64) *float64 {
	return &f
}

var _ metric.Provider = (*Provider)(nil)
