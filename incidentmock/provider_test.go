package incidentmock

import (
	"context"
	"testing"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

func TestListAndGetSeededIncidents(t *testing.T) {
	provAny, err := New(map[string]any{"source": "demo", "defaultSeverity": "sev1"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	list, err := prov.Query(context.Background(), schema.IncidentQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("expected seeded incidents, got %d", len(list))
	}
	if list[0].Service == "" {
		t.Fatalf("expected service to be populated, got %+v", list[0])
	}
	if list[0].Metadata["source"] != "demo" {
		t.Fatalf("expected metadata source to match config, got %v", list[0].Metadata["source"])
	}
	if list[0].Description == "" {
		t.Fatalf("expected description to be set on seeded incidents")
	}

	got, err := prov.Get(context.Background(), "inc-001")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.ID != "inc-001" {
		t.Fatalf("unexpected incident fetched: %+v", got)
	}
	if got.Description == "" {
		t.Fatalf("expected seeded incident to have description: %+v", got)
	}
	if _, err := prov.Get(context.Background(), "missing"); err == nil {
		t.Fatalf("expected error for missing incident")
	}
}

func TestCreateUpdateAndTimeline(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	input := schema.CreateIncidentInput{Title: "New", Description: "New incident impacting web", Status: "open", Service: "svc-web", Fields: map[string]any{"environment": "prod"}}
	created, err := prov.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID == "" || created.Severity == "" {
		t.Fatalf("expected ID and severity to be set: %+v", created)
	}
	if created.Service != input.Service {
		t.Fatalf("expected service %s, got %s", input.Service, created.Service)
	}
	if svc, _ := created.Fields["service"].(string); svc != input.Service {
		t.Fatalf("expected service field to mirror service value, got %v", svc)
	}
	if created.Metadata["source"] != "mock" {
		t.Fatalf("expected default source metadata, got %+v", created.Metadata)
	}
	if created.Description != input.Description {
		t.Fatalf("expected description to be copied, got %+v", created)
	}

	now := time.Now().UTC()
	updateTitle := "Updated"
	updateSeverity := "sev1"
	updateService := "svc-api"
	updateDescription := "Updated incident details"
	updated, err := prov.Update(context.Background(), created.ID, schema.UpdateIncidentInput{Title: &updateTitle, Description: &updateDescription, Severity: &updateSeverity, Service: &updateService})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Title != updateTitle || updated.Severity != updateSeverity {
		t.Fatalf("fields not updated: %+v", updated)
	}
	if updated.Service != updateService {
		t.Fatalf("expected service updated to %s, got %s", updateService, updated.Service)
	}
	if updated.Description != updateDescription {
		t.Fatalf("expected description updated, got %+v", updated)
	}
	if !updated.UpdatedAt.After(now.Add(-1 * time.Second)) {
		t.Fatalf("UpdatedAt should be bumped")
	}

	// Timeline
	appendErr := prov.AppendTimeline(context.Background(), created.ID, schema.TimelineAppendInput{Body: "note", Kind: "note"})
	if appendErr != nil {
		t.Fatalf("AppendTimeline returned error: %v", appendErr)
	}
	entries, err := prov.GetTimeline(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTimeline returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].IncidentID != created.ID {
		t.Fatalf("unexpected timeline: %+v", entries)
	}

	if err := prov.AppendTimeline(context.Background(), "missing", schema.TimelineAppendInput{}); err == nil {
		t.Fatalf("expected error for missing incident timeline append")
	}
}

func TestCloningProtectsState(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	list, _ := prov.Query(context.Background(), schema.IncidentQuery{})
	list[0].Metadata["source"] = "mutated"
	list[0].Fields["team"] = "mutated"

	again, _ := prov.Get(context.Background(), list[0].ID)
	if again.Metadata["source"] == "mutated" || again.Fields["team"] == "mutated" {
		t.Fatalf("provider state should not be mutated by callers: %+v", again)
	}
}

func TestQueryRespectsScope(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	scope := schema.QueryScope{Service: "svc-checkout", Environment: "prod", Team: "team-velocity"}
	list, err := prov.Query(WithScope(context.Background(), scope), schema.IncidentQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(list) != 1 || list[0].ID != "inc-001" {
		t.Fatalf("expected scoped list to return inc-001, got %+v", list)
	}

	list, err = prov.Query(context.Background(), schema.IncidentQuery{Scope: schema.QueryScope{Service: "svc-search", Environment: "staging"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no incidents for mismatched scope, got %+v", list)
	}
}

func TestQueryFiltersStatusAndSearch(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	results, err := prov.Query(context.Background(), schema.IncidentQuery{Statuses: []string{"monitoring"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected monitoring filter to return results, got %+v", results)
	}
	for _, inc := range results {
		if inc.Status != "monitoring" {
			t.Fatalf("expected monitoring status, got %+v", inc)
		}
	}
	found := false
	for _, inc := range results {
		if inc.ID == "inc-002" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected monitoring results to include inc-002, got %+v", results)
	}

	results, err = prov.Query(context.Background(), schema.IncidentQuery{Query: "checkout", Limit: 1})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected limit to apply to search results, got %+v", results)
	}
	results, err = prov.Query(context.Background(), schema.IncidentQuery{Query: "timeouts"})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	found = false
	for _, inc := range results {
		if inc.ID == "inc-001" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected description search to find inc-001, got %+v", results)
	}
}
