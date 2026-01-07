package orchestrationmock

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

func TestNew_DefaultConfig(t *testing.T) {
	p, err := New(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("provider is nil")
	}

	// Verify default source is "mock"
	provider := p.(*Provider)
	if provider.cfg.Source != "mock" {
		t.Errorf("got source %q, want %q", provider.cfg.Source, "mock")
	}
}

func TestNew_CustomConfig(t *testing.T) {
	cfg := map[string]any{
		"source": "custom-source",
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	provider := p.(*Provider)
	if provider.cfg.Source != "custom-source" {
		t.Errorf("got source %q, want %q", provider.cfg.Source, "custom-source")
	}
}

func TestQueryPlans_Empty(t *testing.T) {
	p, _ := New(nil)
	plans, err := p.QueryPlans(context.Background(), schema.OrchestrationPlanQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return all seeded plans (5 playbooks + 7 runbooks + 3 release checklists + 6 complex flows = 21)
	if len(plans) != 21 {
		t.Errorf("got %d plans, want 21", len(plans))
	}
}

func TestQueryPlans_QueryFilter(t *testing.T) {
	p, _ := New(nil)

	tests := []struct {
		name     string
		query    string
		wantMin  int
		validate func([]schema.OrchestrationPlan) bool
	}{
		{
			name:    "database query",
			query:   "database",
			wantMin: 2,
			validate: func(plans []schema.OrchestrationPlan) bool {
				for _, plan := range plans {
					text := strings.ToLower(plan.Title + " " + plan.Description)
					if !strings.Contains(text, "database") {
						return false
					}
				}
				return true
			},
		},
		{
			name:    "failover query",
			query:   "failover",
			wantMin: 1,
			validate: func(plans []schema.OrchestrationPlan) bool {
				for _, plan := range plans {
					text := strings.ToLower(plan.Title + " " + plan.Description)
					if !strings.Contains(text, "failover") {
						return false
					}
				}
				return true
			},
		},
		{
			name:    "nonexistent query",
			query:   "xyznonexistent",
			wantMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plans, err := p.QueryPlans(context.Background(), schema.OrchestrationPlanQuery{Query: tt.query})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(plans) < tt.wantMin {
				t.Errorf("got %d plans, want at least %d", len(plans), tt.wantMin)
			}
			if tt.validate != nil && !tt.validate(plans) {
				t.Error("validation failed")
			}
		})
	}
}

func TestQueryPlans_TagFilter(t *testing.T) {
	p, _ := New(nil)

	tests := []struct {
		name    string
		tags    map[string]string
		wantMin int
	}{
		{
			name:    "type=playbook",
			tags:    map[string]string{"type": "playbook"},
			wantMin: 5,
		},
		{
			name:    "type=runbook",
			tags:    map[string]string{"type": "runbook"},
			wantMin: 7,
		},
		{
			name:    "type=release-checklist",
			tags:    map[string]string{"type": "release-checklist"},
			wantMin: 3,
		},
		{
			name:    "service=svc-database",
			tags:    map[string]string{"service": "svc-database"},
			wantMin: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plans, err := p.QueryPlans(context.Background(), schema.OrchestrationPlanQuery{Tags: tt.tags})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(plans) < tt.wantMin {
				t.Errorf("got %d plans, want at least %d", len(plans), tt.wantMin)
			}
			// Verify all returned plans have matching tags
			for _, plan := range plans {
				for k, v := range tt.tags {
					if plan.Tags[k] != v {
						t.Errorf("plan %s tag %s=%s, want %s", plan.ID, k, plan.Tags[k], v)
					}
				}
			}
		})
	}
}

func TestQueryPlans_Limit(t *testing.T) {
	p, _ := New(nil)

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"limit 1", 1, 1},
		{"limit 3", 3, 3},
		{"limit 5", 5, 5},
		{"limit 0 (no limit)", 0, 21},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plans, err := p.QueryPlans(context.Background(), schema.OrchestrationPlanQuery{Limit: tt.limit})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(plans) != tt.want {
				t.Errorf("got %d plans, want %d", len(plans), tt.want)
			}
		})
	}
}

func TestGetPlan_Valid(t *testing.T) {
	p, _ := New(nil)

	plan, err := p.GetPlan(context.Background(), "plan-playbook-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan == nil {
		t.Fatal("plan is nil")
	}
	if plan.ID != "plan-playbook-001" {
		t.Errorf("got ID %q, want %q", plan.ID, "plan-playbook-001")
	}
	if len(plan.Steps) == 0 {
		t.Error("plan has no steps")
	}
}

func TestGetPlan_NotFound(t *testing.T) {
	p, _ := New(nil)

	plan, err := p.GetPlan(context.Background(), "nonexistent-plan")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if plan != nil {
		t.Fatal("plan should be nil")
	}
	if !strings.Contains(err.Error(), "not_found") {
		t.Errorf("error should contain 'not_found', got %v", err)
	}
}

func TestQueryRuns_Empty(t *testing.T) {
	p, _ := New(nil)
	runs, err := p.QueryRuns(context.Background(), schema.OrchestrationRunQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return all seeded runs (4)
	if len(runs) != 4 {
		t.Errorf("got %d runs, want 4", len(runs))
	}
}

func TestQueryRuns_StatusFilter(t *testing.T) {
	p, _ := New(nil)

	tests := []struct {
		name     string
		statuses []string
		wantMin  int
	}{
		{
			name:     "status=blocked",
			statuses: []string{"blocked"},
			wantMin:  1,
		},
		{
			name:     "status=running",
			statuses: []string{"running"},
			wantMin:  2,
		},
		{
			name:     "status=completed",
			statuses: []string{"completed"},
			wantMin:  1,
		},
		{
			name:     "multiple statuses",
			statuses: []string{"running", "blocked"},
			wantMin:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runs, err := p.QueryRuns(context.Background(), schema.OrchestrationRunQuery{Statuses: tt.statuses})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(runs) < tt.wantMin {
				t.Errorf("got %d runs, want at least %d", len(runs), tt.wantMin)
			}
			// Verify all returned runs have matching status
			for _, run := range runs {
				found := false
				for _, status := range tt.statuses {
					if run.Status == status {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("run %s status %s not in filter", run.ID, run.Status)
				}
			}
		})
	}
}

func TestQueryRuns_PlanIDFilter(t *testing.T) {
	p, _ := New(nil)

	runs, err := p.QueryRuns(context.Background(), schema.OrchestrationRunQuery{
		PlanIDs: []string{"plan-playbook-001"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return 2 runs for this plan (run-001 and run-scenario-001)
	if len(runs) != 2 {
		t.Errorf("got %d runs, want 2", len(runs))
	}

	for _, run := range runs {
		if run.PlanID != "plan-playbook-001" {
			t.Errorf("run %s planID %s, want plan-playbook-001", run.ID, run.PlanID)
		}
	}
}

func TestQueryRuns_Limit(t *testing.T) {
	p, _ := New(nil)

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"limit 1", 1, 1},
		{"limit 2", 2, 2},
		{"limit 0 (no limit)", 0, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runs, err := p.QueryRuns(context.Background(), schema.OrchestrationRunQuery{Limit: tt.limit})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(runs) != tt.want {
				t.Errorf("got %d runs, want %d", len(runs), tt.want)
			}
		})
	}
}

func TestGetRun_Valid(t *testing.T) {
	p, _ := New(nil)

	run, err := p.GetRun(context.Background(), "run-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run == nil {
		t.Fatal("run is nil")
	}
	if run.ID != "run-001" {
		t.Errorf("got ID %q, want %q", run.ID, "run-001")
	}
	if len(run.Steps) == 0 {
		t.Error("run has no steps")
	}
}

func TestGetRun_NotFound(t *testing.T) {
	p, _ := New(nil)

	run, err := p.GetRun(context.Background(), "nonexistent-run")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if run != nil {
		t.Fatal("run should be nil")
	}
	if !strings.Contains(err.Error(), "not_found") {
		t.Errorf("error should contain 'not_found', got %v", err)
	}
}

func TestStartRun_Valid(t *testing.T) {
	p, _ := New(nil)

	run, err := p.StartRun(context.Background(), "plan-playbook-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run == nil {
		t.Fatal("run is nil")
	}

	// Verify run properties
	if run.Status != "created" {
		t.Errorf("got status %q, want %q", run.Status, "created")
	}
	if run.PlanID != "plan-playbook-001" {
		t.Errorf("got planID %q, want %q", run.PlanID, "plan-playbook-001")
	}

	// Verify step initialization
	if len(run.Steps) != 6 {
		t.Errorf("got %d steps, want 6", len(run.Steps))
	}

	// First step should be ready (no dependencies)
	if run.Steps[0].Status != "ready" {
		t.Errorf("step 0 status %q, want %q", run.Steps[0].Status, "ready")
	}

	// Second step should be pending (has dependency)
	if run.Steps[1].Status != "pending" {
		t.Errorf("step 1 status %q, want %q", run.Steps[1].Status, "pending")
	}
}

func TestStartRun_NotFound(t *testing.T) {
	p, _ := New(nil)

	run, err := p.StartRun(context.Background(), "nonexistent-plan")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if run != nil {
		t.Fatal("run should be nil")
	}
	if !strings.Contains(err.Error(), "not_found") {
		t.Errorf("error should contain 'not_found', got %v", err)
	}
}

func TestCompleteStep_Valid(t *testing.T) {
	p, _ := New(nil)

	// Start a run
	run, _ := p.StartRun(context.Background(), "plan-playbook-001")

	// Complete the first step
	err := p.CompleteStep(context.Background(), run.ID, "step-1", "test-user", "test note")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify step is marked as succeeded
	updatedRun, _ := p.GetRun(context.Background(), run.ID)
	if updatedRun.Steps[0].Status != "succeeded" {
		t.Errorf("step 0 status %q, want %q", updatedRun.Steps[0].Status, "succeeded")
	}
	if updatedRun.Steps[0].Actor != "test-user" {
		t.Errorf("step 0 actor %q, want %q", updatedRun.Steps[0].Actor, "test-user")
	}
	if updatedRun.Steps[0].Note != "test note" {
		t.Errorf("step 0 note %q, want %q", updatedRun.Steps[0].Note, "test note")
	}

	// Verify dependent step is now ready
	if updatedRun.Steps[1].Status != "ready" {
		t.Errorf("step 1 status %q, want %q (should be ready after dependency complete)", updatedRun.Steps[1].Status, "ready")
	}
}

func TestCompleteStep_RunNotFound(t *testing.T) {
	p, _ := New(nil)

	err := p.CompleteStep(context.Background(), "nonexistent-run", "step-1", "user", "note")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not_found") {
		t.Errorf("error should contain 'not_found', got %v", err)
	}
}

func TestCompleteStep_StepNotFound(t *testing.T) {
	p, _ := New(nil)

	run, _ := p.StartRun(context.Background(), "plan-playbook-001")

	err := p.CompleteStep(context.Background(), run.ID, "nonexistent-step", "user", "note")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not_found") {
		t.Errorf("error should contain 'not_found', got %v", err)
	}
}

func TestSourceMetadata(t *testing.T) {
	cfg := map[string]any{"source": "test-source"}
	p, _ := New(cfg)

	// Check plans have source metadata
	plans, _ := p.QueryPlans(context.Background(), schema.OrchestrationPlanQuery{})
	for _, plan := range plans {
		if plan.Metadata["source"] != "test-source" {
			t.Errorf("plan %s source %v, want test-source", plan.ID, plan.Metadata["source"])
		}
	}

	// Check runs have source metadata
	runs, _ := p.QueryRuns(context.Background(), schema.OrchestrationRunQuery{})
	for _, run := range runs {
		if run.Metadata["source"] != "test-source" {
			t.Errorf("run %s source %v, want test-source", run.ID, run.Metadata["source"])
		}
	}
}

func TestSeed_Playbooks(t *testing.T) {
	p, _ := New(nil)

	plans, _ := p.QueryPlans(context.Background(), schema.OrchestrationPlanQuery{
		Tags: map[string]string{"type": "playbook"},
	})

	if len(plans) != 5 {
		t.Errorf("got %d playbooks, want 5", len(plans))
	}

	// Verify specific playbooks exist
	playbookIDs := map[string]bool{
		"plan-playbook-001": false,
		"plan-playbook-002": false,
		"plan-playbook-003": false,
		"plan-playbook-004": false,
		"plan-playbook-005": false,
	}

	for _, plan := range plans {
		if _, ok := playbookIDs[plan.ID]; ok {
			playbookIDs[plan.ID] = true
		}
	}

	for id, found := range playbookIDs {
		if !found {
			t.Errorf("playbook %s not found", id)
		}
	}
}

func TestSeed_Runbooks(t *testing.T) {
	p, _ := New(nil)

	plans, _ := p.QueryPlans(context.Background(), schema.OrchestrationPlanQuery{
		Tags: map[string]string{"type": "runbook"},
	})

	if len(plans) != 7 {
		t.Errorf("got %d runbooks, want 7", len(plans))
	}

	// Verify specific runbooks exist
	runbookIDs := map[string]bool{
		"plan-runbook-001": false,
		"plan-runbook-002": false,
		"plan-runbook-003": false,
		"plan-runbook-004": false,
		"plan-runbook-005": false,
		"plan-runbook-006": false,
		"plan-runbook-007": false,
	}

	for _, plan := range plans {
		if _, ok := runbookIDs[plan.ID]; ok {
			runbookIDs[plan.ID] = true
		}
	}

	for id, found := range runbookIDs {
		if !found {
			t.Errorf("runbook %s not found", id)
		}
	}
}

func TestSeed_ReleaseChecklists(t *testing.T) {
	p, _ := New(nil)

	plans, _ := p.QueryPlans(context.Background(), schema.OrchestrationPlanQuery{
		Tags: map[string]string{"type": "release-checklist"},
	})

	if len(plans) != 3 {
		t.Errorf("got %d release checklists, want 3", len(plans))
	}

	// Verify specific checklists exist
	checklistIDs := map[string]bool{
		"plan-release-001": false,
		"plan-release-002": false,
		"plan-release-003": false,
	}

	for _, plan := range plans {
		if _, ok := checklistIDs[plan.ID]; ok {
			checklistIDs[plan.ID] = true
		}
	}

	for id, found := range checklistIDs {
		if !found {
			t.Errorf("checklist %s not found", id)
		}
	}
}

func TestSeed_ActiveRuns(t *testing.T) {
	p, _ := New(nil)

	runs, _ := p.QueryRuns(context.Background(), schema.OrchestrationRunQuery{})

	if len(runs) != 4 {
		t.Errorf("got %d runs, want 4", len(runs))
	}

	// Verify run statuses
	statusCounts := make(map[string]int)
	for _, run := range runs {
		statusCounts[run.Status]++
	}

	if statusCounts["blocked"] != 1 {
		t.Errorf("got %d blocked runs, want 1", statusCounts["blocked"])
	}
	if statusCounts["running"] != 2 {
		t.Errorf("got %d running runs, want 2", statusCounts["running"])
	}
	if statusCounts["completed"] != 1 {
		t.Errorf("got %d completed runs, want 1", statusCounts["completed"])
	}
}

func TestAutomatedExecution(t *testing.T) {
	// Use a short duration for tests
	cfg := map[string]any{
		"step_duration": "200ms",
	}
	p, _ := New(cfg)

	// Start the automated playbook
	run, err := p.StartRun(context.Background(), "plan-playbook-005")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Step 1 is automated and running immediately
	// It should complete automatically after a short delay
	// We wait up to 1 second (5x step duration) for automation to make progress
	time.Sleep(1 * time.Second)

	updatedRun, _ := p.GetRun(context.Background(), run.ID)

	// Step 1 should be succeeded
	if updatedRun.Steps[0].Status != "succeeded" {
		t.Errorf("step 1 status %q, want succeeded", updatedRun.Steps[0].Status)
	}
	if updatedRun.Steps[0].Actor != "system-automation" {
		t.Errorf("step 1 actor %q, want system-automation", updatedRun.Steps[0].Actor)
	}

	// Step 2 depends on Step 1, is automated, so should also be succeeded (or running if we caught it mid-flight)
	// Given 3s wait and ~500ms sleep, step 1, 2, 3 should likely all be done
	if updatedRun.Steps[1].Status != "succeeded" {
		t.Errorf("step 2 status %q, want succeeded", updatedRun.Steps[1].Status)
	}

	if updatedRun.Steps[2].Status != "succeeded" {
		t.Errorf("step 3 status %q, want succeeded", updatedRun.Steps[2].Status)
	}

	// Step 4 is manual, so it should be ready or running but NOT succeeded
	// It depends on step 3.
	if updatedRun.Steps[3].Status != "running" && updatedRun.Steps[3].Status != "ready" {
		// It becomes running automatically because it is manual but provider updates dependent steps to running if ready?
		// check provider.go logic:
		// if allDepsComplete && run.Steps[i].Status == "pending" { ... Status = "running" }
		// So it should be running
		t.Errorf("step 4 status %q, want running", updatedRun.Steps[3].Status)
	}
}
