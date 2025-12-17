package teammock

import (
	"context"
	"testing"

	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-core/team"
)

func TestTeamMockProvider(t *testing.T) {
	// Test provider registration
	t.Run("Provider Registration", func(t *testing.T) {
		constructor, ok := team.LookupProvider("mock")
		if !ok {
			t.Fatal("mock team provider not registered")
		}

		if constructor == nil {
			t.Error("constructor is nil")
		}
	})

	// Test provider creation
	t.Run("Provider Creation", func(t *testing.T) {
		provider, err := New(map[string]any{})
		if err != nil {
			t.Fatalf("failed to create provider: %v", err)
		}

		if provider == nil {
			t.Error("provider is nil")
		}
	})

	// Test configuration parsing
	t.Run("Configuration Parsing", func(t *testing.T) {
		tests := []struct {
			name     string
			config   map[string]any
			expected string
		}{
			{
				name:     "default organization",
				config:   map[string]any{},
				expected: "demo-org",
			},
			{
				name:     "custom organization",
				config:   map[string]any{"organization": "custom-org"},
				expected: "custom-org",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				parsed := parseConfig(tt.config)
				if parsed.Organization != tt.expected {
					t.Errorf("expected organization %q, got %q", tt.expected, parsed.Organization)
				}
			})
		}
	})
}

func TestTeamMockProviderQueries(t *testing.T) {
	provider, err := New(map[string]any{"organization": "test-org"})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()

	// Test Query all teams
	t.Run("Query All Teams", func(t *testing.T) {
		teams, err := provider.Query(ctx, schema.TeamQuery{})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}

		if len(teams) == 0 {
			t.Error("expected teams to be returned")
		}

		// Verify all teams have the correct organization tag
		for _, team := range teams {
			if team.Tags["organization"] != "test-org" {
				t.Errorf("expected organization tag to be 'test-org', got %q", team.Tags["organization"])
			}
		}
	})

	// Test Query by name
	t.Run("Query By Name", func(t *testing.T) {
		teams, err := provider.Query(ctx, schema.TeamQuery{Name: "velocity"})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}

		if len(teams) != 1 {
			t.Errorf("expected 1 team, got %d", len(teams))
		}

		if len(teams) > 0 && teams[0].ID != "team-velocity" {
			t.Errorf("expected team-velocity, got %s", teams[0].ID)
		}
	})

	// Test Query by tags
	t.Run("Query By Tags", func(t *testing.T) {
		teams, err := provider.Query(ctx, schema.TeamQuery{
			Tags: map[string]string{"type": "department"},
		})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}

		if len(teams) != 1 { // only engineering now
			t.Errorf("expected 1 department, got %d", len(teams))
		}

		for _, team := range teams {
			if team.Tags["type"] != "department" {
				t.Errorf("expected department type, got %s", team.Tags["type"])
			}
		}
	})

	// Test Get team
	t.Run("Get Team", func(t *testing.T) {
		team, err := provider.Get(ctx, "engineering")
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}

		if team.ID != "engineering" {
			t.Errorf("expected engineering team, got %s", team.ID)
		}

		if team.Name != "Engineering" {
			t.Errorf("expected Engineering name, got %s", team.Name)
		}
	})

	// Test Get non-existent team
	t.Run("Get Non-Existent Team", func(t *testing.T) {
		_, err := provider.Get(ctx, "nonexistent")
		if err == nil {
			t.Error("expected error for non-existent team")
		}
	})

	// Test Members
	t.Run("Get Team Members", func(t *testing.T) {
		members, err := provider.Members(ctx, "team-velocity")
		if err != nil {
			t.Fatalf("members failed: %v", err)
		}

		if len(members) != 2 {
			t.Errorf("expected 2 members, got %d", len(members))
		}

		// Verify member structure
		for _, member := range members {
			if member.ID == "" {
				t.Error("member ID should not be empty")
			}
			if member.Email == "" {
				t.Error("member email should not be empty")
			}
			if member.Role == "" {
				t.Error("member role should not be empty")
			}
		}
	})

	// Test Members for non-existent team
	t.Run("Get Members Non-Existent Team", func(t *testing.T) {
		members, err := provider.Members(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("members failed: %v", err)
		}

		if len(members) != 0 {
			t.Errorf("expected 0 members, got %d", len(members))
		}
	})

	// Test hierarchical teams
	t.Run("Hierarchical Teams", func(t *testing.T) {
		// Get parent team
		parent, err := provider.Get(ctx, "engineering")
		if err != nil {
			t.Fatalf("get parent failed: %v", err)
		}

		// Get child team
		child, err := provider.Get(ctx, "team-velocity")
		if err != nil {
			t.Fatalf("get child failed: %v", err)
		}

		if child.Parent != parent.ID {
			t.Errorf("expected child parent to be %s, got %s", parent.ID, child.Parent)
		}
	})
}

func TestTeamMockProviderFiltering(t *testing.T) {
	provider, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()

	// Test name filtering
	t.Run("Name Filtering", func(t *testing.T) {
		tests := []struct {
			name     string
			filter   string
			expected int
		}{
			{"exact match", "Velocity Team", 1},
			{"partial match", "team", 10}, // all the team-* teams
			{"case insensitive", "VELOCITY", 1},
			{"no match", "nonexistent", 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				teams, err := provider.Query(ctx, schema.TeamQuery{Name: tt.filter})
				if err != nil {
					t.Fatalf("query failed: %v", err)
				}

				if len(teams) != tt.expected {
					t.Errorf("expected %d teams, got %d", tt.expected, len(teams))
				}
			})
		}
	})

	// Test tag filtering
	t.Run("Tag Filtering", func(t *testing.T) {
		tests := []struct {
			name     string
			tags     map[string]string
			expected int
		}{
			{"type team", map[string]string{"type": "team"}, 10},
			{"type department", map[string]string{"type": "department"}, 1},
			{"focus checkout-web", map[string]string{"focus": "checkout-web"}, 1},
			{"multiple tags", map[string]string{"type": "team", "focus": "payments"}, 1},
			{"no match", map[string]string{"type": "nonexistent"}, 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				teams, err := provider.Query(ctx, schema.TeamQuery{Tags: tt.tags})
				if err != nil {
					t.Fatalf("query failed: %v", err)
				}

				if len(teams) != tt.expected {
					t.Errorf("expected %d teams, got %d", tt.expected, len(teams))
				}
			})
		}
	})
}
