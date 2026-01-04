package deploymentmock

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/opsorch/opsorch-core/deployment"
	"github.com/opsorch/opsorch-core/orcherr"
	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/internal/mockutil"
)

// ProviderName can be referenced via OPSORCH_DEPLOYMENT_PROVIDER.
const ProviderName = "mock"

// Config controls mock deployment metadata.
type Config struct {
	Source string
}

// Provider holds in-memory deployments to support demo flows.
type Provider struct {
	cfg         Config
	mu          sync.Mutex
	nextID      int
	deployments map[string]schema.Deployment
}

// New constructs the mock deployment provider with seeded deployment history.
func New(cfg map[string]any) (deployment.Provider, error) {
	parsed := parseConfig(cfg)
	p := &Provider{cfg: parsed, deployments: map[string]schema.Deployment{}}
	p.seed()
	return p, nil
}

func init() {
	_ = deployment.RegisterProvider(ProviderName, New)
}

// Query returns deployments that match the provided filters.
func (p *Provider) Query(ctx context.Context, query schema.DeploymentQuery) ([]schema.Deployment, error) {
	_ = ctx

	p.mu.Lock()
	defer p.mu.Unlock()

	// Add static scenario-themed deployments
	now := time.Now().UTC()
	scenarioDeployments := getScenarioDeployments(now)
	for _, sd := range scenarioDeployments {
		p.deployments[sd.ID] = sd
	}

	ids := sortedDeploymentIDs(p.deployments)
	results := make([]schema.Deployment, 0, len(p.deployments))
	for _, id := range ids {
		dep := p.deployments[id]
		if !matchesDeployment(query, dep) {
			continue
		}
		results = append(results, cloneDeployment(dep))
		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}
	}

	return results, nil
}

// Get returns a deployment by ID.
func (p *Provider) Get(ctx context.Context, id string) (schema.Deployment, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	dep, ok := p.deployments[id]
	if !ok {
		return schema.Deployment{}, orcherr.New("not_found", "deployment not found", nil)
	}
	return cloneDeployment(dep), nil
}

func (p *Provider) seed() {
	now := time.Now().UTC()
	seed := []schema.Deployment{
		{
			ID:          "deploy-001",
			Service:     "svc-checkout",
			Environment: "prod",
			Version:     "v2.31.3",
			Status:      "success",
			StartedAt:   now.Add(-4 * time.Hour),
			FinishedAt:  now.Add(-4 * time.Hour).Add(3*time.Minute + 45*time.Second),
			URL:         "https://github.com/company/checkout/actions/runs/12345",
			Actor:       map[string]any{"name": "alex", "type": "user"},
			Metadata: map[string]any{
				"source":        p.cfg.Source,
				"commit":        "abc123def456",
				"branch":        "main",
				"duration":      "3m45s",
				"region":        "use1",
				"rollback":      false,
				"canary":        false,
				"blue_green":    true,
				"health_checks": []string{"http", "database", "redis"},
			},
		},
		{
			ID:          "deploy-002",
			Service:     "svc-search",
			Environment: "prod",
			Version:     "v1.8.2",
			Status:      "success",
			StartedAt:   now.Add(-6 * time.Hour),
			FinishedAt:  now.Add(-6 * time.Hour).Add(2*time.Minute + 15*time.Second),
			URL:         "https://github.com/company/search/actions/runs/12346",
			Actor:       map[string]any{"name": "jamie", "type": "user"},
			Metadata: map[string]any{
				"source":        p.cfg.Source,
				"commit":        "def456ghi789",
				"branch":        "main",
				"duration":      "2m15s",
				"region":        "use1",
				"rollback":      false,
				"canary":        true,
				"blue_green":    false,
				"health_checks": []string{"http", "elasticsearch"},
			},
		},
		{
			ID:          "deploy-003",
			Service:     "svc-checkout",
			Environment: "prod",
			Version:     "v2.31.4",
			Status:      "failed",
			StartedAt:   now.Add(-2 * time.Hour),
			FinishedAt:  now.Add(-2 * time.Hour).Add(1*time.Minute + 30*time.Second),
			URL:         "https://github.com/company/checkout/actions/runs/12347",
			Actor:       map[string]any{"name": "deploy-bot", "type": "automation"},
			Metadata: map[string]any{
				"source":        p.cfg.Source,
				"commit":        "ghi789jkl012",
				"branch":        "main",
				"duration":      "1m30s",
				"region":        "use1",
				"rollback":      false,
				"canary":        false,
				"blue_green":    true,
				"error":         "health check failed: database connection timeout",
				"health_checks": []string{"http", "database", "redis"},
			},
		},
		{
			ID:          "deploy-004",
			Service:     "svc-checkout",
			Environment: "prod",
			Version:     "v2.31.3",
			Status:      "success",
			StartedAt:   now.Add(-1 * time.Hour),
			FinishedAt:  now.Add(-1 * time.Hour).Add(4*time.Minute + 12*time.Second),
			URL:         "https://github.com/company/checkout/actions/runs/12348",
			Actor:       map[string]any{"name": "deploy-bot", "type": "automation"},
			Metadata: map[string]any{
				"source":        p.cfg.Source,
				"commit":        "abc123def456",
				"branch":        "main",
				"duration":      "4m12s",
				"region":        "use1",
				"rollback":      true,
				"canary":        false,
				"blue_green":    true,
				"rollback_from": "v2.31.4",
				"health_checks": []string{"http", "database", "redis"},
			},
		},
		{
			ID:          "deploy-005",
			Service:     "svc-notifications",
			Environment: "prod",
			Version:     "v3.2.1",
			Status:      "success",
			StartedAt:   now.Add(-8 * time.Hour),
			FinishedAt:  now.Add(-8 * time.Hour).Add(5*time.Minute + 30*time.Second),
			URL:         "https://github.com/company/notifications/actions/runs/12349",
			Actor:       map[string]any{"name": "taylor", "type": "user"},
			Metadata: map[string]any{
				"source":        p.cfg.Source,
				"commit":        "jkl012mno345",
				"branch":        "main",
				"duration":      "5m30s",
				"region":        "use1",
				"rollback":      false,
				"canary":        true,
				"blue_green":    false,
				"health_checks": []string{"http", "kafka", "redis"},
			},
		},
		{
			ID:          "deploy-006",
			Service:     "svc-identity",
			Environment: "staging",
			Version:     "v1.5.0-rc1",
			Status:      "success",
			StartedAt:   now.Add(-3 * time.Hour),
			FinishedAt:  now.Add(-3 * time.Hour).Add(2*time.Minute + 45*time.Second),
			URL:         "https://github.com/company/identity/actions/runs/12350",
			Actor:       map[string]any{"name": "devon", "type": "user"},
			Metadata: map[string]any{
				"source":        p.cfg.Source,
				"commit":        "mno345pqr678",
				"branch":        "release/v1.5.0",
				"duration":      "2m45s",
				"region":        "use1",
				"rollback":      false,
				"canary":        false,
				"blue_green":    true,
				"health_checks": []string{"http", "database", "redis"},
			},
		},
		{
			ID:          "deploy-007",
			Service:     "svc-analytics",
			Environment: "prod",
			Version:     "v2.8.3",
			Status:      "running",
			StartedAt:   now.Add(-15 * time.Minute),
			FinishedAt:  time.Time{}, // Still running
			URL:         "https://github.com/company/analytics/actions/runs/12351",
			Actor:       map[string]any{"name": "maya", "type": "user"},
			Metadata: map[string]any{
				"source":        p.cfg.Source,
				"commit":        "pqr678stu901",
				"branch":        "main",
				"duration":      "ongoing",
				"region":        "use1",
				"rollback":      false,
				"canary":        true,
				"blue_green":    false,
				"progress":      "75%",
				"health_checks": []string{"http", "database", "kafka"},
			},
		},
		{
			ID:          "deploy-008",
			Service:     "svc-search",
			Environment: "staging",
			Version:     "v1.8.3-beta",
			Status:      "success",
			StartedAt:   now.Add(-5 * time.Hour),
			FinishedAt:  now.Add(-5 * time.Hour).Add(3*time.Minute + 20*time.Second),
			URL:         "https://github.com/company/search/actions/runs/12352",
			Actor:       map[string]any{"name": "riley", "type": "user"},
			Metadata: map[string]any{
				"source":        p.cfg.Source,
				"commit":        "stu901vwx234",
				"branch":        "feature/ml-ranking",
				"duration":      "3m20s",
				"region":        "use1",
				"rollback":      false,
				"canary":        false,
				"blue_green":    true,
				"health_checks": []string{"http", "elasticsearch", "ml-service"},
			},
		},
		{
			ID:          "deploy-009",
			Service:     "svc-realtime",
			Environment: "prod",
			Version:     "v0.9.8",
			Status:      "success",
			StartedAt:   now.Add(-7 * time.Hour),
			FinishedAt:  now.Add(-7 * time.Hour).Add(1*time.Minute + 55*time.Second),
			URL:         "https://github.com/company/realtime/actions/runs/12353",
			Actor:       map[string]any{"name": "samir", "type": "user"},
			Metadata: map[string]any{
				"source":        p.cfg.Source,
				"commit":        "vwx234yza567",
				"branch":        "main",
				"duration":      "1m55s",
				"region":        "use1",
				"rollback":      false,
				"canary":        false,
				"blue_green":    false,
				"rolling":       true,
				"health_checks": []string{"websocket", "redis"},
			},
		},
		{
			ID:          "deploy-010",
			Service:     "svc-warehouse",
			Environment: "prod",
			Version:     "v4.1.2",
			Status:      "success",
			StartedAt:   now.Add(-12 * time.Hour),
			FinishedAt:  now.Add(-12 * time.Hour).Add(8*time.Minute + 45*time.Second),
			URL:         "https://github.com/company/warehouse/actions/runs/12354",
			Actor:       map[string]any{"name": "morgan", "type": "user"},
			Metadata: map[string]any{
				"source":        p.cfg.Source,
				"commit":        "yza567bcd890",
				"branch":        "main",
				"duration":      "8m45s",
				"region":        "use1",
				"rollback":      false,
				"canary":        false,
				"blue_green":    false,
				"rolling":       true,
				"health_checks": []string{"http", "database", "s3"},
			},
		},
	}

	for _, dep := range seed {
		applyDeploymentFlair(&dep, now)
		p.deployments[dep.ID] = dep
		if n, err := fmt.Sscanf(dep.ID, "deploy-%d", &p.nextID); n == 1 && err == nil {
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

func applyDeploymentFlair(dep *schema.Deployment, now time.Time) {
	if dep.Metadata == nil {
		dep.Metadata = map[string]any{}
	}

	// Add deployment-specific metadata
	dep.Metadata["deployment_type"] = getDeploymentType(dep)
	dep.Metadata["estimated_impact"] = getEstimatedImpact(dep.Service)
	dep.Metadata["rollback_available"] = dep.Status == "success" || dep.Status == "failed"
	dep.Metadata["monitoring_links"] = getMonitoringLinks(dep.Service)
	dep.Metadata["related_tickets"] = getRelatedTickets(dep.Service)
	dep.Metadata["deployment_window"] = getDeploymentWindow(dep.Environment)

	// Add deployment metrics
	if dep.Status == "success" {
		dep.Metadata["success_rate"] = "100%"
		dep.Metadata["error_rate"] = "0%"
	} else if dep.Status == "failed" {
		dep.Metadata["success_rate"] = "0%"
		dep.Metadata["error_rate"] = "100%"
	} else {
		dep.Metadata["success_rate"] = "pending"
		dep.Metadata["error_rate"] = "pending"
	}
}

func getDeploymentType(dep *schema.Deployment) string {
	if rollback, ok := dep.Metadata["rollback"].(bool); ok && rollback {
		return "rollback"
	}
	if canary, ok := dep.Metadata["canary"].(bool); ok && canary {
		return "canary"
	}
	if blueGreen, ok := dep.Metadata["blue_green"].(bool); ok && blueGreen {
		return "blue_green"
	}
	if rolling, ok := dep.Metadata["rolling"].(bool); ok && rolling {
		return "rolling"
	}
	return "standard"
}

func getEstimatedImpact(service string) string {
	switch {
	case strings.Contains(service, "checkout") || strings.Contains(service, "payment"):
		return "high"
	case strings.Contains(service, "search") || strings.Contains(service, "identity"):
		return "medium"
	default:
		return "low"
	}
}

func getMonitoringLinks(service string) []string {
	key := strings.TrimPrefix(service, "svc-")
	return []string{
		fmt.Sprintf("https://grafana.demo/d/%s-deployment", key),
		fmt.Sprintf("https://datadog.demo/dashboard/%s-deploy", key),
		fmt.Sprintf("https://newrelic.demo/deployment/%s", key),
	}
}

func getRelatedTickets(service string) []string {
	switch {
	case strings.Contains(service, "checkout"):
		return []string{"TCK-001", "TCK-003", "TCK-009"}
	case strings.Contains(service, "search"):
		return []string{"TCK-002"}
	case strings.Contains(service, "notifications"):
		return []string{"TCK-004"}
	case strings.Contains(service, "identity"):
		return []string{"TCK-005"}
	case strings.Contains(service, "warehouse"):
		return []string{"TCK-006"}
	case strings.Contains(service, "analytics"):
		return []string{"TCK-008"}
	case strings.Contains(service, "realtime"):
		return []string{"TCK-010"}
	default:
		return []string{}
	}
}

func getDeploymentWindow(environment string) string {
	if environment == "prod" {
		return "business_hours"
	}
	return "anytime"
}

func cloneDeployment(in schema.Deployment) schema.Deployment {
	cloned := schema.Deployment{
		ID:          in.ID,
		Service:     in.Service,
		Environment: in.Environment,
		Version:     in.Version,
		Status:      in.Status,
		StartedAt:   in.StartedAt,
		FinishedAt:  in.FinishedAt,
		URL:         generateDeploymentURL(in),
		Actor:       mockutil.CloneMap(in.Actor),
		Fields:      mockutil.CloneMap(in.Fields),
		Metadata:    mockutil.CloneMap(in.Metadata),
	}
	return cloned
}

// generateDeploymentURL creates a GitHub Actions-style URL for the deployment
func generateDeploymentURL(dep schema.Deployment) string {
	serviceName := strings.TrimPrefix(dep.Service, "svc-")

	// Extract run ID from existing URL if present, otherwise generate one
	runID := "12345"
	if dep.URL != "" && strings.Contains(dep.URL, "/actions/runs/") {
		parts := strings.Split(dep.URL, "/actions/runs/")
		if len(parts) > 1 {
			runID = parts[1]
		}
	}

	return fmt.Sprintf("https://github.com/company/%s/actions/runs/%s", serviceName, runID)
}

func sortedDeploymentIDs(deployments map[string]schema.Deployment) []string {
	ids := make([]string, 0, len(deployments))
	for id := range deployments {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func matchesDeployment(query schema.DeploymentQuery, dep schema.Deployment) bool {
	if !matchesQuery(query.Query, dep) {
		return false
	}
	if !matchesScope(query.Scope, dep) {
		return false
	}
	if len(query.Statuses) > 0 && !matchesStatuses(query.Statuses, dep.Status) {
		return false
	}
	if len(query.Versions) > 0 && !matchesVersions(query.Versions, dep.Version) {
		return false
	}
	if len(query.Metadata) > 0 && !matchesMetadata(query.Metadata, dep.Metadata) {
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

func matchesVersions(filter []string, version string) bool {
	for _, v := range filter {
		if v == version {
			return true
		}
	}
	return false
}

func matchesScope(scope schema.QueryScope, dep schema.Deployment) bool {
	if scope == (schema.QueryScope{}) {
		return true
	}

	if scope.Service != "" && dep.Service != scope.Service {
		return false
	}
	if scope.Environment != "" && dep.Environment != scope.Environment {
		return false
	}
	// Team matching would need to be added to deployment metadata if needed

	return true
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

func matchesQuery(filter string, dep schema.Deployment) bool {
	if filter == "" {
		return true
	}

	needle := strings.ToLower(filter)
	fields := []string{dep.ID, dep.Service, dep.Environment, dep.Version, dep.Status}
	for _, field := range fields {
		if field != "" && strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}

	// Search in actor
	if dep.Actor != nil {
		if name, ok := dep.Actor["name"].(string); ok {
			if strings.Contains(strings.ToLower(name), needle) {
				return true
			}
		}
	}

	// Search in metadata
	if dep.Metadata != nil {
		if commit, ok := dep.Metadata["commit"].(string); ok {
			if strings.Contains(strings.ToLower(commit), needle) {
				return true
			}
		}
		if branch, ok := dep.Metadata["branch"].(string); ok {
			if strings.Contains(strings.ToLower(branch), needle) {
				return true
			}
		}
	}

	return false
}

// getScenarioDeployments returns static scenario-themed deployments
func getScenarioDeployments(now time.Time) []schema.Deployment {
	return []schema.Deployment{
		{
			ID:          "deploy-scenario-001",
			Service:     "svc-checkout",
			Environment: "prod",
			Version:     "v2.31.2",
			Status:      "success",
			StartedAt:   now.Add(-30 * time.Minute),
			FinishedAt:  now.Add(-25 * time.Minute),
			URL:         "https://github.com/company/checkout/actions/runs/scenario-001",
			Actor:       map[string]any{"name": "deploy-bot", "type": "automation"},
			Metadata: map[string]any{
				"source":         "mock",
				"commit":         "scenario001abc",
				"branch":         "main",
				"duration":       "5m12s",
				"region":         "use1",
				"rollback":       false,
				"canary":         false,
				"blue_green":     true,
				"scenario_id":    "scenario-001",
				"scenario_name":  "SLO Budget Exhaustion",
				"scenario_stage": "pre-incident",
				"is_scenario":    true,
				"health_checks":  []string{"http", "database", "redis"},
				"success_rate":   "100%",
				"error_rate":     "0%",
			},
		},
		{
			ID:          "deploy-scenario-002",
			Service:     "svc-search",
			Environment: "prod",
			Version:     "v1.8.1",
			Status:      "success",
			StartedAt:   now.Add(-45 * time.Minute),
			FinishedAt:  now.Add(-40 * time.Minute),
			URL:         "https://github.com/company/search/actions/runs/scenario-002",
			Actor:       map[string]any{"name": "jamie", "type": "user"},
			Metadata: map[string]any{
				"source":         "mock",
				"commit":         "scenario002def",
				"branch":         "main",
				"duration":       "3m30s",
				"region":         "use1",
				"rollback":       false,
				"canary":         true,
				"blue_green":     false,
				"scenario_id":    "scenario-002",
				"scenario_name":  "Cascading Database Failure",
				"scenario_stage": "pre-incident",
				"is_scenario":    true,
				"health_checks":  []string{"http", "elasticsearch"},
				"success_rate":   "100%",
				"error_rate":     "0%",
			},
		},
		{
			ID:          "deploy-scenario-003",
			Service:     "svc-checkout",
			Environment: "prod",
			Version:     "v2.31.4",
			Status:      "failed",
			StartedAt:   now.Add(-20 * time.Minute),
			FinishedAt:  now.Add(-15 * time.Minute),
			URL:         "https://github.com/company/checkout/actions/runs/scenario-003",
			Actor:       map[string]any{"name": "deploy-bot", "type": "automation"},
			Metadata: map[string]any{
				"source":         "mock",
				"commit":         "scenario003ghi",
				"branch":         "main",
				"duration":       "2m45s",
				"region":         "use1",
				"rollback":       false,
				"canary":         false,
				"blue_green":     true,
				"scenario_id":    "scenario-003",
				"scenario_name":  "Deployment Rollback",
				"scenario_stage": "incident-trigger",
				"is_scenario":    true,
				"error":          "health check failed: elevated error rate detected",
				"health_checks":  []string{"http", "database", "redis"},
				"success_rate":   "0%",
				"error_rate":     "100%",
			},
		},
		{
			ID:          "deploy-scenario-004",
			Service:     "svc-checkout",
			Environment: "prod",
			Version:     "v2.31.3",
			Status:      "success",
			StartedAt:   now.Add(-10 * time.Minute),
			FinishedAt:  now.Add(-5 * time.Minute),
			URL:         "https://github.com/company/checkout/actions/runs/scenario-004",
			Actor:       map[string]any{"name": "deploy-bot", "type": "automation"},
			Metadata: map[string]any{
				"source":         "mock",
				"commit":         "scenario001abc",
				"branch":         "main",
				"duration":       "4m20s",
				"region":         "use1",
				"rollback":       true,
				"canary":         false,
				"blue_green":     true,
				"rollback_from":  "v2.31.4",
				"scenario_id":    "scenario-003",
				"scenario_name":  "Deployment Rollback",
				"scenario_stage": "mitigation",
				"is_scenario":    true,
				"health_checks":  []string{"http", "database", "redis"},
				"success_rate":   "100%",
				"error_rate":     "0%",
			},
		},
		{
			ID:          "deploy-scenario-005",
			Service:     "svc-search",
			Environment: "prod",
			Version:     "v1.8.2",
			Status:      "running",
			StartedAt:   now.Add(-3 * time.Minute),
			FinishedAt:  time.Time{}, // Still running
			URL:         "https://github.com/company/search/actions/runs/scenario-005",
			Actor:       map[string]any{"name": "kim", "type": "user"},
			Metadata: map[string]any{
				"source":         "mock",
				"commit":         "scenario005jkl",
				"branch":         "main",
				"duration":       "ongoing",
				"region":         "use1",
				"rollback":       false,
				"canary":         true,
				"blue_green":     false,
				"progress":       "60%",
				"scenario_id":    "scenario-005",
				"scenario_name":  "Autoscaling Lag",
				"scenario_stage": "active",
				"is_scenario":    true,
				"health_checks":  []string{"http", "elasticsearch"},
				"success_rate":   "pending",
				"error_rate":     "pending",
			},
		},
		{
			ID:          "deploy-scenario-006",
			Service:     "svc-checkout",
			Environment: "prod",
			Version:     "v2.31.1",
			Status:      "success",
			StartedAt:   now.Add(-35 * time.Minute),
			FinishedAt:  now.Add(-30 * time.Minute),
			URL:         "https://github.com/company/checkout/actions/runs/scenario-006",
			Actor:       map[string]any{"name": "samir", "type": "user"},
			Metadata: map[string]any{
				"source":         "mock",
				"commit":         "scenario006mno",
				"branch":         "main",
				"duration":       "4m15s",
				"region":         "use1",
				"rollback":       false,
				"canary":         false,
				"blue_green":     true,
				"scenario_id":    "scenario-006",
				"scenario_name":  "Circuit Breaker Cascade",
				"scenario_stage": "pre-incident",
				"is_scenario":    true,
				"health_checks":  []string{"http", "database", "redis"},
				"success_rate":   "100%",
				"error_rate":     "0%",
			},
		},
	}
}

var _ deployment.Provider = (*Provider)(nil)
