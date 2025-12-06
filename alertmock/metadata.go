package alertmock

import (
	"time"

	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/internal/mockutil"
)

// enrichAlertMetadata adds metadata fields to an alert based on its service and severity
// This is a wrapper around mockutil.EnrichAlertMetadata for use within alertmock package
func enrichAlertMetadata(alert *schema.Alert) {
	mockutil.EnrichAlertMetadata(alert)
}

// enrichWithContextualInfo adds deployment, config, or user segment info to alerts
// This is a wrapper around mockutil.EnrichWithContextualInfo for use within alertmock package
func enrichWithContextualInfo(alert *schema.Alert, now time.Time) {
	mockutil.EnrichWithContextualInfo(alert, now)
}

// enrichWithMultiRegionFields adds multi-region fields to infrastructure alerts
func enrichWithMultiRegionFields(alert *schema.Alert) {
	mockutil.EnrichWithMultiRegionFields(alert)
}
