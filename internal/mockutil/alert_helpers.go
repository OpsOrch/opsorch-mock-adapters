package mockutil

import (
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

// SummarizeAlerts produces lightweight references for embedding in metadata/fields.
func SummarizeAlerts(alerts []schema.Alert) []map[string]any {
	if len(alerts) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(alerts))
	for _, al := range alerts {
		summary := map[string]any{
			"id":       al.ID,
			"status":   al.Status,
			"severity": al.Severity,
			"service":  al.Service,
			"title":    al.Title,
			"window": map[string]any{
				"createdAt": al.CreatedAt,
				"updatedAt": al.UpdatedAt,
			},
		}
		out = append(out, summary)
	}
	return out
}

// GetAnomalyFactor calculates an anomaly factor for an alert based on its severity and age.
// Returns a multiplier >= 1.0 that can be used to adjust metric values.
func GetAnomalyFactor(alert schema.Alert, ts time.Time) float64 {
	// Base factor by severity
	baseFactor := 1.0
	switch alert.Severity {
	case "critical":
		baseFactor = 3.0
	case "error":
		baseFactor = 2.0
	case "warning":
		baseFactor = 1.5
	default:
		baseFactor = 1.2
	}

	// Increase factor based on alert age (older alerts have more impact)
	age := ts.Sub(alert.CreatedAt)
	if age > 30*time.Minute {
		baseFactor *= 1.5
	} else if age > 15*time.Minute {
		baseFactor *= 1.3
	} else if age > 5*time.Minute {
		baseFactor *= 1.1
	}

	return baseFactor
}

// StrongestAlertFactor returns the highest anomaly factor among alerts active at ts.
func StrongestAlertFactor(service string, ts time.Time, alerts []schema.Alert) (float64, *schema.Alert) {
	var (
		best      = 1.0
		bestAlert *schema.Alert
	)
	for i := range alerts {
		al := alerts[i]
		if al.Service != "" && service != "" && al.Service != service {
			continue
		}
		if ts.Before(al.CreatedAt) || ts.After(al.UpdatedAt.Add(10*time.Minute)) {
			continue
		}
		if al.Status != "firing" && al.Status != "acknowledged" {
			continue
		}
		factor := GetAnomalyFactor(al, ts)
		if factor > best {
			best = factor
			bestAlert = &alerts[i]
		}
	}
	return best, bestAlert
}
