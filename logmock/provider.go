package logmock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/opsorch/opsorch-core/log"
	"github.com/opsorch/opsorch-core/schema"
)

// ProviderName can be referenced via OPSORCH_LOG_PROVIDER.
const ProviderName = "mock"

// Config tunes mock log behavior.
type Config struct {
	DefaultLimit int
	Source       string
}

// Provider returns generated log entries for demo queries.
type Provider struct {
	cfg Config
}

// New constructs the mock log provider.
func New(cfg map[string]any) (log.Provider, error) {
	parsed := parseConfig(cfg)
	return &Provider{cfg: parsed}, nil
}

func init() {
	_ = log.RegisterProvider(ProviderName, New)
}

// Query returns synthetic log entries that echo the query context.
func (p *Provider) Query(ctx context.Context, query schema.LogQuery) ([]schema.LogEntry, error) {
	_ = ctx

	end := query.End
	if end.IsZero() {
		end = time.Now().UTC()
	}
	start := query.Start
	if start.IsZero() {
		start = end.Add(-1 * time.Hour)
	}
	limit := query.Limit
	if limit <= 0 {
		limit = p.cfg.DefaultLimit
	}
	service := inferService(query)
	severity := inferSeverity(query.Query)

	count := limit
	if count > 20 {
		count = 20 // keep responses small for demos
	}
	if count < 3 {
		count = 3
	}
	step := end.Sub(start) / time.Duration(count+1)
	if step <= 0 {
		step = 5 * time.Second
	}

	entries := make([]schema.LogEntry, 0, count)
	for i := 0; i < count; i++ {
		ts := start.Add(time.Duration(i+1) * step)
		labels := scopedLabels(query, service)
		method, path := requestShape(service, i)
		status := responseStatus(severity, i)
		latency := baseLatency(severity, i)
		traceID := fmt.Sprintf("trace-%05d", 4200+i)
		user := []string{"alice", "sam", "casey", "fern"}[i%4]
		entries = append(entries, schema.LogEntry{
			Timestamp: ts,
			Message:   fmt.Sprintf("%s %s %d in %dms | user=%s trace=%s | %s", method, path, status, latency, user, traceID, fallback(query.Query, "service logs")),
			Severity:  severity,
			Service:   service,
			Labels:    labels,
			Fields: map[string]any{
				"requestId": fmt.Sprintf("req-%06d", i+1),
				"path":      path,
				"method":    method,
				"status":    status,
				"latencyMs": latency,
				"traceId":   traceID,
				"user":      user,
			},
			Metadata: scopedMetadata(p.cfg.Source, query),
		})
	}

	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

func parseConfig(cfg map[string]any) Config {
	out := Config{DefaultLimit: 50, Source: "mock-log"}
	if v, ok := cfg["defaultLimit"].(int); ok && v > 0 {
		out.DefaultLimit = v
	}
	if v, ok := cfg["defaultLimit"].(float64); ok && v > 0 { // configs come in via JSON
		out.DefaultLimit = int(v)
	}
	if v, ok := cfg["source"].(string); ok && v != "" {
		out.Source = v
	}
	return out
}

func inferService(q schema.LogQuery) string {
	if q.Scope.Service != "" {
		return q.Scope.Service
	}
	if v, ok := q.Metadata["service"].(string); ok && v != "" {
		return v
	}
	lower := strings.ToLower(q.Query)
	samples := []string{"checkout", "search", "web"}
	for _, s := range samples {
		if strings.Contains(lower, s) {
			return s
		}
	}
	return ""
}

func inferSeverity(query string) string {
	lower := strings.ToLower(query)
	switch {
	case strings.Contains(lower, "error"):
		return "error"
	case strings.Contains(lower, "warn"):
		return "warn"
	default:
		return "info"
	}
}

func fallback(val, def string) string {
	if strings.TrimSpace(val) != "" {
		return val
	}
	return def
}

func requestShape(service string, idx int) (method string, path string) {
	methods := []string{"GET", "POST", "PUT"}
	method = methods[idx%len(methods)]

	perService := map[string][]string{
		"checkout": {"/api/checkout", "/api/checkout/order", "/api/payments/charge"},
		"search":   {"/api/search", "/api/search/suggestions", "/api/search/trending"},
		"web":      {"/", "/healthz", "/static/app.js"},
	}
	paths, ok := perService[service]
	if !ok || len(paths) == 0 {
		paths = []string{"/api/demo", "/api/internal/metrics", "/healthz"}
	}
	path = paths[idx%len(paths)]
	return
}

func responseStatus(severity string, idx int) int {
	if severity == "error" {
		return []int{500, 502, 504}[idx%3]
	}
	if severity == "warn" {
		return []int{200, 206, 429}[idx%3]
	}
	return []int{200, 200, 204, 304}[idx%4]
}

func baseLatency(severity string, idx int) int {
	base := 45 + idx%17
	switch severity {
	case "error":
		return base + 320
	case "warn":
		return base + 90
	default:
		return base + (idx % 5 * 5)
	}
}

func scopedLabels(q schema.LogQuery, service string) map[string]string {
	labels := map[string]string{"env": fallbackEnv(q)}
	if service != "" {
		labels["service"] = service
	}
	if q.Scope.Team != "" {
		labels["team"] = q.Scope.Team
	}
	return labels
}

func fallbackEnv(q schema.LogQuery) string {
	if q.Scope.Environment != "" {
		return q.Scope.Environment
	}
	return "prod"
}

func scopedMetadata(source string, q schema.LogQuery) map[string]any {
	metadata := map[string]any{
		"source":  source,
		"matched": strings.TrimSpace(q.Query),
	}
	if q.Scope != (schema.QueryScope{}) {
		metadata["scope"] = q.Scope
	}
	return metadata
}

// ensure compile-time interface compatibility
var _ log.Provider = (*Provider)(nil)
