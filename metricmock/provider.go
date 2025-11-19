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
	step := query.Step
	if step <= 0 {
		step = 30 * time.Second
	}

	points := generatePoints(start, end, step, query.Expression)
	labels := scopedLabels(query)
	service, _ := labels["service"].(string)
	unit := inferUnit(query.Expression)
	metadata := map[string]any{"source": p.cfg.Source, "step": step.String(), "unit": unit}
	if query.Scope != (schema.QueryScope{}) {
		metadata["scope"] = query.Scope
	}

	series := schema.MetricSeries{
		Name:     fallback(query.Expression, "demo.series"),
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
	count := int(end.Sub(start) / step)
	if count < 3 {
		count = 3
	}

	for i := 0; i <= count; i++ {
		ts := start.Add(time.Duration(i) * step)
		if ts.After(end) {
			break
		}
		// Deterministic wave so charts look interesting but predictable.
		wave := math.Sin(float64(i)/3.5) * profile.amplitude
		trend := profile.trend * float64(i)
		noise := float64((i%4)-1) * 0.35
		val := profile.baseline + wave + trend + noise
		if val < 0 {
			val = 0
		}
		points = append(points, schema.MetricPoint{Timestamp: ts, Value: math.Round(val*100) / 100})
	}
	return points
}

type seriesProfile struct {
	baseline  float64
	amplitude float64
	trend     float64
}

func profileForExpression(expr string) seriesProfile {
	lower := strings.ToLower(expr)
	switch {
	case strings.Contains(lower, "latency"):
		return seriesProfile{baseline: 210, amplitude: 45, trend: 0.4}
	case strings.Contains(lower, "error"):
		return seriesProfile{baseline: 0.3, amplitude: 0.18, trend: 0.02}
	case strings.Contains(lower, "rps") || strings.Contains(lower, "throughput"):
		return seriesProfile{baseline: 1600, amplitude: 220, trend: 3.5}
	case strings.Contains(lower, "active") || strings.Contains(lower, "connections"):
		return seriesProfile{baseline: 320, amplitude: 55, trend: 1.1}
	default:
		seed := float64(len(expr)%7 + 4)
		return seriesProfile{baseline: seed * 4, amplitude: seed, trend: seed * 0.1}
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
	if query.Scope.Service != "" {
		labels["service"] = query.Scope.Service
	} else if svc := inferService(query.Expression); svc != "" {
		labels["service"] = svc
	}

	return labels
}

var _ metric.Provider = (*Provider)(nil)
