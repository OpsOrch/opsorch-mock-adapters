package ticketmock

import (
	"context"
	"testing"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

func TestGetSeededTickets(t *testing.T) {
	provAny, err := New(map[string]any{"source": "demo"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	tk, err := prov.Get(context.Background(), "TCK-001")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if tk.Metadata["source"] != "demo" {
		t.Fatalf("expected metadata source set, got %v", tk.Metadata["source"])
	}
	if _, err := prov.Get(context.Background(), "missing"); err == nil {
		t.Fatalf("expected error for missing ticket")
	}
}

func TestCreateAndUpdate(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	created, err := prov.Create(context.Background(), schema.CreateTicketInput{Title: "new", Description: "desc"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID == "" || created.Key == "" {
		t.Fatalf("expected id/key set, got %+v", created)
	}
	if created.Status != "todo" {
		t.Fatalf("expected default status, got %s", created.Status)
	}

	assignees := []string{"sam"}
	status := "in_progress"
	updated, err := prov.Update(context.Background(), created.ID, schema.UpdateTicketInput{Status: &status, Assignees: &assignees})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Status != status || len(updated.Assignees) != 1 || updated.Assignees[0] != "sam" {
		t.Fatalf("update did not apply: %+v", updated)
	}
	if !updated.UpdatedAt.After(created.CreatedAt.Add(-1 * time.Second)) {
		t.Fatalf("UpdatedAt should move forward")
	}

	// ensure cloning
	updated.Assignees[0] = "mutated"
	again, _ := prov.Get(context.Background(), created.ID)
	if again.Assignees[0] == "mutated" {
		t.Fatalf("stored assignees mutated")
	}

	if _, err := prov.Update(context.Background(), "missing", schema.UpdateTicketInput{}); err == nil {
		t.Fatalf("expected error updating missing ticket")
	}
}

func TestQueryFilters(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)
	ctx := context.Background()

	results, err := prov.Query(ctx, schema.TicketQuery{Query: "search"})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "TCK-002" {
		t.Fatalf("expected search query to match TCK-002, got %+v", results)
	}

	results, err = prov.Query(ctx, schema.TicketQuery{Query: "tck-001", Statuses: []string{"in_progress"}, Reporter: "sre-bot"})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "TCK-001" {
		t.Fatalf("expected filtered query to match TCK-001, got %+v", results)
	}

	results, err = prov.Query(ctx, schema.TicketQuery{Query: "missing"})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no matches for missing query, got %+v", results)
	}
}

func TestQueryRespectsScope(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)
	ctx := context.Background()

	scope := schema.QueryScope{Service: "svc-search", Environment: "prod", Team: "team-aurora"}
	results, err := prov.Query(ctx, schema.TicketQuery{Scope: scope})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "TCK-002" {
		t.Fatalf("expected scoped query to match TCK-002, got %+v", results)
	}

	// Ensure mismatched scope excludes tickets
	results, err = prov.Query(ctx, schema.TicketQuery{Scope: schema.QueryScope{Service: "svc-search", Environment: "staging"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no matches for mismatched scope, got %+v", results)
	}
}
