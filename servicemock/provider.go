package servicemock

import (
	"context"
	"fmt"
	"strings"

	"github.com/opsorch/opsorch-core/schema"
	coreservice "github.com/opsorch/opsorch-core/service"
	"github.com/opsorch/opsorch-mock-adapters/internal/mockutil"
)

// ProviderName can be passed to OPSORCH_SERVICE_PROVIDER.
const ProviderName = "mock"

// Config allows limited customization for demo scenarios.
type Config struct {
	// Environment tag that will be stamped on all demo services.
	Environment string
}

// Provider serves a static set of demo services and applies client-side filtering.
type Provider struct {
	cfg      Config
	services []schema.Service
}

// New constructs the mock service provider.
func New(cfg map[string]any) (coreservice.Provider, error) {
	parsed := parseConfig(cfg)
	services := seedServices(parsed)
	return &Provider{cfg: parsed, services: services}, nil
}

func init() {
	_ = coreservice.RegisterProvider(ProviderName, New)
}

// Query filters demo services by the provided criteria.
func (p *Provider) Query(ctx context.Context, query schema.ServiceQuery) ([]schema.Service, error) {
	_ = ctx

	results := make([]schema.Service, 0, len(p.services))
	for _, svc := range p.services {
		if !matchesIDs(query.IDs, svc.ID) {
			continue
		}
		if !matchesName(query.Name, svc.Name) {
			continue
		}
		if !matchesTags(query.Tags, svc.Tags) {
			continue
		}
		if !matchesScope(query.Scope, svc) {
			continue
		}

		// Clone service for result
		enriched := cloneService(svc)

		results = append(results, enriched)
		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}
	}

	return results, nil
}

func parseConfig(cfg map[string]any) Config {
	out := Config{Environment: "prod"}
	if v, ok := cfg["environment"].(string); ok && v != "" {
		out.Environment = v
	}
	return out
}

func seedServices(cfg Config) []schema.Service {
	base := []schema.Service{
		{
			ID:   "svc-checkout",
			Name: "Checkout API",
			Tags: map[string]string{"env": cfg.Environment, "tier": "backend", "owner": "team-velocity"},
			Metadata: map[string]any{
				"description": "Entry point for cart and payment flows",
				"links":       []string{"https://runbook.demo/checkout", "https://grafana.demo/d/checkout", "https://github.com/opsorch/demo-checkout"},
				"oncall":      "pd:checkout-primary",
				"language":    "go",
			},
		},
		{
			ID:   "svc-search",
			Name: "Search Service",
			Tags: map[string]string{"env": cfg.Environment, "tier": "backend", "owner": "team-aurora"},
			Metadata: map[string]any{
				"description": "Search and discovery pipeline",
				"links":       []string{"https://runbook.demo/search", "https://grafana.demo/d/search"},
				"oncall":      "pd:search-primary",
				"language":    "java",
			},
		},
		{
			ID:   "svc-web",
			Name: "Web Frontend",
			Tags: map[string]string{"env": cfg.Environment, "tier": "frontend", "owner": "team-velocity"},
			Metadata: map[string]any{
				"description": "User-facing React app served via CDN",
				"links":       []string{"https://grafana.demo/d/web", "https://runbook.demo/web"},
				"language":    "typescript",
			},
		},
		{
			ID:   "svc-payments",
			Name: "Payments Service",
			Tags: map[string]string{"env": cfg.Environment, "tier": "backend", "owner": "team-revenue"},
			Metadata: map[string]any{
				"description": "Processes payment intents and handles provider webhooks",
				"links":       []string{"https://runbook.demo/payments", "https://grafana.demo/d/payments"},
				"oncall":      "pd:payments-primary",
				"language":    "go",
			},
		},
		{
			ID:   "svc-notifications",
			Name: "Notification Fanout",
			Tags: map[string]string{"env": cfg.Environment, "tier": "backend", "owner": "team-signal"},
			Metadata: map[string]any{
				"description": "Multichannel fanout for email, SMS, and push",
				"links":       []string{"https://runbook.demo/notifications", "https://grafana.demo/d/notifications"},
				"oncall":      "pd:notifications",
				"language":    "kotlin",
			},
		},
		{
			ID:   "svc-identity",
			Name: "Identity Platform",
			Tags: map[string]string{"env": cfg.Environment, "tier": "backend", "owner": "team-guardian"},
			Metadata: map[string]any{
				"description": "Authentication, MFA, and session management APIs",
				"links":       []string{"https://runbook.demo/identity", "https://grafana.demo/d/identity"},
				"oncall":      "pd:identity",
				"language":    "go",
			},
		},
		{
			ID:   "svc-warehouse",
			Name: "Data Warehouse Jobs",
			Tags: map[string]string{"env": cfg.Environment, "tier": "data", "owner": "team-foundry"},
			Metadata: map[string]any{
				"description": "Scheduled batch jobs loading facts and dimensions",
				"links":       []string{"https://runbook.demo/warehouse", "https://grafana.demo/d/warehouse"},
				"oncall":      "pd:data-platform",
				"language":    "python",
			},
		},
		{
			ID:   "svc-recommendation",
			Name: "Recommendation Engine",
			Tags: map[string]string{"env": cfg.Environment, "tier": "backend", "owner": "team-orion"},
			Metadata: map[string]any{
				"description": "Personalized ranking and candidate generation",
				"links":       []string{"https://runbook.demo/recommendations", "https://grafana.demo/d/reco"},
				"oncall":      "pd:reco",
				"language":    "scala",
			},
		},
		{
			ID:   "svc-analytics",
			Name: "Analytics Pipeline",
			Tags: map[string]string{"env": cfg.Environment, "tier": "data", "owner": "team-lumen"},
			Metadata: map[string]any{
				"description": "Event ingestion, ETL, and warehouse export",
				"links":       []string{"https://runbook.demo/analytics", "https://grafana.demo/d/analytics"},
				"oncall":      "pd:analytics",
				"language":    "python",
			},
		},
		{
			ID:   "svc-order",
			Name: "Order Service",
			Tags: map[string]string{"env": cfg.Environment, "tier": "backend", "owner": "team-velocity"},
			Metadata: map[string]any{
				"description": "Order lifecycle management and payment orchestration",
				"links":       []string{"https://runbook.demo/orders", "https://grafana.demo/d/orders"},
				"oncall":      "pd:orders",
				"language":    "go",
			},
		},
		{
			ID:   "svc-catalog",
			Name: "Catalog Indexer",
			Tags: map[string]string{"env": cfg.Environment, "tier": "data", "owner": "team-atlas"},
			Metadata: map[string]any{
				"description": "Searchable product index and attribute enrichment",
				"links":       []string{"https://runbook.demo/catalog", "https://grafana.demo/d/catalog"},
				"oncall":      "pd:catalog",
				"language":    "rust",
			},
		},
		{
			ID:   "svc-shipping",
			Name: "Shipping Tracker",
			Tags: map[string]string{"env": cfg.Environment, "tier": "backend", "owner": "team-hawkeye"},
			Metadata: map[string]any{
				"description": "Shipment status polling and carrier integrations",
				"links":       []string{"https://runbook.demo/shipping", "https://grafana.demo/d/shipping"},
				"oncall":      "pd:shipping",
				"language":    "go",
			},
		},
		{
			ID:   "svc-realtime",
			Name: "Realtime Gateway",
			Tags: map[string]string{"env": cfg.Environment, "tier": "edge", "owner": "team-nova"},
			Metadata: map[string]any{
				"description": "Websocket fanout for live updates and notifications",
				"links":       []string{"https://runbook.demo/realtime", "https://grafana.demo/d/realtime"},
				"oncall":      "pd:realtime",
				"language":    "go",
			},
		},
	}

	for i := range base {
		applyServiceFlair(&base[i])
	}
	out := make([]schema.Service, len(base))
	for i, svc := range base {
		out[i] = cloneService(svc)
	}
	return out
}

func applyServiceFlair(svc *schema.Service) {
	if svc.Metadata == nil {
		svc.Metadata = map[string]any{}
	}
	slug := serviceSlug(svc.ID)
	owner := svc.Tags["owner"]
	contacts := map[string]string{
		"slack": fmt.Sprintf("#%s", strings.TrimPrefix(owner, "team-")),
		"email": fmt.Sprintf("%s@demo", strings.TrimPrefix(owner, "team-")),
		"pager": fmt.Sprintf("pagerduty://%s", strings.TrimPrefix(owner, "team-")),
	}
	svc.Metadata["contacts"] = contacts
	svc.Metadata["dependencies"] = serviceDependencies(svc.ID)
	svc.Metadata["repositories"] = []string{fmt.Sprintf("https://github.com/opsorch/%s", slug)}
	svc.Metadata["dashboards"] = []string{fmt.Sprintf("https://grafana.demo/d/%s-overview", slug)}
	svc.Metadata["goldenMetrics"] = []string{"latency", "errors", "saturation"}
}

func serviceDependencies(id string) []string {
	switch id {
	case "svc-checkout":
		return []string{"svc-payments", "svc-order", "svc-notifications"}
	case "svc-search":
		return []string{"svc-web", "svc-catalog"}
	case "svc-web":
		return []string{"svc-realtime"}
	case "svc-payments":
		return []string{"svc-identity"}
	case "svc-notifications":
		return []string{"svc-analytics"}
	case "svc-identity":
		return []string{"svc-web"}
	case "svc-warehouse":
		return []string{"svc-analytics"}
	case "svc-recommendation":
		return []string{"svc-catalog", "svc-analytics"}
	case "svc-analytics":
		return []string{"svc-warehouse"}
	case "svc-order":
		return []string{"svc-checkout", "svc-payments"}
	case "svc-catalog":
		return []string{"svc-warehouse"}
	case "svc-shipping":
		return []string{"svc-order"}
	case "svc-realtime":
		return []string{"svc-notifications"}
	default:
		return nil
	}
}

func serviceSlug(id string) string {
	return strings.TrimPrefix(id, "svc-")
}

func cloneService(in schema.Service) schema.Service {
	return schema.Service{
		ID:       in.ID,
		Name:     in.Name,
		Tags:     mockutil.CloneStringMap(in.Tags),
		Metadata: mockutil.CloneMap(in.Metadata),
	}
}

func matchesIDs(filter []string, id string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, candidate := range filter {
		if candidate == id {
			return true
		}
	}
	return false
}

func matchesName(filter string, name string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(filter))
}

func matchesTags(filter map[string]string, tags map[string]string) bool {
	if len(filter) == 0 {
		return true
	}
	for k, v := range filter {
		if tags[k] != v {
			return false
		}
	}
	return true
}

func matchesScope(scope schema.QueryScope, svc schema.Service) bool {
	if scope == (schema.QueryScope{}) {
		return true
	}

	if scope.Service != "" && scope.Service != svc.ID {
		return false
	}
	if scope.Environment != "" && svc.Tags["env"] != scope.Environment {
		return false
	}
	if scope.Team != "" {
		owner := svc.Tags["owner"]
		if owner == "" || owner != scope.Team {
			return false
		}
	}

	return true
}
