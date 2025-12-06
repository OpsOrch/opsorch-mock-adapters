package mockutil

// serviceTeamMap provides service-to-team mapping for alert ownership
// This is used across different mock adapters to ensure consistent team assignments
var serviceTeamMap = map[string]string{
	"svc-checkout":       "team-velocity",
	"svc-search":         "team-aurora",
	"svc-payments":       "team-revenue",
	"svc-notifications":  "team-signal",
	"svc-identity":       "team-guardian",
	"svc-warehouse":      "team-foundry",
	"svc-recommendation": "team-orion",
	"svc-analytics":      "team-lumen",
	"svc-order":          "team-velocity",
	"svc-catalog":        "team-atlas",
	"svc-shipping":       "team-hawkeye",
	"svc-realtime":       "team-nova",
	"svc-web":            "team-velocity",
	"svc-database":       "team-data",
	"svc-api-gateway":    "team-platform",
	"svc-ingress":        "team-platform",
	"svc-dns":            "team-platform",
	"svc-workers":        "team-signal",
	"svc-cache":          "team-platform",
	"svc-logging":        "team-platform",
	"svc-loadbalancer":   "team-platform",
	"svc-feature-flags":  "team-velocity",
	"svc-support":        "team-lumen",
	"svc-api":            "team-platform",
	"svc-slo-monitor":    "team-platform",
}

// teamChannelMap provides team-to-channel mapping for Slack/Teams notifications
var teamChannelMap = map[string]string{
	"team-velocity": "#checkout-alerts",
	"team-aurora":   "#search-eng",
	"team-revenue":  "#payments",
	"team-signal":   "#notifications",
	"team-guardian": "#identity-security",
	"team-foundry":  "#data-warehouse",
	"team-orion":    "#recommendations",
	"team-lumen":    "#analytics",
	"team-atlas":    "#catalog",
	"team-hawkeye":  "#shipping",
	"team-nova":     "#realtime-eng",
	"team-data":     "#database",
	"team-platform": "#platform-alerts",
}

// GetTeamForService returns the team that owns a service
func GetTeamForService(service string) string {
	if team, ok := serviceTeamMap[service]; ok {
		return team
	}
	return "team-platform"
}

// GetChannelForTeam returns the Slack channel for a team
func GetChannelForTeam(team string) string {
	if channel, ok := teamChannelMap[team]; ok {
		return channel
	}
	return "#ops-alerts"
}
