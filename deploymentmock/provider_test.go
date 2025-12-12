package deploymentmock

import (
	"context"
	"testing"

	"github.com/opsorch/opsorch-core/schema"
)

func TestProvider_Query(t *testing.T) {
	prov, err := New(map[string]any{"source": "test"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name    string
		query   schema.DeploymentQuery
		wantLen int
	}{
		{
			name:    "empty query returns all",
			query:   schema.DeploymentQuery{},
			wantLen: 16, // 10 seed + 6 scenario deployments
		},
		{
			name: "filter by service",
			query: schema.DeploymentQuery{
				Scope: schema.QueryScope{Service: "svc-checkout"},
			},
			wantLen: 7, // checkout deployments in seed + scenarios
		},
		{
			name: "filter by environment via scope",
			query: schema.DeploymentQuery{
				Scope: schema.QueryScope{Environment: "prod"},
			},
			wantLen: 14, // prod deployments
		},
		{
			name: "filter by status",
			query: schema.DeploymentQuery{
				Statuses: []string{"success"},
			},
			wantLen: 12, // successful deployments
		},
		{
			name: "limit results",
			query: schema.DeploymentQuery{
				Limit: 5,
			},
			wantLen: 5,
		},
		{
			name: "text search",
			query: schema.DeploymentQuery{
				Query: "checkout",
			},
			wantLen: 7, // deployments matching "checkout"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := prov.Query(context.Background(), tt.query)
			if err != nil {
				t.Errorf("Query() error = %v", err)
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("Query() got %d deployments, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestProvider_Get(t *testing.T) {
	prov, err := New(map[string]any{"source": "test"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing deployment",
			id:      "deploy-001",
			wantErr: false,
		},
		{
			name:    "non-existing deployment",
			id:      "deploy-999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := prov.Get(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.ID != tt.id {
				t.Errorf("Get() got ID = %v, want %v", got.ID, tt.id)
			}
		})
	}
}

func TestProvider_QueryFiltering(t *testing.T) {
	prov, err := New(map[string]any{"source": "test"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test multiple status filtering
	deployments, err := prov.Query(context.Background(), schema.DeploymentQuery{
		Statuses: []string{"success", "failed"},
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	for _, dep := range deployments {
		if dep.Status != "success" && dep.Status != "failed" {
			t.Errorf("Expected status 'success' or 'failed', got %s", dep.Status)
		}
	}

	// Test environment filtering via scope
	prodDeployments, err := prov.Query(context.Background(), schema.DeploymentQuery{
		Scope: schema.QueryScope{Environment: "prod"},
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	for _, dep := range prodDeployments {
		if dep.Environment != "prod" {
			t.Errorf("Expected environment 'prod', got %s", dep.Environment)
		}
	}

	// Test metadata filtering
	rollbackDeployments, err := prov.Query(context.Background(), schema.DeploymentQuery{
		Metadata: map[string]any{"rollback": true},
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	for _, dep := range rollbackDeployments {
		if rollback, ok := dep.Metadata["rollback"].(bool); !ok || !rollback {
			t.Errorf("Expected rollback=true in metadata, got %v", dep.Metadata["rollback"])
		}
	}
}

func TestProvider_DeploymentMetadata(t *testing.T) {
	prov, err := New(map[string]any{"source": "test"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	deployment, err := prov.Get(context.Background(), "deploy-001")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Check that deployment has expected metadata fields
	expectedFields := []string{
		"source", "commit", "branch", "duration", "region",
		"deployment_type", "estimated_impact", "rollback_available",
		"monitoring_links", "related_tickets", "deployment_window",
		"success_rate", "error_rate",
	}

	for _, field := range expectedFields {
		if _, ok := deployment.Metadata[field]; !ok {
			t.Errorf("Expected metadata field %s not found", field)
		}
	}

	// Check that monitoring links are properly formatted
	if links, ok := deployment.Metadata["monitoring_links"].([]string); ok {
		if len(links) == 0 {
			t.Error("Expected monitoring_links to be non-empty")
		}
		for _, link := range links {
			if link == "" {
				t.Error("Expected monitoring link to be non-empty")
			}
		}
	} else {
		t.Error("Expected monitoring_links to be []string")
	}
}

func TestProvider_ScenarioDeployments(t *testing.T) {
	prov, err := New(map[string]any{"source": "test"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Query for scenario deployments
	deployments, err := prov.Query(context.Background(), schema.DeploymentQuery{
		Metadata: map[string]any{"is_scenario": true},
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(deployments) == 0 {
		t.Error("Expected to find scenario deployments")
	}

	// Check that scenario deployments have scenario metadata
	for _, dep := range deployments {
		if scenarioID, ok := dep.Metadata["scenario_id"].(string); !ok || scenarioID == "" {
			t.Errorf("Expected scenario_id in deployment %s", dep.ID)
		}
		if scenarioName, ok := dep.Metadata["scenario_name"].(string); !ok || scenarioName == "" {
			t.Errorf("Expected scenario_name in deployment %s", dep.ID)
		}
		if scenarioStage, ok := dep.Metadata["scenario_stage"].(string); !ok || scenarioStage == "" {
			t.Errorf("Expected scenario_stage in deployment %s", dep.ID)
		}
	}
}
