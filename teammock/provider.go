package teammock

import (
	"context"
	"fmt"
	"strings"

	"github.com/opsorch/opsorch-core/schema"
	coreteam "github.com/opsorch/opsorch-core/team"
	"github.com/opsorch/opsorch-mock-adapters/internal/mockutil"
)

// ProviderName can be passed to OPSORCH_TEAM_PROVIDER.
const ProviderName = "mock"

// Config allows limited customization for demo scenarios.
type Config struct {
	// Organization name that will be used in team metadata.
	Organization string
}

// Provider serves a static set of demo teams and applies client-side filtering.
type Provider struct {
	cfg     Config
	teams   []schema.Team
	members map[string][]schema.TeamMember
}

// New constructs the mock team provider.
func New(cfg map[string]any) (coreteam.Provider, error) {
	parsed := parseConfig(cfg)
	teams, members := seedTeams(parsed)
	return &Provider{cfg: parsed, teams: teams, members: members}, nil
}

func init() {
	_ = coreteam.RegisterProvider(ProviderName, New)
}

// generateTeamURL creates a realistic GitHub-style team URL
func generateTeamURL(teamID string) string {
	return fmt.Sprintf("https://github.demo.com/orgs/opsorch/teams/%s", teamID)
}

// Query filters demo teams by the provided criteria.
func (p *Provider) Query(ctx context.Context, query schema.TeamQuery) ([]schema.Team, error) {
	_ = ctx

	results := make([]schema.Team, 0, len(p.teams))
	for _, team := range p.teams {
		if !matchesName(query.Name, team.Name) {
			continue
		}
		if !matchesTags(query.Tags, team.Tags) {
			continue
		}
		if !matchesScope(query.Scope, team) {
			continue
		}

		// Clone team for result
		enriched := cloneTeam(team)
		results = append(results, enriched)
	}

	return results, nil
}

// Get returns a single team by its ID.
func (p *Provider) Get(ctx context.Context, id string) (schema.Team, error) {
	_ = ctx

	for _, team := range p.teams {
		if team.ID == id {
			return cloneTeam(team), nil
		}
	}

	return schema.Team{}, fmt.Errorf("team not found: %s", id)
}

// Members returns the members of a team.
func (p *Provider) Members(ctx context.Context, teamID string) ([]schema.TeamMember, error) {
	_ = ctx

	members, exists := p.members[teamID]
	if !exists {
		return []schema.TeamMember{}, nil
	}

	// Clone members for result
	result := make([]schema.TeamMember, len(members))
	for i, member := range members {
		result[i] = cloneTeamMember(member)
	}

	return result, nil
}

func parseConfig(cfg map[string]any) Config {
	out := Config{Organization: "demo-org"}
	if v, ok := cfg["organization"].(string); ok && v != "" {
		out.Organization = v
	}
	return out
}

func seedTeams(cfg Config) ([]schema.Team, map[string][]schema.TeamMember) {
	teams := []schema.Team{
		{
			ID:   "engineering",
			Name: "Engineering",
			Tags: map[string]string{
				"type":         "department",
				"provider":     "mock",
				"organization": cfg.Organization,
			},
			Metadata: map[string]any{
				"description":   "Software engineering department",
				"slack_channel": "#engineering",
				"email":         "engineering@opsorch.com",
				"members_count": 25,
				"created_at":    "2023-01-15T10:00:00Z",
				"services":      []string{"svc-checkout", "svc-search", "svc-web", "svc-payments", "svc-identity", "svc-warehouse", "svc-recommendation", "svc-analytics", "svc-order", "svc-catalog", "svc-shipping", "svc-realtime"},
			},
		},
		// Teams that match the service ownership patterns
		{
			ID:     "team-velocity",
			Name:   "Velocity Team",
			Parent: "engineering",
			Tags: map[string]string{
				"type":         "team",
				"provider":     "mock",
				"organization": cfg.Organization,
				"focus":        "checkout-web",
			},
			Metadata: map[string]any{
				"description":   "Checkout and web frontend development",
				"slack_channel": "#velocity",
				"email":         "velocity@opsorch.com",
				"members_count": 4,
				"created_at":    "2023-02-01T14:30:00Z",
				"services":      []string{"svc-checkout", "svc-web", "svc-order"},
				"repositories":  []string{"checkout-api", "web-frontend", "order-service"},
			},
		},
		{
			ID:     "team-aurora",
			Name:   "Aurora Team",
			Parent: "engineering",
			Tags: map[string]string{
				"type":         "team",
				"provider":     "mock",
				"organization": cfg.Organization,
				"focus":        "search-discovery",
			},
			Metadata: map[string]any{
				"description":   "Search and discovery systems",
				"slack_channel": "#aurora",
				"email":         "aurora@opsorch.com",
				"members_count": 3,
				"created_at":    "2023-02-01T14:30:00Z",
				"services":      []string{"svc-search"},
				"repositories":  []string{"search-service", "discovery-engine"},
			},
		},
		{
			ID:     "team-revenue",
			Name:   "Revenue Team",
			Parent: "engineering",
			Tags: map[string]string{
				"type":         "team",
				"provider":     "mock",
				"organization": cfg.Organization,
				"focus":        "payments",
			},
			Metadata: map[string]any{
				"description":   "Payment processing and revenue systems",
				"slack_channel": "#revenue",
				"email":         "revenue@opsorch.com",
				"members_count": 3,
				"created_at":    "2023-02-15T09:00:00Z",
				"services":      []string{"svc-payments"},
				"repositories":  []string{"payments-service", "billing-api"},
			},
		},
		{
			ID:     "team-signal",
			Name:   "Signal Team",
			Parent: "engineering",
			Tags: map[string]string{
				"type":         "team",
				"provider":     "mock",
				"organization": cfg.Organization,
				"focus":        "notifications",
			},
			Metadata: map[string]any{
				"description":   "Notification and communication systems",
				"slack_channel": "#signal",
				"email":         "signal@opsorch.com",
				"members_count": 2,
				"created_at":    "2023-03-01T10:00:00Z",
				"services":      []string{"svc-notifications"},
				"repositories":  []string{"notification-service", "messaging-api"},
			},
		},
		{
			ID:     "team-guardian",
			Name:   "Guardian Team",
			Parent: "engineering",
			Tags: map[string]string{
				"type":         "team",
				"provider":     "mock",
				"organization": cfg.Organization,
				"focus":        "identity-security",
			},
			Metadata: map[string]any{
				"description":   "Identity and security platform",
				"slack_channel": "#guardian",
				"email":         "guardian@opsorch.com",
				"members_count": 3,
				"created_at":    "2023-01-10T08:00:00Z",
				"services":      []string{"svc-identity"},
				"repositories":  []string{"identity-platform", "auth-service"},
				"oncall":        "24/7",
			},
		},
		{
			ID:     "team-foundry",
			Name:   "Foundry Team",
			Parent: "engineering",
			Tags: map[string]string{
				"type":         "team",
				"provider":     "mock",
				"organization": cfg.Organization,
				"focus":        "data-platform",
			},
			Metadata: map[string]any{
				"description":   "Data platform and warehouse systems",
				"slack_channel": "#foundry",
				"email":         "foundry@opsorch.com",
				"members_count": 4,
				"created_at":    "2023-01-20T11:00:00Z",
				"services":      []string{"svc-warehouse", "svc-analytics"},
				"repositories":  []string{"data-warehouse", "etl-pipeline", "analytics-service"},
			},
		},
		{
			ID:     "team-orion",
			Name:   "Orion Team",
			Parent: "engineering",
			Tags: map[string]string{
				"type":         "team",
				"provider":     "mock",
				"organization": cfg.Organization,
				"focus":        "ml-recommendations",
			},
			Metadata: map[string]any{
				"description":   "Machine learning and recommendation systems",
				"slack_channel": "#orion",
				"email":         "orion@opsorch.com",
				"members_count": 3,
				"created_at":    "2023-03-15T10:00:00Z",
				"services":      []string{"svc-recommendation"},
				"repositories":  []string{"recommendation-engine", "ml-pipeline"},
			},
		},
		{
			ID:     "team-atlas",
			Name:   "Atlas Team",
			Parent: "engineering",
			Tags: map[string]string{
				"type":         "team",
				"provider":     "mock",
				"organization": cfg.Organization,
				"focus":        "catalog-data",
			},
			Metadata: map[string]any{
				"description":   "Product catalog and data indexing",
				"slack_channel": "#atlas",
				"email":         "atlas@opsorch.com",
				"members_count": 2,
				"created_at":    "2023-04-01T10:00:00Z",
				"services":      []string{"svc-catalog"},
				"repositories":  []string{"catalog-indexer", "product-data"},
			},
		},
		{
			ID:     "team-hawkeye",
			Name:   "Hawkeye Team",
			Parent: "engineering",
			Tags: map[string]string{
				"type":         "team",
				"provider":     "mock",
				"organization": cfg.Organization,
				"focus":        "shipping-logistics",
			},
			Metadata: map[string]any{
				"description":   "Shipping and logistics systems",
				"slack_channel": "#hawkeye",
				"email":         "hawkeye@opsorch.com",
				"members_count": 2,
				"created_at":    "2023-04-15T10:00:00Z",
				"services":      []string{"svc-shipping"},
				"repositories":  []string{"shipping-tracker", "logistics-api"},
			},
		},
		{
			ID:     "team-nova",
			Name:   "Nova Team",
			Parent: "engineering",
			Tags: map[string]string{
				"type":         "team",
				"provider":     "mock",
				"organization": cfg.Organization,
				"focus":        "realtime-edge",
			},
			Metadata: map[string]any{
				"description":   "Real-time systems and edge computing",
				"slack_channel": "#nova",
				"email":         "nova@opsorch.com",
				"members_count": 2,
				"created_at":    "2023-05-01T10:00:00Z",
				"services":      []string{"svc-realtime"},
				"repositories":  []string{"realtime-gateway", "edge-services"},
			},
		},
	}

	members := map[string][]schema.TeamMember{
		"engineering": {
			{
				ID:     "alice.johnson@opsorch.com",
				Name:   "Alice Johnson",
				Email:  "alice.johnson@opsorch.com",
				Handle: "alice.johnson",
				Role:   "owner",
				Metadata: map[string]any{
					"title":     "VP of Engineering",
					"location":  "San Francisco, CA",
					"timezone":  "America/Los_Angeles",
					"github":    "alice-johnson",
					"joined_at": "2022-06-01T09:00:00Z",
				},
			},
		},
		"team-velocity": {
			{
				ID:     "charlie.brown@opsorch.com",
				Name:   "Charlie Brown",
				Email:  "charlie.brown@opsorch.com",
				Handle: "charlie.brown",
				Role:   "owner",
				Metadata: map[string]any{
					"title":     "Senior Full-Stack Engineer",
					"location":  "New York, NY",
					"timezone":  "America/New_York",
					"github":    "charlie-brown",
					"languages": []string{"Go", "TypeScript", "React"},
					"services":  []string{"svc-checkout", "svc-web", "svc-order"},
					"joined_at": "2023-01-15T09:00:00Z",
				},
			},
			{
				ID:     "diana.prince@opsorch.com",
				Name:   "Diana Prince",
				Email:  "diana.prince@opsorch.com",
				Handle: "diana.prince",
				Role:   "member",
				Metadata: map[string]any{
					"title":     "Frontend Engineer",
					"location":  "Seattle, WA",
					"timezone":  "America/Los_Angeles",
					"github":    "diana-prince",
					"languages": []string{"TypeScript", "React", "Vue"},
					"services":  []string{"svc-web"},
					"joined_at": "2023-03-01T09:00:00Z",
				},
			},
		},
		"team-aurora": {
			{
				ID:     "eve.wilson@opsorch.com",
				Name:   "Eve Wilson",
				Email:  "eve.wilson@opsorch.com",
				Handle: "eve.wilson",
				Role:   "owner",
				Metadata: map[string]any{
					"title":     "Senior Search Engineer",
					"location":  "Portland, OR",
					"timezone":  "America/Los_Angeles",
					"github":    "eve-wilson",
					"languages": []string{"Java", "Elasticsearch", "Kafka"},
					"services":  []string{"svc-search"},
					"joined_at": "2023-02-01T09:00:00Z",
				},
			},
		},
		"team-revenue": {
			{
				ID:     "frank.miller@opsorch.com",
				Name:   "Frank Miller",
				Email:  "frank.miller@opsorch.com",
				Handle: "frank.miller",
				Role:   "owner",
				Metadata: map[string]any{
					"title":     "Senior Payments Engineer",
					"location":  "Denver, CO",
					"timezone":  "America/Denver",
					"github":    "frank-miller",
					"languages": []string{"Go", "PostgreSQL"},
					"services":  []string{"svc-payments"},
					"joined_at": "2023-04-15T09:00:00Z",
				},
			},
		},
		"team-signal": {
			{
				ID:     "grace.hopper@opsorch.com",
				Name:   "Grace Hopper",
				Email:  "grace.hopper@opsorch.com",
				Handle: "grace.hopper",
				Role:   "owner",
				Metadata: map[string]any{
					"title":     "Senior Backend Engineer",
					"location":  "Boston, MA",
					"timezone":  "America/New_York",
					"github":    "grace-hopper",
					"languages": []string{"Kotlin", "RabbitMQ", "Redis"},
					"services":  []string{"svc-notifications"},
					"joined_at": "2022-12-01T09:00:00Z",
				},
			},
		},
		"team-guardian": {
			{
				ID:     "henry.ford@opsorch.com",
				Name:   "Henry Ford",
				Email:  "henry.ford@opsorch.com",
				Handle: "henry.ford",
				Role:   "owner",
				Metadata: map[string]any{
					"title":          "Security Engineer",
					"location":       "Los Angeles, CA",
					"timezone":       "America/Los_Angeles",
					"github":         "henry-ford",
					"languages":      []string{"Go", "OAuth", "JWT"},
					"services":       []string{"svc-identity"},
					"certifications": []string{"CISSP", "CEH"},
					"joined_at":      "2023-01-01T09:00:00Z",
				},
			},
		},
		"team-foundry": {
			{
				ID:     "iris.chang@opsorch.com",
				Name:   "Iris Chang",
				Email:  "iris.chang@opsorch.com",
				Handle: "iris.chang",
				Role:   "owner",
				Metadata: map[string]any{
					"title":     "Data Platform Lead",
					"location":  "Remote",
					"timezone":  "America/Los_Angeles",
					"github":    "iris-chang",
					"languages": []string{"Python", "Spark", "Airflow"},
					"services":  []string{"svc-warehouse", "svc-analytics"},
					"joined_at": "2022-11-15T09:00:00Z",
				},
			},
		},
		"team-orion": {
			{
				ID:     "jack.sparrow@opsorch.com",
				Name:   "Jack Sparrow",
				Email:  "jack.sparrow@opsorch.com",
				Handle: "jack.sparrow",
				Role:   "owner",
				Metadata: map[string]any{
					"title":     "ML Engineer",
					"location":  "Austin, TX",
					"timezone":  "America/Chicago",
					"github":    "jack-sparrow",
					"languages": []string{"Python", "TensorFlow", "Scala"},
					"services":  []string{"svc-recommendation"},
					"joined_at": "2023-03-15T09:00:00Z",
				},
			},
		},
		"team-atlas": {
			{
				ID:     "kate.bishop@opsorch.com",
				Name:   "Kate Bishop",
				Email:  "kate.bishop@opsorch.com",
				Handle: "kate.bishop",
				Role:   "owner",
				Metadata: map[string]any{
					"title":     "Data Engineer",
					"location":  "Chicago, IL",
					"timezone":  "America/Chicago",
					"github":    "kate-bishop",
					"languages": []string{"Rust", "PostgreSQL", "Redis"},
					"services":  []string{"svc-catalog"},
					"joined_at": "2023-04-01T09:00:00Z",
				},
			},
		},
		"team-hawkeye": {
			{
				ID:     "luke.cage@opsorch.com",
				Name:   "Luke Cage",
				Email:  "luke.cage@opsorch.com",
				Handle: "luke.cage",
				Role:   "owner",
				Metadata: map[string]any{
					"title":     "Logistics Engineer",
					"location":  "Miami, FL",
					"timezone":  "America/New_York",
					"github":    "luke-cage",
					"languages": []string{"Go", "GraphQL"},
					"services":  []string{"svc-shipping"},
					"joined_at": "2023-04-15T09:00:00Z",
				},
			},
		},
		"team-nova": {
			{
				ID:     "maria.hill@opsorch.com",
				Name:   "Maria Hill",
				Email:  "maria.hill@opsorch.com",
				Handle: "maria.hill",
				Role:   "owner",
				Metadata: map[string]any{
					"title":     "Real-time Systems Engineer",
					"location":  "Phoenix, AZ",
					"timezone":  "America/Phoenix",
					"github":    "maria-hill",
					"languages": []string{"Go", "WebSocket", "Redis"},
					"services":  []string{"svc-realtime"},
					"joined_at": "2023-05-01T09:00:00Z",
				},
			},
		},
	}

	return teams, members
}

func cloneTeam(in schema.Team) schema.Team {
	// Generate URL if not already present
	url := in.URL
	if url == "" {
		url = generateTeamURL(in.ID)
	}

	return schema.Team{
		ID:       in.ID,
		Name:     in.Name,
		Parent:   in.Parent,
		URL:      url,
		Tags:     mockutil.CloneStringMap(in.Tags),
		Metadata: mockutil.CloneMap(in.Metadata),
	}
}

func cloneTeamMember(in schema.TeamMember) schema.TeamMember {
	return schema.TeamMember{
		ID:       in.ID,
		Name:     in.Name,
		Email:    in.Email,
		Handle:   in.Handle,
		Role:     in.Role,
		Metadata: mockutil.CloneMap(in.Metadata),
	}
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

func matchesScope(scope schema.QueryScope, team schema.Team) bool {
	if scope == (schema.QueryScope{}) {
		return true
	}

	if scope.Team != "" && scope.Team != team.ID {
		return false
	}

	// Teams don't typically have environment or service scope,
	// but we can check if they're mentioned in metadata
	if scope.Environment != "" {
		if env, ok := team.Metadata["environment"].(string); ok && env != scope.Environment {
			return false
		}
	}

	if scope.Service != "" {
		if services, ok := team.Metadata["services"].([]string); ok {
			found := false
			for _, svc := range services {
				if svc == scope.Service {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}
