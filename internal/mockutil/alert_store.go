package mockutil

import (
	"sync"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

var (
	alertStoreMu sync.RWMutex
	alertStore   []schema.Alert
)

func init() {
	alertStore = buildDefaultAlerts()
}

func PublishAlerts(alerts []schema.Alert) {
	alertStoreMu.Lock()
	defer alertStoreMu.Unlock()
	alertStore = CloneAlerts(alerts)
	if alertStore == nil {
		alertStore = buildDefaultAlerts()
	}
}

func SnapshotAlerts() []schema.Alert {
	alertStoreMu.RLock()
	defer alertStoreMu.RUnlock()
	return CloneAlerts(alertStore)
}

func buildDefaultAlerts() []schema.Alert {
	now := time.Now().UTC()
	fallback := []schema.Alert{
		{
			ID:          "fixture-checkout-latency",
			Title:       "Checkout latency elevated",
			Description: "Synthetic fallback alert to correlate checkout latency demos",
			Status:      "firing",
			Severity:    "critical",
			Service:     "svc-checkout",
			CreatedAt:   now.Add(-30 * time.Minute),
			UpdatedAt:   now.Add(-5 * time.Minute),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-velocity",
				"metric":      "http_request_duration_seconds:p95",
			},
			Metadata: map[string]any{"source": "mock-fixture"},
		},
		{
			ID:          "fixture-search-errors",
			Title:       "Search 5xx ratio climbing",
			Description: "Synthetic fallback alert to correlate search error demos",
			Status:      "acknowledged",
			Severity:    "error",
			Service:     "svc-search",
			CreatedAt:   now.Add(-50 * time.Minute),
			UpdatedAt:   now.Add(-15 * time.Minute),
			Fields: map[string]any{
				"environment": "prod",
				"team":        "team-aurora",
				"metric":      "http_requests_total:5xx",
			},
			Metadata: map[string]any{"source": "mock-fixture"},
		},
	}
	return CloneAlerts(fallback)
}
