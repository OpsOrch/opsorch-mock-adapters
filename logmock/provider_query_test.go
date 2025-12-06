package logmock

import (
	"context"
	"testing"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

func TestQuery_WithSearchExpression_ReturnsMatchingLogs(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	// Test the exact query from the issue
	now := time.Now().UTC()
	query := schema.LogQuery{
		Expression: &schema.LogExpression{
			Search: "redis OR \"Connection refused\" OR \"OOM command not allowed\" OR \"MISCONF Redis\"",
			Filters: []schema.LogFilter{
				{Field: "env", Operator: "=", Value: "prod"},
			},
			SeverityIn: []string{"error", "warn"},
		},
		Start: now.Add(-1 * time.Hour),
		End:   now,
		Scope: schema.QueryScope{
			Environment: "prod",
		},
		Limit: 20,
	}

	logs, err := prov.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected logs to be returned, got none")
	}

	// Verify all logs match the search terms
	for _, log := range logs {
		matched := false
		searchText := log.Message + " " + log.Service
		if containsAny(searchText, []string{"redis", "connection refused", "oom command not allowed", "misconf redis"}) {
			matched = true
		}
		// Also check fields
		for _, v := range log.Fields {
			if s, ok := v.(string); ok {
				if containsAny(s, []string{"redis", "connection refused", "oom", "misconf"}) {
					matched = true
					break
				}
			}
		}
		if !matched {
			t.Errorf("Log doesn't contain any search terms: %s", log.Message)
		}

		// Verify environment filter
		if env, ok := log.Labels["env"]; !ok || env != "prod" {
			t.Errorf("Log doesn't have env=prod in labels")
		}

		// Verify severity filter
		if log.Severity != "error" && log.Severity != "warn" {
			t.Errorf("Log has unexpected severity: %s", log.Severity)
		}
	}

	// Verify limit is respected
	if len(logs) > query.Limit {
		t.Errorf("Expected at most %d logs, got %d", query.Limit, len(logs))
	}
}

func TestQuery_WithQuotedPhrase_ReturnsMatchingLogs(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	now := time.Now().UTC()
	query := schema.LogQuery{
		Expression: &schema.LogExpression{
			Search: "\"Connection refused\"",
		},
		Start: now.Add(-1 * time.Hour),
		End:   now,
		Limit: 10,
	}

	logs, err := prov.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected logs to be returned, got none")
	}

	// At least one log should contain the exact phrase
	foundPhrase := false
	for _, log := range logs {
		if containsIgnoreCase(log.Message, "connection refused") {
			foundPhrase = true
			break
		}
	}

	if !foundPhrase {
		t.Error("No log contains the quoted phrase 'Connection refused'")
	}
}

func TestQuery_WithServiceFilter_ReturnsServiceLogs(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	now := time.Now().UTC()
	query := schema.LogQuery{
		Expression: &schema.LogExpression{
			Search: "error",
			Filters: []schema.LogFilter{
				{Field: "service", Operator: "=", Value: "svc-cache"},
			},
		},
		Start: now.Add(-1 * time.Hour),
		End:   now,
		Limit: 10,
	}

	logs, err := prov.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected logs to be returned, got none")
	}

	// All logs should be for svc-cache
	for _, log := range logs {
		if log.Service != "svc-cache" {
			t.Errorf("Expected service=svc-cache, got %s", log.Service)
		}
	}
}

func TestQuery_WithSeverityFilter_ReturnsFilteredLogs(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	now := time.Now().UTC()
	query := schema.LogQuery{
		Expression: &schema.LogExpression{
			Search:     "redis",
			SeverityIn: []string{"error"},
		},
		Start: now.Add(-1 * time.Hour),
		End:   now,
		Limit: 10,
	}

	logs, err := prov.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected logs to be returned, got none")
	}

	// All logs should have error severity
	for _, log := range logs {
		if log.Severity != "error" {
			t.Errorf("Expected severity=error, got %s", log.Severity)
		}
	}
}

func TestQuery_WithMultipleFilters_ReturnsMatchingLogs(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	now := time.Now().UTC()
	query := schema.LogQuery{
		Expression: &schema.LogExpression{
			Search: "cache",
			Filters: []schema.LogFilter{
				{Field: "env", Operator: "=", Value: "prod"},
			},
			SeverityIn: []string{"error", "warn"},
		},
		Start: now.Add(-1 * time.Hour),
		End:   now,
		Scope: schema.QueryScope{
			Environment: "prod",
		},
		Limit: 10,
	}

	logs, err := prov.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected logs to be returned, got none")
	}

	// Verify all filters are applied
	for _, log := range logs {
		// Check search term
		matched := containsIgnoreCase(log.Message, "cache") || containsIgnoreCase(log.Service, "cache")
		if !matched {
			for _, v := range log.Fields {
				if s, ok := v.(string); ok && containsIgnoreCase(s, "cache") {
					matched = true
					break
				}
			}
		}
		if !matched {
			t.Errorf("Log doesn't contain 'cache': %s", log.Message)
		}

		// Check environment
		if env, ok := log.Labels["env"]; !ok || env != "prod" {
			t.Errorf("Log doesn't have env=prod")
		}

		// Check severity
		if log.Severity != "error" && log.Severity != "warn" {
			t.Errorf("Log has unexpected severity: %s", log.Severity)
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
