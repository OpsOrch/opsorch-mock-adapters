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

var _ coreticket.Provider = (*Provider)(nil)
