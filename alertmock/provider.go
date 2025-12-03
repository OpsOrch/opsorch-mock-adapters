package alertmock

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/opsorch/opsorch-core/alert"
	"github.com/opsorch/opsorch-core/orcherr"
	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/internal/mockutil"
)

// ProviderName can be referenced via OPSORCH_ALERT_PROVIDER.
const ProviderName = "mock"

// Config controls mock alert behavior.
type Config struct {
	Source string
}

// Provider serves seeded alerts for demo purposes.
type Provider struct {
	cfg    Config
	mu     sync.Mutex
	alerts map[string]schema.Alert
}

// New constructs the provider with seeded demo alerts.
func New(cfg map[string]any) (alert.Provider, error) {
	parsed := parseConfig(cfg)
	p := &Provider{cfg: parsed, alerts: map[string]schema.Alert{}}
	p.seed()
	return p, nil
}

func init() {
	_ = alert.RegisterProvider(ProviderName, New)
}

// WithScope attaches a QueryScope so Query can merge it with inline filters.
func WithScope(ctx context.Context, scope schema.QueryScope) context.Context {
	return context.WithValue(ctx, scopeKey{}, scope)
}

type scopeKey struct{}

// Query returns alerts filtered by status/severity/scope/query.
func (p *Provider) Query(ctx context.Context, query schema.AlertQuery) ([]schema.Alert, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	combinedScope := mergeScope(extractScope(ctx), query.Scope)
	statusFilter := toSet(query.Statuses)
	severityFilter := toSet(query.Severities)
	needle := strings.ToLower(strings.TrimSpace(query.Query))

	out := make([]schema.Alert, 0, len(p.alerts))
	for _, al := range p.alerts {
		if !matchesScope(combinedScope, al) {
			continue
		}
		if len(statusFilter) > 0 && !statusFilter[al.Status] {
			continue
		}
		if len(severityFilter) > 0 && !severityFilter[al.Severity] {
			continue
		}
		if needle != "" && !matchesQuery(needle, al) {
			continue
		}

		out = append(out, cloneAlert(al))
		if query.Limit > 0 && len(out) >= query.Limit {
			break
		}
	}
	return out, nil
}

// Get fetches an alert by ID.
func (p *Provider) Get(ctx context.Context, id string) (schema.Alert, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	al, ok := p.alerts[id]
	if !ok {
		return schema.Alert{}, orcherr.New("not_found", "alert not found", nil)
	}
	return cloneAlert(al), nil
}

func (p *Provider) seed() {
	now := time.Now().UTC()
	seed := []schema.Alert{
		{
			ID:          "al-001",
			Title:       "Checkout latency SLO breach",
			Description: "Checkout p95 latency exceeded 1.2s for the past 15 minutes",
			Status:      "firing",
			Severity:    "critical",
			Service:     "svc-checkout",
			CreatedAt:   now.Add(-25 * time.Minute),
			UpdatedAt:   now.Add(-5 * time.Minute),
			Fields: map[string]any{
				"metric":      "http_request_duration_seconds:p95",
				"environment": "prod",
				"team":        "team-velocity",
				"region":      "euw1",
				"threshold":   "1.2s",
			},
			Metadata: map[string]any{"ruleId": "mon-checkout-latency", "dashboard": "dash-checkout"},
		},
		{
			ID:          "al-002",
			Title:       "Search 5xx spike on cluster ares",
			Description: "Search error budget is being consumed due to node instability",
			Status:      "firing",
			Severity:    "error",
			Service:     "svc-search",
			CreatedAt:   now.Add(-45 * time.Minute),
			UpdatedAt:   now.Add(-10 * time.Minute),
			Fields: map[string]any{
				"metric":      "http_requests_total:5xx",
				"environment": "prod",
				"team":        "team-aurora",
				"cluster":     "ares",
			},
			Metadata: map[string]any{"ruleId": "mon-search-5xx", "runbook": "https://runbook.demo/search-5xx"},
		},
		{
			ID:          "al-003",
			Title:       "Payments webhook retries exhausted",
			Description: "Stripe webhook deliveries repeated 5 times without success",
			Status:      "acknowledged",
			Severity:    "warning",
			Service:     "svc-payments",
			CreatedAt:   now.Add(-2 * time.Hour),
			UpdatedAt:   now.Add(-40 * time.Minute),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-revenue",
				"provider":    "stripe",
				"region":      "us-east-1",
			},
			Metadata: map[string]any{"ruleId": "mon-payments-webhook", "channel": "#payments"},
		},
		{
			ID:          "al-004",
			Title:       "Notification queue depth high",
			Description: "Promo notification fanout queue depth above 40k messages",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-notifications",
			CreatedAt:   now.Add(-90 * time.Minute),
			UpdatedAt:   now.Add(-20 * time.Minute),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-signal",
				"queue":       "promo-delivery",
				"depth":       48000,
			},
			Metadata: map[string]any{"ruleId": "mon-notify-queue", "dashboard": "dash-notifications"},
		},
		{
			ID:          "al-005",
			Title:       "Auth token issuance drop",
			Description: "Token issuance rate dropped below 250/s for mobile clients",
			Status:      "resolved",
			Severity:    "critical",
			Service:     "svc-identity",
			CreatedAt:   now.Add(-4 * time.Hour),
			UpdatedAt:   now.Add(-30 * time.Minute),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-guardian",
				"metric":      "issued_tokens_total",
				"platform":    "mobile",
			},
			Metadata: map[string]any{"ruleId": "mon-auth-issuance", "runbook": "https://runbook.demo/auth-tokens"},
		},
		{
			ID:          "al-006",
			Title:       "Warehouse ETL runtime variance",
			Description: "ETL job runtime variance exceeded 3x baseline",
			Status:      "firing",
			Severity:    "info",
			Service:     "svc-warehouse",
			CreatedAt:   now.Add(-3 * time.Hour),
			UpdatedAt:   now.Add(-50 * time.Minute),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-foundry",
				"job":         "warehouse-etl",
				"region":      "us-west-2",
			},
			Metadata: map[string]any{"ruleId": "mon-warehouse-duration"},
		},
		{
			ID:          "al-007",
			Title:       "Realtime websocket disconnects",
			Description: "Firefox clients disconnect after ~45s with close code 1006",
			Status:      "acknowledged",
			Severity:    "error",
			Service:     "svc-realtime",
			CreatedAt:   now.Add(-75 * time.Minute),
			UpdatedAt:   now.Add(-15 * time.Minute),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-nova",
				"browser":     "firefox",
			},
			Metadata: map[string]any{"ruleId": "mon-realtime-disconnect"},
		},
		{
			ID:          "al-008",
			Title:       "Analytics pipeline APAC gap",
			Description: "APAC tracking stream produced zero events for 12 minutes",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-analytics",
			CreatedAt:   now.Add(-110 * time.Minute),
			UpdatedAt:   now.Add(-12 * time.Minute),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-lumen",
				"region":      "apac",
				"stream":      "analytics-tracking",
			},
			Metadata: map[string]any{"ruleId": "mon-analytics-gap", "channel": "#analytics"},
		},
	}

	for _, al := range seed {
		alertCopy := al
		if alertCopy.Metadata == nil {
			alertCopy.Metadata = map[string]any{}
		}
		alertCopy.Metadata["source"] = p.cfg.Source
		p.alerts[alertCopy.ID] = alertCopy
	}
}

func parseConfig(cfg map[string]any) Config {
	out := Config{Source: "mock-alert"}
	if v, ok := cfg["source"].(string); ok && v != "" {
		out.Source = v
	}
	return out
}

func cloneAlert(in schema.Alert) schema.Alert {
	return schema.Alert{
		ID:          in.ID,
		Title:       in.Title,
		Description: in.Description,
		Status:      in.Status,
		Severity:    in.Severity,
		Service:     in.Service,
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
		Fields:      mockutil.CloneMap(in.Fields),
		Metadata:    mockutil.CloneMap(in.Metadata),
	}
}

func extractScope(ctx context.Context) schema.QueryScope {
	if ctx == nil {
		return schema.QueryScope{}
	}
	if v, ok := ctx.Value(scopeKey{}).(schema.QueryScope); ok {
		return v
	}
	return schema.QueryScope{}
}

func mergeScope(ctxScope, queryScope schema.QueryScope) schema.QueryScope {
	out := ctxScope
	if queryScope.Service != "" {
		out.Service = queryScope.Service
	}
	if queryScope.Environment != "" {
		out.Environment = queryScope.Environment
	}
	if queryScope.Team != "" {
		out.Team = queryScope.Team
	}
	return out
}

func matchesScope(scope schema.QueryScope, al schema.Alert) bool {
	if scope == (schema.QueryScope{}) {
		return true
	}

	if scope.Service != "" {
		svc := al.Service
		if svc == "" && al.Fields != nil {
			if v, ok := al.Fields["service"].(string); ok {
				svc = v
			}
		}
		if svc != scope.Service {
			return false
		}
	}
	if scope.Environment != "" {
		if env, ok := al.Fields["environment"].(string); !ok || env != scope.Environment {
			return false
		}
	}
	if scope.Team != "" {
		if team, ok := al.Fields["team"].(string); !ok || team != scope.Team {
			return false
		}
	}
	return true
}

func matchesQuery(needle string, al schema.Alert) bool {
	if needle == "" {
		return true
	}
	lowerTitle := strings.ToLower(al.Title)
	if strings.Contains(lowerTitle, needle) {
		return true
	}
	if al.Description != "" && strings.Contains(strings.ToLower(al.Description), needle) {
		return true
	}
	if al.Service != "" && strings.Contains(strings.ToLower(al.Service), needle) {
		return true
	}
	if strings.Contains(strings.ToLower(al.ID), needle) {
		return true
	}
	for _, v := range al.Fields {
		if s, ok := v.(string); ok {
			if strings.Contains(strings.ToLower(s), needle) {
				return true
			}
		}
	}
	return false
}

func toSet(vals []string) map[string]bool {
	if len(vals) == 0 {
		return nil
	}
	out := make(map[string]bool, len(vals))
	for _, v := range vals {
		if v == "" {
			continue
		}
		out[v] = true
	}
	return out
}
