package alertmock

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

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
	cfg       Config
	mu        sync.Mutex
	alerts    map[string]schema.Alert
	lifecycle map[string]*alertLifecycle
}

// New constructs the provider with seeded demo alerts.
func New(cfg map[string]any) (alert.Provider, error) {
	parsed := parseConfig(cfg)
	p := &Provider{cfg: parsed, alerts: map[string]schema.Alert{}, lifecycle: map[string]*alertLifecycle{}}
	p.seed()
	return p, nil
}

func init() {
	_ = alert.RegisterProvider(ProviderName, New)
}

// generateAlertURL creates a realistic Prometheus-style alert URL
func generateAlertURL(alertID, service string, isScenario bool) string {
	if isScenario {
		return fmt.Sprintf("https://prometheus.demo.com/alerts/%s?scenario=true", alertID)
	}
	return fmt.Sprintf("https://prometheus.demo.com/alerts/%s", alertID)
}

// isScenarioAlert checks if an alert has scenario metadata
func isScenarioAlert(metadata map[string]any, fields map[string]any) bool {
	// Check for scenario markers in metadata
	if metadata != nil {
		if isScenario, ok := metadata["is_scenario"].(bool); ok && isScenario {
			return true
		}
		if _, ok := metadata["scenario_id"]; ok {
			return true
		}
		if _, ok := metadata["scenario_name"]; ok {
			return true
		}
		if _, ok := metadata["scenario_effects"]; ok {
			return true
		}
	}

	// Check for scenario markers in fields
	if fields != nil {
		if isScenario, ok := fields["is_scenario"].(bool); ok && isScenario {
			return true
		}
		if _, ok := fields["scenario_id"]; ok {
			return true
		}
		if _, ok := fields["scenario_name"]; ok {
			return true
		}
	}

	return false
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

	now := time.Now().UTC()
	p.refreshLifecycleLocked(now)

	combinedScope := mergeScope(extractScope(ctx), query.Scope)
	statusFilter := toSet(query.Statuses)
	severityFilter := toSet(query.Severities)
	needle := strings.ToLower(strings.TrimSpace(query.Query))

	// Parse the search query
	parsedQuery := mockutil.ParseSearchQuery(query.Query)

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

	// If we have a search query but no results, generate mock alerts that match
	if query.Query != "" && len(out) == 0 {
		limit := query.Limit
		if limit <= 0 {
			limit = 5
		}
		generated := p.generateAlertsForQuery(parsedQuery, combinedScope, statusFilter, severityFilter, limit, now)
		out = append(out, generated...)
	}

	return out, nil
}

// Get fetches an alert by ID.
func (p *Provider) Get(ctx context.Context, id string) (schema.Alert, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.refreshLifecycleLocked(time.Now().UTC())

	al, ok := p.alerts[id]
	if !ok {
		return schema.Alert{}, orcherr.New("not_found", "alert not found", nil)
	}
	return cloneAlert(al), nil
}

func (p *Provider) seed() {
	now := time.Now().UTC()
	seed := []schema.Alert{
		// CRITICAL ALERTS (10% - 3 alerts)
		{
			ID:          "al-001",
			Title:       "Checkout latency SLO breach",
			Description: "Checkout p95 latency exceeded 1.2s for the past 15 minutes affecting 45% of transactions",
			Status:      "firing",
			Severity:    "critical",
			Service:     "svc-checkout",
			CreatedAt:   now.Add(-25 * time.Minute),
			UpdatedAt:   now.Add(-5 * time.Minute),
			Fields: map[string]any{
				"metric":           "http_request_duration_seconds:p95",
				"environment":      "prod",
				"team":             "team-velocity",
				"region":           "euw1",
				"threshold":        "1.2s",
				"impactedSegments": []string{"checkout-web", "checkout-mobile"},
				"affectedUsers":    "~12,500 users",
				"impactPercent":    45,
				"lastDeploy":       map[string]any{"version": "checkout-v2.31.4", "at": now.Add(-70 * time.Minute).Format(time.RFC3339), "author": "release-bot"},
				"slo":              map[string]any{"targetMs": 1200, "currentMs": 1530, "breachFor": "15m"},
			},
			Metadata: map[string]any{
				"ruleId":     "mon-checkout-latency",
				"dashboard":  "dash-checkout",
				"links":      []string{"https://grafana.demo/d/checkout-latency", "https://runbook.demo/checkout-latency"},
				"pagerDuty":  "pagerduty://PRD123",
				"escalation": []string{"@oncall-velocity", "pagerduty://PRD123"},
			},
		},
		{
			ID:          "al-002",
			Title:       "Database primary failover initiated",
			Description: "Primary database node unresponsive, automatic failover to replica in progress",
			Status:      "firing",
			Severity:    "critical",
			Service:     "svc-database",
			CreatedAt:   now.Add(-8 * time.Minute),
			UpdatedAt:   now.Add(-2 * time.Minute),
			Fields: map[string]any{
				"environment":      "prod",
				"team":             "team-data",
				"region":           "us-east-1",
				"cluster":          "prod-primary",
				"failoverStatus":   "in-progress",
				"estimatedRTO":     "5 minutes",
				"affectedServices": []string{"svc-checkout", "svc-catalog", "svc-orders"},
				"impactedRegions":  []string{"us-east-1", "us-west-2"},
			},
			Metadata: map[string]any{
				"ruleId":     "mon-db-failover",
				"runbook":    "https://runbook.demo/db-failover",
				"dashboard":  "dash-database-health",
				"escalation": []string{"@oncall-data", "@db-admin", "pagerduty://DB-CRIT"},
			},
		},
		{
			ID:          "al-003",
			Title:       "Payment processing complete outage",
			Description: "Payment gateway returning 503 errors, zero successful transactions in last 10 minutes",
			Status:      "firing",
			Severity:    "critical",
			Service:     "svc-payments",
			CreatedAt:   now.Add(-12 * time.Minute),
			UpdatedAt:   now.Add(-1 * time.Minute),
			Fields: map[string]any{
				"environment":          "prod",
				"team":                 "team-revenue",
				"region":               "global",
				"provider":             "stripe",
				"errorRate":            1.0,
				"affectedUsers":        "all active checkout sessions",
				"estimatedRevenueLoss": "$8,400/minute",
				"lastSuccessfulTx":     now.Add(-10 * time.Minute).Format(time.RFC3339),
			},
			Metadata: map[string]any{
				"ruleId":     "mon-payment-outage",
				"runbook":    "https://runbook.demo/payment-outage",
				"dashboard":  "dash-payments",
				"escalation": []string{"@oncall-revenue", "@payment-lead", "pagerduty://PAY-CRIT"},
				"links":      []string{"https://status.stripe.com"},
			},
		},
		// ERROR ALERTS (25% - 7 alerts)
		{
			ID:          "al-004",
			Title:       "Search 5xx spike on cluster ares",
			Description: "Search error budget is being consumed due to node instability, 4.2% error rate",
			Status:      "firing",
			Severity:    "error",
			Service:     "svc-search",
			CreatedAt:   now.Add(-3 * time.Hour),
			UpdatedAt:   now.Add(-10 * time.Minute),
			Fields: map[string]any{
				"metric":          "http_requests_total:5xx",
				"environment":     "prod",
				"team":            "team-aurora",
				"cluster":         "ares",
				"impactedRegions": []string{"use1", "euw1"},
				"errorRatio":      0.042,
				"affectedQueries": "~8,500 searches/min",
				"configChange":    map[string]any{"change": "index-refresh-interval", "at": now.Add(-50 * time.Minute).Format(time.RFC3339)},
			},
			Metadata: map[string]any{
				"ruleId":  "mon-search-5xx",
				"runbook": "https://runbook.demo/search-5xx",
				"links":   []string{"https://kibana.demo/app/r/logs/search-5xx"},
				"team":    "@team-aurora",
				"channel": "#search-alerts",
			},
		},
		{
			ID:          "al-005",
			Title:       "Catalog inventory sync drift",
			Description: "Inventory deltas from ERP lagging beyond 20 minutes in us-east, 2,400 SKUs out of sync",
			Status:      "firing",
			Severity:    "error",
			Service:     "svc-catalog",
			CreatedAt:   now.Add(-4 * time.Hour),
			UpdatedAt:   now.Add(-8 * time.Minute),
			Fields: map[string]any{
				"environment":     "prod",
				"team":            "team-atlas",
				"region":          "us-east-1",
				"lagMinutes":      22,
				"impactedFeeds":   []string{"inventory-full", "inventory-delta"},
				"skusAffected":    2400,
				"upstreamService": "erp-sync-service",
			},
			Metadata: map[string]any{
				"ruleId":    "mon-catalog-drift",
				"dashboard": "dash-catalog",
				"channel":   "#catalog",
				"runbook":   "https://runbook.demo/catalog-sync",
			},
		},
		{
			ID:          "al-006",
			Title:       "Realtime websocket disconnects",
			Description: "Firefox clients disconnect after ~45s with close code 1006, affecting 27% of connections",
			Status:      "acknowledged",
			Severity:    "error",
			Service:     "svc-realtime",
			CreatedAt:   now.Add(-75 * time.Minute),
			UpdatedAt:   now.Add(-15 * time.Minute),
			Fields: map[string]any{
				"environment":    "prod",
				"team":           "team-nova",
				"browser":        "firefox",
				"disconnectRate": 0.27,
				"userAgent":      "Firefox/128",
				"affectedUsers":  "~3,200 users",
				"acknowledgedBy": "alice@demo.com",
				"acknowledgedAt": now.Add(-15 * time.Minute).Format(time.RFC3339),
				"notes":          "Investigating Firefox WebSocket timeout issue",
			},
			Metadata: map[string]any{
				"ruleId":  "mon-realtime-disconnect",
				"links":   []string{"https://grafana.demo/d/realtime-connections"},
				"channel": "#realtime-eng",
			},
		},
		{
			ID:          "al-007",
			Title:       "Container restart loop on pod checkout-7d4f",
			Description: "Pod restarted 8 times in 15 minutes due to OOMKilled events",
			Status:      "firing",
			Severity:    "error",
			Service:     "svc-checkout",
			CreatedAt:   now.Add(-18 * time.Minute),
			UpdatedAt:   now.Add(-3 * time.Minute),
			Fields: map[string]any{
				"environment":   "prod",
				"team":          "team-velocity",
				"region":        "us-west-2",
				"namespace":     "production",
				"pod":           "checkout-7d4f9c8b-xk2m",
				"node":          "node-prod-42",
				"restartCount":  8,
				"exitReason":    "OOMKilled",
				"memoryLimit":   "512Mi",
				"memoryRequest": "256Mi",
			},
			Metadata: map[string]any{
				"ruleId":  "mon-pod-restart-loop",
				"runbook": "https://runbook.demo/pod-restart",
				"links":   []string{"https://k8s.demo/pod/checkout-7d4f9c8b-xk2m"},
				"channel": "#k8s-alerts",
			},
		},
		{
			ID:          "al-008",
			Title:       "API rate limit exhaustion for mobile clients",
			Description: "Mobile API rate limits hit 95% capacity, throttling beginning",
			Status:      "firing",
			Severity:    "error",
			Service:     "svc-api-gateway",
			CreatedAt:   now.Add(-32 * time.Minute),
			UpdatedAt:   now.Add(-7 * time.Minute),
			Fields: map[string]any{
				"environment":     "prod",
				"team":            "team-platform",
				"region":          "global",
				"clientType":      "mobile",
				"utilizationPct":  95,
				"throttledReqs":   1240,
				"affectedClients": []string{"ios-app", "android-app"},
			},
			Metadata: map[string]any{
				"ruleId":    "mon-rate-limit",
				"runbook":   "https://runbook.demo/rate-limits",
				"dashboard": "dash-api-gateway",
			},
		},
		{
			ID:          "al-009",
			Title:       "Certificate expiration warning",
			Description: "TLS certificate for api.demo.com expires in 14 days",
			Status:      "firing",
			Severity:    "error",
			Service:     "svc-ingress",
			CreatedAt:   now.Add(-6 * time.Hour),
			UpdatedAt:   now.Add(-2 * time.Hour),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-platform",
				"domain":      "api.demo.com",
				"expiresIn":   "14 days",
				"expiresAt":   now.Add(14 * 24 * time.Hour).Format(time.RFC3339),
				"issuer":      "Let's Encrypt",
				"autoRenewal": false,
			},
			Metadata: map[string]any{
				"ruleId":  "mon-cert-expiry",
				"runbook": "https://runbook.demo/cert-renewal",
				"channel": "#security",
			},
		},
		{
			ID:          "al-010",
			Title:       "DNS resolution failures spiking",
			Description: "DNS lookup failures increased to 2.1% of requests in eu-west-1",
			Status:      "firing",
			Severity:    "error",
			Service:     "svc-dns",
			CreatedAt:   now.Add(-28 * time.Minute),
			UpdatedAt:   now.Add(-6 * time.Minute),
			Fields: map[string]any{
				"environment":   "prod",
				"team":          "team-platform",
				"region":        "eu-west-1",
				"failureRate":   0.021,
				"affectedZones": []string{"internal.demo.com", "api.demo.com"},
				"dnsProvider":   "route53",
			},
			Metadata: map[string]any{
				"ruleId":  "mon-dns-failures",
				"runbook": "https://runbook.demo/dns-issues",
				"links":   []string{"https://console.aws.amazon.com/route53"},
			},
		},
		// WARNING ALERTS (45% - 13 alerts)
		{
			ID:          "al-011",
			Title:       "Notification queue depth high",
			Description: "Promo notification fanout queue depth above 40k messages affecting 15,000 users",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-notifications",
			CreatedAt:   now.Add(-5 * time.Hour),
			UpdatedAt:   now.Add(-20 * time.Minute),
			Fields: map[string]any{
				"environment":    "prod",
				"team":           "team-signal",
				"queue":          "promo-delivery",
				"depth":          48000,
				"lagByPartition": map[string]any{"0": 12000, "1": 9600, "2": 8800},
				"fanoutChannels": []string{"email", "push", "sms"},
				"affectedUsers":  "~15,000 users",
			},
			Metadata: map[string]any{
				"ruleId":    "mon-notify-queue",
				"dashboard": "dash-notifications",
				"links":     []string{"https://grafana.demo/d/notifications-queue"},
			},
		},
		{
			ID:          "al-012",
			Title:       "Payments webhook retries exhausted",
			Description: "Stripe webhook deliveries repeated 5 times without success for 18 events",
			Status:      "acknowledged",
			Severity:    "warning",
			Service:     "svc-payments",
			CreatedAt:   now.Add(-2 * time.Hour),
			UpdatedAt:   now.Add(-40 * time.Minute),
			Fields: map[string]any{
				"environment":    "prod",
				"team":           "team-revenue",
				"provider":       "stripe",
				"region":         "us-east-1",
				"retryPolicy":    map[string]any{"maxAttempts": 5, "backoff": "exponential"},
				"recentErrors":   []string{"HTTP 504", "context deadline exceeded"},
				"failedEvents":   18,
				"acknowledgedBy": "bob@demo.com",
				"acknowledgedAt": now.Add(-40 * time.Minute).Format(time.RFC3339),
				"notes":          "Investigating webhook endpoint timeout",
			},
			Metadata: map[string]any{
				"ruleId":     "mon-payments-webhook",
				"channel":    "#payments",
				"escalation": []string{"pagerduty://PAY-99", "@revenue-lead"},
			},
		},
		{
			ID:          "al-013",
			Title:       "Web vitals CLS regression on 8.2",
			Description: "Core web vitals degrade for EU mobile traffic on release 8.2",
			Status:      "acknowledged",
			Severity:    "warning",
			Service:     "svc-web",
			CreatedAt:   now.Add(-95 * time.Minute),
			UpdatedAt:   now.Add(-25 * time.Minute),
			Fields: map[string]any{
				"environment":    "prod",
				"team":           "team-velocity",
				"release":        "web-8.2.0",
				"coreWebVitals":  map[string]any{"cls": 0.36, "lcpMs": 3100},
				"clientSegments": []string{"eu-mobile", "na-desktop"},
				"affectedUsers":  "~8,900 users",
				"acknowledgedBy": "carol@demo.com",
				"acknowledgedAt": now.Add(-25 * time.Minute).Format(time.RFC3339),
				"notes":          "Rolling back to 8.1.9",
			},
			Metadata: map[string]any{
				"ruleId": "mon-web-cls",
				"links":  []string{"https://grafana.demo/d/web-vitals"},
			},
		},
		{
			ID:          "al-014",
			Title:       "Analytics pipeline APAC gap",
			Description: "APAC tracking stream produced zero events for 12 minutes",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-analytics",
			CreatedAt:   now.Add(-6 * time.Hour),
			UpdatedAt:   now.Add(-12 * time.Minute),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-lumen",
				"region":      "apac",
				"stream":      "analytics-tracking",
				"eventTypes":  []string{"browse", "purchase"},
				"gapMinutes":  12,
			},
			Metadata: map[string]any{
				"ruleId":          "mon-analytics-gap",
				"channel":         "#analytics",
				"linkedDashboard": "dash-analytics-gaps",
			},
		},
		{
			ID:          "al-015",
			Title:       "Data warehouse compaction backlog",
			Description: "Compact job queue length exceeding 3x expected baseline",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-warehouse",
			CreatedAt:   now.Add(-3 * time.Hour),
			UpdatedAt:   now.Add(-40 * time.Minute),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-foundry",
				"queueDepth":  950,
				"job":         "compact-partitions",
				"region":      "us-west-2",
			},
			Metadata: map[string]any{
				"ruleId":    "mon-warehouse-compaction",
				"dashboard": "dash-warehouse-maint",
			},
		},
		{
			ID:          "al-016",
			Title:       "Cache hit rate degradation",
			Description: "Redis cache hit rate dropped from 94% to 67% over last hour",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-cache",
			CreatedAt:   now.Add(-7 * time.Hour),
			UpdatedAt:   now.Add(-11 * time.Minute),
			Fields: map[string]any{
				"environment":  "prod",
				"team":         "team-platform",
				"region":       "us-east-1",
				"cluster":      "redis-prod-01",
				"hitRate":      0.67,
				"baselineRate": 0.94,
				"missedKeys":   "~45,000/min",
				"memoryUsage":  "78%",
			},
			Metadata: map[string]any{
				"ruleId":    "mon-cache-hit-rate",
				"dashboard": "dash-cache",
				"runbook":   "https://runbook.demo/cache-degradation",
			},
		},
		{
			ID:          "al-017",
			Title:       "Disk space warning on log aggregator",
			Description: "Log aggregator disk usage at 82% capacity in us-west-2",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-logging",
			CreatedAt:   now.Add(-4 * time.Hour),
			UpdatedAt:   now.Add(-90 * time.Minute),
			Fields: map[string]any{
				"environment":  "prod",
				"team":         "team-platform",
				"region":       "us-west-2",
				"host":         "log-aggregator-03",
				"diskUsagePct": 82,
				"availableGB":  180,
				"totalGB":      1000,
				"growthRate":   "2.1 GB/hour",
			},
			Metadata: map[string]any{
				"ruleId":  "mon-disk-space",
				"runbook": "https://runbook.demo/disk-cleanup",
			},
		},
		{
			ID:          "al-018",
			Title:       "Load balancer health check failures",
			Description: "3 of 12 backend instances failing health checks in eu-central-1",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-loadbalancer",
			CreatedAt:   now.Add(-35 * time.Minute),
			UpdatedAt:   now.Add(-9 * time.Minute),
			Fields: map[string]any{
				"environment":      "prod",
				"team":             "team-platform",
				"region":           "eu-central-1",
				"failingInstances": []string{"i-0a1b2c3d", "i-0e4f5g6h", "i-0i7j8k9l"},
				"totalInstances":   12,
				"healthyPct":       75,
			},
			Metadata: map[string]any{
				"ruleId":  "mon-lb-health",
				"runbook": "https://runbook.demo/lb-health-check",
				"links":   []string{"https://console.aws.amazon.com/ec2/v2/home?region=eu-central-1#LoadBalancers"},
			},
		},
		{
			ID:          "al-019",
			Title:       "Circuit breaker tripped for recommendation service",
			Description: "Circuit breaker open after 15 consecutive failures to recommendation API",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-web",
			CreatedAt:   now.Add(-22 * time.Minute),
			UpdatedAt:   now.Add(-4 * time.Minute),
			Fields: map[string]any{
				"environment":       "prod",
				"team":              "team-velocity",
				"region":            "global",
				"downstreamService": "svc-recommendations",
				"failureCount":      15,
				"circuitState":      "open",
				"fallbackActive":    true,
			},
			Metadata: map[string]any{
				"ruleId":  "mon-circuit-breaker",
				"runbook": "https://runbook.demo/circuit-breaker",
				"channel": "#web-eng",
			},
		},
		{
			ID:          "al-020",
			Title:       "Unusual authentication pattern detected",
			Description: "Login attempts from 47 different countries in 10 minutes for user segment",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-identity",
			CreatedAt:   now.Add(-14 * time.Minute),
			UpdatedAt:   now.Add(-2 * time.Minute),
			Fields: map[string]any{
				"environment":      "prod",
				"team":             "team-security",
				"region":           "global",
				"affectedAccounts": 23,
				"countryCount":     47,
				"suspiciousIPs":    []string{"203.0.113.42", "198.51.100.88", "192.0.2.156"},
				"pattern":          "credential-stuffing",
			},
			Metadata: map[string]any{
				"ruleId":  "mon-auth-anomaly",
				"runbook": "https://runbook.demo/security-incident",
				"channel": "#security-alerts",
			},
		},
		{
			ID:          "al-021",
			Title:       "Background job processing lag",
			Description: "Email delivery job queue lag increased to 45 minutes",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-workers",
			CreatedAt:   now.Add(-8 * time.Hour),
			UpdatedAt:   now.Add(-13 * time.Minute),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-signal",
				"region":      "us-east-1",
				"jobType":     "email-delivery",
				"queueDepth":  12400,
				"lagMinutes":  45,
				"workerCount": 8,
			},
			Metadata: map[string]any{
				"ruleId":    "mon-job-lag",
				"dashboard": "dash-workers",
				"runbook":   "https://runbook.demo/job-lag",
			},
		},
		{
			ID:          "al-022",
			Title:       "Database connection pool saturation",
			Description: "Connection pool utilization at 92% for catalog database",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-catalog",
			CreatedAt:   now.Add(-38 * time.Minute),
			UpdatedAt:   now.Add(-7 * time.Minute),
			Fields: map[string]any{
				"environment":       "prod",
				"team":              "team-atlas",
				"region":            "us-east-1",
				"poolUtilization":   0.92,
				"activeConnections": 92,
				"maxConnections":    100,
				"waitingQueries":    14,
			},
			Metadata: map[string]any{
				"ruleId":    "mon-db-pool",
				"dashboard": "dash-database",
				"runbook":   "https://runbook.demo/db-connections",
			},
		},
		{
			ID:          "al-023",
			Title:       "API key usage anomaly",
			Description: "API key abc123 exceeded normal usage by 340% in last hour",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-api-gateway",
			CreatedAt:   now.Add(-48 * time.Minute),
			UpdatedAt:   now.Add(-12 * time.Minute),
			Fields: map[string]any{
				"environment":   "prod",
				"team":          "team-security",
				"region":        "global",
				"apiKey":        "abc123...xyz",
				"usageIncrease": 3.4,
				"requestCount":  45600,
				"baselineCount": 13400,
				"clientName":    "partner-acme",
			},
			Metadata: map[string]any{
				"ruleId":  "mon-api-key-anomaly",
				"runbook": "https://runbook.demo/api-abuse",
				"channel": "#security-alerts",
			},
		},
		// INFO ALERTS (20% - 6 alerts)
		{
			ID:          "al-024",
			Title:       "Warehouse ETL runtime variance",
			Description: "ETL job runtime variance exceeded 3x baseline but within acceptable limits",
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
				"runtimeP95":  "47m",
				"variance":    3.1,
				"baseline":    "15m",
			},
			Metadata: map[string]any{
				"ruleId":    "mon-warehouse-duration",
				"dashboard": "dash-warehouse",
			},
		},
		{
			ID:          "al-025",
			Title:       "Support chatbot deflection drop",
			Description: "Chatbot deflection under 30% causing live agent queue growth",
			Status:      "silenced",
			Severity:    "info",
			Service:     "svc-support",
			CreatedAt:   now.Add(-6 * time.Hour),
			UpdatedAt:   now.Add(-55 * time.Minute),
			Fields: map[string]any{
				"environment":   "prod",
				"team":          "team-lumen",
				"channel":       "support-chat",
				"deflection":    0.28,
				"nlpVersion":    "nlp-v5",
				"experiments":   []string{"reply-suggestions"},
				"silencedBy":    "dave@demo.com",
				"silencedAt":    now.Add(-55 * time.Minute).Format(time.RFC3339),
				"silenceUntil":  now.Add(18 * time.Hour).Format(time.RFC3339),
				"silenceReason": "Known issue, ML team working on model improvements",
			},
			Metadata: map[string]any{
				"ruleId":  "mon-support-deflection",
				"channel": "#support-intel",
			},
		},
		{
			ID:          "al-026",
			Title:       "Feature flag rollout progressing",
			Description: "New checkout flow feature flag at 25% rollout, monitoring for issues",
			Status:      "firing",
			Severity:    "info",
			Service:     "svc-feature-flags",
			CreatedAt:   now.Add(-2 * time.Hour),
			UpdatedAt:   now.Add(-30 * time.Minute),
			Fields: map[string]any{
				"environment":   "prod",
				"team":          "team-velocity",
				"region":        "global",
				"flagName":      "new-checkout-flow",
				"rolloutPct":    25,
				"targetPct":     100,
				"affectedUsers": "~6,250 users",
				"errorRate":     0.008,
			},
			Metadata: map[string]any{
				"ruleId":    "mon-feature-rollout",
				"dashboard": "dash-feature-flags",
			},
		},
		{
			ID:          "al-027",
			Title:       "Scheduled maintenance window approaching",
			Description: "Database maintenance window scheduled in 4 hours for replica upgrades",
			Status:      "firing",
			Severity:    "info",
			Service:     "svc-database",
			CreatedAt:   now.Add(-20 * time.Hour),
			UpdatedAt:   now.Add(-1 * time.Hour),
			Fields: map[string]any{
				"environment":       "prod",
				"team":              "team-data",
				"region":            "us-east-1",
				"maintenanceType":   "replica-upgrade",
				"scheduledAt":       now.Add(4 * time.Hour).Format(time.RFC3339),
				"estimatedDuration": "45 minutes",
				"impactLevel":       "low",
			},
			Metadata: map[string]any{
				"ruleId":  "mon-maintenance-window",
				"channel": "#ops-notifications",
			},
		},
		{
			ID:          "al-028",
			Title:       "Conversion rate trending below target",
			Description: "Weekly conversion rate at 2.4%, below 2.7% target but within normal variance",
			Status:      "firing",
			Severity:    "info",
			Service:     "svc-analytics",
			CreatedAt:   now.Add(-8 * time.Hour),
			UpdatedAt:   now.Add(-2 * time.Hour),
			Fields: map[string]any{
				"environment":     "prod",
				"team":            "team-growth",
				"region":          "global",
				"conversionRate":  0.024,
				"targetRate":      0.027,
				"weekOverWeek":    -0.003,
				"affectedFunnels": []string{"checkout", "signup"},
			},
			Metadata: map[string]any{
				"ruleId":    "mon-conversion-rate",
				"dashboard": "dash-business-metrics",
			},
		},
		{
			ID:          "al-029",
			Title:       "Deployment completed successfully",
			Description: "Catalog service v3.14.2 deployed to production, monitoring for issues",
			Status:      "resolved",
			Severity:    "info",
			Service:     "svc-catalog",
			CreatedAt:   now.Add(-90 * time.Minute),
			UpdatedAt:   now.Add(-30 * time.Minute),
			Fields: map[string]any{
				"environment":    "prod",
				"team":           "team-atlas",
				"region":         "global",
				"version":        "v3.14.2",
				"deployedBy":     "deploy-bot",
				"deployDuration": "8 minutes",
				"healthStatus":   "healthy",
				"resolvedAt":     now.Add(-30 * time.Minute).Format(time.RFC3339),
			},
			Metadata: map[string]any{
				"ruleId":  "mon-deployment",
				"channel": "#deployments",
				"links":   []string{"https://deploy.demo.com/catalog/v3.14.2"},
			},
		},
		// KAFKA/MESSAGING ALERTS
		{
			ID:          "al-030",
			Title:       "Kafka consumer lag increasing",
			Description: "Analytics consumer group lag exceeded 500k messages on user-events topic",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-analytics",
			CreatedAt:   now.Add(-2 * time.Hour),
			UpdatedAt:   now.Add(-15 * time.Minute),
			Fields: map[string]any{
				"environment":    "prod",
				"team":           "team-lumen",
				"region":         "us-east-1",
				"topic":          "user-events",
				"consumerGroup":  "analytics-processor",
				"partition":      0,
				"lag":            520000,
				"lagThreshold":   500000,
				"messagesPerSec": 1200,
				"consumeRate":    850,
			},
			Metadata: map[string]any{
				"ruleId":    "mon-kafka-consumer-lag",
				"dashboard": "dash-kafka",
				"runbook":   "https://runbook.demo/kafka-lag",
			},
		},
		{
			ID:          "al-031",
			Title:       "Message queue depth critical",
			Description: "Notification delivery queue depth at 85k messages, approaching capacity",
			Status:      "firing",
			Severity:    "error",
			Service:     "svc-notifications",
			CreatedAt:   now.Add(-45 * time.Minute),
			UpdatedAt:   now.Add(-8 * time.Minute),
			Fields: map[string]any{
				"environment":    "prod",
				"team":           "team-signal",
				"region":         "global",
				"queue":          "notification-delivery",
				"depth":          85000,
				"capacity":       100000,
				"utilizationPct": 85,
				"oldestMessage":  now.Add(-35 * time.Minute).Format(time.RFC3339),
				"consumerCount":  8,
			},
			Metadata: map[string]any{
				"ruleId":    "mon-queue-depth",
				"dashboard": "dash-messaging",
				"runbook":   "https://runbook.demo/queue-depth",
			},
		},
		{
			ID:          "al-032",
			Title:       "Kafka broker disk usage high",
			Description: "Broker kafka-3 disk usage at 88% in us-west-2",
			Status:      "acknowledged",
			Severity:    "warning",
			Service:     "svc-notifications",
			CreatedAt:   now.Add(-3 * time.Hour),
			UpdatedAt:   now.Add(-45 * time.Minute),
			Fields: map[string]any{
				"environment":    "prod",
				"team":           "team-signal",
				"region":         "us-west-2",
				"broker":         "kafka-3",
				"diskUsagePct":   88,
				"availableGB":    120,
				"totalGB":        1000,
				"retentionHours": 168,
				"topTopics":      []string{"user-events", "order-events", "notification-events"},
				"acknowledgedBy": "ops@demo.com",
				"acknowledgedAt": now.Add(-45 * time.Minute).Format(time.RFC3339),
				"notes":          "Reviewing retention policies",
			},
			Metadata: map[string]any{
				"ruleId":    "mon-kafka-disk",
				"dashboard": "dash-kafka-brokers",
				"runbook":   "https://runbook.demo/kafka-disk",
			},
		},
		{
			ID:          "al-033",
			Title:       "Message producer throttling active",
			Description: "Event producer experiencing throttling due to quota limits",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-analytics",
			CreatedAt:   now.Add(-55 * time.Minute),
			UpdatedAt:   now.Add(-12 * time.Minute),
			Fields: map[string]any{
				"environment":     "prod",
				"team":            "team-lumen",
				"region":          "us-east-1",
				"producer":        "event-collector",
				"topic":           "raw-events",
				"throttleTimeMs":  450,
				"quotaViolations": 23,
				"producerRate":    "15MB/s",
				"quotaLimit":      "10MB/s",
			},
			Metadata: map[string]any{
				"ruleId":    "mon-producer-throttle",
				"dashboard": "dash-kafka-producers",
				"runbook":   "https://runbook.demo/producer-throttle",
			},
		},
		// SCENARIO-THEMED ALERTS
		{
			ID:          "al-scenario-001",
			Title:       "SLO budget exhaustion - Checkout service",
			Description: "Error budget for checkout service exhausted, 99.9% SLO breached for 30 minutes",
			Status:      "firing",
			Severity:    "critical",
			Service:     "svc-checkout",
			CreatedAt:   now.Add(-35 * time.Minute),
			UpdatedAt:   now.Add(-5 * time.Minute),
			Fields: map[string]any{
				"scenario_id":      "slo-exhaustion",
				"scenario_name":    "SLO Budget Exhaustion",
				"scenario_stage":   "budget-depleted",
				"environment":      "prod",
				"team":             "team-velocity",
				"region":           "global",
				"slo_target":       0.999,
				"current_slo":      0.992,
				"budget_remaining": 0.0,
				"breach_duration":  "30m",
				"affected_users":   "~25,000 users",
			},
			Metadata: map[string]any{
				"root_cause":  "increased traffic without autoscaling",
				"is_scenario": true,
			},
		},
		{
			ID:          "al-scenario-002",
			Title:       "Cascading failure - Database connection pool exhaustion",
			Description: "Database connection pool exhausted causing downstream service failures",
			Status:      "firing",
			Severity:    "critical",
			Service:     "svc-database",
			CreatedAt:   now.Add(-20 * time.Minute),
			UpdatedAt:   now.Add(-3 * time.Minute),
			Fields: map[string]any{
				"scenario_id":        "cascading-failure",
				"scenario_name":      "Cascading Failure",
				"scenario_stage":     "propagation",
				"environment":        "prod",
				"team":               "team-data",
				"region":             "us-east-1",
				"pool_size":          100,
				"active_connections": 100,
				"waiting_queries":    450,
			},
			Metadata: map[string]any{
				"root_cause":   "connection leak in checkout service",
				"is_scenario":  true,
				"affects":      []string{"svc-checkout", "svc-order", "svc-catalog"},
				"is_cascading": true,
			},
		},
		{
			ID:          "al-scenario-003",
			Title:       "Deployment rollback triggered - Payment service",
			Description: "Automated rollback initiated due to elevated error rates post-deployment",
			Status:      "acknowledged",
			Severity:    "error",
			Service:     "svc-payments",
			CreatedAt:   now.Add(-45 * time.Minute),
			UpdatedAt:   now.Add(-10 * time.Minute),
			Fields: map[string]any{
				"scenario_id":      "deployment-rollback",
				"scenario_name":    "Deployment Rollback",
				"scenario_stage":   "rollback-initiated",
				"environment":      "prod",
				"team":             "team-revenue",
				"region":           "global",
				"deployed_version": "v2.8.3",
				"rollback_version": "v2.8.2",
				"error_rate_spike": 0.15,
				"baseline_error":   0.01,
				"acknowledgedBy":   "sre-oncall@demo.com",
				"acknowledgedAt":   now.Add(-10 * time.Minute).Format(time.RFC3339),
				"notes":            "Rollback in progress, monitoring error rates",
			},
			Metadata: map[string]any{
				"root_cause":  "incompatible API change",
				"is_scenario": true,
			},
		},
		{
			ID:          "al-scenario-004",
			Title:       "External dependency failure - Stripe API degradation",
			Description: "Stripe payment API experiencing elevated latency and timeouts",
			Status:      "firing",
			Severity:    "error",
			Service:     "svc-payments",
			CreatedAt:   now.Add(-60 * time.Minute),
			UpdatedAt:   now.Add(-15 * time.Minute),
			Fields: map[string]any{
				"scenario_id":      "external-dependency",
				"scenario_name":    "External Dependency Failure",
				"scenario_stage":   "degraded",
				"environment":      "prod",
				"team":             "team-revenue",
				"region":           "global",
				"provider":         "stripe",
				"baseline_latency": "250ms",
				"current_latency":  "4500ms",
				"timeout_rate":     0.12,
				"fallback_active":  true,
			},
			Metadata: map[string]any{
				"root_cause":           "stripe infrastructure issue",
				"is_scenario":          true,
				"external_status_page": "https://status.stripe.com",
			},
		},
		{
			ID:          "al-scenario-005",
			Title:       "Autoscaling lag - Traffic spike exceeds capacity",
			Description: "Traffic spike detected, autoscaling in progress but lagging behind demand",
			Status:      "firing",
			Severity:    "warning",
			Service:     "svc-web",
			CreatedAt:   now.Add(-15 * time.Minute),
			UpdatedAt:   now.Add(-2 * time.Minute),
			Fields: map[string]any{
				"scenario_id":       "autoscaling-lag",
				"scenario_name":     "Autoscaling Lag",
				"scenario_stage":    "scaling-up",
				"environment":       "prod",
				"team":              "team-velocity",
				"region":            "global",
				"current_instances": 12,
				"target_instances":  24,
				"scaling_progress":  "50%",
				"traffic_increase":  "180%",
				"response_time_p95": "850ms",
			},
			Metadata: map[string]any{
				"root_cause":  "viral social media post",
				"is_scenario": true,
			},
		},
		{
			ID:          "al-scenario-006",
			Title:       "Circuit breaker cascade - Recommendation service",
			Description: "Circuit breakers tripping across multiple services due to recommendation service failure",
			Status:      "firing",
			Severity:    "error",
			Service:     "svc-recommendation",
			CreatedAt:   now.Add(-30 * time.Minute),
			UpdatedAt:   now.Add(-5 * time.Minute),
			Fields: map[string]any{
				"scenario_id":        "circuit-breaker-cascade",
				"scenario_name":      "Circuit Breaker Cascade",
				"scenario_stage":     "cascade-active",
				"environment":        "prod",
				"team":               "team-orion",
				"region":             "global",
				"circuit_state":      "open",
				"failure_count":      250,
				"affected_endpoints": []string{"/recommendations", "/personalized-feed", "/similar-items"},
			},
			Metadata: map[string]any{
				"root_cause":   "ML model inference timeout",
				"is_scenario":  true,
				"affects":      []string{"svc-web", "svc-catalog"},
				"is_cascading": true,
			},
		},
	}

	for _, al := range seed {
		alertCopy := al
		if alertCopy.Metadata == nil {
			alertCopy.Metadata = map[string]any{}
		}
		alertCopy.Metadata["source"] = p.cfg.Source

		// Enrich with metadata fields (runbook, dashboard, channel, escalation)
		enrichAlertMetadata(&alertCopy)

		// Enrich with contextual information (deployment, config, user impact)
		enrichWithContextualInfo(&alertCopy, now)

		// Enrich with multi-region fields for infrastructure alerts
		enrichWithMultiRegionFields(&alertCopy)

		p.alerts[alertCopy.ID] = alertCopy
		if steps, ok := lifecycleScenarios[alertCopy.ID]; ok {
			p.lifecycle[alertCopy.ID] = &alertLifecycle{steps: steps}
		}
	}

	// Add analytics alert
	analyticsAlertID := "alert-analytics-001"
	p.alerts[analyticsAlertID] = schema.Alert{
		ID:          analyticsAlertID,
		Title:       "Analytics Correlation Lag",
		Description: "Correlation lag exceeds 30 minutes in svc-analytics.",
		Status:      "firing",
		Severity:    "sev2",
		Service:     "svc-analytics",
		CreatedAt:   now.Add(-20 * time.Minute),
		UpdatedAt:   now.Add(-20 * time.Minute),
		URL:         generateAlertURL(analyticsAlertID, "svc-analytics", false),
		Fields: map[string]any{
			"service":     "svc-analytics",
			"team":        "data-platform",
			"environment": "prod",
			"alert_name":  "correlation_lag_high",
		},
		Metadata: map[string]any{
			"source": p.cfg.Source,
		},
	}
	p.lifecycle["alert-analytics-001"] = &alertLifecycle{steps: lifecycleScenarios["al-013"]}

	// Add payment latency alert
	paymentAlertID := "alert-payment-001"
	p.alerts[paymentAlertID] = schema.Alert{
		ID:          paymentAlertID,
		Title:       "Payment Service Latency",
		Description: "P99 latency for svc-payments exceeds 500ms.",
		Status:      "firing",
		Severity:    "sev1",
		Service:     "svc-payments",
		CreatedAt:   now.Add(-10 * time.Minute),
		UpdatedAt:   now.Add(-10 * time.Minute),
		URL:         generateAlertURL(paymentAlertID, "svc-payments", false),
		Fields: map[string]any{
			"service":     "svc-payments",
			"team":        "payments",
			"environment": "prod",
			"alert_name":  "payment_latency_high",
		},
		Metadata: map[string]any{
			"source": p.cfg.Source,
		},
	}
	p.lifecycle[paymentAlertID] = &alertLifecycle{steps: lifecycleScenarios["al-001"]}

	p.publishLocked()
}

type lifecycleStep struct {
	After    time.Duration
	Status   string
	Severity string
}

type alertLifecycle struct {
	steps   []lifecycleStep
	applied int
}

func (p *Provider) refreshLifecycleLocked(now time.Time) {
	if len(p.lifecycle) == 0 {
		return
	}
	changed := false
	for id, plan := range p.lifecycle {
		alertState, ok := p.alerts[id]
		if !ok {
			continue
		}
		if plan.advance(now, &alertState) {
			p.alerts[id] = alertState
			changed = true
		}
	}
	if changed {
		p.publishLocked()
	}
}

func (plan *alertLifecycle) advance(now time.Time, alertState *schema.Alert) bool {
	if plan == nil || len(plan.steps) == 0 {
		return false
	}
	elapsed := now.Sub(alertState.CreatedAt)
	changed := false
	for plan.applied < len(plan.steps) {
		step := plan.steps[plan.applied]
		if elapsed < step.After {
			break
		}
		if step.Status != "" && alertState.Status != step.Status {
			alertState.Status = step.Status
			changed = true
			if step.Status == "resolved" {
				if alertState.Metadata == nil {
					alertState.Metadata = map[string]any{}
				}
				alertState.Metadata["resolvedAt"] = now.Format(time.RFC3339)
			}
			if step.Status == "acknowledged" {
				if alertState.Fields == nil {
					alertState.Fields = map[string]any{}
				}
				if _, ok := alertState.Fields["acknowledgedBy"].(string); !ok {
					alertState.Fields["acknowledgedBy"] = ackContactForService(alertState.Service)
				}
				alertState.Fields["acknowledgedAt"] = now.Format(time.RFC3339)
				alertState.Fields["notes"] = fmt.Sprintf("Auto-acknowledged by %s", alertState.Fields["acknowledgedBy"])
			}
		}
		if step.Severity != "" && alertState.Severity != step.Severity {
			alertState.Severity = step.Severity
			changed = true
		}
		plan.applied++
	}
	if changed {
		alertState.UpdatedAt = now
	}
	return changed
}

func (p *Provider) publishLocked() {
	snapshot := make([]schema.Alert, 0, len(p.alerts))
	for _, al := range p.alerts {
		snapshot = append(snapshot, cloneAlert(al))
	}
	mockutil.PublishAlerts(snapshot)
}

var lifecycleScenarios = map[string][]lifecycleStep{
	"al-001": {
		{After: 40 * time.Minute, Status: "acknowledged"},
		{After: 70 * time.Minute, Status: "resolved"},
	},
	"al-004": {
		{After: 35 * time.Minute, Status: "acknowledged"},
	},
	"al-009": {
		{After: 50 * time.Minute, Status: "mitigating"},
		{After: 95 * time.Minute, Status: "resolved", Severity: "info"},
	},
	"al-012": {
		{After: 30 * time.Minute, Status: "acknowledged"},
	},
	"al-013": {
		{After: 15 * time.Minute, Status: "acknowledged"},
		{After: 45 * time.Minute, Status: "resolved"},
	},
}

func ackContactForService(service string) string {
	switch service {
	case "svc-checkout", "svc-order":
		return "checkout-oncall@demo.com"
	case "svc-search":
		return "search-oncall@demo.com"
	case "svc-realtime":
		return "realtime-oncall@demo.com"
	default:
		return "oncall@demo.com"
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
	// Generate URL if not already present
	url := in.URL
	if url == "" {
		isScenario := isScenarioAlert(in.Metadata, in.Fields)
		url = generateAlertURL(in.ID, in.Service, isScenario)
	}

	return schema.Alert{
		ID:          in.ID,
		Title:       in.Title,
		Description: in.Description,
		Status:      in.Status,
		Severity:    in.Severity,
		Service:     in.Service,
		URL:         url,
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

// generateAlertsForQuery creates mock alerts that match the search query
func (p *Provider) generateAlertsForQuery(parsed mockutil.ParsedQuery, scope schema.QueryScope, statusFilter, severityFilter map[string]bool, limit int, now time.Time) []schema.Alert {
	if limit <= 0 {
		limit = 5
	}

	// Determine service from scope or infer from query
	service := scope.Service
	if service == "" {
		service = mockutil.InferServiceFromQuery(parsed)
	}
	if service == "" {
		service = "svc-cache" // Default for queries without clear service
	}

	// Determine environment
	environment := scope.Environment
	if environment == "" {
		environment = "prod"
	}

	// Determine team
	team := scope.Team
	if team == "" {
		team = mockutil.GetTeamForService(service)
	}

	// Determine statuses to generate
	statuses := []string{"firing"}
	if len(statusFilter) > 0 {
		statuses = make([]string, 0, len(statusFilter))
		for status := range statusFilter {
			statuses = append(statuses, status)
		}
	}

	// Determine severities to generate
	severities := []string{"warning", "error", "critical"}
	if len(severityFilter) > 0 {
		severities = make([]string, 0, len(severityFilter))
		for severity := range severityFilter {
			severities = append(severities, severity)
		}
	}

	// Generate alerts
	alerts := make([]schema.Alert, 0, limit)
	for i := 0; i < limit; i++ {
		status := statuses[i%len(statuses)]
		severity := severities[i%len(severities)]

		// Build title and description incorporating search terms
		title, description := buildAlertContent(parsed, service, severity, i)

		alert := schema.Alert{
			ID:          fmt.Sprintf("al-gen-%d", now.Unix()+int64(i)),
			Title:       title,
			Description: description,
			Status:      status,
			Severity:    severity,
			Service:     service,
			URL:         generateAlertURL(fmt.Sprintf("al-gen-%d", now.Unix()+int64(i)), service, false),
			CreatedAt:   now.Add(-time.Duration(30-i*5) * time.Minute),
			UpdatedAt:   now.Add(-time.Duration(5-i) * time.Minute),
			Fields: map[string]any{
				"environment": environment,
				"team":        team,
				"region":      "us-east-1",
				"generated":   true,
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"generated":   true,
				"searchTerms": parsed.Terms,
			},
		}

		// Add search terms to fields for better matching
		for idx, term := range parsed.Terms {
			if idx < 3 { // Limit to first 3 terms
				alert.Fields[fmt.Sprintf("term_%d", idx)] = term
			}
		}

		alerts = append(alerts, alert)
	}

	return alerts
}

// buildAlertContent creates alert title and description incorporating search terms
func buildAlertContent(parsed mockutil.ParsedQuery, service, severity string, index int) (string, string) {
	// Use first term or quoted phrase as primary content
	primaryTerm := ""
	if len(parsed.QuotedPhrases) > 0 {
		primaryTerm = parsed.QuotedPhrases[0]
	} else if len(parsed.Terms) > 0 {
		primaryTerm = parsed.Terms[0]
	}

	if primaryTerm == "" {
		primaryTerm = "service issue"
	}

	// Build title based on severity and search terms
	var title string
	switch severity {
	case "critical":
		title = fmt.Sprintf("%s critical failure detected", cases.Title(language.English).String(primaryTerm))
	case "error":
		title = fmt.Sprintf("%s error rate elevated", cases.Title(language.English).String(primaryTerm))
	default:
		title = fmt.Sprintf("%s performance degradation", cases.Title(language.English).String(primaryTerm))
	}

	// Build description incorporating multiple search terms
	description := fmt.Sprintf("Service %s experiencing issues related to %s", service, primaryTerm)
	if len(parsed.Terms) > 1 {
		description += fmt.Sprintf(" and %s", strings.Join(parsed.Terms[1:], ", "))
	}
	if len(parsed.QuotedPhrases) > 1 {
		description += fmt.Sprintf(". Specific errors include: %s", strings.Join(parsed.QuotedPhrases[1:], ", "))
	}

	return title, description
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
