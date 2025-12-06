package mockutil

import (
	"fmt"
	"strings"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

// EnrichAlertMetadata adds metadata fields to an alert based on its service and severity
// This ensures alerts have runbook, dashboard, channel, and escalation information
func EnrichAlertMetadata(alert *schema.Alert) {
	if alert.Metadata == nil {
		alert.Metadata = make(map[string]any)
	}

	// Get team for the service
	team := getTeamForService(alert.Service)

	// Add runbook if not present
	if _, ok := alert.Metadata["runbook"]; !ok {
		alertType := getAlertTypeFromTitle(alert.Title)
		alert.Metadata["runbook"] = generateRunbookURL(alert.Service, alertType)
	}

	// Add dashboard if not present
	if _, ok := alert.Metadata["dashboard"]; !ok {
		alert.Metadata["dashboard"] = generateDashboardURL(alert.Service)
	}

	// Add channel if not present
	if _, ok := alert.Metadata["channel"]; !ok {
		alert.Metadata["channel"] = generateSlackChannel(team)
	}

	// Add escalation if not present
	if _, ok := alert.Metadata["escalation"]; !ok {
		alert.Metadata["escalation"] = generateEscalationPolicy(alert.Service, alert.Severity)
	}

	// Add team to metadata if not present
	if _, ok := alert.Metadata["team"]; !ok {
		alert.Metadata["team"] = team
	}
}

// generateRunbookURL creates a runbook URL for a service and alert type
func generateRunbookURL(service, alertType string) string {
	if alertType == "" {
		alertType = "general"
	}
	// Normalize alert type for URL
	urlType := strings.ToLower(strings.ReplaceAll(alertType, " ", "-"))
	return fmt.Sprintf("https://runbook.demo/%s-%s", service, urlType)
}

// generateDashboardURL creates a dashboard URL for a service
func generateDashboardURL(service string) string {
	// Remove svc- prefix for cleaner dashboard names
	dashName := strings.TrimPrefix(service, "svc-")
	return fmt.Sprintf("dash-%s", dashName)
}

// generateSlackChannel returns the appropriate Slack channel for a team
func generateSlackChannel(team string) string {
	return GetChannelForTeam(team)
}

// generateEscalationPolicy returns escalation contacts for a service and severity
func generateEscalationPolicy(service, severity string) []string {
	team := getTeamForService(service)

	escalation := []string{}

	// Add team oncall
	oncall := fmt.Sprintf("@oncall-%s", strings.TrimPrefix(team, "team-"))
	escalation = append(escalation, oncall)

	// Add PagerDuty for critical/error
	if severity == "critical" || severity == "error" {
		// Generate PagerDuty ID based on service
		pdID := strings.ToUpper(strings.TrimPrefix(service, "svc-"))
		if len(pdID) > 3 {
			pdID = pdID[:3]
		}
		escalation = append(escalation, fmt.Sprintf("pagerduty://%s-CRIT", pdID))
	}

	// Add team lead for critical
	if severity == "critical" {
		lead := fmt.Sprintf("@%s-lead", strings.TrimPrefix(team, "team-"))
		escalation = append(escalation, lead)
	}

	return escalation
}

// getTeamForService returns the team that owns a service
func getTeamForService(service string) string {
	return GetTeamForService(service)
}

// getAlertTypeFromTitle extracts a simplified alert type from the title
func getAlertTypeFromTitle(title string) string {
	lower := strings.ToLower(title)

	// Common alert type patterns
	if strings.Contains(lower, "latency") || strings.Contains(lower, "slow") {
		return "latency"
	}
	if strings.Contains(lower, "error") || strings.Contains(lower, "5xx") {
		return "error-rate"
	}
	if strings.Contains(lower, "outage") || strings.Contains(lower, "down") {
		return "outage"
	}
	if strings.Contains(lower, "failover") {
		return "failover"
	}
	if strings.Contains(lower, "disk") || strings.Contains(lower, "space") {
		return "disk-space"
	}
	if strings.Contains(lower, "memory") || strings.Contains(lower, "oom") {
		return "memory"
	}
	if strings.Contains(lower, "cpu") {
		return "cpu"
	}
	if strings.Contains(lower, "queue") || strings.Contains(lower, "lag") {
		return "queue-lag"
	}
	if strings.Contains(lower, "certificate") || strings.Contains(lower, "cert") {
		return "certificate"
	}
	if strings.Contains(lower, "deployment") || strings.Contains(lower, "deploy") {
		return "deployment"
	}
	if strings.Contains(lower, "budget") || strings.Contains(lower, "slo") {
		return "slo"
	}

	return "general"
}

// EnrichWithContextualInfo adds deployment, config, or user segment info to alerts
func EnrichWithContextualInfo(alert *schema.Alert, now time.Time) {
	if alert.Fields == nil {
		alert.Fields = make(map[string]any)
	}

	// Add deployment context for service-related alerts (if not already present)
	if _, hasLastDeploy := alert.Fields["lastDeploy"]; !hasLastDeploy {
		if _, hasRelease := alert.Fields["release"]; !hasRelease {
			if _, hasVersion := alert.Fields["version"]; !hasVersion {
				// Add deployment info to alerts that might be deployment-related
				if strings.Contains(strings.ToLower(alert.Title), "latency") ||
					strings.Contains(strings.ToLower(alert.Title), "error") ||
					strings.Contains(strings.ToLower(alert.Title), "5xx") ||
					strings.Contains(strings.ToLower(alert.Title), "spike") ||
					strings.Contains(strings.ToLower(alert.Description), "regression") ||
					strings.Contains(strings.ToLower(alert.Description), "deploy") {

					// Generate a realistic version number
					version := fmt.Sprintf("%s-v2.%d.%d", alert.Service, (now.Unix()%50)+1, (now.Unix() % 20))
					deployTime := alert.CreatedAt.Add(-time.Duration(30+now.Unix()%90) * time.Minute)

					alert.Fields["lastDeploy"] = map[string]any{
						"version": version,
						"at":      deployTime.Format(time.RFC3339),
						"author":  "deploy-bot",
					}
				}
			}
		}
	}

	// Add user impact context if not present
	if _, hasAffectedUsers := alert.Fields["affectedUsers"]; !hasAffectedUsers {
		if _, hasImpactedSegments := alert.Fields["impactedSegments"]; !hasImpactedSegments {
			if _, hasClientSegments := alert.Fields["clientSegments"]; !hasClientSegments {
				// Add user impact for customer-facing services or critical/error alerts
				if alert.Service == "svc-checkout" || alert.Service == "svc-web" ||
					alert.Service == "svc-payments" || alert.Service == "svc-search" ||
					alert.Service == "svc-catalog" || alert.Service == "svc-identity" ||
					alert.Service == "svc-api" || alert.Severity == "critical" || alert.Severity == "error" {

					userCount := 1000 + (now.Unix() % 15000)
					alert.Fields["affectedUsers"] = fmt.Sprintf("~%d users", userCount)
				}
			}
		}
	}

	// Add affected services for infrastructure components
	if _, hasAffectedServices := alert.Fields["affectedServices"]; !hasAffectedServices {
		if alert.Service == "svc-database" || alert.Service == "svc-cache" ||
			alert.Service == "svc-loadbalancer" || alert.Service == "svc-api-gateway" ||
			alert.Service == "svc-dns" || alert.Service == "svc-ingress" {

			// Infrastructure services typically affect multiple downstream services
			downstreamServices := []string{"svc-checkout", "svc-catalog", "svc-order"}
			alert.Fields["affectedServices"] = downstreamServices
		}
	}
}

// EnrichWithMultiRegionFields adds multi-region fields to infrastructure alerts
func EnrichWithMultiRegionFields(alert *schema.Alert) {
	if alert.Fields == nil {
		alert.Fields = make(map[string]any)
	}

	// Check if already has multi-region fields
	if regions, ok := alert.Fields["impactedRegions"].([]string); ok && len(regions) >= 2 {
		return
	}

	// Add multi-region fields for infrastructure services or global services
	isInfrastructure := alert.Service == "svc-database" || alert.Service == "svc-cache" ||
		alert.Service == "svc-api-gateway" || alert.Service == "svc-dns" ||
		alert.Service == "svc-loadbalancer" || alert.Service == "svc-cdn" ||
		alert.Service == "svc-ingress"

	// Check if region is global
	isGlobal := false
	if region, ok := alert.Fields["region"].(string); ok {
		isGlobal = strings.Contains(region, "global")
	}

	if isInfrastructure || isGlobal {
		// Add multiple regions
		regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
		alert.Fields["impactedRegions"] = regions

		// Add availability zones
		alert.Fields["availabilityZones"] = []string{"us-east-1a", "us-east-1b", "us-west-2a"}
	}
}
