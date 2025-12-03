package alertmock

import (
	"context"
	"testing"

	"github.com/opsorch/opsorch-core/schema"
)

func TestSeededAlertsAndGet(t *testing.T) {
	provAny, err := New(map[string]any{"source": "demo"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	list, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("expected seeded alerts, got %d", len(list))
	}
	if list[0].Service == "" {
		t.Fatalf("expected service populated, got %+v", list[0])
	}
	if list[0].Description == "" {
		t.Fatalf("expected description populated, got %+v", list[0])
	}
	if list[0].Metadata["source"] != "demo" {
		t.Fatalf("expected metadata source to match config, got %v", list[0].Metadata["source"])
	}

	got, err := prov.Get(context.Background(), "al-001")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.ID != "al-001" {
		t.Fatalf("unexpected alert fetched: %+v", got)
	}
	if _, err := prov.Get(context.Background(), "missing"); err == nil {
		t.Fatalf("expected error for missing alert")
	}
}

func TestQueryFilters(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	results, err := prov.Query(context.Background(), schema.AlertQuery{Statuses: []string{"firing"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected firing alerts, got 0")
	}
	for _, al := range results {
		if al.Status != "firing" {
			t.Fatalf("expected firing status, got %+v", al)
		}
	}

	results, err = prov.Query(context.Background(), schema.AlertQuery{Severities: []string{"critical"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected critical alerts, got 0")
	}
	for _, al := range results {
		if al.Severity != "critical" {
			t.Fatalf("expected critical severity, got %+v", al)
		}
	}

	results, err = prov.Query(context.Background(), schema.AlertQuery{Query: "latency", Limit: 1})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected limit to apply, got %+v", results)
	}

	results, err = prov.Query(context.Background(), schema.AlertQuery{Query: "stripe"})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	found := false
	for _, al := range results {
		if al.ID == "al-003" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected description search to find al-003, got %+v", results)
	}
}

func TestScopeFiltering(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	scope := schema.QueryScope{Service: "svc-checkout", Environment: "prod", Team: "team-velocity"}
	scopedList, err := prov.Query(WithScope(context.Background(), scope), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(scopedList) != 1 || scopedList[0].ID != "al-001" {
		t.Fatalf("expected scoped query to return al-001, got %+v", scopedList)
	}

	scopedList, err = prov.Query(context.Background(), schema.AlertQuery{Scope: schema.QueryScope{Service: "svc-checkout", Environment: "staging"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(scopedList) != 0 {
		t.Fatalf("expected no alerts for mismatched scope, got %+v", scopedList)
	}
}

func TestCloningPreventsMutation(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	list, _ := prov.Query(context.Background(), schema.AlertQuery{})
	if len(list) == 0 {
		t.Fatalf("expected seeded alerts")
	}
	list[0].Metadata["source"] = "mutated"
	list[0].Fields["team"] = "mutated"

	again, _ := prov.Get(context.Background(), list[0].ID)
	if again.Metadata["source"] == "mutated" || again.Fields["team"] == "mutated" {
		t.Fatalf("provider state should not be mutated: %+v", again)
	}
}
