package incidentmock

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/opsorch/opsorch-core/incident"
	"github.com/opsorch/opsorch-core/orcherr"
	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/internal/mockutil"
)

// ProviderName can be referenced via OPSORCH_INCIDENT_PROVIDER.
const ProviderName = "mock"

// Config controls mock incident behavior.
type Config struct {
	Source          string
	DefaultSeverity string
}

// Provider keeps an in-memory incident list for demo purposes.
type Provider struct {
	cfg       Config
	mu        sync.Mutex
	nextID    int
	incidents map[string]schema.Incident
	timeline  map[string][]schema.TimelineEntry
}

// New constructs the provider with seeded demo incidents.
func New(cfg map[string]any) (incident.Provider, error) {
	parsed := parseConfig(cfg)
	p := &Provider{cfg: parsed, incidents: map[string]schema.Incident{}, timeline: map[string][]schema.TimelineEntry{}}
	p.seed()
	return p, nil
}

func init() {
	_ = incident.RegisterProvider(ProviderName, New)
}

// WithScope attaches a QueryScope to the context so Query/List can filter incidents client-side.
func WithScope(ctx context.Context, scope schema.QueryScope) context.Context {
	return context.WithValue(ctx, scopeKey{}, scope)
}

type scopeKey struct{}

// Query returns incidents filtered by query parameters. If a QueryScope was attached to the context
// with WithScope, it is merged with the provided query.Scope (query takes precedence).
func (p *Provider) Query(ctx context.Context, query schema.IncidentQuery) ([]schema.Incident, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	combinedScope := mergeScope(extractScope(ctx), query.Scope)
	statusFilter := toSet(query.Statuses)
	severityFilter := toSet(query.Severities)
	needle := strings.ToLower(strings.TrimSpace(query.Query))

	out := make([]schema.Incident, 0, len(p.incidents))
	for _, inc := range p.incidents {
		if !matchesScope(combinedScope, inc) {
			continue
		}
		if len(statusFilter) > 0 && !statusFilter[inc.Status] {
			continue
		}
		if len(severityFilter) > 0 && !severityFilter[inc.Severity] {
			continue
		}
		if needle != "" && !matchesQuery(needle, inc) {
			continue
		}

		out = append(out, cloneIncident(inc))
		if query.Limit > 0 && len(out) >= query.Limit {
			break
		}
	}
	return out, nil
}

// List returns incidents currently tracked. If a QueryScope was attached to the context
// with WithScope, the results are filtered to match service/team/environment.
func (p *Provider) List(ctx context.Context) ([]schema.Incident, error) {
	return p.Query(ctx, schema.IncidentQuery{})
}

// Get fetches an incident by ID.
func (p *Provider) Get(ctx context.Context, id string) (schema.Incident, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	inc, ok := p.incidents[id]
	if !ok {
		return schema.Incident{}, orcherr.New("not_found", "incident not found", nil)
	}
	return cloneIncident(inc), nil
}

// Create inserts a new incident with generated ID and enriched metadata.
func (p *Provider) Create(ctx context.Context, in schema.CreateIncidentInput) (schema.Incident, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.nextID++
	id := fmt.Sprintf("inc-%03d", p.nextID)
	now := time.Now().UTC()

	incident := schema.Incident{
		ID:          id,
		Title:       in.Title,
		Description: in.Description,
		Status:      emptyFallback(in.Status, "open"),
		Severity:    emptyFallback(in.Severity, p.cfg.DefaultSeverity),
		Service:     inferService(in),
		CreatedAt:   now,
		UpdatedAt:   now,
		Fields:      mockutil.CloneMap(in.Fields),
		Metadata:    mockutil.CloneMap(in.Metadata),
	}
	if incident.Metadata == nil {
		incident.Metadata = map[string]any{}
	}
	incident.Metadata["source"] = p.cfg.Source
	if incident.Service != "" {
		if incident.Fields == nil {
			incident.Fields = map[string]any{}
		}
		incident.Fields["service"] = incident.Service
	}

	p.incidents[id] = incident
	return cloneIncident(incident), nil
}

// Update mutates an incident in place.
func (p *Provider) Update(ctx context.Context, id string, in schema.UpdateIncidentInput) (schema.Incident, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	inc, ok := p.incidents[id]
	if !ok {
		return schema.Incident{}, orcherr.New("not_found", "incident not found", nil)
	}

	if in.Title != nil {
		inc.Title = *in.Title
	}
	if in.Description != nil {
		inc.Description = *in.Description
	}
	if in.Status != nil {
		inc.Status = *in.Status
	}
	if in.Severity != nil {
		inc.Severity = *in.Severity
	}
	if in.Service != nil {
		inc.Service = *in.Service
	}
	if in.Fields != nil {
		inc.Fields = mockutil.CloneMap(in.Fields)
	}
	if in.Metadata != nil {
		inc.Metadata = mockutil.CloneMap(in.Metadata)
	}
	if inc.Service != "" {
		if inc.Fields == nil {
			inc.Fields = map[string]any{}
		}
		inc.Fields["service"] = inc.Service
	}
	inc.UpdatedAt = time.Now().UTC()

	p.incidents[id] = inc
	return cloneIncident(inc), nil
}

// GetTimeline returns timeline entries for an incident.
func (p *Provider) GetTimeline(ctx context.Context, id string) ([]schema.TimelineEntry, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.incidents[id]; !ok {
		return nil, orcherr.New("not_found", "incident not found", nil)
	}

	// Get base timeline entries
	entries := cloneTimeline(p.timeline[id])

	return entries, nil
}

// AppendTimeline adds a timeline entry to an incident.
func (p *Provider) AppendTimeline(ctx context.Context, id string, entry schema.TimelineAppendInput) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.incidents[id]; !ok {
		return orcherr.New("not_found", "incident not found", nil)
	}

	n := len(p.timeline[id]) + 1
	at := entry.At
	if at.IsZero() {
		at = time.Now().UTC()
	}

	p.timeline[id] = append(p.timeline[id], schema.TimelineEntry{
		ID:         fmt.Sprintf("%s-t%d", id, n),
		IncidentID: id,
		At:         at,
		Kind:       emptyFallback(entry.Kind, "note"),
		Body:       entry.Body,
		Actor:      mockutil.CloneMap(entry.Actor),
		Metadata:   mockutil.CloneMap(entry.Metadata),
	})
	return nil
}

func (p *Provider) seed() {
	now := time.Now().UTC()

	seed := []schema.Incident{
		{
			ID:          "inc-001",
			Title:       "Checkout latency impacting EU customers",
			Description: "High checkout latency causing timeouts for a slice of EU traffic",
			Status:      "mitigating",
			Severity:    p.cfg.DefaultSeverity,
			Service:     "svc-checkout",
			CreatedAt:   now.Add(-55 * time.Minute),
			UpdatedAt:   now.Add(-10 * time.Minute),
			Fields: map[string]any{
				"service":        "svc-checkout",
				"team":           "team-velocity",
				"environment":    "prod",
				"customerImpact": "Checkout timing out for ~8% of EU sessions",
				"alertId":        "pagerduty:PRD123",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "channel": "#inc-123", "runbook": "https://runbook.demo/checkout-latency"},
		},
		{
			ID:          "inc-002",
			Title:       "Search results intermittently empty",
			Description: "Search API intermittently returns empty result sets",
			Status:      "monitoring",
			Severity:    "sev3",
			Service:     "svc-search",
			CreatedAt:   now.Add(-2 * time.Hour),
			UpdatedAt:   now.Add(-30 * time.Minute),
			Fields: map[string]any{
				"service":       "svc-search",
				"team":          "team-aurora",
				"environment":   "prod",
				"correlationId": "corr-9481",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "linkedDashboard": "dash-search"},
		},
		{
			ID:          "inc-003",
			Title:       "Payments webhook timeouts from Stripe",
			Description: "Stripe webhook requests timing out from payments provider",
			Status:      "open",
			Severity:    "sev1",
			Service:     "svc-payments",
			CreatedAt:   now.Add(-3*time.Hour - 45*time.Minute),
			UpdatedAt:   now.Add(-15 * time.Minute),
			Fields: map[string]any{
				"service":     "svc-payments",
				"team":        "team-revenue",
				"environment": "prod",
				"provider":    "stripe",
				"region":      "us-east-1",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "runbook": "https://runbook.demo/payments-webhooks", "alertId": "pagerduty:PAY-99"},
		},
		{
			ID:          "inc-004",
			Title:       "Notification fanout lagging for promos",
			Description: "Notification fanout workers are lagging promo campaigns",
			Status:      "investigating",
			Severity:    "sev2",
			Service:     "svc-notifications",
			CreatedAt:   now.Add(-95 * time.Minute),
			UpdatedAt:   now.Add(-35 * time.Minute),
			Fields: map[string]any{
				"service":     "svc-notifications",
				"team":        "team-signal",
				"environment": "prod",
				"experiment":  "promo-day",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "channel": "#notify-incident", "linkedDashboard": "dash-notifications"},
		},
		{
			ID:          "inc-005",
			Title:       "Auth latency spikes for mobile logins",
			Description: "Mobile login auth latency spiking for the identity service",
			Status:      "mitigating",
			Severity:    "sev2",
			Service:     "svc-identity",
			CreatedAt:   now.Add(-4*time.Hour - 10*time.Minute),
			UpdatedAt:   now.Add(-50 * time.Minute),
			Fields: map[string]any{
				"service":     "svc-identity",
				"team":        "team-guardian",
				"environment": "prod",
				"platform":    "mobile",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "runbook": "https://runbook.demo/auth-latency", "linkedDashboard": "dash-auth"},
		},
		{
			ID:          "inc-006",
			Title:       "Warehouse batch stuck on partition 7",
			Description: "Warehouse batch job stuck processing partition 7",
			Status:      "identified",
			Severity:    "sev3",
			Service:     "svc-warehouse",
			CreatedAt:   now.Add(-7*time.Hour - 20*time.Minute),
			UpdatedAt:   now.Add(-5 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-warehouse",
				"team":        "team-foundry",
				"environment": "prod",
				"batchId":     "2024-09-12-07",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "linkedDashboard": "dash-warehouse", "oncall": "pd:data-platform"},
		},
		{
			ID:          "inc-007",
			Title:       "Recommendation quality drop after rollout",
			Description: "New recommendation rollout degraded quality metrics",
			Status:      "monitoring",
			Severity:    "sev3",
			Service:     "svc-recommendation",
			CreatedAt:   now.Add(-26 * time.Hour),
			UpdatedAt:   now.Add(-3 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-recommendation",
				"team":        "team-orion",
				"environment": "prod",
				"model":       "reco-v5",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "experiment": "recotype-b", "linkedDashboard": "dash-reco"},
		},
		{
			ID:          "inc-008",
			Title:       "Analytics pipeline missing events from APAC",
			Description: "Analytics pipeline missing events originating from APAC region",
			Status:      "investigating",
			Severity:    "sev2",
			Service:     "svc-analytics",
			CreatedAt:   now.Add(-8 * time.Hour),
			UpdatedAt:   now.Add(-2 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-analytics",
				"team":        "team-lumen",
				"environment": "prod",
				"region":      "apac",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "linkedDashboard": "dash-analytics", "correlationId": "etl-apac-2219"},
		},
		{
			ID:          "inc-009",
			Title:       "Order placement errors for prepaid cards",
			Description: "Orders paid with prepaid cards fail due to processor errors",
			Status:      "open",
			Severity:    "sev2",
			Service:     "svc-order",
			CreatedAt:   now.Add(-3 * time.Hour),
			UpdatedAt:   now.Add(-1 * time.Hour),
			Fields: map[string]any{
				"service":       "svc-order",
				"team":          "team-velocity",
				"environment":   "prod",
				"paymentMethod": "prepaid",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "channel": "#orders", "linkedDashboard": "dash-orders"},
		},
		{
			ID:          "inc-010",
			Title:       "Catalog indexer backlog after schema change",
			Description: "Catalog indexing backlog building after schema migration",
			Status:      "mitigating",
			Severity:    "sev4",
			Service:     "svc-catalog",
			CreatedAt:   now.Add(-12 * time.Hour),
			UpdatedAt:   now.Add(-4 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-catalog",
				"team":        "team-atlas",
				"environment": "prod",
				"release":     "catalog-2.8.0",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "runbook": "https://runbook.demo/catalog-index", "linkedDashboard": "dash-catalog"},
		},
		{
			ID:          "inc-011",
			Title:       "Shipping tracker returns stale data",
			Description: "Shipping tracker caches are returning stale tracking payloads",
			Status:      "monitoring",
			Severity:    "sev4",
			Service:     "svc-shipping",
			CreatedAt:   now.Add(-18 * time.Hour),
			UpdatedAt:   now.Add(-30 * time.Minute),
			Fields: map[string]any{
				"service":     "svc-shipping",
				"team":        "team-hawkeye",
				"environment": "prod",
				"carrier":     "fast-ship",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "channel": "#shipping", "linkedDashboard": "dash-shipping"},
		},
		{
			ID:          "inc-012",
			Title:       "Realtime updates disconnect in Firefox",
			Description: "Realtime websocket connections drop for Firefox clients",
			Status:      "open",
			Severity:    "sev3",
			Service:     "svc-realtime",
			CreatedAt:   now.Add(-2*time.Hour - 20*time.Minute),
			UpdatedAt:   now.Add(-15 * time.Minute),
			Fields: map[string]any{
				"service":     "svc-realtime",
				"team":        "team-nova",
				"environment": "prod",
				"browser":     "firefox-128",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "linkedDashboard": "dash-realtime", "alertId": "pagerduty:RT-77"},
		},
		// SCENARIO-THEMED INCIDENTS
		{
			ID:          "inc-scenario-001",
			Title:       "SLO Budget Exhaustion - Checkout Service",
			Description: "Error budget for checkout service completely exhausted, 99.9% SLO breached",
			Status:      "mitigating",
			Severity:    "sev1",
			Service:     "svc-checkout",
			CreatedAt:   now.Add(-45 * time.Minute),
			UpdatedAt:   now.Add(-10 * time.Minute),
			Fields: map[string]any{
				"scenario_id":      "slo-exhaustion",
				"scenario_name":    "SLO Budget Exhaustion",
				"scenario_stage":   "mitigation",
				"service":          "svc-checkout",
				"team":             "team-velocity",
				"environment":      "prod",
				"oncall_assignee":  "alex",
				"escalation_level": 1,
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"root_cause":  "increased traffic without autoscaling",
				"is_scenario": true,
			},
		},
		{
			ID:          "inc-scenario-002",
			Title:       "Cascading Failure - Database Connection Pool Exhaustion",
			Description: "Database connection pool exhausted causing cascading failures across dependent services",
			Status:      "investigating",
			Severity:    "sev1",
			Service:     "svc-database",
			CreatedAt:   now.Add(-30 * time.Minute),
			UpdatedAt:   now.Add(-5 * time.Minute),
			Fields: map[string]any{
				"scenario_id":      "cascading-failure",
				"scenario_name":    "Cascading Failure",
				"scenario_stage":   "propagation",
				"service":          "svc-database",
				"team":             "team-data",
				"environment":      "prod",
				"oncall_assignee":  "morgan",
				"escalation_level": 2,
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"root_cause":  "connection leak in checkout service",
				"is_scenario": true,
			},
		},
		{
			ID:          "inc-scenario-003",
			Title:       "Deployment Rollback - Payment Service v2.8.3",
			Description: "Automated rollback triggered due to elevated error rates after deployment",
			Status:      "monitoring",
			Severity:    "sev2",
			Service:     "svc-payments",
			CreatedAt:   now.Add(-90 * time.Minute),
			UpdatedAt:   now.Add(-20 * time.Minute),
			Fields: map[string]any{
				"scenario_id":      "deployment-rollback",
				"scenario_name":    "Deployment Rollback",
				"scenario_stage":   "rollback-complete",
				"service":          "svc-payments",
				"team":             "team-revenue",
				"environment":      "prod",
				"oncall_assignee":  "sam",
				"escalation_level": 0,
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"root_cause":  "incompatible API change",
				"is_scenario": true,
			},
		},
		{
			ID:          "inc-scenario-004",
			Title:       "External Dependency Failure - Stripe API Degradation",
			Description: "Stripe payment API experiencing elevated latency and rate limiting, impacting checkout flow",
			Status:      "investigating",
			Severity:    "sev2",
			Service:     "svc-checkout",
			CreatedAt:   now.Add(-15 * time.Minute),
			UpdatedAt:   now.Add(-2 * time.Minute),
			Fields: map[string]any{
				"scenario_id":       "external-dependency-failure",
				"scenario_name":     "External Dependency Failure - Stripe",
				"scenario_stage":    "active",
				"service":           "svc-checkout",
				"team":              "team-velocity",
				"environment":       "prod",
				"oncall_assignee":   "fern",
				"escalation_level":  1,
				"external_provider": "stripe",
			},
			Metadata: map[string]any{
				"source":            p.cfg.Source,
				"root_cause":        "stripe API rate limiting",
				"affected_services": []string{"svc-checkout", "svc-payments"},
				"is_scenario":       true,
			},
		},
		{
			ID:          "inc-scenario-005",
			Title:       "Autoscaling Lag - Traffic Spike Exceeds Capacity",
			Description: "Sudden traffic spike detected, autoscaling in progress but lagging behind demand",
			Status:      "mitigating",
			Severity:    "sev3",
			Service:     "svc-search",
			CreatedAt:   now.Add(-12 * time.Minute),
			UpdatedAt:   now.Add(-1 * time.Minute),
			Fields: map[string]any{
				"scenario_id":      "autoscaling-lag",
				"scenario_name":    "Autoscaling Lag",
				"scenario_stage":   "scaling",
				"service":          "svc-search",
				"team":             "team-aurora",
				"environment":      "prod",
				"oncall_assignee":  "lena",
				"escalation_level": 0,
				"current_replicas": 3,
				"target_replicas":  8,
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"root_cause":  "traffic spike from marketing campaign",
				"is_scenario": true,
			},
		},
		{
			ID:          "inc-scenario-006",
			Title:       "Circuit Breaker Cascade - Recommendation Service Failure",
			Description: "Circuit breakers tripping across multiple services due to recommendation service degradation",
			Status:      "investigating",
			Severity:    "sev1",
			Service:     "svc-recommendation",
			CreatedAt:   now.Add(-8 * time.Minute),
			UpdatedAt:   now.Add(-1 * time.Minute),
			Fields: map[string]any{
				"scenario_id":      "circuit-breaker-cascade",
				"scenario_name":    "Circuit Breaker Cascade",
				"scenario_stage":   "escalating",
				"service":          "svc-recommendation",
				"team":             "team-orion",
				"environment":      "prod",
				"oncall_assignee":  "milo",
				"escalation_level": 2,
			},
			Metadata: map[string]any{
				"source":            p.cfg.Source,
				"root_cause":        "recommendation model inference timeout",
				"affected_services": []string{"svc-checkout", "svc-catalog", "svc-web"},
				"is_scenario":       true,
			},
		},
	}

	for _, inc := range seed {
		p.incidents[inc.ID] = inc
		if n, err := fmt.Sscanf(inc.ID, "inc-%d", &p.nextID); n == 1 && err == nil {
			// keep the largest parsed ID for incremental IDs
		}
	}

	p.timeline["inc-001"] = []schema.TimelineEntry{
		{ID: "inc-001-t1", IncidentID: "inc-001", At: now.Add(-50 * time.Minute), Kind: "note", Body: "PagerDuty triggered by checkout p95 > 1.2s", Actor: map[string]any{"type": "system", "name": "pd-bot"}},
		{ID: "inc-001-t2", IncidentID: "inc-001", At: now.Add(-35 * time.Minute), Kind: "link", Body: "Runbook https://runbook.demo/checkout-latency", Actor: map[string]any{"type": "user", "name": "alex"}},
		{ID: "inc-001-t3", IncidentID: "inc-001", At: now.Add(-18 * time.Minute), Kind: "note", Body: "Rolled back checkout v2.31.4 in EUW1", Actor: map[string]any{"type": "user", "name": "alex"}},
	}

	p.timeline["inc-002"] = []schema.TimelineEntry{
		{ID: "inc-002-t1", IncidentID: "inc-002", At: now.Add(-110 * time.Minute), Kind: "note", Body: "Search cluster scaled up from 12 -> 16 nodes", Actor: map[string]any{"type": "user", "name": "jamie"}},
		{ID: "inc-002-t2", IncidentID: "inc-002", At: now.Add(-70 * time.Minute), Kind: "note", Body: "Cache warmup reduces 500s, monitoring", Actor: map[string]any{"type": "user", "name": "taylor"}},
	}

	p.timeline["inc-003"] = []schema.TimelineEntry{
		{ID: "inc-003-t1", IncidentID: "inc-003", At: now.Add(-3*time.Hour - 40*time.Minute), Kind: "note", Body: "Stripe webhook errors above 40% (HTTP 504) in us-east-1", Actor: map[string]any{"type": "system", "name": "pd-bot"}},
		{ID: "inc-003-t2", IncidentID: "inc-003", At: now.Add(-3*time.Hour - 10*time.Minute), Kind: "note", Body: "Acknowledged by oncall, tracing requests through new ALB", Actor: map[string]any{"type": "user", "name": "sam"}},
		{ID: "inc-003-t3", IncidentID: "inc-003", At: now.Add(-2*time.Hour - 20*time.Minute), Kind: "note", Body: "Shifted 30% traffic to standby workers and increased webhook timeout to 8s", Actor: map[string]any{"type": "user", "name": "sam"}},
		{ID: "inc-003-t4", IncidentID: "inc-003", At: now.Add(-1 * time.Hour), Kind: "note", Body: "Stripe confirms transient network degradation resolved", Actor: map[string]any{"type": "user", "name": "partner-relations"}},
		{ID: "inc-003-t5", IncidentID: "inc-003", At: now.Add(-20 * time.Minute), Kind: "note", Body: "Errors back to baseline, watching queues for 30m", Actor: map[string]any{"type": "user", "name": "sam"}},
	}

	p.timeline["inc-004"] = []schema.TimelineEntry{
		{ID: "inc-004-t1", IncidentID: "inc-004", At: now.Add(-90 * time.Minute), Kind: "note", Body: "Promo notification latency spiked above 6m", Actor: map[string]any{"type": "system", "name": "alertmanager"}},
		{ID: "inc-004-t2", IncidentID: "inc-004", At: now.Add(-80 * time.Minute), Kind: "note", Body: "Kafka partitions imbalanced after promo re-shard; consumer lag rising", Actor: map[string]any{"type": "user", "name": "lee"}},
		{ID: "inc-004-t3", IncidentID: "inc-004", At: now.Add(-55 * time.Minute), Kind: "note", Body: "Rerouted promo fanout to gcp-europe and throttled attachments", Actor: map[string]any{"type": "user", "name": "lee"}},
		{ID: "inc-004-t4", IncidentID: "inc-004", At: now.Add(-35 * time.Minute), Kind: "note", Body: "Consumer lag trending down, announcement paused", Actor: map[string]any{"type": "user", "name": "taylor"}},
	}

	p.timeline["inc-005"] = []schema.TimelineEntry{
		{ID: "inc-005-t1", IncidentID: "inc-005", At: now.Add(-4 * time.Hour), Kind: "note", Body: "p95 auth latency 1.1s for mobile sign-ins", Actor: map[string]any{"type": "system", "name": "apm"}},
		{ID: "inc-005-t2", IncidentID: "inc-005", At: now.Add(-3*time.Hour - 45*time.Minute), Kind: "note", Body: "Rolled back mobile-auth service to 1.14.2", Actor: map[string]any{"type": "user", "name": "devon"}},
		{ID: "inc-005-t3", IncidentID: "inc-005", At: now.Add(-2 * time.Hour), Kind: "note", Body: "Enabled per-region Redis pools to reduce contention", Actor: map[string]any{"type": "user", "name": "devon"}},
		{ID: "inc-005-t4", IncidentID: "inc-005", At: now.Add(-50 * time.Minute), Kind: "note", Body: "Latency normalizing; keeping increased autoscale minimums", Actor: map[string]any{"type": "user", "name": "devon"}},
	}

	p.timeline["inc-006"] = []schema.TimelineEntry{
		{ID: "inc-006-t1", IncidentID: "inc-006", At: now.Add(-7 * time.Hour), Kind: "note", Body: "Batch 2024-09-12-07 stuck at 57% due to lock on partition 7", Actor: map[string]any{"type": "system", "name": "scheduler"}},
		{ID: "inc-006-t2", IncidentID: "inc-006", At: now.Add(-6*time.Hour - 45*time.Minute), Kind: "note", Body: "Restarted worker batch-wrk-02 without progress", Actor: map[string]any{"type": "user", "name": "morgan"}},
		{ID: "inc-006-t3", IncidentID: "inc-006", At: now.Add(-6 * time.Hour), Kind: "note", Body: "Moved partition 7 to queue-b; verifying checkpoints", Actor: map[string]any{"type": "user", "name": "morgan"}},
		{ID: "inc-006-t4", IncidentID: "inc-006", At: now.Add(-5 * time.Hour), Kind: "note", Body: "Scheduled backfill for missing segments post-unlock", Actor: map[string]any{"type": "user", "name": "data-eng"}},
	}

	p.timeline["inc-007"] = []schema.TimelineEntry{
		{ID: "inc-007-t1", IncidentID: "inc-007", At: now.Add(-25*time.Hour - 50*time.Minute), Kind: "note", Body: "CTR drop 18% after reco-v5 canary", Actor: map[string]any{"type": "system", "name": "metrics-bot"}},
		{ID: "inc-007-t2", IncidentID: "inc-007", At: now.Add(-25 * time.Hour), Kind: "note", Body: "Rolled back to reco-v4 for US region", Actor: map[string]any{"type": "user", "name": "riley"}},
		{ID: "inc-007-t3", IncidentID: "inc-007", At: now.Add(-22 * time.Hour), Kind: "note", Body: "Retrained feature store with refreshed catalog data", Actor: map[string]any{"type": "user", "name": "riley"}},
		{ID: "inc-007-t4", IncidentID: "inc-007", At: now.Add(-3 * time.Hour), Kind: "note", Body: "Traffic steady; re-enabling 10% canary", Actor: map[string]any{"type": "user", "name": "riley"}},
	}

	p.timeline["inc-008"] = []schema.TimelineEntry{
		{ID: "inc-008-t1", IncidentID: "inc-008", At: now.Add(-7*time.Hour - 45*time.Minute), Kind: "note", Body: "APAC ingestion gap detected: 0 events in past 20m", Actor: map[string]any{"type": "system", "name": "spark-monitor"}},
		{ID: "inc-008-t2", IncidentID: "inc-008", At: now.Add(-7 * time.Hour), Kind: "note", Body: "Spark job failing with expired OAuth token for storage bucket", Actor: map[string]any{"type": "user", "name": "maya"}},
		{ID: "inc-008-t3", IncidentID: "inc-008", At: now.Add(-5 * time.Hour), Kind: "note", Body: "Rotated service account and replayed backlog from sequence 220", Actor: map[string]any{"type": "user", "name": "maya"}},
		{ID: "inc-008-t4", IncidentID: "inc-008", At: now.Add(-2 * time.Hour), Kind: "note", Body: "Repartitioned APAC shards to 6 executors; latency stable", Actor: map[string]any{"type": "user", "name": "maya"}},
	}

	p.timeline["inc-009"] = []schema.TimelineEntry{
		{ID: "inc-009-t1", IncidentID: "inc-009", At: now.Add(-2*time.Hour - 50*time.Minute), Kind: "note", Body: "Order prepaid auth failures exceeded 3% of traffic", Actor: map[string]any{"type": "system", "name": "ops-alerts"}},
		{ID: "inc-009-t2", IncidentID: "inc-009", At: now.Add(-2 * time.Hour), Kind: "note", Body: "Gateway rejecting prepaid BIN range 5523", Actor: map[string]any{"type": "user", "name": "kim"}},
		{ID: "inc-009-t3", IncidentID: "inc-009", At: now.Add(-90 * time.Minute), Kind: "note", Body: "Added fallback provider for prepaid and draining queue", Actor: map[string]any{"type": "user", "name": "kim"}},
		{ID: "inc-009-t4", IncidentID: "inc-009", At: now.Add(-60 * time.Minute), Kind: "note", Body: "QA validating affected orders in sandbox", Actor: map[string]any{"type": "user", "name": "jordan"}},
	}

	p.timeline["inc-010"] = []schema.TimelineEntry{
		{ID: "inc-010-t1", IncidentID: "inc-010", At: now.Add(-11*time.Hour - 50*time.Minute), Kind: "note", Body: "Indexer backlog climbed to 450k items after schema deploy", Actor: map[string]any{"type": "system", "name": "indexer"}},
		{ID: "inc-010-t2", IncidentID: "inc-010", At: now.Add(-10 * time.Hour), Kind: "note", Body: "Schema change introduced null category for legacy SKUs", Actor: map[string]any{"type": "user", "name": "casey"}},
		{ID: "inc-010-t3", IncidentID: "inc-010", At: now.Add(-6 * time.Hour), Kind: "note", Body: "Added secondary shard and requeued failed jobs", Actor: map[string]any{"type": "user", "name": "casey"}},
		{ID: "inc-010-t4", IncidentID: "inc-010", At: now.Add(-4 * time.Hour), Kind: "note", Body: "Backlog clearing at 40k/min, ETA 90m", Actor: map[string]any{"type": "user", "name": "casey"}},
	}

	p.timeline["inc-011"] = []schema.TimelineEntry{
		{ID: "inc-011-t1", IncidentID: "inc-011", At: now.Add(-17*time.Hour - 50*time.Minute), Kind: "note", Body: "Shipment ETA endpoints serving stale cache (>2h)", Actor: map[string]any{"type": "system", "name": "status-bot"}},
		{ID: "inc-011-t2", IncidentID: "inc-011", At: now.Add(-16 * time.Hour), Kind: "note", Body: "Paused CDN cache invalidations to stop thrash", Actor: map[string]any{"type": "user", "name": "alexis"}},
		{ID: "inc-011-t3", IncidentID: "inc-011", At: now.Add(-2 * time.Hour), Kind: "note", Body: "Hotfix to shorten cache TTL for status lookups", Actor: map[string]any{"type": "user", "name": "alexis"}},
		{ID: "inc-011-t4", IncidentID: "inc-011", At: now.Add(-30 * time.Minute), Kind: "note", Body: "Customer care confirms fresh ETAs; keeping monitors elevated", Actor: map[string]any{"type": "user", "name": "alexis"}},
	}

	p.timeline["inc-012"] = []schema.TimelineEntry{
		{ID: "inc-012-t1", IncidentID: "inc-012", At: now.Add(-2*time.Hour - 10*time.Minute), Kind: "note", Body: "Firefox clients disconnect after 45s with websocket close 1006", Actor: map[string]any{"type": "system", "name": "browser-watch"}},
		{ID: "inc-012-t2", IncidentID: "inc-012", At: now.Add(-100 * time.Minute), Kind: "note", Body: "Disabled permessage-deflate for Firefox user agent", Actor: map[string]any{"type": "user", "name": "samir"}},
		{ID: "inc-012-t3", IncidentID: "inc-012", At: now.Add(-40 * time.Minute), Kind: "note", Body: "Added 25s keepalive ping to websocket gateway", Actor: map[string]any{"type": "user", "name": "samir"}},
		{ID: "inc-012-t4", IncidentID: "inc-012", At: now.Add(-15 * time.Minute), Kind: "note", Body: "User retry reports stable connections; preparing hotfix release", Actor: map[string]any{"type": "user", "name": "samir"}},
	}

	// Scenario incident timelines
	p.timeline["inc-scenario-001"] = []schema.TimelineEntry{
		{ID: "inc-scenario-001-t1", IncidentID: "inc-scenario-001", At: now.Add(-45 * time.Minute), Kind: "note", Body: "Incident detected: SLO Budget Exhaustion", Actor: map[string]any{"type": "system", "name": "alertmanager"}},
		{ID: "inc-scenario-001-t2", IncidentID: "inc-scenario-001", At: now.Add(-40 * time.Minute), Kind: "note", Body: "Investigation started by alex", Actor: map[string]any{"type": "user", "name": "alex"}},
		{ID: "inc-scenario-001-t3", IncidentID: "inc-scenario-001", At: now.Add(-30 * time.Minute), Kind: "note", Body: "Mitigation actions in progress", Actor: map[string]any{"type": "user", "name": "alex"}},
		{ID: "inc-scenario-001-t4", IncidentID: "inc-scenario-001", At: now.Add(-10 * time.Minute), Kind: "note", Body: "Scaled up checkout service instances from 12 to 24", Actor: map[string]any{"type": "user", "name": "alex"}},
	}

	p.timeline["inc-scenario-002"] = []schema.TimelineEntry{
		{ID: "inc-scenario-002-t1", IncidentID: "inc-scenario-002", At: now.Add(-30 * time.Minute), Kind: "note", Body: "Incident detected: Cascading Failure", Actor: map[string]any{"type": "system", "name": "alertmanager"}},
		{ID: "inc-scenario-002-t2", IncidentID: "inc-scenario-002", At: now.Add(-25 * time.Minute), Kind: "note", Body: "Investigation started by morgan", Actor: map[string]any{"type": "user", "name": "morgan"}},
		{ID: "inc-scenario-002-t3", IncidentID: "inc-scenario-002", At: now.Add(-15 * time.Minute), Kind: "note", Body: "Identified connection leak in checkout service", Actor: map[string]any{"type": "user", "name": "morgan"}},
		{ID: "inc-scenario-002-t4", IncidentID: "inc-scenario-002", At: now.Add(-5 * time.Minute), Kind: "note", Body: "Restarted checkout service pods to release connections", Actor: map[string]any{"type": "user", "name": "morgan"}},
	}

	p.timeline["inc-scenario-003"] = []schema.TimelineEntry{
		{ID: "inc-scenario-003-t1", IncidentID: "inc-scenario-003", At: now.Add(-90 * time.Minute), Kind: "note", Body: "Incident detected: Deployment Rollback", Actor: map[string]any{"type": "system", "name": "alertmanager"}},
		{ID: "inc-scenario-003-t2", IncidentID: "inc-scenario-003", At: now.Add(-85 * time.Minute), Kind: "note", Body: "Investigation started by sam", Actor: map[string]any{"type": "user", "name": "sam"}},
		{ID: "inc-scenario-003-t3", IncidentID: "inc-scenario-003", At: now.Add(-75 * time.Minute), Kind: "note", Body: "Mitigation actions in progress", Actor: map[string]any{"type": "user", "name": "sam"}},
		{ID: "inc-scenario-003-t4", IncidentID: "inc-scenario-003", At: now.Add(-60 * time.Minute), Kind: "note", Body: "Rolled back payment service from v2.8.3 to v2.8.2", Actor: map[string]any{"type": "user", "name": "sam"}},
		{ID: "inc-scenario-003-t5", IncidentID: "inc-scenario-003", At: now.Add(-20 * time.Minute), Kind: "note", Body: "Mitigation applied, monitoring for stability", Actor: map[string]any{"type": "user", "name": "sam"}},
	}

	p.timeline["inc-scenario-004"] = []schema.TimelineEntry{
		{ID: "inc-scenario-004-t1", IncidentID: "inc-scenario-004", At: now.Add(-15 * time.Minute), Kind: "note", Body: "Incident detected: External Dependency Failure", Actor: map[string]any{"type": "system", "name": "alertmanager"}},
		{ID: "inc-scenario-004-t2", IncidentID: "inc-scenario-004", At: now.Add(-12 * time.Minute), Kind: "note", Body: "Investigation started by fern", Actor: map[string]any{"type": "user", "name": "fern"}},
		{ID: "inc-scenario-004-t3", IncidentID: "inc-scenario-004", At: now.Add(-8 * time.Minute), Kind: "note", Body: "Confirmed Stripe API rate limiting affecting checkout", Actor: map[string]any{"type": "user", "name": "fern"}},
		{ID: "inc-scenario-004-t4", IncidentID: "inc-scenario-004", At: now.Add(-5 * time.Minute), Kind: "note", Body: "Enabled circuit breaker for Stripe API calls", Actor: map[string]any{"type": "user", "name": "fern"}},
	}

	p.timeline["inc-scenario-005"] = []schema.TimelineEntry{
		{ID: "inc-scenario-005-t1", IncidentID: "inc-scenario-005", At: now.Add(-12 * time.Minute), Kind: "note", Body: "Incident detected: Autoscaling Lag", Actor: map[string]any{"type": "system", "name": "alertmanager"}},
		{ID: "inc-scenario-005-t2", IncidentID: "inc-scenario-005", At: now.Add(-10 * time.Minute), Kind: "note", Body: "Investigation started by lena", Actor: map[string]any{"type": "user", "name": "lena"}},
		{ID: "inc-scenario-005-t3", IncidentID: "inc-scenario-005", At: now.Add(-6 * time.Minute), Kind: "note", Body: "Identified traffic spike from marketing campaign", Actor: map[string]any{"type": "user", "name": "lena"}},
		{ID: "inc-scenario-005-t4", IncidentID: "inc-scenario-005", At: now.Add(-3 * time.Minute), Kind: "note", Body: "Manually scaled search service from 3 to 8 replicas", Actor: map[string]any{"type": "user", "name": "lena"}},
	}

	p.timeline["inc-scenario-006"] = []schema.TimelineEntry{
		{ID: "inc-scenario-006-t1", IncidentID: "inc-scenario-006", At: now.Add(-8 * time.Minute), Kind: "note", Body: "Incident detected: Circuit Breaker Cascade", Actor: map[string]any{"type": "system", "name": "alertmanager"}},
		{ID: "inc-scenario-006-t2", IncidentID: "inc-scenario-006", At: now.Add(-6 * time.Minute), Kind: "note", Body: "Investigation started by milo", Actor: map[string]any{"type": "user", "name": "milo"}},
		{ID: "inc-scenario-006-t3", IncidentID: "inc-scenario-006", At: now.Add(-4 * time.Minute), Kind: "note", Body: "Identified recommendation model inference timeout", Actor: map[string]any{"type": "user", "name": "milo"}},
		{ID: "inc-scenario-006-t4", IncidentID: "inc-scenario-006", At: now.Add(-2 * time.Minute), Kind: "note", Body: "Restarting recommendation service pods", Actor: map[string]any{"type": "user", "name": "milo"}},
	}
}

func parseConfig(cfg map[string]any) Config {
	out := Config{Source: "mock", DefaultSeverity: "sev2"}
	if v, ok := cfg["source"].(string); ok && v != "" {
		out.Source = v
	}
	if v, ok := cfg["defaultSeverity"].(string); ok && v != "" {
		out.DefaultSeverity = v
	}
	return out
}

func emptyFallback(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}

func inferService(in schema.CreateIncidentInput) string {
	if in.Service != "" {
		return in.Service
	}
	if in.Fields != nil {
		if svc, ok := in.Fields["service"].(string); ok {
			return svc
		}
	}
	return ""
}

func cloneIncident(in schema.Incident) schema.Incident {
	return schema.Incident{
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

func cloneTimeline(in []schema.TimelineEntry) []schema.TimelineEntry {
	if in == nil {
		return nil
	}
	out := make([]schema.TimelineEntry, len(in))
	for i, e := range in {
		out[i] = schema.TimelineEntry{
			ID:         e.ID,
			IncidentID: e.IncidentID,
			At:         e.At,
			Kind:       e.Kind,
			Body:       e.Body,
			Actor:      mockutil.CloneMap(e.Actor),
			Metadata:   mockutil.CloneMap(e.Metadata),
		}
	}
	return out
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

func matchesQuery(needle string, inc schema.Incident) bool {
	if needle == "" {
		return true
	}
	if strings.Contains(strings.ToLower(inc.Title), needle) {
		return true
	}
	if inc.Description != "" && strings.Contains(strings.ToLower(inc.Description), needle) {
		return true
	}
	if inc.Service != "" && strings.Contains(strings.ToLower(inc.Service), needle) {
		return true
	}
	if strings.Contains(strings.ToLower(inc.ID), needle) {
		return true
	}
	return false
}

func matchesScope(scope schema.QueryScope, inc schema.Incident) bool {
	if scope == (schema.QueryScope{}) {
		return true
	}

	if scope.Service != "" {
		svc := inc.Service
		if svc == "" {
			if f, ok := inc.Fields["service"].(string); ok {
				svc = f
			}
		}
		if svc != scope.Service {
			return false
		}
	}
	if scope.Environment != "" {
		if env, ok := inc.Fields["environment"].(string); !ok || env != scope.Environment {
			return false
		}
	}
	if scope.Team != "" {
		if team, ok := inc.Fields["team"].(string); !ok || team != scope.Team {
			return false
		}
	}

	return true
}
