package logmock

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

func TestQueryGeneratesEntries(t *testing.T) {
	provAny, err := New(map[string]any{"defaultLimit": 5, "source": "demo"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)
	entries, err := prov.Query(context.Background(), schema.LogQuery{Expression: &schema.LogExpression{Search: "checkout error"}, Start: start, End: end, Limit: 4})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected entries to respect limit, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Timestamp.Before(start) || e.Timestamp.After(end) {
			t.Fatalf("timestamp outside range: %v", e.Timestamp)
		}
	}
	if entries[0].Severity != "error" {
		t.Fatalf("expected severity inference, got %s", entries[0].Severity)
	}
	if entries[0].Service != "checkout" {
		t.Fatalf("expected service field to be set, got %s", entries[0].Service)
	}
	if entries[0].Labels["service"] != "checkout" {
		t.Fatalf("expected service detection in labels, got %v", entries[0].Labels["service"])
	}
	if entries[0].Metadata["source"] != "demo" {
		t.Fatalf("expected metadata source, got %v", entries[0].Metadata["source"])
	}
}

func TestDefaultValues(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	entries, err := prov.Query(context.Background(), schema.LogQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected default entries")
	}
}

func TestQueryRespectsScope(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	scope := schema.QueryScope{Service: "svc-checkout", Environment: "staging", Team: "team-velocity"}
	entries, err := prov.Query(context.Background(), schema.LogQuery{Limit: 2, Scope: scope})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected entries for scoped query")
	}
	if entries[0].Service != scope.Service {
		t.Fatalf("expected service field %s, got %s", scope.Service, entries[0].Service)
	}
	if got := entries[0].Labels["service"]; got != scope.Service {
		t.Fatalf("expected service label %s, got %v", scope.Service, got)
	}
	if got := entries[0].Labels["env"]; got != scope.Environment {
		t.Fatalf("expected env label %s, got %v", scope.Environment, got)
	}
	if got := entries[0].Labels["team"]; got != scope.Team {
		t.Fatalf("expected team label %s, got %v", scope.Team, got)
	}
	if scopeMeta, ok := entries[0].Metadata["scope"].(schema.QueryScope); !ok || scopeMeta != scope {
		t.Fatalf("expected scope metadata %+v, got %+v", scope, entries[0].Metadata["scope"])
	}
}

func TestLogsIncludeAlertContext(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-20 * time.Minute)
	entries, err := prov.Query(context.Background(), schema.LogQuery{
		Scope: schema.QueryScope{Service: "svc-checkout"},
		Start: start,
		End:   end,
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected entries for checkout scope")
	}
	found := false
	for _, entry := range entries {
		if _, ok := entry.Fields["alerts"]; ok {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected at least one log entry to include alert context")
	}
}

func TestScenarioLogsStaticSeeding(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)
	entries, err := prov.Query(context.Background(), schema.LogQuery{
		Start: start,
		End:   end,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Verify scenario logs are present
	scenarioCount := 0
	for _, entry := range entries {
		if isScenario, ok := entry.Fields["is_scenario"].(bool); ok && isScenario {
			scenarioCount++
			// Verify scenario metadata
			if entry.Fields["scenario_id"] == nil {
				t.Errorf("scenario log missing scenario_id field")
			}
			if entry.Fields["scenario_name"] == nil {
				t.Errorf("scenario log missing scenario_name field")
			}
			if entry.Fields["scenario_stage"] == nil {
				t.Errorf("scenario log missing scenario_stage field")
			}
			// Verify metadata
			if entry.Metadata["scenario_id"] == nil {
				t.Errorf("scenario log missing scenario_id in metadata")
			}
		}
	}

	if scenarioCount == 0 {
		t.Fatalf("expected scenario logs to be present, got 0")
	}
	if scenarioCount != 6 {
		t.Errorf("expected 6 scenario logs, got %d", scenarioCount)
	}
}

func TestScenarioLogVariety(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)
	entries, err := prov.Query(context.Background(), schema.LogQuery{
		Start: start,
		End:   end,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Collect scenario logs
	scenarioLogs := []schema.LogEntry{}
	for _, entry := range entries {
		if isScenario, ok := entry.Fields["is_scenario"].(bool); ok && isScenario {
			scenarioLogs = append(scenarioLogs, entry)
		}
	}

	if len(scenarioLogs) == 0 {
		t.Fatalf("expected scenario logs")
	}

	// Check severity variety
	severities := make(map[string]bool)
	for _, log := range scenarioLogs {
		severities[log.Severity] = true
	}
	if !severities["error"] {
		t.Errorf("expected error severity in scenario logs")
	}
	if !severities["warn"] {
		t.Errorf("expected warn severity in scenario logs")
	}

	// Check service variety
	services := make(map[string]bool)
	for _, log := range scenarioLogs {
		services[log.Service] = true
	}
	if len(services) < 2 {
		t.Errorf("expected multiple services in scenario logs, got %d", len(services))
	}
}

func TestScenarioErrorLogsHaveStackTraces(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)
	entries, err := prov.Query(context.Background(), schema.LogQuery{
		Start: start,
		End:   end,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Find scenario error logs
	errorCount := 0
	stackTraceCount := 0
	errorCodeCount := 0
	for _, entry := range entries {
		if isScenario, ok := entry.Fields["is_scenario"].(bool); ok && isScenario {
			if entry.Severity == "error" {
				errorCount++
				if stackTrace, ok := entry.Fields["stackTrace"].(string); ok && stackTrace != "" {
					stackTraceCount++
				}
				if errorCode, ok := entry.Fields["errorCode"].(string); ok && errorCode != "" {
					errorCodeCount++
				}
			}
		}
	}

	if errorCount == 0 {
		t.Fatalf("expected scenario error logs")
	}
	if stackTraceCount != errorCount {
		t.Errorf("expected all %d error logs to have stack traces, got %d", errorCount, stackTraceCount)
	}
	if errorCodeCount != errorCount {
		t.Errorf("expected all %d error logs to have error codes, got %d", errorCount, errorCodeCount)
	}
}

func TestQueryScenarioLogsByService(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)
	entries, err := prov.Query(context.Background(), schema.LogQuery{
		Scope: schema.QueryScope{Service: "svc-checkout"},
		Start: start,
		End:   end,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Verify all logs are for svc-checkout
	scenarioCount := 0
	for _, entry := range entries {
		if entry.Service != "svc-checkout" {
			t.Errorf("expected service svc-checkout, got %s", entry.Service)
		}
		if isScenario, ok := entry.Fields["is_scenario"].(bool); ok && isScenario {
			scenarioCount++
		}
	}

	if scenarioCount == 0 {
		t.Errorf("expected scenario logs for svc-checkout")
	}
}

func TestQueryRespectsFilters(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)

	// Test service filter
	entries, err := prov.Query(context.Background(), schema.LogQuery{
		Expression: &schema.LogExpression{
			Filters: []schema.LogFilter{
				{Field: "service", Operator: "=", Value: "svc-cache"},
			},
		},
		Start: start,
		End:   end,
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(entries) == 0 {
		t.Fatalf("expected entries with service filter")
	}

	// Verify all entries match the filter
	for _, entry := range entries {
		if entry.Service != "svc-cache" {
			t.Errorf("expected service=svc-cache, got %s", entry.Service)
		}
	}

	// Test severity filter
	entries, err = prov.Query(context.Background(), schema.LogQuery{
		Expression: &schema.LogExpression{
			Filters: []schema.LogFilter{
				{Field: "severity", Operator: "=", Value: "error"},
			},
		},
		Start: start,
		End:   end,
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	for _, entry := range entries {
		if entry.Severity != "error" {
			t.Errorf("expected severity=error, got %s", entry.Severity)
		}
	}

	// Test multiple filters (AND logic)
	entries, err = prov.Query(context.Background(), schema.LogQuery{
		Expression: &schema.LogExpression{
			Filters: []schema.LogFilter{
				{Field: "service", Operator: "=", Value: "svc-checkout"},
				{Field: "severity", Operator: "=", Value: "error"},
			},
		},
		Start: start,
		End:   end,
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	for _, entry := range entries {
		if entry.Service != "svc-checkout" {
			t.Errorf("expected service=svc-checkout, got %s", entry.Service)
		}
		if entry.Severity != "error" {
			t.Errorf("expected severity=error, got %s", entry.Severity)
		}
	}
}

func TestCacheServiceSpecificLogs(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)

	entries, err := prov.Query(context.Background(), schema.LogQuery{
		Scope: schema.QueryScope{Service: "svc-cache"},
		Start: start,
		End:   end,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(entries) == 0 {
		t.Fatalf("expected entries for svc-cache")
	}

	// Verify cache-specific paths and components
	for _, entry := range entries {
		if entry.Service != "svc-cache" {
			t.Errorf("expected service=svc-cache, got %s", entry.Service)
		}

		// Check for cache-specific paths
		if path, ok := entry.Fields["path"].(string); ok {
			if !strings.Contains(path, "/cache/") {
				t.Errorf("expected cache-specific path, got %s", path)
			}
		}

		// Check for cache-specific components
		if component, ok := entry.Fields["component"].(string); ok {
			validComponents := []string{"redis-primary", "redis-replica", "eviction-manager"}
			found := false
			for _, valid := range validComponents {
				if component == valid {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected cache-specific component, got %s", component)
			}
		}

		// Check message format
		if !strings.Contains(entry.Message, "Cache") && !strings.Contains(entry.Message, "redis") {
			t.Errorf("expected cache-specific message, got %s", entry.Message)
		}
	}
}

func TestSearchInfluencesLogContent(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)

	// Test 1: Search for "recommendation quality drop rollout"
	entries, err := prov.Query(context.Background(), schema.LogQuery{
		Expression: &schema.LogExpression{
			Search: "Recommendation quality drop after rollout",
		},
		Scope: schema.QueryScope{Service: "svc-recommendation"},
		Start: start,
		End:   end,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(entries) == 0 {
		t.Fatalf("expected entries for recommendation search")
	}

	// Verify service is correct
	for _, entry := range entries {
		if entry.Service != "svc-recommendation" {
			t.Errorf("expected service=svc-recommendation, got %s", entry.Service)
		}
	}

	// Verify messages contain relevant context
	foundRelevant := false
	for _, entry := range entries {
		msg := strings.ToLower(entry.Message)
		if strings.Contains(msg, "quality") || strings.Contains(msg, "rollout") ||
			strings.Contains(msg, "recommendation") || strings.Contains(msg, "degrad") {
			foundRelevant = true
			break
		}
	}
	if !foundRelevant {
		t.Errorf("expected at least one log message to contain search-relevant terms")
	}

	// Test 2: Error search should bias toward error severity
	entries, err = prov.Query(context.Background(), schema.LogQuery{
		Expression: &schema.LogExpression{
			Search: "error failure crash",
		},
		Scope: schema.QueryScope{Service: "svc-checkout"},
		Start: start,
		End:   end,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	errorCount := 0
	for _, entry := range entries {
		if entry.Severity == "error" {
			errorCount++
		}
	}

	// Should have more errors than other severities due to search bias
	if errorCount < len(entries)/2 {
		t.Errorf("expected error bias in results, got %d errors out of %d entries", errorCount, len(entries))
	}
}

func TestUsesConsistentTeamMapping(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)

	// Test that svc-recommendation uses team-orion (from mockutil maps)
	entries, err := prov.Query(context.Background(), schema.LogQuery{
		Scope: schema.QueryScope{Service: "svc-recommendation"},
		Start: start,
		End:   end,
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(entries) == 0 {
		t.Fatalf("expected entries for svc-recommendation")
	}

	for _, entry := range entries {
		if team, ok := entry.Labels["team"]; ok {
			if team != "team-orion" {
				t.Errorf("expected team=team-orion for svc-recommendation, got %s", team)
			}
		}
	}
}
func TestLogURLGeneration(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	prov := provAny.(*Provider)

	now := time.Now()
	logs, err := prov.Query(context.Background(), schema.LogQuery{
		Start: now.Add(-1 * time.Hour),
		End:   now,
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("failed to query logs: %v", err)
	}

	for _, log := range logs {
		if log.URL == "" {
			t.Errorf("log entry has empty URL")
		}
		if !strings.HasPrefix(log.URL, "https://kibana.demo.com/app/logs/stream?") {
			t.Errorf("log entry has invalid URL format: %s", log.URL)
		}
		if !strings.Contains(log.URL, "logId=") {
			t.Errorf("log URL should contain logId parameter: %s", log.URL)
		}
		if !strings.Contains(log.URL, "timestamp=") {
			t.Errorf("log URL should contain timestamp parameter: %s", log.URL)
		}
	}
}
