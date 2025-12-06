package mockutil

import (
	"strings"
)

// ParsedQuery represents a parsed search query with extracted terms
type ParsedQuery struct {
	Terms         []string // Individual search terms (lowercased)
	QuotedPhrases []string // Exact phrases in quotes (lowercased)
	IsORQuery     bool     // Whether query uses OR operator
}

// ParseSearchQuery parses a search query string and extracts terms, quoted phrases, and OR operators
func ParseSearchQuery(query string) ParsedQuery {
	if query == "" {
		return ParsedQuery{}
	}

	result := ParsedQuery{
		Terms:         make([]string, 0),
		QuotedPhrases: make([]string, 0),
	}

	// Check if this is an OR query
	result.IsORQuery = strings.Contains(strings.ToUpper(query), " OR ")

	// Extract quoted phrases first
	remaining := query
	for {
		startQuote := strings.Index(remaining, "\"")
		if startQuote == -1 {
			break
		}
		endQuote := strings.Index(remaining[startQuote+1:], "\"")
		if endQuote == -1 {
			// Unmatched quote - treat rest as literal
			break
		}
		endQuote += startQuote + 1

		phrase := remaining[startQuote+1 : endQuote]
		if phrase != "" {
			result.QuotedPhrases = append(result.QuotedPhrases, strings.ToLower(strings.TrimSpace(phrase)))
		}

		// Remove the quoted phrase from remaining text
		remaining = remaining[:startQuote] + " " + remaining[endQuote+1:]
	}

	// Split remaining text by OR operator
	var parts []string
	if result.IsORQuery {
		parts = strings.Split(remaining, " OR ")
	} else {
		parts = []string{remaining}
	}

	// Extract individual terms from each part
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by whitespace to get individual words
		words := strings.Fields(part)
		for _, word := range words {
			word = strings.ToLower(strings.TrimSpace(word))
			if word != "" && word != "or" {
				result.Terms = append(result.Terms, word)
			}
		}
	}

	return result
}

// MatchesSearchQuery checks if a text contains any of the search terms or quoted phrases
func MatchesSearchQuery(text string, parsed ParsedQuery) bool {
	if len(parsed.Terms) == 0 && len(parsed.QuotedPhrases) == 0 {
		return true
	}

	lowerText := strings.ToLower(text)

	// Check quoted phrases first (exact match required)
	for _, phrase := range parsed.QuotedPhrases {
		if strings.Contains(lowerText, phrase) {
			return true
		}
	}

	// Check individual terms
	for _, term := range parsed.Terms {
		if strings.Contains(lowerText, term) {
			return true
		}
	}

	return false
}

// InferServiceFromQuery attempts to extract a service name from search terms
func InferServiceFromQuery(parsed ParsedQuery) string {
	// Common service keywords
	serviceKeywords := map[string]string{
		"redis":          "svc-cache",
		"cache":          "svc-cache",
		"checkout":       "svc-checkout",
		"payment":        "svc-payments",
		"payments":       "svc-payments",
		"search":         "svc-search",
		"database":       "svc-database",
		"db":             "svc-database",
		"postgres":       "svc-database",
		"notification":   "svc-notifications",
		"notifications":  "svc-notifications",
		"identity":       "svc-identity",
		"auth":           "svc-identity",
		"warehouse":      "svc-warehouse",
		"recommendation": "svc-recommendation",
		"analytics":      "svc-analytics",
		"order":          "svc-order",
		"orders":         "svc-order",
		"catalog":        "svc-catalog",
		"realtime":       "svc-realtime",
		"websocket":      "svc-realtime",
		"web":            "svc-web",
		"api":            "svc-api-gateway",
		"gateway":        "svc-api-gateway",
		"dns":            "svc-dns",
		"loadbalancer":   "svc-loadbalancer",
		"kafka":          "svc-notifications",
	}

	// Check all terms and quoted phrases
	allTerms := append(parsed.Terms, parsed.QuotedPhrases...)
	for _, term := range allTerms {
		if service, ok := serviceKeywords[term]; ok {
			return service
		}
	}

	return ""
}
