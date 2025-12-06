package mockutil

import "github.com/opsorch/opsorch-core/schema"

// CloneMap returns a shallow copy of a string->any map.
func CloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// CloneStringMap returns a shallow copy of a string->string map.
func CloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// CloneStringSlice copies a slice of strings.
func CloneStringSlice(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

// CloneAlerts performs a copy of alerts so callers can safely mutate them.
func CloneAlerts(in []schema.Alert) []schema.Alert {
	if in == nil {
		return nil
	}
	out := make([]schema.Alert, len(in))
	for i, al := range in {
		out[i] = schema.Alert{
			ID:          al.ID,
			Title:       al.Title,
			Description: al.Description,
			Status:      al.Status,
			Severity:    al.Severity,
			Service:     al.Service,
			CreatedAt:   al.CreatedAt,
			UpdatedAt:   al.UpdatedAt,
			Fields:      CloneMap(al.Fields),
			Metadata:    CloneMap(al.Metadata),
		}
	}
	return out
}
