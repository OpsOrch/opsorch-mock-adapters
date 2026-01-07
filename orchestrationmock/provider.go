package orchestrationmock

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/opsorch/opsorch-core/orcherr"
	"github.com/opsorch/opsorch-core/orchestration"
	"github.com/opsorch/opsorch-core/schema"
)

// ProviderName can be referenced via OPSORCH_ORCHESTRATION_PROVIDER.
const ProviderName = "mock"

// Config controls mock orchestration behavior.
type Config struct {
	Source       string
	StepDuration time.Duration
}

// Provider keeps an in-memory plan and run store for demo purposes.
type Provider struct {
	cfg    Config
	mu     sync.Mutex
	nextID int
	plans  map[string]schema.OrchestrationPlan
	runs   map[string]schema.OrchestrationRun
}

// New constructs the provider with seeded demo plans and runs.
func New(cfg map[string]any) (orchestration.Provider, error) {
	parsed := parseConfig(cfg)
	p := &Provider{
		cfg:   parsed,
		plans: map[string]schema.OrchestrationPlan{},
		runs:  map[string]schema.OrchestrationRun{},
	}
	p.seed()
	return p, nil
}

func init() {
	_ = orchestration.RegisterProvider(ProviderName, New)
}

// parseConfig extracts configuration from a map.
func parseConfig(cfg map[string]any) Config {
	parsed := Config{
		Source:       "mock",
		StepDuration: 10 * time.Second,
	}
	if cfg == nil {
		return parsed
	}
	if source, ok := cfg["source"].(string); ok && source != "" {
		parsed.Source = source
	}
	if durationStr, ok := cfg["step_duration"].(string); ok && durationStr != "" {
		if d, err := time.ParseDuration(durationStr); err == nil {
			parsed.StepDuration = d
		}
	}
	return parsed
}

// QueryPlans returns plans matching the query parameters.
func (p *Provider) QueryPlans(ctx context.Context, query schema.OrchestrationPlanQuery) ([]schema.OrchestrationPlan, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	needle := strings.ToLower(strings.TrimSpace(query.Query))
	tagFilter := query.Tags
	scopeFilter := query.Scope

	out := make([]schema.OrchestrationPlan, 0, len(p.plans))
	for _, plan := range p.plans {
		// Filter by query string (title or description)
		if needle != "" {
			planText := strings.ToLower(plan.Title + " " + plan.Description)
			if !strings.Contains(planText, needle) {
				continue
			}
		}

		// Filter by tags
		if len(tagFilter) > 0 {
			if !matchesTags(plan.Tags, tagFilter) {
				continue
			}
		}

		// Filter by scope
		if !matchesScopeForPlan(scopeFilter, plan) {
			continue
		}

		out = append(out, clonePlan(plan))
		if query.Limit > 0 && len(out) >= query.Limit {
			break
		}
	}
	return out, nil
}

// GetPlan returns a single plan by ID.
func (p *Provider) GetPlan(ctx context.Context, planID string) (*schema.OrchestrationPlan, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	plan, ok := p.plans[planID]
	if !ok {
		return nil, orcherr.New("not_found", "plan not found", nil)
	}
	cloned := clonePlan(plan)
	return &cloned, nil
}

// QueryRuns returns runs matching the query parameters.
func (p *Provider) QueryRuns(ctx context.Context, query schema.OrchestrationRunQuery) ([]schema.OrchestrationRun, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	needle := strings.ToLower(strings.TrimSpace(query.Query))
	statusFilter := toSet(query.Statuses)
	planIDFilter := toSet(query.PlanIDs)
	scopeFilter := query.Scope

	out := make([]schema.OrchestrationRun, 0, len(p.runs))
	for _, run := range p.runs {
		// Filter by query string
		if needle != "" {
			runText := strings.ToLower(run.ID + " " + run.PlanID)
			if !strings.Contains(runText, needle) {
				continue
			}
		}

		// Filter by statuses
		if len(statusFilter) > 0 && !statusFilter[run.Status] {
			continue
		}

		// Filter by plan IDs
		if len(planIDFilter) > 0 && !planIDFilter[run.PlanID] {
			continue
		}

		// Filter by scope
		if !matchesScopeForRun(scopeFilter, run) {
			continue
		}

		out = append(out, cloneRun(run))
		if query.Limit > 0 && len(out) >= query.Limit {
			break
		}
	}
	return out, nil
}

// GetRun returns a single run by ID with current step states.
func (p *Provider) GetRun(ctx context.Context, runID string) (*schema.OrchestrationRun, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	run, ok := p.runs[runID]
	if !ok {
		return nil, orcherr.New("not_found", "run not found", nil)
	}
	cloned := cloneRun(run)
	return &cloned, nil
}

// StartRun creates a new run from a plan.
func (p *Provider) StartRun(ctx context.Context, planID string) (*schema.OrchestrationRun, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	plan, ok := p.plans[planID]
	if !ok {
		return nil, orcherr.New("not_found", "plan not found", nil)
	}

	p.nextID++
	runID := fmt.Sprintf("run-%03d", p.nextID)
	now := time.Now().UTC()

	// Initialize step states
	stepStates := make([]schema.OrchestrationStepState, len(plan.Steps))
	runStatus := "created"
	for i, step := range plan.Steps {
		status := "pending"
		if len(step.DependsOn) == 0 {
			if step.Type == "automated" {
				status = "running"
			} else {
				status = "ready"
			}
		}
		var startedAt *time.Time
		if status == "running" {
			startedAt = &now
			runStatus = "running"
		}
		stepStates[i] = schema.OrchestrationStepState{
			StepID:    step.ID,
			Status:    status,
			StartedAt: startedAt,
			UpdatedAt: &now,
		}
	}

	run := schema.OrchestrationRun{
		ID:        runID,
		PlanID:    planID,
		Plan:      &plan,
		Status:    runStatus,
		Steps:     stepStates,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata: map[string]any{
			"source": p.cfg.Source,
		},
	}

	p.runs[runID] = run
	cloned := cloneRun(run)

	// Check for automated steps to trigger
	p.checkAutomatedSteps(context.Background(), &cloned)

	return &cloned, nil
}

// CompleteStep marks a step as complete and updates dependent steps.
func (p *Provider) CompleteStep(ctx context.Context, runID string, stepID string, actor string, note string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	run, ok := p.runs[runID]
	if !ok {
		return orcherr.New("not_found", "run not found", nil)
	}

	// Find the step state
	stepIdx := -1
	for i, s := range run.Steps {
		if s.StepID == stepID {
			stepIdx = i
			break
		}
	}
	if stepIdx == -1 {
		return orcherr.New("not_found", "step not found", nil)
	}

	// Mark step as succeeded
	now := time.Now().UTC()
	run.Steps[stepIdx].Status = "succeeded"
	run.Steps[stepIdx].Actor = actor
	run.Steps[stepIdx].Note = note
	run.Steps[stepIdx].FinishedAt = &now
	run.Steps[stepIdx].UpdatedAt = &now

	// Update dependent steps
	p.updateDependentSteps(&run, stepID)

	// Check if all steps are complete
	allStepsComplete := true
	for _, s := range run.Steps {
		if s.Status != "succeeded" {
			allStepsComplete = false
			break
		}
	}
	if allStepsComplete {
		run.Status = "completed"
	}

	run.UpdatedAt = now
	run.UpdatedAt = now
	p.runs[runID] = run

	// Check for further automated steps to trigger
	// Note: We need a fresh clone or the updated run structure
	updatedRun := cloneRun(run)
	p.checkAutomatedSteps(ctx, &updatedRun)

	return nil
}

// updateDependentSteps marks dependent steps as ready when all their dependencies are complete.
func (p *Provider) updateDependentSteps(run *schema.OrchestrationRun, completedStepID string) {
	plan := p.plans[run.PlanID]

	for i, step := range plan.Steps {
		// Check if this step depends on the completed step
		dependsOnCompleted := false
		for _, depID := range step.DependsOn {
			if depID == completedStepID {
				dependsOnCompleted = true
				break
			}
		}
		if !dependsOnCompleted {
			continue
		}

		// Check if all dependencies are now complete
		allDepsComplete := true
		for _, depID := range step.DependsOn {
			depState := findStepState(run.Steps, depID)
			if depState == nil || depState.Status != "succeeded" {
				allDepsComplete = false
				break
			}
		}

		// If all deps complete and step is pending, mark as ready (manual) or running (automated)
		if allDepsComplete && run.Steps[i].Status == "pending" {
			now := time.Now().UTC()
			if step.Type == "automated" {
				run.Steps[i].Status = "running"
				run.Steps[i].StartedAt = &now
			} else {
				run.Steps[i].Status = "ready"
			}
			run.Steps[i].UpdatedAt = &now
		}
	}
}

// checkAutomatedSteps identifies and triggers running steps marked as automated.
func (p *Provider) checkAutomatedSteps(ctx context.Context, run *schema.OrchestrationRun) {
	for _, step := range run.Steps {
		if step.Status != "running" {
			continue
		}

		// Find the step definition to check type
		var stepDef *schema.OrchestrationStep
		if run.Plan != nil {
			for _, s := range run.Plan.Steps {
				if s.ID == step.StepID {
					stepDef = &s
					break
				}
			}
		}

		isAutomated := false
		if stepDef != nil {
			// Check Type
			if stepDef.Type == "automated" {
				isAutomated = true
			}
			// Fallback check for metadata (legacy/compat)
			if !isAutomated && stepDef.Metadata != nil {
				if val, ok := stepDef.Metadata["automated"].(bool); ok && val {
					isAutomated = true
				}
			}
		}

		if isAutomated {
			// Spawn a goroutine to execute the step
			go func(runID, stepID string) {
				// Simulate some work duration
				time.Sleep(p.cfg.StepDuration)

				// Complete the step
				// We create a background context since original ctx might cancel
				_ = p.CompleteStep(context.Background(), runID, stepID, "system-automation", "Automated execution completed")
			}(run.ID, step.StepID)
		}
	}
}

// Helper functions

func toSet(items []string) map[string]bool {
	m := make(map[string]bool)
	for _, item := range items {
		m[item] = true
	}
	return m
}

func matchesTags(planTags map[string]string, filterTags map[string]string) bool {
	for k, v := range filterTags {
		if planTags[k] != v {
			return false
		}
	}
	return true
}

func matchesScopeForPlan(scope schema.QueryScope, plan schema.OrchestrationPlan) bool {
	if scope.Service != "" && plan.Tags["service"] != scope.Service {
		return false
	}
	if scope.Team != "" && plan.Tags["team"] != scope.Team {
		return false
	}
	if scope.Environment != "" && plan.Tags["environment"] != scope.Environment {
		return false
	}
	return true
}

func matchesScopeForRun(scope schema.QueryScope, run schema.OrchestrationRun) bool {
	if scope.Service != "" && run.Scope.Service != scope.Service {
		return false
	}
	if scope.Team != "" && run.Scope.Team != scope.Team {
		return false
	}
	if scope.Environment != "" && run.Scope.Environment != scope.Environment {
		return false
	}
	return true
}

func findStepState(states []schema.OrchestrationStepState, stepID string) *schema.OrchestrationStepState {
	for i, s := range states {
		if s.StepID == stepID {
			return &states[i]
		}
	}
	return nil
}

func clonePlan(plan schema.OrchestrationPlan) schema.OrchestrationPlan {
	cloned := plan
	cloned.Steps = make([]schema.OrchestrationStep, len(plan.Steps))
	copy(cloned.Steps, plan.Steps)
	cloned.Tags = cloneStringMap(plan.Tags)
	cloned.Fields = cloneMap(plan.Fields)
	cloned.Metadata = cloneMap(plan.Metadata)
	return cloned
}

func cloneRun(run schema.OrchestrationRun) schema.OrchestrationRun {
	cloned := run
	cloned.Steps = make([]schema.OrchestrationStepState, len(run.Steps))
	copy(cloned.Steps, run.Steps)
	if run.Plan != nil {
		planClone := clonePlan(*run.Plan)
		cloned.Plan = &planClone
	}
	cloned.Fields = cloneMap(run.Fields)
	cloned.Metadata = cloneMap(run.Metadata)
	return cloned
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	cloned := make(map[string]any)
	for k, v := range m {
		cloned[k] = v
	}
	return cloned
}

func cloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	cloned := make(map[string]string)
	for k, v := range m {
		cloned[k] = v
	}
	return cloned
}
