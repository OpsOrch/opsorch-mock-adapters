package metricmock

import (
	"context"
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

// New constructs the mock metric provider.
func New(cfg map[string]any) (metric.Provider, error) {
	parsed := parseConfig(cfg)
	return &Provider{cfg: parsed}, nil
}

func init() {
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

	points := generatePoints(start, end, step, metricName)
	labels := scopedLabels(query)
	service, _ := labels["service"].(string)
	unit := inferUnit(metricName)
	metadata := map[string]any{"source": p.cfg.Source, "step": step.String(), "unit": unit}
	if query.Scope != (schema.QueryScope{}) {
		metadata["scope"] = query.Scope
	}

	series := schema.MetricSeries{
		Name:     fallback(metricName, "demo.series"),
		Service:  service,
		Labels:   labels,
		Points:   points,
		Metadata: metadata,
	}
	// Provide a comparative shadow series to help dashboards show multiple lines
	comparison := series
	comparison.Service = service
	comparison.Name = series.Name + ".baseline"
	comparison.Labels = mockutil.CloneMap(series.Labels)
	comparison.Labels["variant"] = "baseline"
	comparison.Points = make([]schema.MetricPoint, len(series.Points))
	copy(comparison.Points, series.Points)
	for i := range comparison.Points {
		comparison.Points[i].Value = comparison.Points[i].Value * 0.92
	}

	return []schema.MetricSeries{series, comparison}, nil
}

// Describe lists available metrics.
func (p *Provider) Describe(ctx context.Context, scope schema.QueryScope) ([]schema.MetricDescriptor, error) {
	return []schema.MetricDescriptor{
		{
			Name:        "http_requests_total",
			Type:        "counter",
			Description: "Total number of HTTP requests",
			Labels:      []string{"service", "method", "status"},
			Unit:        "requests",
		},
		{
			Name:        "http_request_duration_seconds",
			Type:        "histogram",
			Description: "HTTP request latency",
			Labels:      []string{"service", "method", "status"},
			Unit:        "seconds",
		},
		{
			Name:        "process_resident_memory_bytes",
			Type:        "gauge",
			Description: "Resident memory size in bytes",
			Labels:      []string{"service", "instance"},
			Unit:        "bytes",
		},
		{
			Name:        "process_cpu_seconds_total",
			Type:        "counter",
			Description: "Total user and system CPU time spent in seconds",
			Labels:      []string{"service", "instance"},
			Unit:        "seconds",
		},
	}, nil
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

func generatePoints(start, end time.Time, step time.Duration, expr string) []schema.MetricPoint {
	points := []schema.MetricPoint{}
	profile := profileForExpression(expr)
	metricType := inferType(expr)

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

func scopedLabels(query schema.MetricQuery) map[string]any {
	labels := map[string]any{"env": "prod"}

	// Respect scope overrides first.
	if query.Scope.Environment != "" {
		labels["env"] = query.Scope.Environment
	}
	if query.Scope.Team != "" {
		labels["team"] = query.Scope.Team
	}

	metricName := ""
	if query.Expression != nil {
		metricName = query.Expression.MetricName
	}

	if query.Scope.Service != "" {
		labels["service"] = query.Scope.Service
	} else if svc := inferService(metricName); svc != "" {
		labels["service"] = svc
	}

	return labels
}

var _ metric.Provider = (*Provider)(nil)
