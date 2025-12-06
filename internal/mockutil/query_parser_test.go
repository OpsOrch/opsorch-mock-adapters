package mockutil

import (
	"testing"
)

func TestParseSearchQuery_Empty(t *testing.T) {
	result := ParseSearchQuery("")
	if len(result.Terms) != 0 || len(result.QuotedPhrases) != 0 {
		t.Errorf("Expected empty result for empty query")
	}
}

func TestParseSearchQuery_SimpleTerms(t *testing.T) {
	result := ParseSearchQuery("redis cache")
	if len(result.Terms) != 2 {
		t.Errorf("Expected 2 terms, got %d", len(result.Terms))
	}
	if result.Terms[0] != "redis" || result.Terms[1] != "cache" {
		t.Errorf("Expected [redis, cache], got %v", result.Terms)
	}
	if result.IsORQuery {
		t.Errorf("Expected IsORQuery to be false")
	}
}

func TestParseSearchQuery_OROperator(t *testing.T) {
	result := ParseSearchQuery("redis OR cache OR database")
	if !result.IsORQuery {
		t.Errorf("Expected IsORQuery to be true")
	}
	if len(result.Terms) != 3 {
		t.Errorf("Expected 3 terms, got %d: %v", len(result.Terms), result.Terms)
	}
	expectedTerms := map[string]bool{"redis": true, "cache": true, "database": true}
	for _, term := range result.Terms {
		if !expectedTerms[term] {
			t.Errorf("Unexpected term: %s", term)
		}
	}
}

func TestParseSearchQuery_QuotedPhrase(t *testing.T) {
	result := ParseSearchQuery(`"Connection refused"`)
	if len(result.QuotedPhrases) != 1 {
		t.Errorf("Expected 1 quoted phrase, got %d", len(result.QuotedPhrases))
	}
	if result.QuotedPhrases[0] != "connection refused" {
		t.Errorf("Expected 'connection refused', got '%s'", result.QuotedPhrases[0])
	}
}

func TestParseSearchQuery_MixedORAndQuotes(t *testing.T) {
	result := ParseSearchQuery(`redis OR "Connection refused" OR "OOM command not allowed"`)
	if !result.IsORQuery {
		t.Errorf("Expected IsORQuery to be true")
	}
	if len(result.QuotedPhrases) != 2 {
		t.Errorf("Expected 2 quoted phrases, got %d", len(result.QuotedPhrases))
	}
	if len(result.Terms) != 1 {
		t.Errorf("Expected 1 term (redis), got %d: %v", len(result.Terms), result.Terms)
	}
	if result.Terms[0] != "redis" {
		t.Errorf("Expected 'redis', got '%s'", result.Terms[0])
	}
}

func TestParseSearchQuery_UnmatchedQuote(t *testing.T) {
	result := ParseSearchQuery(`redis "unclosed quote`)
	// Should handle gracefully - extract what we can
	if len(result.Terms) == 0 {
		t.Errorf("Expected at least 'redis' term")
	}
}

func TestMatchesSearchQuery_SimpleTerm(t *testing.T) {
	parsed := ParseSearchQuery("redis")
	if !MatchesSearchQuery("Redis cache hit rate degradation", parsed) {
		t.Errorf("Expected match for 'redis' in text")
	}
	if MatchesSearchQuery("Database connection failed", parsed) {
		t.Errorf("Expected no match for 'redis' in text without redis")
	}
}

func TestMatchesSearchQuery_QuotedPhrase(t *testing.T) {
	parsed := ParseSearchQuery(`"Connection refused"`)
	if !MatchesSearchQuery("Error: Connection refused by server", parsed) {
		t.Errorf("Expected match for quoted phrase")
	}
	if MatchesSearchQuery("Connection to server failed", parsed) {
		t.Errorf("Expected no match - 'refused' is missing")
	}
}

func TestMatchesSearchQuery_OROperator(t *testing.T) {
	parsed := ParseSearchQuery("redis OR cache OR database")
	if !MatchesSearchQuery("Cache hit rate low", parsed) {
		t.Errorf("Expected match for 'cache'")
	}
	if !MatchesSearchQuery("Redis connection timeout", parsed) {
		t.Errorf("Expected match for 'redis'")
	}
	if !MatchesSearchQuery("Database query slow", parsed) {
		t.Errorf("Expected match for 'database'")
	}
	if MatchesSearchQuery("Network error occurred", parsed) {
		t.Errorf("Expected no match - no matching terms")
	}
}

func TestInferServiceFromQuery_Redis(t *testing.T) {
	parsed := ParseSearchQuery("redis OR cache")
	service := InferServiceFromQuery(parsed)
	if service != "svc-cache" {
		t.Errorf("Expected 'svc-cache', got '%s'", service)
	}
}

func TestInferServiceFromQuery_Payment(t *testing.T) {
	parsed := ParseSearchQuery("payment error")
	service := InferServiceFromQuery(parsed)
	if service != "svc-payments" {
		t.Errorf("Expected 'svc-payments', got '%s'", service)
	}
}

func TestInferServiceFromQuery_NoMatch(t *testing.T) {
	parsed := ParseSearchQuery("unknown service error")
	service := InferServiceFromQuery(parsed)
	if service != "" {
		t.Errorf("Expected empty string, got '%s'", service)
	}
}
