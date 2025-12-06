package alertmock

import (
	"context"
	"testing"

	"github.com/opsorch/opsorch-core/schema"
)

func TestQuery_WithSearchTerms_ReturnsMatchingAlerts(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	// Test the exact query from the issue
	query := schema.AlertQuery{
		Query:      "redis OR cache OR \"cache_hits_total\"",
		Statuses:   []string{"open", "investigating"},
		Severities: []string{"sev1", "sev2", "sev3"},
		Scope: schema.QueryScope{
			Environment: "prod",
		},
		Limit: 5,
	}

	alerts, err := prov.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(alerts) == 0 {
		t.Fatal("Expected alerts to be returned, got none")
	}

	// Verify all alerts match the search terms
	for _, alert := range alerts {
		matched := false
		searchText := alert.Title + " " + alert.Description + " " + alert.Service
		if containsAny(searchText, []string{"redis", "cache", "cache_hits_total"}) {
			matched = true
		}
		if !matched {
			t.Errorf("Alert %s doesn't contain any search terms: %s", alert.ID, alert.Title)
		}

		// Verify environment filter
		if env, ok := alert.Fields["environment"].(string); !ok || env != "prod" {
			t.Errorf("Alert %s doesn't have environment=prod", alert.ID)
		}
	}

	// Verify limit is respected
	if len(alerts) > query.Limit {
		t.Errorf("Expected at most %d alerts, got %d", query.Limit, len(alerts))
	}
}

func TestQuery_WithQuotedPhrase_ReturnsMatchingAlerts(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	query := schema.AlertQuery{
		Query: "\"Connection refused\"",
		Limit: 3,
	}

	alerts, err := prov.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(alerts) == 0 {
		t.Fatal("Expected alerts to be returned, got none")
	}

	// At least one alert should contain the exact phrase
	foundPhrase := false
	for _, alert := range alerts {
		searchText := alert.Title + " " + alert.Description
		if containsIgnoreCase(searchText, "connection refused") {
			foundPhrase = true
			break
		}
	}

	if !foundPhrase {
		t.Error("No alert contains the quoted phrase 'Connection refused'")
	}
}

func TestQuery_WithServiceScope_ReturnsServiceAlerts(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	query := schema.AlertQuery{
		Query: "error",
		Scope: schema.QueryScope{
			Service: "svc-cache",
		},
		Limit: 3,
	}

	alerts, err := prov.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(alerts) == 0 {
		t.Fatal("Expected alerts to be returned, got none")
	}

	// All alerts should be for svc-cache
	for _, alert := range alerts {
		if alert.Service != "svc-cache" {
			t.Errorf("Expected service=svc-cache, got %s", alert.Service)
		}
	}
}

func TestQuery_WithSeverityFilter_ReturnsFilteredAlerts(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	query := schema.AlertQuery{
		Query:      "redis",
		Severities: []string{"critical"},
		Limit:      3,
	}

	alerts, err := prov.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(alerts) == 0 {
		t.Fatal("Expected alerts to be returned, got none")
	}

	// All alerts should have critical severity
	for _, alert := range alerts {
		if alert.Severity != "critical" {
			t.Errorf("Expected severity=critical, got %s", alert.Severity)
		}
	}
}

func TestQuery_WithStatusFilter_ReturnsFilteredAlerts(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	query := schema.AlertQuery{
		Query:    "cache",
		Statuses: []string{"firing"},
		Limit:    3,
	}

	alerts, err := prov.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(alerts) == 0 {
		t.Fatal("Expected alerts to be returned, got none")
	}

	// All alerts should have firing status
	for _, alert := range alerts {
		if alert.Status != "firing" {
			t.Errorf("Expected status=firing, got %s", alert.Status)
		}
	}
}

// Helper functions
func containsAny(text string, terms []string) bool {
	lowerText := toLower(text)
	for _, term := range terms {
		if containsIgnoreCase(lowerText, term) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(text, substr string) bool {
	return contains(toLower(text), toLower(substr))
}

func contains(text, substr string) bool {
	return len(text) >= len(substr) && (text == substr || indexOf(text, substr) >= 0)
}

func indexOf(text, substr string) int {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
