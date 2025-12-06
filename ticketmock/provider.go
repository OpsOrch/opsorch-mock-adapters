package ticketmock

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/opsorch/opsorch-core/orcherr"
	"github.com/opsorch/opsorch-core/schema"
	coreticket "github.com/opsorch/opsorch-core/ticket"
	"github.com/opsorch/opsorch-mock-adapters/internal/mockutil"
)

// ProviderName can be referenced via OPSORCH_TICKET_PROVIDER.
const ProviderName = "mock"

// Config controls mock ticket metadata.
type Config struct {
	Source string
}

// Provider holds in-memory tickets to support demo flows.
type Provider struct {
	cfg     Config
	mu      sync.Mutex
	nextID  int
	tickets map[string]schema.Ticket
}

// New constructs the mock ticket provider with seeded work items.
func New(cfg map[string]any) (coreticket.Provider, error) {
	parsed := parseConfig(cfg)
	p := &Provider{cfg: parsed, tickets: map[string]schema.Ticket{}}
	p.seed()
	return p, nil
}

func init() {
	_ = coreticket.RegisterProvider(ProviderName, New)
}

// Query returns tickets that match the provided filters.
func (p *Provider) Query(ctx context.Context, query schema.TicketQuery) ([]schema.Ticket, error) {
	_ = ctx

	p.mu.Lock()
	defer p.mu.Unlock()

	// Add static scenario-themed tickets
	now := time.Now().UTC()
	scenarioTickets := getScenarioTickets(now)
	for _, st := range scenarioTickets {
		p.tickets[st.ID] = st
	}

	ids := sortedTicketIDs(p.tickets)
	results := make([]schema.Ticket, 0, len(p.tickets))
	for _, id := range ids {
		tk := p.tickets[id]
		if !matchesTicket(query, tk) {
			continue
		}
		results = append(results, cloneTicket(tk))
		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}
	}

	return results, nil
}

// Get returns a ticket by ID.
func (p *Provider) Get(ctx context.Context, id string) (schema.Ticket, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tk, ok := p.tickets[id]
	if !ok {
		return schema.Ticket{}, orcherr.New("not_found", "ticket not found", nil)
	}
	return cloneTicket(tk), nil
}

// Create inserts a new ticket.
func (p *Provider) Create(ctx context.Context, in schema.CreateTicketInput) (schema.Ticket, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.nextID++
	id := fmt.Sprintf("TCK-%03d", p.nextID)
	now := time.Now().UTC()

	tk := schema.Ticket{
		ID:          id,
		Key:         id,
		Title:       in.Title,
		Description: in.Description,
		Status:      "todo",
		CreatedAt:   now,
		UpdatedAt:   now,
		Fields:      mockutil.CloneMap(in.Fields),
		Metadata:    mockutil.CloneMap(in.Metadata),
	}
	if tk.Metadata == nil {
		tk.Metadata = map[string]any{}
	}
	tk.Metadata["source"] = p.cfg.Source

	p.tickets[id] = tk
	return cloneTicket(tk), nil
}

// Update mutates ticket fields.
func (p *Provider) Update(ctx context.Context, id string, in schema.UpdateTicketInput) (schema.Ticket, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tk, ok := p.tickets[id]
	if !ok {
		return schema.Ticket{}, orcherr.New("not_found", "ticket not found", nil)
	}

	if in.Title != nil {
		tk.Title = *in.Title
	}
	if in.Description != nil {
		tk.Description = *in.Description
	}
	if in.Status != nil {
		tk.Status = *in.Status
	}
	if in.Assignees != nil {
		tk.Assignees = mockutil.CloneStringSlice(*in.Assignees)
	}
	if in.Fields != nil {
		tk.Fields = mockutil.CloneMap(in.Fields)
	}
	if in.Metadata != nil {
		tk.Metadata = mockutil.CloneMap(in.Metadata)
	}
	tk.UpdatedAt = time.Now().UTC()

	p.tickets[id] = tk
	return cloneTicket(tk), nil
}

func (p *Provider) seed() {
	now := time.Now().UTC()
	seed := []schema.Ticket{
		{
			ID:          "TCK-001",
			Key:         "TCK-001",
			Title:       "Checkout circuit breaker postmortem",
			Description: "Document mitigation steps and attach graphs from EUW1 outage",
			Status:      "in_progress",
			Assignees:   []string{"alex"},
			Reporter:    "sre-bot",
			CreatedAt:   now.Add(-24 * time.Hour),
			UpdatedAt:   now.Add(-2 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-checkout",
				"environment": "prod",
				"team":        "team-velocity",
				"priority":    "P1",
				"sprint":      "2024-09-a",
			},
			Metadata: map[string]any{"source": p.cfg.Source},
		},
		{
			ID:        "TCK-002",
			Key:       "TCK-002",
			Title:     "Search ranking tweaks for seasonal terms",
			Status:    "in_review",
			Assignees: []string{"jamie", "taylor"},
			Reporter:  "product",
			CreatedAt: now.Add(-72 * time.Hour),
			UpdatedAt: now.Add(-48 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-search",
				"environment": "prod",
				"team":        "team-aurora",
				"priority":    "P2",
				"labels":      []string{"relevance", "experiment"},
			},
			Metadata: map[string]any{"source": p.cfg.Source},
		},
		{
			ID:          "TCK-003",
			Key:         "TCK-003",
			Title:       "Payments webhook retry policy adjustments",
			Description: "Tune exponential backoff for Stripe webhooks and add DLQ metrics",
			Status:      "todo",
			Assignees:   []string{"sam"},
			Reporter:    "revenue-lead",
			CreatedAt:   now.Add(-12 * time.Hour),
			UpdatedAt:   now.Add(-6 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-payments",
				"environment": "prod",
				"team":        "team-revenue",
				"priority":    "P1",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "epic": "PAY-121"},
		},
		{
			ID:          "TCK-004",
			Key:         "TCK-004",
			Title:       "Notification fanout Kafka partition rebalance",
			Description: "Implement partition assignment strategy for promo campaigns",
			Status:      "in_progress",
			Assignees:   []string{"lee", "taylor"},
			Reporter:    "sre-bot",
			CreatedAt:   now.Add(-36 * time.Hour),
			UpdatedAt:   now.Add(-8 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-notifications",
				"environment": "prod",
				"team":        "team-signal",
				"priority":    "P1",
				"labels":      []string{"kafka", "throughput"},
			},
			Metadata: map[string]any{"source": p.cfg.Source},
		},
		{
			ID:          "TCK-005",
			Key:         "TCK-005",
			Title:       "Identity service Redis pool tuning",
			Description: "Lower max_idle_conns and add per-region dashboards",
			Status:      "todo",
			Assignees:   []string{"devon"},
			Reporter:    "platform",
			CreatedAt:   now.Add(-48 * time.Hour),
			UpdatedAt:   now.Add(-24 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-identity",
				"environment": "prod",
				"team":        "team-guardian",
				"priority":    "P2",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "sprint": "2024-09-b"},
		},
		{
			ID:          "TCK-006",
			Key:         "TCK-006",
			Title:       "Warehouse stuck batch RCA",
			Description: "Document partition lock issue and add checkpoint alarms",
			Status:      "in_review",
			Assignees:   []string{"morgan"},
			Reporter:    "data-eng",
			CreatedAt:   now.Add(-72 * time.Hour),
			UpdatedAt:   now.Add(-6 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-warehouse",
				"environment": "prod",
				"team":        "team-foundry",
				"priority":    "P2",
				"labels":      []string{"rca"},
			},
			Metadata: map[string]any{"source": p.cfg.Source},
		},
		{
			ID:          "TCK-007",
			Key:         "TCK-007",
			Title:       "Recommendation model rollback automation",
			Description: "Add one-click rollback and feature store pinning",
			Status:      "in_progress",
			Assignees:   []string{"riley"},
			Reporter:    "ml-lead",
			CreatedAt:   now.Add(-30 * time.Hour),
			UpdatedAt:   now.Add(-4 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-recommendation",
				"environment": "prod",
				"team":        "team-orion",
				"priority":    "P1",
				"labels":      []string{"ml", "rollback"},
			},
			Metadata: map[string]any{"source": p.cfg.Source, "epic": "RECO-88"},
		},
		{
			ID:          "TCK-008",
			Key:         "TCK-008",
			Title:       "Analytics APAC ingestion auth refresh",
			Description: "Ensure service accounts rotate before expiration; add alerting",
			Status:      "todo",
			Assignees:   []string{"maya"},
			Reporter:    "data-platform",
			CreatedAt:   now.Add(-20 * time.Hour),
			UpdatedAt:   now.Add(-10 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-analytics",
				"environment": "prod",
				"team":        "team-lumen",
				"priority":    "P1",
				"labels":      []string{"auth", "rotation"},
			},
			Metadata: map[string]any{"source": p.cfg.Source},
		},
		{
			ID:          "TCK-009",
			Key:         "TCK-009",
			Title:       "Order prepaid fallback validation",
			Description: "Add integration tests for prepaid BIN ranges and gateway failover",
			Status:      "in_review",
			Assignees:   []string{"kim", "jordan"},
			Reporter:    "checkout-pm",
			CreatedAt:   now.Add(-18 * time.Hour),
			UpdatedAt:   now.Add(-2 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-order",
				"environment": "prod",
				"team":        "team-velocity",
				"priority":    "P1",
				"labels":      []string{"payments", "tests"},
			},
			Metadata: map[string]any{"source": p.cfg.Source},
		},
		{
			ID:          "TCK-010",
			Key:         "TCK-010",
			Title:       "Realtime gateway Firefox hotfix rollout",
			Description: "Package websocket keepalive changes and verify cross-browser",
			Status:      "todo",
			Assignees:   []string{"samir"},
			Reporter:    "edge-team",
			CreatedAt:   now.Add(-6 * time.Hour),
			UpdatedAt:   now.Add(-1 * time.Hour),
			Fields: map[string]any{
				"service":     "svc-realtime",
				"environment": "prod",
				"team":        "team-nova",
				"priority":    "P1",
			},
			Metadata: map[string]any{"source": p.cfg.Source, "release": "edge-1.8.1"},
		},
	}

	for _, tk := range seed {
		applyTicketFlair(&tk, now)
		p.tickets[tk.ID] = tk
		if n, err := fmt.Sscanf(tk.ID, "TCK-%d", &p.nextID); n == 1 && err == nil {
			// keep last parsed id
		}
	}
}

func parseConfig(cfg map[string]any) Config {
	out := Config{Source: "mock"}
	if v, ok := cfg["source"].(string); ok && v != "" {
		out.Source = v
	}
	return out
}

func applyTicketFlair(tk *schema.Ticket, now time.Time) {
	if tk.Fields == nil {
		tk.Fields = map[string]any{}
	}
	service, _ := tk.Fields["service"].(string)
	links := serviceLinks(service)
	tk.Fields["links"] = links
	tk.Fields["checklist"] = []map[string]any{
		{"label": "Review dashboards", "done": tk.Status != "todo"},
		{"label": "Post update in #ops", "done": tk.Status == "in_progress" || tk.Status == "in_review"},
	}
	tk.Fields["dueDate"] = now.Add(dueDurationForStatus(tk.Status))
	tk.Fields["dependencies"] = dependencyHints(service)
	tk.Fields["effortHours"] = 6 + len(tk.Assignees)*2
	if tk.Metadata == nil {
		tk.Metadata = map[string]any{}
	}
	tk.Metadata["relatedIncidents"] = relatedIncidents(service)
	clonedLinks := append([]string(nil), links...)
	tk.Metadata["links"] = clonedLinks
	if len(tk.Assignees) > 0 {
		tk.Metadata["lastUpdatedBy"] = tk.Assignees[0]
	}
}

func serviceLinks(service string) []string {
	key := ticketServiceKey(service)
	if key == "" {
		key = "platform"
	}
	return []string{
		fmt.Sprintf("https://runbook.demo/%s", key),
		fmt.Sprintf("https://grafana.demo/d/%s", key),
	}
}

func dueDurationForStatus(status string) time.Duration {
	switch status {
	case "todo":
		return 72 * time.Hour
	case "in_progress":
		return 36 * time.Hour
	case "in_review":
		return 12 * time.Hour
	default:
		return 48 * time.Hour
	}
}

func dependencyHints(service string) []string {
	switch ticketServiceKey(service) {
	case "checkout":
		return []string{"svc-payments", "svc-order"}
	case "search":
		return []string{"svc-web"}
	case "payments":
		return []string{"svc-notifications", "svc-order"}
	case "notifications":
		return []string{"svc-analytics"}
	case "identity":
		return []string{"svc-web", "svc-notifications"}
	case "warehouse":
		return []string{"svc-analytics"}
	case "realtime":
		return []string{"svc-web"}
	default:
		return []string{"platform"}
	}
}

func relatedIncidents(service string) []string {
	switch ticketServiceKey(service) {
	case "checkout":
		return []string{"inc-001"}
	case "search":
		return []string{"inc-002"}
	case "payments":
		return []string{"inc-003"}
	case "notifications":
		return []string{"inc-004"}
	case "identity":
		return []string{"inc-005"}
	case "warehouse":
		return []string{"inc-006"}
	case "analytics":
		return []string{"inc-008"}
	case "order":
		return []string{"inc-009"}
	case "realtime":
		return []string{"inc-012"}
	default:
		return nil
	}
}

func ticketServiceKey(service string) string {
	if service == "" {
		return ""
	}
	return strings.TrimPrefix(service, "svc-")
}

func cloneTicket(in schema.Ticket) schema.Ticket {
	return schema.Ticket{
		ID:          in.ID,
		Key:         in.Key,
		Title:       in.Title,
		Description: in.Description,
		Status:      in.Status,
		Assignees:   mockutil.CloneStringSlice(in.Assignees),
		Reporter:    in.Reporter,
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
		Fields:      mockutil.CloneMap(in.Fields),
		Metadata:    mockutil.CloneMap(in.Metadata),
	}
}

func sortedTicketIDs(tickets map[string]schema.Ticket) []string {
	ids := make([]string, 0, len(tickets))
	for id := range tickets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func matchesTicket(query schema.TicketQuery, tk schema.Ticket) bool {
	if !matchesQuery(query.Query, tk) {
		return false
	}
	if !matchesScope(query.Scope, tk) {
		return false
	}
	if len(query.Statuses) > 0 && !matchesStatuses(query.Statuses, tk.Status) {
		return false
	}
	if len(query.Assignees) > 0 && !matchesAssignees(query.Assignees, tk.Assignees) {
		return false
	}
	if query.Reporter != "" && query.Reporter != tk.Reporter {
		return false
	}
	if len(query.Metadata) > 0 && !matchesMetadata(query.Metadata, tk.Metadata) {
		return false
	}

	return true
}

func matchesStatuses(filter []string, status string) bool {
	for _, s := range filter {
		if s == status {
			return true
		}
	}
	return false
}

func matchesScope(scope schema.QueryScope, tk schema.Ticket) bool {
	if scope == (schema.QueryScope{}) {
		return true
	}

	if scope.Service != "" {
		if svc, ok := tk.Fields["service"].(string); !ok || svc != scope.Service {
			return false
		}
	}
	if scope.Environment != "" {
		if env, ok := tk.Fields["environment"].(string); !ok || env != scope.Environment {
			return false
		}
	}
	if scope.Team != "" {
		if team, ok := tk.Fields["team"].(string); !ok || team != scope.Team {
			return false
		}
	}

	return true
}

func matchesAssignees(filter []string, assignees []string) bool {
	for _, desired := range filter {
		for _, actual := range assignees {
			if desired == actual {
				return true
			}
		}
	}
	return false
}

func matchesMetadata(filter map[string]any, metadata map[string]any) bool {
	if len(filter) == 0 {
		return true
	}
	if len(metadata) == 0 {
		return false
	}
	for k, v := range filter {
		if !reflect.DeepEqual(metadata[k], v) {
			return false
		}
	}
	return true
}

func matchesQuery(filter string, tk schema.Ticket) bool {
	if filter == "" {
		return true
	}

	needle := strings.ToLower(filter)
	fields := []string{tk.ID, tk.Key, tk.Title, tk.Description, tk.Reporter}
	for _, field := range fields {
		if field != "" && strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}
	for _, a := range tk.Assignees {
		if strings.Contains(strings.ToLower(a), needle) {
			return true
		}
	}

	return false
}

// getScenarioTickets returns static scenario-themed tickets
func getScenarioTickets(now time.Time) []schema.Ticket {
	return []schema.Ticket{
		{
			ID:          "TCK-SCENARIO-001",
			Key:         "TCK-SCENARIO-001",
			Title:       "SLO Budget Exhaustion - Payment Gateway Timeouts",
			Description: "Investigate and mitigate payment gateway timeouts causing SLO budget exhaustion. Error rate exceeded threshold at 15:30 UTC.",
			Status:      "in_progress",
			Assignees:   []string{"alex", "sam"},
			Reporter:    "sre-bot",
			CreatedAt:   now.Add(-25 * time.Minute),
			UpdatedAt:   now.Add(-5 * time.Minute),
			Fields: map[string]any{
				"service":           "svc-checkout",
				"environment":       "prod",
				"team":              "team-velocity",
				"priority":          "P0",
				"sprint":            "2024-12-a",
				"incident_id":       "inc-scenario-001",
				"scenario_id":       "scenario-001",
				"scenario_name":     "SLO Budget Exhaustion",
				"scenario_stage":    "active",
				"is_scenario":       true,
				"error_budget_burn": "85%",
				"affected_regions":  []string{"use1", "euw1"},
				"labels":            []string{"slo", "payments", "urgent"},
			},
			Metadata: map[string]any{
				"source":         "mock",
				"scenario_id":    "scenario-001",
				"scenario_name":  "SLO Budget Exhaustion",
				"scenario_stage": "active",
				"incident_id":    "inc-scenario-001",
				"links": []string{
					"https://runbook.demo/checkout",
					"https://grafana.demo/d/checkout",
				},
				"relatedIncidents": []string{"inc-scenario-001"},
			},
		},
		{
			ID:          "TCK-SCENARIO-002",
			Key:         "TCK-SCENARIO-002",
			Title:       "Database Connection Pool Exhaustion - Search Service",
			Description: "Cascading failure in search service due to database connection pool exhaustion. Implement connection pooling improvements and add monitoring.",
			Status:      "todo",
			Assignees:   []string{"jamie"},
			Reporter:    "incident-manager",
			CreatedAt:   now.Add(-20 * time.Minute),
			UpdatedAt:   now.Add(-10 * time.Minute),
			Fields: map[string]any{
				"service":          "svc-search",
				"environment":      "prod",
				"team":             "team-aurora",
				"priority":         "P0",
				"sprint":           "2024-12-a",
				"incident_id":      "inc-scenario-002",
				"scenario_id":      "scenario-002",
				"scenario_name":    "Cascading Database Failure",
				"scenario_stage":   "escalating",
				"is_scenario":      true,
				"db_connections":   "100/100",
				"affected_queries": []string{"search", "autocomplete", "trending"},
				"labels":           []string{"database", "cascading-failure", "critical"},
			},
			Metadata: map[string]any{
				"source":         "mock",
				"scenario_id":    "scenario-002",
				"scenario_name":  "Cascading Database Failure",
				"scenario_stage": "escalating",
				"incident_id":    "inc-scenario-002",
				"links": []string{
					"https://runbook.demo/search",
					"https://grafana.demo/d/search",
				},
				"relatedIncidents": []string{"inc-scenario-002"},
			},
		},
		{
			ID:          "TCK-SCENARIO-003",
			Key:         "TCK-SCENARIO-003",
			Title:       "Deployment Rollback - Payment Service v2.31.4",
			Description: "Rollback payment service deployment due to elevated error rates. Document root cause and implement additional pre-deployment checks.",
			Status:      "in_review",
			Assignees:   []string{"taylor", "devon"},
			Reporter:    "deploy-bot",
			CreatedAt:   now.Add(-15 * time.Minute),
			UpdatedAt:   now.Add(-3 * time.Minute),
			Fields: map[string]any{
				"service":          "svc-checkout",
				"environment":      "prod",
				"team":             "team-velocity",
				"priority":         "P1",
				"sprint":           "2024-12-a",
				"incident_id":      "inc-scenario-003",
				"scenario_id":      "scenario-003",
				"scenario_name":    "Deployment Rollback",
				"scenario_stage":   "mitigating",
				"is_scenario":      true,
				"deployment_id":    "deploy-2024-12-07-003",
				"rollback_reason":  "error_rate_threshold_exceeded",
				"previous_version": "v2.31.3",
				"failed_version":   "v2.31.4",
				"labels":           []string{"deployment", "rollback", "postmortem"},
			},
			Metadata: map[string]any{
				"source":         "mock",
				"scenario_id":    "scenario-003",
				"scenario_name":  "Deployment Rollback",
				"scenario_stage": "mitigating",
				"incident_id":    "inc-scenario-003",
				"links": []string{
					"https://runbook.demo/checkout",
					"https://grafana.demo/d/checkout",
				},
				"relatedIncidents": []string{"inc-scenario-003"},
			},
		},
		{
			ID:          "TCK-SCENARIO-004",
			Key:         "TCK-SCENARIO-004",
			Title:       "External Dependency Failure - Stripe API Rate Limiting",
			Description: "Implement retry logic and circuit breaker for Stripe API calls. Add monitoring for external dependency health.",
			Status:      "in_progress",
			Assignees:   []string{"morgan", "riley"},
			Reporter:    "platform-team",
			CreatedAt:   now.Add(-12 * time.Minute),
			UpdatedAt:   now.Add(-2 * time.Minute),
			Fields: map[string]any{
				"service":          "svc-checkout",
				"environment":      "prod",
				"team":             "team-velocity",
				"priority":         "P1",
				"sprint":           "2024-12-a",
				"incident_id":      "inc-scenario-004",
				"scenario_id":      "scenario-004",
				"scenario_name":    "External Dependency Failure - Stripe",
				"scenario_stage":   "active",
				"is_scenario":      true,
				"external_service": "stripe",
				"external_error":   "rate_limit_exceeded",
				"retry_strategy":   "exponential_backoff",
				"circuit_breaker":  "implementing",
				"labels":           []string{"external-dependency", "stripe", "resilience"},
			},
			Metadata: map[string]any{
				"source":         "mock",
				"scenario_id":    "scenario-004",
				"scenario_name":  "External Dependency Failure - Stripe",
				"scenario_stage": "active",
				"incident_id":    "inc-scenario-004",
				"links": []string{
					"https://runbook.demo/checkout",
					"https://grafana.demo/d/checkout",
				},
				"relatedIncidents": []string{"inc-scenario-004"},
			},
		},
		{
			ID:          "TCK-SCENARIO-005",
			Key:         "TCK-SCENARIO-005",
			Title:       "Autoscaling Lag - Search Service Capacity",
			Description: "Tune autoscaling policies for search service to reduce lag during traffic spikes. Add predictive scaling based on historical patterns.",
			Status:      "todo",
			Assignees:   []string{"kim"},
			Reporter:    "capacity-planning",
			CreatedAt:   now.Add(-8 * time.Minute),
			UpdatedAt:   now.Add(-1 * time.Minute),
			Fields: map[string]any{
				"service":            "svc-search",
				"environment":        "prod",
				"team":               "team-aurora",
				"priority":           "P2",
				"sprint":             "2024-12-a",
				"scenario_id":        "scenario-005",
				"scenario_name":      "Autoscaling Lag",
				"scenario_stage":     "active",
				"is_scenario":        true,
				"current_instances":  3,
				"target_instances":   8,
				"scaling_duration":   "5m",
				"autoscaling_policy": "cpu-based",
				"labels":             []string{"autoscaling", "capacity", "performance"},
			},
			Metadata: map[string]any{
				"source":         "mock",
				"scenario_id":    "scenario-005",
				"scenario_name":  "Autoscaling Lag",
				"scenario_stage": "active",
				"links": []string{
					"https://runbook.demo/search",
					"https://grafana.demo/d/search",
				},
			},
		},
		{
			ID:          "TCK-SCENARIO-006",
			Key:         "TCK-SCENARIO-006",
			Title:       "Circuit Breaker Cascade - Payment Gateway",
			Description: "Investigate circuit breaker cascade in payment gateway. Implement bulkhead pattern and improve failure isolation.",
			Status:      "in_progress",
			Assignees:   []string{"samir", "maya"},
			Reporter:    "sre-lead",
			CreatedAt:   now.Add(-5 * time.Minute),
			UpdatedAt:   now.Add(-1 * time.Minute),
			Fields: map[string]any{
				"service":            "svc-checkout",
				"environment":        "prod",
				"team":               "team-velocity",
				"priority":           "P0",
				"sprint":             "2024-12-a",
				"incident_id":        "inc-scenario-006",
				"scenario_id":        "scenario-006",
				"scenario_name":      "Circuit Breaker Cascade",
				"scenario_stage":     "escalating",
				"is_scenario":        true,
				"circuit_state":      "open",
				"failure_threshold":  "5/10",
				"downstream_service": "payment-gateway",
				"affected_regions":   []string{"use1", "aps1"},
				"labels":             []string{"circuit-breaker", "cascading-failure", "resilience"},
			},
			Metadata: map[string]any{
				"source":         "mock",
				"scenario_id":    "scenario-006",
				"scenario_name":  "Circuit Breaker Cascade",
				"scenario_stage": "escalating",
				"incident_id":    "inc-scenario-006",
				"links": []string{
					"https://runbook.demo/checkout",
					"https://grafana.demo/d/checkout",
				},
				"relatedIncidents": []string{"inc-scenario-006"},
			},
		},
	}
}

var _ coreticket.Provider = (*Provider)(nil)
