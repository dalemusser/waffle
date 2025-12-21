// pantry/query/query.go
package query

import (
	"net/http"
	"strings"
)

// MaxSearchLen is the default maximum length for search query parameters.
const MaxSearchLen = 200

// Get returns the trimmed value of a query parameter.
// Returns an empty string if the parameter is missing or blank.
func Get(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

// GetMax returns the trimmed value of a query parameter, truncated to maxLen.
// Returns an empty string if the parameter is missing or blank.
func GetMax(r *http.Request, key string, maxLen int) string {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if len(v) > maxLen {
		return v[:maxLen]
	}
	return v
}

// Search returns a search query parameter, truncated to MaxSearchLen.
// Use this for search/filter inputs that will be used in database queries.
func Search(r *http.Request, key string) string {
	return GetMax(r, key, MaxSearchLen)
}
