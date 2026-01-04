package servicemock

import (
	"context"
	"strings"
	"testing"

	"github.com/opsorch/opsorch-core/schema"
)

func TestQueryFiltersAndCloning(t *testing.T) {
	provAny, err := New(map[string]any{"environment": "staging"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	// Name filter
	out, err := prov.Query(context.Background(), schema.ServiceQuery{Name: "search"})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(out) != 1 || out[0].ID != "svc-search" {
		t.Fatalf("expected search service, got %+v", out)
	}
	if out[0].Tags["env"] != "staging" {
		t.Fatalf("expected env tag to follow config, got %q", out[0].Tags["env"])
	}

	// Tag filter
	out, err = prov.Query(context.Background(), schema.ServiceQuery{Tags: map[string]string{"owner": "team-velocity"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(out) < 2 {
		t.Fatalf("expected services for owner, got %d", len(out))
	}

	// IDs filter and limit
	out, err = prov.Query(context.Background(), schema.ServiceQuery{IDs: []string{"svc-web", "svc-checkout"}, Limit: 1})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("limit should restrict size, got %d", len(out))
	}

	// Ensure clones are returned
	out[0].Tags["mutated"] = "true"
	again, err := prov.Query(context.Background(), schema.ServiceQuery{IDs: []string{"svc-web"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if _, ok := again[0].Tags["mutated"]; ok {
		t.Fatalf("returned services should be cloned, got %+v", again[0].Tags)
	}
}

func TestQueryRespectsScope(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	// Environment and team filters should be applied via scope.
	scope := schema.QueryScope{Environment: "prod", Team: "team-velocity"}
	out, err := prov.Query(context.Background(), schema.ServiceQuery{Scope: scope})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(out) < 2 {
		t.Fatalf("expected services for scope %+v, got %d", scope, len(out))
	}
	for _, svc := range out {
		if svc.Tags["env"] != scope.Environment {
			t.Fatalf("expected env %s, got %s", scope.Environment, svc.Tags["env"])
		}
		if svc.Tags["owner"] != scope.Team {
			t.Fatalf("expected owner %s, got %s", scope.Team, svc.Tags["owner"])
		}
	}

	// Service scope should narrow to a single match.
	out, err = prov.Query(context.Background(), schema.ServiceQuery{Scope: schema.QueryScope{Service: "svc-web"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(out) != 1 || out[0].ID != "svc-web" {
		t.Fatalf("expected svc-web for scoped query, got %+v", out)
	}
}
func TestServiceURLGeneration(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	services, err := prov.Query(context.Background(), schema.ServiceQuery{Limit: 5})
	if err != nil {
		t.Fatalf("failed to query services: %v", err)
	}

	for _, service := range services {
		if service.URL == "" {
			t.Errorf("service %s has empty URL", service.ID)
		}
		if !strings.HasPrefix(service.URL, "https://grafana.demo.com/d/service-") {
			t.Errorf("service %s has invalid URL format: %s", service.ID, service.URL)
		}
		if !strings.Contains(service.URL, "/service-overview") {
			t.Errorf("service %s URL should contain service-overview: %s", service.ID, service.URL)
		}
	}

	// Test specific service URL format
	if len(services) > 0 {
		service := services[0]
		expectedPrefix := "https://grafana.demo.com/d/service-"
		if !strings.HasPrefix(service.URL, expectedPrefix) {
			t.Errorf("service URL should start with %s, got %s", expectedPrefix, service.URL)
		}
		if service.ID == "svc-checkout" {
			expectedURL := "https://grafana.demo.com/d/service-checkout/service-overview"
			if service.URL != expectedURL {
				t.Errorf("checkout service has incorrect URL: got %s, want %s", service.URL, expectedURL)
			}
		}
	}
}
