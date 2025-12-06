package alertmock

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/internal/mockutil"
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
		if al.ID == "al-012" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected description search to find al-012 (stripe webhook), got %+v", results)
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
	if len(scopedList) == 0 {
		t.Fatalf("expected scoped query to return alerts for svc-checkout, got none")
	}
	// Verify all returned alerts match the scope
	for _, al := range scopedList {
		if al.Service != "svc-checkout" {
			t.Fatalf("expected service svc-checkout, got %s", al.Service)
		}
		if env, ok := al.Fields["environment"].(string); !ok || env != "prod" {
			t.Fatalf("expected environment prod, got %v", al.Fields["environment"])
		}
		if team, ok := al.Fields["team"].(string); !ok || team != "team-velocity" {
			t.Fatalf("expected team team-velocity, got %v", al.Fields["team"])
		}
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

func TestLifecycleAutoAdvancePublishesSnapshots(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	targetID := "al-001"
	prov.mu.Lock()
	plan, ok := prov.lifecycle[targetID]
	if !ok {
		prov.mu.Unlock()
		t.Fatalf("expected lifecycle plan for %s", targetID)
	}
	state := prov.alerts[targetID]
	state.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
	state.UpdatedAt = state.CreatedAt
	prov.alerts[targetID] = state
	plan.applied = 0
	prov.mu.Unlock()

	resolved, err := prov.Get(context.Background(), targetID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("expected lifecycle to resolve %s, got status %s", targetID, resolved.Status)
	}
	if _, ok := resolved.Metadata["resolvedAt"]; !ok {
		t.Fatalf("expected resolvedAt metadata to be present")
	}

	snapshot := mockutil.SnapshotAlerts()
	found := false
	for _, al := range snapshot {
		if al.ID == targetID && al.Status == resolved.Status {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected snapshot to include resolved alert state")
	}
}

// Feature: enhanced-mock-data, Property 1: Severity distribution follows realistic proportions
func TestProperty_SeverityDistribution(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(alerts) < 25 {
		t.Fatalf("expected at least 25 alerts, got %d", len(alerts))
	}

	// Count severities
	counts := map[string]int{
		"critical": 0,
		"error":    0,
		"warning":  0,
		"info":     0,
	}

	for _, al := range alerts {
		counts[al.Severity]++
	}

	total := float64(len(alerts))

	// Check proportions with ±5% tolerance
	criticalPct := float64(counts["critical"]) / total
	errorPct := float64(counts["error"]) / total
	warningPct := float64(counts["warning"]) / total
	infoPct := float64(counts["info"]) / total

	// Critical: 10% ± 5% (5% to 15%)
	if criticalPct < 0.05 || criticalPct > 0.15 {
		t.Errorf("critical severity proportion %.2f%% outside expected range 5%%-15%%", criticalPct*100)
	}

	// Error: 25% ± 5% (20% to 30%)
	if errorPct < 0.20 || errorPct > 0.30 {
		t.Errorf("error severity proportion %.2f%% outside expected range 20%%-30%%", errorPct*100)
	}

	// Warning: 45% ± 5% (40% to 50%)
	if warningPct < 0.40 || warningPct > 0.50 {
		t.Errorf("warning severity proportion %.2f%% outside expected range 40%%-50%%", warningPct*100)
	}

	// Info: 20% ± 5% (15% to 25%)
	if infoPct < 0.15 || infoPct > 0.25 {
		t.Errorf("info severity proportion %.2f%% outside expected range 15%%-25%%", infoPct*100)
	}

	t.Logf("Severity distribution: critical=%.1f%%, error=%.1f%%, warning=%.1f%%, info=%.1f%%",
		criticalPct*100, errorPct*100, warningPct*100, infoPct*100)
}

// Feature: enhanced-mock-data, Property 2: Alert timestamps maintain temporal consistency
func TestProperty_TimestampConsistency(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	now := time.Now().UTC()

	for _, al := range alerts {
		// CreatedAt <= UpdatedAt
		if al.CreatedAt.After(al.UpdatedAt) {
			t.Errorf("alert %s: CreatedAt (%v) is after UpdatedAt (%v)", al.ID, al.CreatedAt, al.UpdatedAt)
		}

		// Timestamps should be within 24 hours of now
		if al.CreatedAt.After(now) {
			t.Errorf("alert %s: CreatedAt (%v) is in the future", al.ID, al.CreatedAt)
		}
		if al.UpdatedAt.After(now) {
			t.Errorf("alert %s: UpdatedAt (%v) is in the future", al.ID, al.UpdatedAt)
		}
		if now.Sub(al.CreatedAt) > 24*time.Hour {
			t.Errorf("alert %s: CreatedAt (%v) is more than 24 hours ago", al.ID, al.CreatedAt)
		}

		// For resolved alerts, check resolution timestamp
		if al.Status == "resolved" {
			if resolvedAt, ok := al.Fields["resolvedAt"].(time.Time); ok {
				if resolvedAt.Before(al.CreatedAt) {
					t.Errorf("alert %s: resolvedAt (%v) is before CreatedAt (%v)", al.ID, resolvedAt, al.CreatedAt)
				}
				if resolvedAt.After(now) {
					t.Errorf("alert %s: resolvedAt (%v) is in the future", al.ID, resolvedAt)
				}
			}
		}

		// For acknowledged alerts, check acknowledgment timestamp
		if al.Status == "acknowledged" {
			if ackAt, ok := al.Fields["acknowledgedAt"].(time.Time); ok {
				if ackAt.Before(al.CreatedAt) {
					t.Errorf("alert %s: acknowledgedAt (%v) is before CreatedAt (%v)", al.ID, ackAt, al.CreatedAt)
				}
				if ackAt.After(now) {
					t.Errorf("alert %s: acknowledgedAt (%v) is in the future", al.ID, ackAt)
				}
			}
		}

		// For silenced alerts, check silence timestamp
		if al.Status == "silenced" {
			if silencedAt, ok := al.Fields["silencedAt"].(time.Time); ok {
				if silencedAt.Before(al.CreatedAt) {
					t.Errorf("alert %s: silencedAt (%v) is before CreatedAt (%v)", al.ID, silencedAt, al.CreatedAt)
				}
				if silencedAt.After(now) {
					t.Errorf("alert %s: silencedAt (%v) is in the future", al.ID, silencedAt)
				}
			}
		}
	}
}

// Feature: enhanced-mock-data, Property 3: Alert metadata contains required fields
func TestProperty_MetadataFields(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	alertsWithMetadata := 0
	for _, al := range alerts {
		hasRequiredField := false

		// Check for runbook links
		if runbook, ok := al.Metadata["runbook"].(string); ok && runbook != "" {
			hasRequiredField = true
		}

		// Check for dashboard URLs
		if dashboard, ok := al.Metadata["dashboard"].(string); ok && dashboard != "" {
			hasRequiredField = true
		}

		// Check for team assignments (in metadata or fields)
		if _, ok := al.Metadata["team"].(string); ok {
			hasRequiredField = true
		}

		// Check for escalation policies
		if escalation, ok := al.Metadata["escalation"]; ok && escalation != nil {
			hasRequiredField = true
		}

		// Check for channel
		if channel, ok := al.Metadata["channel"].(string); ok && channel != "" {
			hasRequiredField = true
		}

		if hasRequiredField {
			alertsWithMetadata++
		}
	}

	// At least 90% of alerts should have metadata fields
	requiredPct := 0.90
	actualPct := float64(alertsWithMetadata) / float64(len(alerts))
	if actualPct < requiredPct {
		t.Errorf("only %.1f%% of alerts have required metadata fields, expected at least %.1f%%", actualPct*100, requiredPct*100)
	}
}

// Feature: enhanced-mock-data, Property 12: Alerts include service and environment metadata
func TestProperty_ServiceAndEnvironmentMetadata(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	for _, al := range alerts {
		// Check service name
		if al.Service == "" {
			t.Errorf("alert %s: missing service name", al.ID)
		}

		// Check environment tag
		if env, ok := al.Fields["environment"].(string); !ok || env == "" {
			t.Errorf("alert %s: missing or empty environment field", al.ID)
		}
	}
}

// Feature: enhanced-mock-data, Property 14: Alert fields include contextual information
func TestProperty_ContextualInformation(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	alertsWithContext := 0
	for _, al := range alerts {
		hasContextualInfo := false

		// Check for recent deployment info
		if _, ok := al.Fields["lastDeploy"]; ok {
			hasContextualInfo = true
		}
		if _, ok := al.Fields["previousDeploy"]; ok {
			hasContextualInfo = true
		}
		if _, ok := al.Fields["release"]; ok {
			hasContextualInfo = true
		}
		if _, ok := al.Fields["version"]; ok {
			hasContextualInfo = true
		}

		// Check for configuration change
		if _, ok := al.Fields["configChange"]; ok {
			hasContextualInfo = true
		}

		// Check for affected user segments
		if _, ok := al.Fields["impactedSegments"]; ok {
			hasContextualInfo = true
		}
		if _, ok := al.Fields["affectedUsers"]; ok {
			hasContextualInfo = true
		}
		if _, ok := al.Fields["clientSegments"]; ok {
			hasContextualInfo = true
		}
		if _, ok := al.Fields["affectedClients"]; ok {
			hasContextualInfo = true
		}
		if _, ok := al.Fields["affectedServices"]; ok {
			hasContextualInfo = true
		}

		if hasContextualInfo {
			alertsWithContext++
		}
	}

	// At least 30% of alerts should have contextual information
	requiredPct := 0.30
	actualPct := float64(alertsWithContext) / float64(len(alerts))
	if actualPct < requiredPct {
		t.Errorf("only %.1f%% of alerts have contextual information, expected at least %.1f%%", actualPct*100, requiredPct*100)
	}
}

// Feature: enhanced-mock-data, Property 25: Alerts include impact descriptions
func TestProperty_ImpactDescriptions(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	alertsWithImpact := 0
	for _, al := range alerts {
		hasImpactInfo := false

		// Check description for impact keywords
		descLower := strings.ToLower(al.Description)
		if strings.Contains(descLower, "affecting") ||
			strings.Contains(descLower, "impacted") ||
			strings.Contains(descLower, "users") ||
			strings.Contains(descLower, "transactions") {
			hasImpactInfo = true
		}

		// Check fields for impact information
		if _, ok := al.Fields["affectedUsers"]; ok {
			hasImpactInfo = true
		}
		if _, ok := al.Fields["impactedRegions"]; ok {
			hasImpactInfo = true
		}
		if _, ok := al.Fields["impactedSegments"]; ok {
			hasImpactInfo = true
		}
		if _, ok := al.Fields["impactPercent"]; ok {
			hasImpactInfo = true
		}
		if _, ok := al.Fields["affectedServices"]; ok {
			hasImpactInfo = true
		}

		if hasImpactInfo {
			alertsWithImpact++
		}
	}

	// At least 24% of alerts should have impact descriptions
	requiredPct := 0.24
	actualPct := float64(alertsWithImpact) / float64(len(alerts))
	if actualPct < requiredPct {
		t.Errorf("only %.1f%% of alerts have impact descriptions, expected at least %.1f%%", actualPct*100, requiredPct*100)
	}
}

// Feature: enhanced-mock-data, Property 17: Firing alerts have varied durations
func TestProperty_FiringAlertDurationVariety(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{Statuses: []string{"firing"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	now := time.Now().UTC()
	recentCount := 0    // < 30 min
	prolongedCount := 0 // > 2 hours

	for _, al := range alerts {
		duration := now.Sub(al.CreatedAt)
		if duration < 30*time.Minute {
			recentCount++
		}
		if duration > 2*time.Hour {
			prolongedCount++
		}
	}

	totalFiring := len(alerts)
	if totalFiring == 0 {
		t.Fatal("expected at least some firing alerts")
	}

	recentPct := float64(recentCount) / float64(totalFiring)
	prolongedPct := float64(prolongedCount) / float64(totalFiring)

	// At least 20% should be in each category
	if recentPct < 0.20 {
		t.Errorf("only %.1f%% of firing alerts are recent (< 30 min), expected at least 20%%", recentPct*100)
	}
	if prolongedPct < 0.20 {
		t.Errorf("only %.1f%% of firing alerts are prolonged (> 2 hours), expected at least 20%%", prolongedPct*100)
	}

	t.Logf("Firing alert durations: %.1f%% recent, %.1f%% prolonged", recentPct*100, prolongedPct*100)
}

// Feature: enhanced-mock-data, Property 18: Acknowledged alerts include acknowledgment details
func TestProperty_AcknowledgedAlertDetails(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{Statuses: []string{"acknowledged"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	for _, al := range alerts {
		hasAckBy := false
		hasAckNotes := false

		if ackBy, ok := al.Fields["acknowledgedBy"].(string); ok && ackBy != "" {
			hasAckBy = true
		}
		if notes, ok := al.Fields["notes"].(string); ok && notes != "" {
			hasAckNotes = true
		}

		if !hasAckBy {
			t.Errorf("acknowledged alert %s: missing acknowledgedBy field", al.ID)
		}
		if !hasAckNotes {
			t.Errorf("acknowledged alert %s: missing notes field", al.ID)
		}
	}
}

// Feature: enhanced-mock-data, Property 19: Silenced alerts include silencing metadata
func TestProperty_SilencedAlertMetadata(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{Statuses: []string{"silenced"}})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	for _, al := range alerts {
		hasSilenceReason := false
		hasSilenceDuration := false

		if reason, ok := al.Fields["silenceReason"].(string); ok && reason != "" {
			hasSilenceReason = true
		}
		// silenceUntil is now a string (RFC3339 format) for JSON serialization
		if silenceUntil, ok := al.Fields["silenceUntil"].(string); ok && silenceUntil != "" {
			hasSilenceDuration = true
		}

		if !hasSilenceReason {
			t.Errorf("silenced alert %s: missing silenceReason field", al.ID)
		}
		if !hasSilenceDuration {
			t.Errorf("silenced alert %s: missing silenceUntil field", al.ID)
		}
	}
}

// Feature: enhanced-mock-data, Property 26: Multi-region alerts list affected regions
func TestProperty_MultiRegionAlertFields(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	multiRegionCount := 0
	for _, al := range alerts {
		if regions, ok := al.Fields["impactedRegions"].([]string); ok && len(regions) >= 2 {
			multiRegionCount++
		}
	}

	// At least 6% of alerts should be multi-region
	requiredPct := 0.06
	actualPct := float64(multiRegionCount) / float64(len(alerts))
	if actualPct < requiredPct {
		t.Errorf("only %.1f%% of alerts are multi-region, expected at least %.1f%%", actualPct*100, requiredPct*100)
	}
}

// Feature: enhanced-mock-data, Property 27: Degradation alerts include impact metrics
func TestProperty_DegradationAlertImpactMetrics(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	degradationCount := 0
	degradationWithImpact := 0

	for _, al := range alerts {
		// Check if it's a degradation alert
		descLower := strings.ToLower(al.Description)
		titleLower := strings.ToLower(al.Title)
		if strings.Contains(descLower, "degrad") || strings.Contains(titleLower, "degrad") ||
			strings.Contains(descLower, "drop") || strings.Contains(titleLower, "drop") ||
			strings.Contains(descLower, "slow") || strings.Contains(titleLower, "slow") {
			degradationCount++

			// Check for impact metrics
			hasImpact := false
			if _, ok := al.Fields["impactPercent"]; ok {
				hasImpact = true
			}
			if _, ok := al.Fields["affectedRequests"]; ok {
				hasImpact = true
			}
			if _, ok := al.Fields["affectedUsers"]; ok {
				hasImpact = true
			}
			if _, ok := al.Fields["errorRate"]; ok {
				hasImpact = true
			}
			if _, ok := al.Fields["errorRatio"]; ok {
				hasImpact = true
			}

			if hasImpact {
				degradationWithImpact++
			}
		}
	}

	if degradationCount > 0 {
		impactPct := float64(degradationWithImpact) / float64(degradationCount)
		if impactPct < 0.30 {
			t.Errorf("only %.1f%% of degradation alerts have impact metrics, expected at least 30%%", impactPct*100)
		}
	}
}

// Feature: enhanced-mock-data, Property 28: Dependency alerts reference related services
func TestProperty_DependencyAlertServiceReferences(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	dependencyCount := 0
	for _, al := range alerts {
		// Check for dependency-related fields
		if _, ok := al.Fields["upstreamService"]; ok {
			dependencyCount++
		}
		if _, ok := al.Fields["downstreamService"]; ok {
			dependencyCount++
		}
		if _, ok := al.Fields["affectedServices"]; ok {
			dependencyCount++
		}
	}

	// At least 5% of alerts should reference dependencies
	requiredPct := 0.05
	actualPct := float64(dependencyCount) / float64(len(alerts))
	if actualPct < requiredPct {
		t.Errorf("only %.1f%% of alerts reference service dependencies, expected at least %.1f%%", actualPct*100, requiredPct*100)
	}
}

// Feature: enhanced-mock-data, Property 29: Customer-facing alerts include user impact
func TestProperty_CustomerFacingAlertUserImpact(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	customerFacingCount := 0
	for _, al := range alerts {
		// Check for user impact fields
		if _, ok := al.Fields["affectedUsers"]; ok {
			customerFacingCount++
		}
		if _, ok := al.Fields["estimatedRevenueLoss"]; ok {
			customerFacingCount++
		}
		if _, ok := al.Fields["impactPercent"]; ok {
			customerFacingCount++
		}
	}

	// At least 10% of alerts should have user impact
	requiredPct := 0.10
	actualPct := float64(customerFacingCount) / float64(len(alerts))
	if actualPct < requiredPct {
		t.Errorf("only %.1f%% of alerts include user impact, expected at least %.1f%%", actualPct*100, requiredPct*100)
	}
}

// Test for scenario-themed alerts without calling scenario methods
func TestScenarioAlertsStaticSeeding(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	// Query should return scenario alerts without calling scenario methods
	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Find scenario-themed alerts
	scenarioAlerts := []schema.Alert{}
	for _, al := range alerts {
		if strings.HasPrefix(al.ID, "al-scenario-") {
			scenarioAlerts = append(scenarioAlerts, al)
		}
	}

	if len(scenarioAlerts) == 0 {
		t.Fatalf("expected scenario-themed alerts, got none")
	}

	t.Logf("Found %d scenario-themed alerts", len(scenarioAlerts))

	// Verify scenario metadata fields are present
	for _, al := range scenarioAlerts {
		if scenarioID, ok := al.Fields["scenario_id"].(string); !ok || scenarioID == "" {
			t.Errorf("scenario alert %s: missing scenario_id field", al.ID)
		}
		if scenarioName, ok := al.Fields["scenario_name"].(string); !ok || scenarioName == "" {
			t.Errorf("scenario alert %s: missing scenario_name field", al.ID)
		}
		if scenarioStage, ok := al.Fields["scenario_stage"].(string); !ok || scenarioStage == "" {
			t.Errorf("scenario alert %s: missing scenario_stage field", al.ID)
		}
		if isScenario, ok := al.Metadata["is_scenario"].(bool); !ok || !isScenario {
			t.Errorf("scenario alert %s: missing or false is_scenario metadata", al.ID)
		}
	}
}

// Test that scenario alerts have variety in severities and statuses
func TestScenarioAlertVariety(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Find scenario-themed alerts
	scenarioAlerts := []schema.Alert{}
	for _, al := range alerts {
		if strings.HasPrefix(al.ID, "al-scenario-") {
			scenarioAlerts = append(scenarioAlerts, al)
		}
	}

	if len(scenarioAlerts) < 3 {
		t.Fatalf("expected at least 3 scenario alerts for variety testing, got %d", len(scenarioAlerts))
	}

	// Check for severity variety
	severities := make(map[string]bool)
	for _, al := range scenarioAlerts {
		severities[al.Severity] = true
	}

	if len(severities) < 2 {
		t.Errorf("scenario alerts should have at least 2 different severities, got %d", len(severities))
	}

	// Check for status variety
	statuses := make(map[string]bool)
	for _, al := range scenarioAlerts {
		statuses[al.Status] = true
	}

	if len(statuses) < 2 {
		t.Errorf("scenario alerts should have at least 2 different statuses, got %d", len(statuses))
	}

	t.Logf("Scenario alerts have %d severities and %d statuses", len(severities), len(statuses))
}

// Test that cascading failure alerts have proper metadata
func TestScenarioCascadingFailureMetadata(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	alerts, err := prov.Query(context.Background(), schema.AlertQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	// Find cascading failure scenario alerts
	cascadingAlerts := []schema.Alert{}
	for _, al := range alerts {
		if isCascading, ok := al.Metadata["is_cascading"].(bool); ok && isCascading {
			cascadingAlerts = append(cascadingAlerts, al)
		}
	}

	if len(cascadingAlerts) == 0 {
		t.Skip("no cascading failure alerts found, skipping test")
	}

	for _, al := range cascadingAlerts {
		// Cascading alerts should have affects metadata
		if affects, ok := al.Metadata["affects"]; !ok || affects == nil {
			t.Errorf("cascading alert %s: missing affects metadata", al.ID)
		}
	}
}

// Test that scenario alerts can be queried by scenario_id
func TestQueryScenarioAlertsByScenarioID(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	// Query for SLO exhaustion scenario
	alerts, err := prov.Query(context.Background(), schema.AlertQuery{
		Query: "slo-exhaustion",
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	found := false
	for _, al := range alerts {
		if scenarioID, ok := al.Fields["scenario_id"].(string); ok && scenarioID == "slo-exhaustion" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected to find alert with scenario_id 'slo-exhaustion'")
	}
}
