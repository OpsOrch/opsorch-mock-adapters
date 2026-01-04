package logmock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/opsorch/opsorch-core/log"
	"github.com/opsorch/opsorch-core/schema"
	"github.com/opsorch/opsorch-mock-adapters/internal/mockutil"
)

// ProviderName can be referenced via OPSORCH_LOG_PROVIDER.
const ProviderName = "mock"

// Config tunes mock log behavior.
type Config struct {
	DefaultLimit int
	Source       string
}

// Provider returns generated log entries for demo queries.
type Provider struct {
	cfg Config
}

type logInsight struct {
	Summary string
	Cause   string
	Tag     string
	Extra   map[string]any
}

// New constructs the mock log provider.
func New(cfg map[string]any) (log.Provider, error) {
	parsed := parseConfig(cfg)
	return &Provider{cfg: parsed}, nil
}

func init() {
	_ = log.RegisterProvider(ProviderName, New)
}

// generateLogURL creates a realistic Kibana-style log URL
func generateLogURL(logID, service string, timestamp time.Time) string {
	params := []string{}
	params = append(params, fmt.Sprintf("logId=%s", logID))
	if service != "" {
		params = append(params, fmt.Sprintf("service=%s", service))
	}
	params = append(params, fmt.Sprintf("timestamp=%s", timestamp.Format(time.RFC3339)))

	return fmt.Sprintf("https://kibana.demo.com/app/logs/stream?%s", strings.Join(params, "&"))
}

// matchesFilters checks if a log entry matches all the provided filters
func matchesFilters(entry schema.LogEntry, filters []schema.LogFilter) bool {
	if len(filters) == 0 {
		return true
	}

	for _, filter := range filters {
		var fieldValue string

		// Get the field value based on the filter field name
		switch filter.Field {
		case "service":
			fieldValue = entry.Service
		case "severity":
			fieldValue = entry.Severity
		case "message":
			fieldValue = entry.Message
		default:
			// Check in Labels
			if val, ok := entry.Labels[filter.Field]; ok {
				fieldValue = val
			} else if val, ok := entry.Fields[filter.Field]; ok {
				// Check in Fields - convert to string
				fieldValue = fmt.Sprintf("%v", val)
			} else {
				// Field not found, filter doesn't match
				return false
			}
		}

		// Apply the operator
		matches := false
		switch filter.Operator {
		case "=", "==":
			matches = fieldValue == filter.Value
		case "!=":
			matches = fieldValue != filter.Value
		case "contains":
			matches = strings.Contains(strings.ToLower(fieldValue), strings.ToLower(filter.Value))
		case "regex":
			// Simple regex support - for now just treat as contains
			matches = strings.Contains(strings.ToLower(fieldValue), strings.ToLower(filter.Value))
		default:
			// Unknown operator, treat as equals
			matches = fieldValue == filter.Value
		}

		if !matches {
			return false
		}
	}

	return true
}

// Query returns synthetic log entries that echo the query context.
func (p *Provider) Query(ctx context.Context, query schema.LogQuery) ([]schema.LogEntry, error) {
	_ = ctx

	end := query.End
	if end.IsZero() {
		end = time.Now().UTC()
	}
	start := query.Start
	if start.IsZero() {
		start = end.Add(-1 * time.Hour)
	}
	limit := query.Limit
	if limit <= 0 {
		limit = p.cfg.DefaultLimit
	}

	search := ""
	var severityFilter []string
	var filters []schema.LogFilter
	if query.Expression != nil {
		search = query.Expression.Search
		severityFilter = query.Expression.SeverityIn
		filters = query.Expression.Filters
	}

	service := inferService(query)
	alertSnapshot := mockutil.SnapshotAlerts()
	// Filter alerts for this service - check if alert is active during the time window
	serviceAlerts := make([]schema.Alert, 0)
	for _, alert := range alertSnapshot {
		if service != "" && alert.Service != service {
			continue
		}
		// Alert is relevant if it was created before end and updated after start
		if alert.CreatedAt.Before(end) && alert.UpdatedAt.After(start) {
			serviceAlerts = append(serviceAlerts, alert)
		}
	}

	// Determine which severities to generate
	severities := []string{"info", "warn", "error"}
	if len(severityFilter) > 0 {
		// Normalize the filter values to lowercase
		normalized := make([]string, 0, len(severityFilter))
		for _, s := range severityFilter {
			normalized = append(normalized, strings.ToLower(s))
		}
		severities = normalized
	} else {
		// Try to infer from search query
		inferred := inferSeverity(search)
		if inferred != "info" {
			severities = []string{inferred}
		}
	}

	count := limit
	if count > 20 {
		count = 20 // keep responses small for demos
	}
	if count < 3 {
		count = 3
	}
	step := end.Sub(start) / time.Duration(count+1)
	if step <= 0 {
		step = 5 * time.Second
	}

	entries := make([]schema.LogEntry, 0, count)
	generatedCount := 0

	// Use search query to seed deterministic generation
	searchSeed := hashString(search)

	// Parse search query for better matching
	parsedQuery := mockutil.ParseSearchQuery(search)

	for i := 0; generatedCount < count; i++ {
		ts := start.Add(time.Duration(i+1) * step)
		region := regionForService(service, i)
		labels := scopedLabels(query, service, region)
		method, path := requestShape(service, i)

		// Make severity distribution influenced by search terms
		severity := selectSeverityForSearch(search, severities, i, searchSeed)
		status := responseStatus(severity, i)
		latency := baseLatency(severity, i)
		traceID := fmt.Sprintf("trace-%05d", 4200+i)
		user := []string{"alice", "sam", "casey", "fern", "lena", "milo"}[i%6]
		component := componentForService(service, i)
		release := releaseForService(service)
		fields := buildFields(method, path, status, latency, traceID, user, component, region, release, severity, i, service)

		// Enhance fields with search terms if present
		if search != "" {
			enhanceFieldsWithSearchTerms(fields, parsedQuery)
		}
		if len(serviceAlerts) > 0 {
			factor, activeAlert := mockutil.StrongestAlertFactor(service, ts, serviceAlerts)
			if factor > 1.05 {
				severity = escalateSeverity(severity)
				status = promoteStatus(status)
				latency = int(float64(latency) * factor)
				fields["anomalyFactor"] = fmt.Sprintf("%.2f", factor)
				active := activeAlertsAt(ts, serviceAlerts)
				if len(active) > 0 {
					fields["alerts"] = mockutil.SummarizeAlerts(active)
				}
				if activeAlert != nil {
					if existing, ok := fields["insight"].(string); ok {
						fields["insight"] = fallback(existing, fmt.Sprintf("Active alert %s", activeAlert.ID))
					} else {
						fields["insight"] = fmt.Sprintf("Active alert %s", activeAlert.ID)
					}
				}
			}
		}
		insight := analyzeSearchContext(search, service, severity, component)
		if insight.Summary != "" {
			if _, exists := fields["insight"]; !exists {
				fields["insight"] = insight.Summary
			}
		}
		if insight.Cause != "" {
			if _, exists := fields["cause"]; !exists {
				fields["cause"] = insight.Cause
			}
		}
		for k, v := range insight.Extra {
			fields[k] = v
		}

		// Make message content reflect search terms
		contextHint := buildContextHint(search, insight.Summary)
		message := renderMessage(service, severity, method, path, status, latency, user, component, traceID, contextHint)

		// If we have search terms, ensure the message includes them
		if search != "" && !mockutil.MatchesSearchQuery(message, parsedQuery) {
			message = enhanceMessageWithSearchTerms(message, parsedQuery, service, component, traceID)
		}
		entry := schema.LogEntry{
			Timestamp: ts,
			Message:   message,
			Severity:  severity,
			Service:   service,
			URL:       generateLogURL(fmt.Sprintf("log-%d-%d", ts.Unix(), i), service, ts),
			Labels:    labels,
			Fields:    fields,
			Metadata:  scopedMetadata(p.cfg.Source, query, start, end, service, region, insight, serviceAlerts),
		}

		// Apply filters to generated entry
		if matchesFilters(entry, filters) {
			entries = append(entries, entry)
			generatedCount++
		}

		// Prevent infinite loop if filters are too restrictive
		if i > count*10 {
			break
		}
	}

	// Add static scenario-themed logs
	scenarioLogs := getScenarioLogs(end)
	for _, sl := range scenarioLogs {
		// Only include logs within the query time range
		if (sl.Timestamp.Equal(start) || sl.Timestamp.After(start)) &&
			(sl.Timestamp.Equal(end) || sl.Timestamp.Before(end)) {
			// Filter by service if specified
			if service == "" || sl.Service == service {
				// Apply filters to scenario logs
				if matchesFilters(sl, filters) {
					entries = append(entries, sl)
				}
			}
		}
	}

	// If we have a search query but no results, generate logs that match
	if search != "" && len(entries) == 0 {
		parsedQuery := mockutil.ParseSearchQuery(search)
		generated := p.generateLogsForQuery(parsedQuery, query, service, start, end, limit)
		entries = append(entries, generated...)
	}

	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

// hashString creates a simple hash from a string for seeding
func hashString(s string) int {
	if s == "" {
		return 0
	}
	hash := 0
	for _, c := range s {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// selectSeverityForSearch picks severity based on search keywords
func selectSeverityForSearch(search string, severities []string, idx, seed int) string {
	lower := strings.ToLower(search)

	// If search contains error/failure keywords, bias toward errors
	errorKeywords := []string{"error", "fail", "crash", "exception", "panic", "fatal", "critical"}
	for _, kw := range errorKeywords {
		if strings.Contains(lower, kw) {
			// 70% errors, 30% other severities
			if (idx+seed)%10 < 7 {
				return "error"
			}
		}
	}

	// If search contains warning keywords, bias toward warnings
	warnKeywords := []string{"warn", "slow", "degrad", "timeout", "retry", "lag"}
	for _, kw := range warnKeywords {
		if strings.Contains(lower, kw) {
			// 60% warnings, 40% other severities
			if (idx+seed)%10 < 6 {
				return "warn"
			}
		}
	}

	// Default: use provided severities in order
	return severities[idx%len(severities)]
}

// buildContextHint creates a context string that incorporates search terms
func buildContextHint(search, insightSummary string) string {
	if insightSummary != "" {
		return insightSummary
	}
	if search == "" {
		return "service logs"
	}

	// Extract key terms from search and make them part of the context
	lower := strings.ToLower(search)

	// Common patterns to extract meaningful context
	if strings.Contains(lower, "rollout") || strings.Contains(lower, "deploy") {
		return "deployment rollout activity"
	}
	if strings.Contains(lower, "quality") && strings.Contains(lower, "drop") {
		return "quality degradation detected"
	}
	if strings.Contains(lower, "recommendation") {
		return "recommendation engine processing"
	}
	if strings.Contains(lower, "timeout") {
		return "request timeout handling"
	}
	if strings.Contains(lower, "cache") {
		return "cache operation"
	}
	if strings.Contains(lower, "database") || strings.Contains(lower, "db") {
		return "database operation"
	}

	// Default: use first few words of search
	words := strings.Fields(search)
	if len(words) > 0 {
		if len(words) > 3 {
			return strings.Join(words[:3], " ")
		}
		return strings.Join(words, " ")
	}

	return "service logs"
}

func parseConfig(cfg map[string]any) Config {
	out := Config{DefaultLimit: 50, Source: "mock-log"}
	if v, ok := cfg["defaultLimit"].(int); ok && v > 0 {
		out.DefaultLimit = v
	}
	if v, ok := cfg["defaultLimit"].(float64); ok && v > 0 { // configs come in via JSON
		out.DefaultLimit = int(v)
	}
	if v, ok := cfg["source"].(string); ok && v != "" {
		out.Source = v
	}
	return out
}

func inferService(q schema.LogQuery) string {
	if q.Scope.Service != "" {
		return q.Scope.Service
	}

	// Check filters for service field
	if q.Expression != nil && len(q.Expression.Filters) > 0 {
		for _, filter := range q.Expression.Filters {
			if filter.Field == "service" && (filter.Operator == "=" || filter.Operator == "==") {
				return filter.Value
			}
		}
	}

	if v, ok := q.Metadata["service"].(string); ok && v != "" {
		return v
	}

	search := ""
	if q.Expression != nil {
		search = q.Expression.Search
	}

	lower := strings.ToLower(search)
	samples := []string{"checkout", "search", "web", "identity", "notification", "realtime", "analytics", "cache", "database"}
	for _, s := range samples {
		if strings.Contains(lower, s) {
			return s
		}
	}
	return ""
}

func inferSeverity(query string) string {
	lower := strings.ToLower(query)
	switch {
	case strings.Contains(lower, "error"):
		return "error"
	case strings.Contains(lower, "warn"):
		return "warn"
	default:
		return "info"
	}
}

func fallback(val, def string) string {
	if strings.TrimSpace(val) != "" {
		return val
	}
	return def
}

func buildFields(method, path string, status, latency int, traceID, user, component, region, release, severity string, idx int, service string) map[string]any {
	session := fmt.Sprintf("sess-%04d", idx*5+7)
	instance := fmt.Sprintf("%s-%s-%02d", strings.ReplaceAll(component, "_", "-"), region, (idx%3)+1)
	fields := map[string]any{
		"requestId":     fmt.Sprintf("req-%06d", idx+1),
		"path":          path,
		"method":        method,
		"status":        status,
		"latencyMs":     latency,
		"traceId":       traceID,
		"user":          user,
		"sessionId":     session,
		"component":     component,
		"instance":      instance,
		"region":        region,
		"release":       release,
		"spanId":        fmt.Sprintf("span-%05d", idx+17),
		"customerTier":  customerForIndex(idx),
		"featureFlag":   featureFlagForService(service, idx),
		"correlationId": fmt.Sprintf("corr-%05d", idx+230),
	}
	if service != "" {
		fields["service"] = service
	}
	switch severity {
	case "error":
		fields["error"] = fmt.Sprintf("%s_upstream_failure", component)
		fields["retryable"] = idx%2 == 0
		fields["attempts"] = 1 + idx%3
	case "warn":
		fields["cacheHit"] = idx%3 == 0
		fields["slowPath"] = latency > 200
	default:
		fields["cacheHit"] = idx%2 == 0
	}
	return fields
}

func renderMessage(service, severity, method, path string, status, latency int, user, component, traceID, context string) string {
	serviceLabel := fallback(service, "service")

	// Create service-specific messages
	key := normalizeServiceName(service)
	switch key {
	case "cache":
		return renderCacheMessage(severity, method, path, component, status, latency, traceID)
	case "database":
		return renderDatabaseMessage(severity, method, path, component, status, latency, traceID)
	case "api-gateway":
		return renderGatewayMessage(severity, method, path, component, status, latency, user, traceID)
	case "loadbalancer":
		return renderLoadBalancerMessage(severity, component, status, latency, traceID)
	case "dns":
		return renderDNSMessage(severity, component, status, latency, traceID)
	default:
		// Default format for application services
		base := fmt.Sprintf("%s %s (%s) -> %d in %dms", method, path, component, status, latency)
		switch severity {
		case "error":
			return fmt.Sprintf("%s | %s request failed for user=%s trace=%s", base, serviceLabel, user, traceID)
		case "warn":
			return fmt.Sprintf("%s | %s degraded: matched='%s'", base, serviceLabel, context)
		default:
			return fmt.Sprintf("%s | %s activity trace=%s", base, serviceLabel, traceID)
		}
	}
}

func renderCacheMessage(severity, method, path, component string, status, latency int, traceID string) string {
	operation := "operation"
	if strings.Contains(path, "/get") {
		operation = "GET"
	} else if strings.Contains(path, "/set") {
		operation = "SET"
	} else if strings.Contains(path, "/delete") {
		operation = "DELETE"
	} else if strings.Contains(path, "/flush") {
		operation = "FLUSH"
	}

	switch severity {
	case "error":
		return fmt.Sprintf("[%s] Cache %s failed on %s | status=%d latency=%dms trace=%s", component, operation, path, status, latency, traceID)
	case "warn":
		return fmt.Sprintf("[%s] Cache %s slow on %s | status=%d latency=%dms trace=%s", component, operation, path, status, latency, traceID)
	default:
		return fmt.Sprintf("[%s] Cache %s completed on %s | status=%d latency=%dms trace=%s", component, operation, path, status, latency, traceID)
	}
}

func renderDatabaseMessage(severity, method, path, component string, status, latency int, traceID string) string {
	operation := "query"
	if strings.Contains(path, "/transaction") {
		operation = "transaction"
	} else if strings.Contains(path, "/connection") {
		operation = "connection"
	} else if strings.Contains(path, "/healthcheck") {
		operation = "healthcheck"
	}

	switch severity {
	case "error":
		return fmt.Sprintf("[%s] Database %s failed | status=%d latency=%dms trace=%s", component, operation, status, latency, traceID)
	case "warn":
		return fmt.Sprintf("[%s] Database %s slow | status=%d latency=%dms trace=%s", component, operation, status, latency, traceID)
	default:
		return fmt.Sprintf("[%s] Database %s completed | status=%d latency=%dms trace=%s", component, operation, status, latency, traceID)
	}
}

func renderGatewayMessage(severity, method, path, component string, status, latency int, user, traceID string) string {
	switch severity {
	case "error":
		return fmt.Sprintf("[%s] Gateway routing failed: %s %s | user=%s status=%d latency=%dms trace=%s", component, method, path, user, status, latency, traceID)
	case "warn":
		return fmt.Sprintf("[%s] Gateway routing degraded: %s %s | user=%s status=%d latency=%dms trace=%s", component, method, path, user, status, latency, traceID)
	default:
		return fmt.Sprintf("[%s] Gateway routed: %s %s | user=%s status=%d latency=%dms trace=%s", component, method, path, user, status, latency, traceID)
	}
}

func renderLoadBalancerMessage(severity, component string, status, latency int, traceID string) string {
	switch severity {
	case "error":
		return fmt.Sprintf("[%s] Load balancer backend check failed | status=%d latency=%dms trace=%s", component, status, latency, traceID)
	case "warn":
		return fmt.Sprintf("[%s] Load balancer backend degraded | status=%d latency=%dms trace=%s", component, status, latency, traceID)
	default:
		return fmt.Sprintf("[%s] Load balancer backend healthy | status=%d latency=%dms trace=%s", component, status, latency, traceID)
	}
}

func renderDNSMessage(severity, component string, status, latency int, traceID string) string {
	switch severity {
	case "error":
		return fmt.Sprintf("[%s] DNS resolution failed | status=%d latency=%dms trace=%s", component, status, latency, traceID)
	case "warn":
		return fmt.Sprintf("[%s] DNS resolution slow | status=%d latency=%dms trace=%s", component, status, latency, traceID)
	default:
		return fmt.Sprintf("[%s] DNS resolution completed | status=%d latency=%dms trace=%s", component, status, latency, traceID)
	}
}

func requestShape(service string, idx int) (method string, path string) {
	methods := []string{"GET", "POST", "PUT", "PATCH"}
	method = methods[idx%len(methods)]

	perService := map[string][]string{
		"checkout":           {"/api/checkout", "/api/checkout/order", "/api/payments/charge"},
		"svc-checkout":       {"/v2/checkout", "/v2/checkout/cart", "/v2/payments/capture"},
		"search":             {"/api/search", "/api/search/suggestions", "/api/search/trending"},
		"svc-search":         {"/v1/query", "/v1/query/trending", "/v1/query/top"},
		"web":                {"/", "/healthz", "/static/app.js"},
		"svc-web":            {"/app-shell", "/content/home", "/static/index.html"},
		"svc-identity":       {"/v1/tokens", "/v1/sessions", "/v1/rotations"},
		"svc-notifications":  {"/dispatch/email", "/dispatch/push", "/dispatch/sms"},
		"svc-realtime":       {"/socket/connect", "/socket/keepalive", "/socket/metrics"},
		"svc-cache":          {"/cache/get", "/cache/set", "/cache/delete", "/cache/flush"},
		"cache":              {"/api/cache/get", "/api/cache/set", "/api/cache/invalidate"},
		"svc-database":       {"/db/query", "/db/transaction", "/db/healthcheck"},
		"database":           {"/api/db/query", "/api/db/connection", "/api/db/pool"},
		"svc-api-gateway":    {"/gateway/route", "/gateway/auth", "/gateway/ratelimit"},
		"svc-loadbalancer":   {"/lb/health", "/lb/backend", "/lb/metrics"},
		"svc-dns":            {"/dns/resolve", "/dns/zone", "/dns/record"},
		"svc-payments":       {"/payments/charge", "/payments/refund", "/payments/webhook"},
		"payments":           {"/api/payments/process", "/api/payments/verify", "/api/payments/status"},
		"svc-order":          {"/orders/create", "/orders/update", "/orders/status"},
		"order":              {"/api/orders", "/api/orders/history", "/api/orders/cancel"},
		"svc-catalog":        {"/catalog/products", "/catalog/search", "/catalog/inventory"},
		"catalog":            {"/api/catalog", "/api/catalog/items", "/api/catalog/categories"},
		"svc-analytics":      {"/analytics/track", "/analytics/report", "/analytics/export"},
		"analytics":          {"/api/analytics/events", "/api/analytics/metrics", "/api/analytics/dashboard"},
		"svc-warehouse":      {"/warehouse/ingest", "/warehouse/query", "/warehouse/export"},
		"warehouse":          {"/api/warehouse/data", "/api/warehouse/schema", "/api/warehouse/jobs"},
		"svc-recommendation": {"/recommend/items", "/recommend/personalized", "/recommend/similar", "/recommend/trending"},
		"recommendation":     {"/api/recommend", "/api/recommend/user", "/api/recommend/product"},
	}
	paths, ok := perService[service]
	if !ok || len(paths) == 0 {
		paths = []string{"/api/demo", "/api/internal/metrics", "/healthz"}
	}
	path = paths[idx%len(paths)]
	return
}

func regionForService(service string, idx int) string {
	key := normalizeServiceName(service)
	perService := map[string][]string{
		"checkout":      {"use1", "euw1", "aps1"},
		"search":        {"use1", "usw2"},
		"web":           {"global"},
		"identity":      {"use1", "euw1"},
		"notifications": {"use1", "gcp-europe"},
		"realtime":      {"use1", "apse2"},
		"analytics":     {"use1", "apse1"},
		"warehouse":     {"usw2"},
	}
	regions := perService[key]
	if len(regions) == 0 {
		regions = []string{"use1", "euw1"}
	}
	return regions[idx%len(regions)]
}

func componentForService(service string, idx int) string {
	key := normalizeServiceName(service)
	perService := map[string][]string{
		"checkout":       {"payments", "cart", "promotions"},
		"search":         {"query", "ranker", "autosuggest"},
		"web":            {"cdn", "edge", "spa"},
		"identity":       {"oauth", "sessions", "mfa"},
		"notifications":  {"fanout", "renderer", "provider"},
		"realtime":       {"gateway", "presence", "subscriptions"},
		"analytics":      {"ingest", "aggregation", "export"},
		"warehouse":      {"loader", "compactor", "scheduler"},
		"cache":          {"redis-primary", "redis-replica", "eviction-manager"},
		"database":       {"query-engine", "connection-pool", "replication"},
		"api-gateway":    {"router", "auth-middleware", "rate-limiter"},
		"loadbalancer":   {"health-checker", "backend-pool", "traffic-manager"},
		"dns":            {"resolver", "zone-manager", "record-cache"},
		"payments":       {"processor", "fraud-detection", "settlement"},
		"order":          {"order-manager", "inventory-sync", "fulfillment"},
		"catalog":        {"product-index", "search-engine", "inventory-tracker"},
		"recommendation": {"collaborative-filter", "content-based", "ranking-engine"},
	}
	components := perService[key]
	if len(components) == 0 {
		components = []string{"api", "worker", "scheduler"}
	}
	return components[idx%len(components)]
}

func releaseForService(service string) string {
	switch normalizeServiceName(service) {
	case "checkout":
		return "checkout-v2.31.4"
	case "search":
		return "search-v5.12.0"
	case "web":
		return "web-8.2.0"
	case "identity":
		return "identity-v4.7.2"
	case "notifications":
		return "notify-v3.3.1"
	case "realtime":
		return "realtime-v1.18.0"
	case "recommendation":
		return "recommendation-v3.8.2"
	case "cache":
		return "redis-v7.0.5"
	case "database":
		return "postgres-v14.2"
	case "analytics":
		return "analytics-v2.4.1"
	case "warehouse":
		return "warehouse-v1.12.0"
	default:
		return "service-v1.0.0"
	}
}

func defaultTeamForService(service string) string {
	// Use the centralized service-to-team mapping
	return mockutil.GetTeamForService(service)
}

func featureFlagForService(service string, idx int) string {
	return fmt.Sprintf("%s-rollout-%d", normalizeServiceName(service), idx%4)
}

func customerForIndex(idx int) string {
	customers := []string{"enterprise", "growth", "consumer", "beta"}
	return customers[idx%len(customers)]
}

func normalizeServiceName(service string) string {
	if service == "" {
		return "service"
	}
	return strings.TrimPrefix(service, "svc-")
}

type logScenarioTemplate struct {
	keyword string
	summary string
	cause   string
	extra   map[string]any
}

var logScenarioTemplates = []logScenarioTemplate{
	{keyword: "timeout", summary: "Gateway timeout contacting upstream", cause: "upstream_timeout", extra: map[string]any{"timeoutMs": 9000, "upstream": "payments"}},
	{keyword: "payment", summary: "Payment processor error response", cause: "payment_provider_error", extra: map[string]any{"provider": "stripe", "errorCode": 504}},
	{keyword: "cache", summary: "Cache miss surge on redis-primary", cause: "cache_miss", extra: map[string]any{"cacheLayer": "redis-primary", "missRatio": 0.42}},
	{keyword: "auth", summary: "Authentication failures reported", cause: "auth_failure", extra: map[string]any{"status": 401, "provider": "identity"}},
	{keyword: "deploy", summary: "Recent deploy correlated with errors", cause: "deploy_regression", extra: map[string]any{"release": ""}},
	{keyword: "rollout", summary: "Feature rollout causing quality drop", cause: "rollout_regression", extra: map[string]any{"feature": "new-algorithm", "qualityDrop": "15%"}},
	{keyword: "quality", summary: "Recommendation quality degradation", cause: "quality_drop", extra: map[string]any{"metric": "precision", "baseline": 0.85, "current": 0.72}},
	{keyword: "lag", summary: "Consumer lag increasing", cause: "consumer_lag", extra: map[string]any{"lag": 8500}},
	{keyword: "latency", summary: "Latency regression observed", cause: "latency_regression", extra: map[string]any{"p99": 3800}},
	{keyword: "recommendation", summary: "Recommendation engine processing", cause: "recommendation_processing", extra: map[string]any{"algorithm": "collaborative-filtering", "itemsProcessed": 15000}},
}

func analyzeSearchContext(search, service, severity, component string) logInsight {
	lower := strings.ToLower(strings.TrimSpace(search))
	for _, tmpl := range logScenarioTemplates {
		if tmpl.keyword != "" && strings.Contains(lower, tmpl.keyword) {
			return buildInsightFromTemplate(tmpl, service)
		}
	}
	serviceLabel := fallback(service, "service")
	switch severity {
	case "error":
		return logInsight{Summary: fmt.Sprintf("%s %s returned error", component, serviceLabel), Cause: "generic_error", Tag: "error", Extra: map[string]any{"retry": true}}
	case "warn":
		return logInsight{Summary: fmt.Sprintf("%s degraded for %s", component, serviceLabel), Cause: "degraded", Tag: "warning", Extra: map[string]any{"slowPath": true}}
	default:
		return logInsight{Extra: map[string]any{"observation": fmt.Sprintf("%s activity", component)}}
	}
}

func buildInsightFromTemplate(tmpl logScenarioTemplate, service string) logInsight {
	extra := cloneExtraFields(tmpl.extra)
	if extra == nil {
		extra = map[string]any{}
	}
	if _, ok := extra["release"]; ok && extra["release"] == "" {
		extra["release"] = releaseForService(service)
	}
	if service != "" {
		extra["service"] = service
	}
	return logInsight{Summary: tmpl.summary, Cause: tmpl.cause, Tag: tmpl.keyword, Extra: extra}
}

func cloneExtraFields(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func responseStatus(severity string, idx int) int {
	if severity == "error" {
		return []int{500, 502, 504}[idx%3]
	}
	if severity == "warn" {
		return []int{200, 206, 429}[idx%3]
	}
	return []int{200, 200, 204, 304}[idx%4]
}

func promoteStatus(current int) int {
	if current >= 500 {
		return current
	}
	if current >= 400 {
		return 500
	}
	return 502
}

func escalateSeverity(current string) string {
	switch current {
	case "info":
		return "warn"
	case "warn":
		return "error"
	default:
		return current
	}
}

func baseLatency(severity string, idx int) int {
	base := 45 + idx%17
	switch severity {
	case "error":
		return base + 320
	case "warn":
		return base + 90
	default:
		return base + (idx % 5 * 5)
	}
}

func scopedLabels(q schema.LogQuery, service, region string) map[string]string {
	labels := map[string]string{"env": fallbackEnv(q)}
	if service != "" {
		labels["service"] = service
	}
	if q.Scope.Team != "" {
		labels["team"] = q.Scope.Team
	} else if team := defaultTeamForService(service); team != "" {
		labels["team"] = team
	}
	if region != "" {
		labels["region"] = region
	}
	return labels
}

func activeAlertsAt(ts time.Time, alerts []schema.Alert) []schema.Alert {
	if len(alerts) == 0 {
		return nil
	}
	active := make([]schema.Alert, 0, len(alerts))
	for _, al := range alerts {
		if ts.Before(al.CreatedAt) || ts.After(al.UpdatedAt.Add(10*time.Minute)) {
			continue
		}
		if al.Status != "firing" && al.Status != "acknowledged" {
			continue
		}
		active = append(active, al)
	}
	return active
}

func fallbackEnv(q schema.LogQuery) string {
	if q.Scope.Environment != "" {
		return q.Scope.Environment
	}
	return "prod"
}

func scopedMetadata(source string, q schema.LogQuery, start, end time.Time, service, region string, insight logInsight, alerts []schema.Alert) map[string]any {
	search := ""
	if q.Expression != nil {
		search = q.Expression.Search
	}

	metadata := map[string]any{
		"source":  source,
		"matched": strings.TrimSpace(search),
		"window":  map[string]string{"start": start.Format(time.RFC3339), "end": end.Format(time.RFC3339)},
	}
	if service != "" {
		metadata["service"] = service
	}
	if region != "" {
		metadata["region"] = region
	}
	if q.Scope != (schema.QueryScope{}) {
		metadata["scope"] = q.Scope
	}
	if q.Expression != nil && len(q.Expression.SeverityIn) > 0 {
		metadata["severityFilter"] = q.Expression.SeverityIn
	}
	if q.Scope.Team != "" {
		metadata["team"] = q.Scope.Team
	} else if team := defaultTeamForService(service); team != "" {
		metadata["team"] = team
	}
	if insight.Tag != "" {
		metadata["scenario"] = insight.Tag
	}
	if insight.Cause != "" {
		metadata["cause"] = insight.Cause
	}
	if insight.Summary != "" {
		metadata["insight"] = insight.Summary
	}
	if len(alerts) > 0 {
		metadata["alerts"] = mockutil.SummarizeAlerts(alerts)
	}
	return metadata
}

// generateLogsForQuery creates mock log entries that match the search query
func (p *Provider) generateLogsForQuery(parsed mockutil.ParsedQuery, query schema.LogQuery, service string, start, end time.Time, limit int) []schema.LogEntry {
	if limit <= 0 {
		limit = 20
	}
	if limit > 20 {
		limit = 20
	}

	// Determine service from scope or infer from query
	if service == "" {
		service = mockutil.InferServiceFromQuery(parsed)
	}
	if service == "" {
		service = "svc-cache" // Default for queries without clear service
	}

	// Determine environment
	environment := "prod"
	if query.Scope.Environment != "" {
		environment = query.Scope.Environment
	}

	// Determine team
	team := mockutil.GetTeamForService(service)
	if query.Scope.Team != "" {
		team = query.Scope.Team
	}

	// Determine severities to generate
	severities := []string{"error", "warn", "info"}
	if query.Expression != nil && len(query.Expression.SeverityIn) > 0 {
		severities = query.Expression.SeverityIn
	}

	// Generate log entries
	entries := make([]schema.LogEntry, 0, limit)
	step := end.Sub(start) / time.Duration(limit+1)
	if step <= 0 {
		step = 5 * time.Second
	}

	for i := 0; i < limit; i++ {
		ts := start.Add(time.Duration(i+1) * step)
		severity := severities[i%len(severities)]

		// Build message incorporating search terms
		message := buildLogMessage(parsed, service, severity, i)

		// Build fields
		fields := map[string]any{
			"requestId": fmt.Sprintf("req-gen-%06d", i+1),
			"traceId":   fmt.Sprintf("trace-gen-%05d", i+1),
			"component": componentForService(service, i),
			"region":    regionForService(service, i),
			"release":   releaseForService(service),
			"generated": true,
			"service":   service,
			"latencyMs": baseLatency(severity, i),
			"status":    responseStatus(severity, i),
		}

		// Add search terms to fields for better matching
		for idx, term := range parsed.Terms {
			if idx < 3 { // Limit to first 3 terms
				fields[fmt.Sprintf("term_%d", idx)] = term
			}
		}

		// Add specific fields based on search terms
		if mockutil.MatchesSearchQuery("redis", parsed) || mockutil.MatchesSearchQuery("cache", parsed) {
			fields["cacheHit"] = i%2 == 0
			fields["cacheLayer"] = "redis-primary"
		}
		if mockutil.MatchesSearchQuery("connection refused", parsed) {
			fields["error"] = "connection_refused"
			fields["errorCode"] = "ECONNREFUSED"
		}
		if mockutil.MatchesSearchQuery("oom", parsed) {
			fields["error"] = "out_of_memory"
			fields["errorCode"] = "OOM"
		}

		entry := schema.LogEntry{
			Timestamp: ts,
			Message:   message,
			Severity:  severity,
			Service:   service,
			URL:       generateLogURL(fmt.Sprintf("log-gen-%06d", i+1), service, ts),
			Labels: map[string]string{
				"env":     environment,
				"service": service,
				"team":    team,
				"region":  regionForService(service, i),
			},
			Fields: fields,
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"generated":   true,
				"searchTerms": parsed.Terms,
			},
		}

		entries = append(entries, entry)
	}

	return entries
}

// buildLogMessage creates a log message incorporating search terms
func buildLogMessage(parsed mockutil.ParsedQuery, service, severity string, index int) string {
	// Use first term or quoted phrase as primary content
	primaryTerm := ""
	if len(parsed.QuotedPhrases) > 0 {
		primaryTerm = parsed.QuotedPhrases[0]
	} else if len(parsed.Terms) > 0 {
		primaryTerm = parsed.Terms[0]
	}

	if primaryTerm == "" {
		primaryTerm = "service operation"
	}

	// Build message based on severity and search terms
	component := componentForService(service, index)
	traceID := fmt.Sprintf("trace-gen-%05d", index+1)

	switch severity {
	case "error":
		return fmt.Sprintf("[%s] %s error: %s | service=%s trace=%s", component, cases.Title(language.English).String(primaryTerm), buildErrorDetail(parsed), service, traceID)
	case "warn":
		return fmt.Sprintf("[%s] %s warning: performance degradation detected | service=%s trace=%s", component, cases.Title(language.English).String(primaryTerm), service, traceID)
	default:
		return fmt.Sprintf("[%s] %s operation completed | service=%s trace=%s", component, cases.Title(language.English).String(primaryTerm), service, traceID)
	}
}

// buildErrorDetail creates error details from search terms
func buildErrorDetail(parsed mockutil.ParsedQuery) string {
	if len(parsed.QuotedPhrases) > 0 {
		return parsed.QuotedPhrases[0]
	}
	if len(parsed.Terms) > 1 {
		return strings.Join(parsed.Terms, " ")
	}
	if len(parsed.Terms) > 0 {
		return parsed.Terms[0] + " failure"
	}
	return "operation failed"
}

// enhanceFieldsWithSearchTerms adds search terms to log fields
func enhanceFieldsWithSearchTerms(fields map[string]any, parsed mockutil.ParsedQuery) {
	// Add search terms to fields for better matching
	for idx, term := range parsed.Terms {
		if idx < 3 { // Limit to first 3 terms
			fields[fmt.Sprintf("searchTerm_%d", idx)] = term
		}
	}

	// Add specific fields based on search terms
	for _, term := range parsed.Terms {
		switch term {
		case "redis", "cache":
			fields["cacheLayer"] = "redis-primary"
			fields["cacheOperation"] = "get"
		case "connection":
			fields["connectionPool"] = "active"
		case "oom":
			fields["memoryUsage"] = "high"
		case "refused":
			fields["connectionStatus"] = "refused"
		}
	}

	// Handle quoted phrases
	for _, phrase := range parsed.QuotedPhrases {
		if strings.Contains(phrase, "connection refused") {
			fields["error"] = "connection_refused"
			fields["errorCode"] = "ECONNREFUSED"
		}
		if strings.Contains(phrase, "oom") {
			fields["error"] = "out_of_memory"
			fields["errorCode"] = "OOM"
		}
	}
}

// enhanceMessageWithSearchTerms modifies a message to include search terms
func enhanceMessageWithSearchTerms(message string, parsed mockutil.ParsedQuery, service, component, traceID string) string {
	// Use first term or quoted phrase as primary content
	primaryTerm := ""
	if len(parsed.QuotedPhrases) > 0 {
		primaryTerm = parsed.QuotedPhrases[0]
	} else if len(parsed.Terms) > 0 {
		primaryTerm = parsed.Terms[0]
	}

	if primaryTerm == "" {
		return message
	}

	// Build a new message that includes the search term
	return fmt.Sprintf("[%s] %s: %s | service=%s trace=%s", component, cases.Title(language.English).String(primaryTerm), message, service, traceID)
}

// getScenarioLogs returns static scenario-themed log entries
func getScenarioLogs(now time.Time) []schema.LogEntry {
	return []schema.LogEntry{
		{
			Timestamp: now.Add(-25 * time.Minute),
			Message:   "POST /api/checkout/order (payments) -> 504 in 9012ms | svc-checkout request failed for user=alice trace=trace-scenario-001",
			Severity:  "error",
			Service:   "svc-checkout",
			URL:       generateLogURL("log-scenario-001", "svc-checkout", now.Add(-25*time.Minute)),
			Labels: map[string]string{
				"env":     "prod",
				"service": "svc-checkout",
				"team":    "team-velocity",
				"region":  "use1",
			},
			Fields: map[string]any{
				"requestId":      "req-scenario-001",
				"path":           "/api/checkout/order",
				"method":         "POST",
				"status":         504,
				"latencyMs":      9012,
				"traceId":        "trace-scenario-001",
				"user":           "alice",
				"component":      "payments",
				"instance":       "payments-use1-01",
				"region":         "use1",
				"release":        "checkout-v2.31.4",
				"error":          "payments_upstream_failure",
				"errorCode":      "GATEWAY_TIMEOUT",
				"retryable":      true,
				"attempts":       3,
				"stackTrace":     "at PaymentGateway.charge (gateway.go:142)\nat OrderService.processPayment (order.go:89)\nat CheckoutHandler.handleOrder (handler.go:56)",
				"scenario_id":    "scenario-001",
				"scenario_name":  "SLO Budget Exhaustion",
				"scenario_stage": "active",
				"is_scenario":    true,
			},
			Metadata: map[string]any{
				"source":         "mock-log",
				"scenario_id":    "scenario-001",
				"scenario_name":  "SLO Budget Exhaustion",
				"scenario_stage": "active",
				"cause":          "upstream_timeout",
				"insight":        "Payment gateway timeout during SLO budget exhaustion",
			},
		},
		{
			Timestamp: now.Add(-20 * time.Minute),
			Message:   "GET /api/search (query) -> 500 in 2845ms | svc-search request failed for user=sam trace=trace-scenario-002",
			Severity:  "error",
			Service:   "svc-search",
			URL:       generateLogURL("log-scenario-002", "svc-search", now.Add(-20*time.Minute)),
			Labels: map[string]string{
				"env":     "prod",
				"service": "svc-search",
				"team":    "team-aurora",
				"region":  "use1",
			},
			Fields: map[string]any{
				"requestId":      "req-scenario-002",
				"path":           "/api/search",
				"method":         "GET",
				"status":         500,
				"latencyMs":      2845,
				"traceId":        "trace-scenario-002",
				"user":           "sam",
				"component":      "query",
				"instance":       "query-use1-02",
				"region":         "use1",
				"release":        "search-v5.12.0",
				"error":          "database_connection_pool_exhausted",
				"errorCode":      "DB_POOL_EXHAUSTED",
				"retryable":      false,
				"attempts":       1,
				"stackTrace":     "at QueryExecutor.execute (executor.go:234)\nat SearchService.query (search.go:156)\nat SearchHandler.handleQuery (handler.go:78)",
				"dbConnections":  "100/100",
				"scenario_id":    "scenario-002",
				"scenario_name":  "Cascading Database Failure",
				"scenario_stage": "escalating",
				"is_scenario":    true,
			},
			Metadata: map[string]any{
				"source":         "mock-log",
				"scenario_id":    "scenario-002",
				"scenario_name":  "Cascading Database Failure",
				"scenario_stage": "escalating",
				"cause":          "db_pool_exhausted",
				"insight":        "Database connection pool exhausted during cascading failure",
			},
		},
		{
			Timestamp: now.Add(-15 * time.Minute),
			Message:   "POST /api/payments/charge (payments) -> 502 in 5234ms | svc-checkout request failed for user=casey trace=trace-scenario-003",
			Severity:  "error",
			Service:   "svc-checkout",
			URL:       generateLogURL("log-scenario-003", "svc-checkout", now.Add(-15*time.Minute)),
			Labels: map[string]string{
				"env":     "prod",
				"service": "svc-checkout",
				"team":    "team-velocity",
				"region":  "euw1",
			},
			Fields: map[string]any{
				"requestId":      "req-scenario-003",
				"path":           "/api/payments/charge",
				"method":         "POST",
				"status":         502,
				"latencyMs":      5234,
				"traceId":        "trace-scenario-003",
				"user":           "casey",
				"component":      "payments",
				"instance":       "payments-euw1-01",
				"region":         "euw1",
				"release":        "checkout-v2.31.3",
				"error":          "deployment_rollback_triggered",
				"errorCode":      "BAD_GATEWAY",
				"retryable":      true,
				"attempts":       2,
				"stackTrace":     "at PaymentProcessor.process (processor.go:189)\nat CheckoutService.charge (checkout.go:234)\nat PaymentHandler.handleCharge (handler.go:92)",
				"deploymentId":   "deploy-2024-12-07-003",
				"rollbackReason": "error_rate_threshold_exceeded",
				"scenario_id":    "scenario-003",
				"scenario_name":  "Deployment Rollback",
				"scenario_stage": "mitigating",
				"is_scenario":    true,
			},
			Metadata: map[string]any{
				"source":         "mock-log",
				"scenario_id":    "scenario-003",
				"scenario_name":  "Deployment Rollback",
				"scenario_stage": "mitigating",
				"cause":          "deploy_regression",
				"insight":        "Payment service errors triggered deployment rollback",
			},
		},
		{
			Timestamp: now.Add(-12 * time.Minute),
			Message:   "GET /api/checkout (cart) -> 503 in 8567ms | svc-checkout request failed for user=fern trace=trace-scenario-004",
			Severity:  "error",
			Service:   "svc-checkout",
			Labels: map[string]string{
				"env":     "prod",
				"service": "svc-checkout",
				"team":    "team-velocity",
				"region":  "use1",
			},
			Fields: map[string]any{
				"requestId":       "req-scenario-004",
				"path":            "/api/checkout",
				"method":          "GET",
				"status":          503,
				"latencyMs":       8567,
				"traceId":         "trace-scenario-004",
				"user":            "fern",
				"component":       "cart",
				"instance":        "cart-use1-03",
				"region":          "use1",
				"release":         "checkout-v2.31.4",
				"error":           "external_dependency_failure",
				"errorCode":       "SERVICE_UNAVAILABLE",
				"retryable":       true,
				"attempts":        3,
				"stackTrace":      "at StripeClient.charge (stripe.go:78)\nat PaymentService.processCharge (payment.go:145)\nat CartHandler.checkout (handler.go:112)",
				"externalService": "stripe",
				"externalError":   "rate_limit_exceeded",
				"scenario_id":     "scenario-004",
				"scenario_name":   "External Dependency Failure - Stripe",
				"scenario_stage":  "active",
				"is_scenario":     true,
			},
			Metadata: map[string]any{
				"source":         "mock-log",
				"scenario_id":    "scenario-004",
				"scenario_name":  "External Dependency Failure - Stripe",
				"scenario_stage": "active",
				"cause":          "external_dependency_failure",
				"insight":        "Stripe API rate limit exceeded causing checkout failures",
			},
		},
		{
			Timestamp: now.Add(-8 * time.Minute),
			Message:   "GET /api/search/suggestions (autosuggest) -> 429 in 1234ms | svc-search degraded: matched='autoscaling lag'",
			Severity:  "warn",
			Service:   "svc-search",
			Labels: map[string]string{
				"env":     "prod",
				"service": "svc-search",
				"team":    "team-aurora",
				"region":  "usw2",
			},
			Fields: map[string]any{
				"requestId":         "req-scenario-005",
				"path":              "/api/search/suggestions",
				"method":            "GET",
				"status":            429,
				"latencyMs":         1234,
				"traceId":           "trace-scenario-005",
				"user":              "lena",
				"component":         "autosuggest",
				"instance":          "autosuggest-usw2-01",
				"region":            "usw2",
				"release":           "search-v5.12.0",
				"cacheHit":          false,
				"slowPath":          true,
				"autoscalingStatus": "scaling_up",
				"currentInstances":  3,
				"targetInstances":   8,
				"scenario_id":       "scenario-005",
				"scenario_name":     "Autoscaling Lag",
				"scenario_stage":    "active",
				"is_scenario":       true,
			},
			Metadata: map[string]any{
				"source":         "mock-log",
				"scenario_id":    "scenario-005",
				"scenario_name":  "Autoscaling Lag",
				"scenario_stage": "active",
				"cause":          "autoscaling_lag",
				"insight":        "Rate limiting due to autoscaling lag",
			},
		},
		{
			Timestamp: now.Add(-5 * time.Minute),
			Message:   "POST /api/checkout/order (payments) -> 500 in 4567ms | svc-checkout request failed for user=milo trace=trace-scenario-006",
			Severity:  "error",
			Service:   "svc-checkout",
			Labels: map[string]string{
				"env":     "prod",
				"service": "svc-checkout",
				"team":    "team-velocity",
				"region":  "aps1",
			},
			Fields: map[string]any{
				"requestId":         "req-scenario-006",
				"path":              "/api/checkout/order",
				"method":            "POST",
				"status":            500,
				"latencyMs":         4567,
				"traceId":           "trace-scenario-006",
				"user":              "milo",
				"component":         "payments",
				"instance":          "payments-aps1-02",
				"region":            "aps1",
				"release":           "checkout-v2.31.4",
				"error":             "circuit_breaker_cascade",
				"errorCode":         "CIRCUIT_OPEN",
				"retryable":         false,
				"attempts":          1,
				"stackTrace":        "at CircuitBreaker.execute (breaker.go:156)\nat PaymentService.charge (payment.go:234)\nat OrderHandler.processOrder (handler.go:89)",
				"circuitState":      "open",
				"failureThreshold":  "5/10",
				"downstreamService": "payment-gateway",
				"scenario_id":       "scenario-006",
				"scenario_name":     "Circuit Breaker Cascade",
				"scenario_stage":    "escalating",
				"is_scenario":       true,
			},
			Metadata: map[string]any{
				"source":         "mock-log",
				"scenario_id":    "scenario-006",
				"scenario_name":  "Circuit Breaker Cascade",
				"scenario_stage": "escalating",
				"cause":          "circuit_breaker_cascade",
				"insight":        "Circuit breaker opened due to cascading failures",
			},
		},
	}
}

// ensure compile-time interface compatibility
var _ log.Provider = (*Provider)(nil)
