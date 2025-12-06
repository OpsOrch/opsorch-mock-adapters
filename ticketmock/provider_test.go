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
	// Should match TCK-002 and possibly scenario tickets with "search" in title
	found := false
	for _, r := range results {
		if r.ID == "TCK-002" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected search query to include TCK-002, got %+v", results)
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
	// Should match TCK-002 and possibly scenario tickets for svc-search
	found := false
	for _, r := range results {
		if r.ID == "TCK-002" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected scoped query to include TCK-002, got %+v", results)
	}
	// Verify all results match the scope
	for _, r := range results {
		if svc, ok := r.Fields["service"].(string); ok && svc != scope.Service {
			t.Errorf("expected service %s, got %s", scope.Service, svc)
		}
		if env, ok := r.Fields["environment"].(string); ok && env != scope.Environment {
			t.Errorf("expected environment %s, got %s", scope.Environment, env)
		}
		if team, ok := r.Fields["team"].(string); ok && team != scope.Team {
			t.Errorf("expected team %s, got %s", scope.Team, team)
		}
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

func TestScenarioTicketsStaticSeeding(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	results, err := prov.Query(context.Background(), schema.TicketQuery{Limit: 100})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Verify scenario tickets are present
	scenarioCount := 0
	for _, ticket := range results {
		if isScenario, ok := ticket.Fields["is_scenario"].(bool); ok && isScenario {
			scenarioCount++
			// Verify scenario metadata
			if ticket.Fields["scenario_id"] == nil {
				t.Errorf("scenario ticket missing scenario_id field")
			}
			if ticket.Fields["scenario_name"] == nil {
				t.Errorf("scenario ticket missing scenario_name field")
			}
			if ticket.Fields["scenario_stage"] == nil {
				t.Errorf("scenario ticket missing scenario_stage field")
			}
			// Verify metadata
			if ticket.Metadata["scenario_id"] == nil {
				t.Errorf("scenario ticket missing scenario_id in metadata")
			}
		}
	}

	if scenarioCount == 0 {
		t.Fatalf("expected scenario tickets to be present, got 0")
	}
	if scenarioCount != 6 {
		t.Errorf("expected 6 scenario tickets, got %d", scenarioCount)
	}
}

func TestScenarioTicketVariety(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	results, err := prov.Query(context.Background(), schema.TicketQuery{Limit: 100})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Collect scenario tickets
	scenarioTickets := []schema.Ticket{}
	for _, ticket := range results {
		if isScenario, ok := ticket.Fields["is_scenario"].(bool); ok && isScenario {
			scenarioTickets = append(scenarioTickets, ticket)
		}
	}

	if len(scenarioTickets) == 0 {
		t.Fatalf("expected scenario tickets")
	}

	// Check status variety
	statuses := make(map[string]bool)
	for _, ticket := range scenarioTickets {
		statuses[ticket.Status] = true
	}
	if !statuses["todo"] {
		t.Errorf("expected todo status in scenario tickets")
	}
	if !statuses["in_progress"] {
		t.Errorf("expected in_progress status in scenario tickets")
	}
	if !statuses["in_review"] {
		t.Errorf("expected in_review status in scenario tickets")
	}

	// Check priority variety
	priorities := make(map[string]bool)
	for _, ticket := range scenarioTickets {
		if priority, ok := ticket.Fields["priority"].(string); ok {
			priorities[priority] = true
		}
	}
	if !priorities["P0"] {
		t.Errorf("expected P0 priority in scenario tickets")
	}
	if !priorities["P1"] {
		t.Errorf("expected P1 priority in scenario tickets")
	}
}

func TestScenarioTicketIncidentRelationships(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	results, err := prov.Query(context.Background(), schema.TicketQuery{Limit: 100})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Find scenario tickets with incident relationships
	incidentRelationshipCount := 0
	for _, ticket := range results {
		if isScenario, ok := ticket.Fields["is_scenario"].(bool); ok && isScenario {
			if incidentID, ok := ticket.Fields["incident_id"].(string); ok && incidentID != "" {
				incidentRelationshipCount++
				// Verify metadata also has the relationship
				if relatedIncidents, ok := ticket.Metadata["relatedIncidents"].([]string); ok {
					if len(relatedIncidents) == 0 {
						t.Errorf("ticket %s has incident_id but no relatedIncidents in metadata", ticket.ID)
					}
				}
			}
		}
	}

	if incidentRelationshipCount == 0 {
		t.Errorf("expected scenario tickets with incident relationships")
	}
}

func TestQueryScenarioTicketsByTitle(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	results, err := prov.Query(context.Background(), schema.TicketQuery{
		Query: "SLO Budget",
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Verify we found the SLO Budget scenario ticket
	found := false
	for _, ticket := range results {
		if isScenario, ok := ticket.Fields["is_scenario"].(bool); ok && isScenario {
			if ticket.Fields["scenario_id"] == "scenario-001" {
				found = true
				break
			}
		}
	}

	if !found {
		t.Errorf("expected to find SLO Budget scenario ticket by title search")
	}
}

func TestQueryScenarioTicketsByService(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	results, err := prov.Query(context.Background(), schema.TicketQuery{
		Scope: schema.QueryScope{Service: "svc-checkout"},
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Verify all tickets are for svc-checkout
	scenarioCount := 0
	for _, ticket := range results {
		if service, ok := ticket.Fields["service"].(string); ok {
			if service != "svc-checkout" {
				t.Errorf("expected service svc-checkout, got %s", service)
			}
		}
		if isScenario, ok := ticket.Fields["is_scenario"].(bool); ok && isScenario {
			scenarioCount++
		}
	}

	if scenarioCount == 0 {
		t.Errorf("expected scenario tickets for svc-checkout")
	}
}
