package logmock

import (
	"context"
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
	entries, err := prov.Query(context.Background(), schema.LogQuery{Query: "checkout error", Start: start, End: end, Limit: 4})
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
