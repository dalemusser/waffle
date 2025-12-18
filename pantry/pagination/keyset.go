// pagination/keyset.go
package pagination

import (
	"fmt"
	"net/http"
	"strings"
)

// Keyset represents keyset-based pagination (seek method).
// More efficient than offset pagination for large datasets.
type Keyset struct {
	// Limit is the number of items to return
	Limit int

	// After is the value to start after (for forward pagination)
	After string

	// Before is the value to start before (for backward pagination)
	Before string

	// OrderBy is the column to order by
	OrderBy string

	// Descending indicates sort order
	Descending bool
}

// KeysetFromRequest extracts keyset pagination from request.
// Supports: ?limit=20&after=xxx or ?limit=20&before=xxx
func KeysetFromRequest(r *http.Request) Keyset {
	q := r.URL.Query()

	limit := parseInt(q.Get("limit"), DefaultPerPage)
	if limit > MaxPerPage {
		limit = MaxPerPage
	}

	return Keyset{
		Limit:      limit,
		After:      q.Get("after"),
		Before:     q.Get("before"),
		OrderBy:    q.Get("order_by"),
		Descending: q.Get("order") == "desc",
	}
}

// IsForward returns true if paginating forward.
func (k Keyset) IsForward() bool {
	return k.Before == "" || k.After != ""
}

// IsBackward returns true if paginating backward.
func (k Keyset) IsBackward() bool {
	return k.Before != "" && k.After == ""
}

// IsFirst returns true if this is the first page.
func (k Keyset) IsFirst() bool {
	return k.After == "" && k.Before == ""
}

// KeysetResult wraps keyset-paginated data.
type KeysetResult[T any] struct {
	Data    []T    `json:"data"`
	HasMore bool   `json:"has_more"`
	StartID string `json:"start_id,omitempty"`
	EndID   string `json:"end_id,omitempty"`
}

// NewKeysetResult creates a keyset result.
// firstID and lastID are the IDs of the first and last items in data.
func NewKeysetResult[T any](data []T, hasMore bool, firstID, lastID string) KeysetResult[T] {
	return KeysetResult[T]{
		Data:    data,
		HasMore: hasMore,
		StartID: firstID,
		EndID:   lastID,
	}
}

// KeysetLinks contains keyset pagination links.
type KeysetLinks struct {
	Self string `json:"self,omitempty"`
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

// KeysetResultWithLinks wraps keyset data with navigation links.
type KeysetResultWithLinks[T any] struct {
	Data    []T         `json:"data"`
	HasMore bool        `json:"has_more"`
	Links   KeysetLinks `json:"links"`
}

// NewKeysetResultWithLinks creates a keyset result with links.
func NewKeysetResultWithLinks[T any](data []T, k Keyset, hasMore bool, firstID, lastID, baseURL string) KeysetResultWithLinks[T] {
	links := KeysetLinks{
		Self: buildKeysetURL(baseURL, k.Limit, k.After, k.Before),
	}

	if hasMore && lastID != "" {
		links.Next = buildKeysetURL(baseURL, k.Limit, lastID, "")
	}
	if firstID != "" && !k.IsFirst() {
		links.Prev = buildKeysetURL(baseURL, k.Limit, "", firstID)
	}

	return KeysetResultWithLinks[T]{
		Data:    data,
		HasMore: hasMore,
		Links:   links,
	}
}

// buildKeysetURL builds a keyset pagination URL.
func buildKeysetURL(baseURL string, limit int, after, before string) string {
	sep := "?"
	if strings.Contains(baseURL, "?") {
		sep = "&"
	}

	url := fmt.Sprintf("%s%slimit=%d", baseURL, sep, limit)
	if after != "" {
		url += "&after=" + after
	}
	if before != "" {
		url += "&before=" + before
	}
	return url
}

// SetKeysetLinkHeader sets Link header for keyset pagination.
func SetKeysetLinkHeader(w http.ResponseWriter, k Keyset, hasMore bool, firstID, lastID, baseURL string) {
	var links []string

	// Next
	if hasMore && lastID != "" {
		links = append(links, fmt.Sprintf(`<%s>; rel="next"`, buildKeysetURL(baseURL, k.Limit, lastID, "")))
	}

	// Prev
	if firstID != "" && !k.IsFirst() {
		links = append(links, fmt.Sprintf(`<%s>; rel="prev"`, buildKeysetURL(baseURL, k.Limit, "", firstID)))
	}

	if len(links) > 0 {
		w.Header().Set("Link", strings.Join(links, ", "))
	}
}

// SQLBuilder helps build keyset pagination SQL clauses.
type SQLBuilder struct {
	Column     string
	Descending bool
	Placeholder string // e.g., "$1" for postgres, "?" for mysql
}

// NewSQLBuilder creates a SQL builder for keyset pagination.
func NewSQLBuilder(column string, descending bool) *SQLBuilder {
	return &SQLBuilder{
		Column:     column,
		Descending: descending,
		Placeholder: "?",
	}
}

// WithPlaceholder sets the SQL placeholder style.
func (b *SQLBuilder) WithPlaceholder(p string) *SQLBuilder {
	b.Placeholder = p
	return b
}

// WhereClause returns the WHERE clause for keyset pagination.
func (b *SQLBuilder) WhereClause(k Keyset) (clause string, args []any) {
	if k.After != "" {
		if b.Descending {
			return fmt.Sprintf("%s < %s", b.Column, b.Placeholder), []any{k.After}
		}
		return fmt.Sprintf("%s > %s", b.Column, b.Placeholder), []any{k.After}
	}
	if k.Before != "" {
		if b.Descending {
			return fmt.Sprintf("%s > %s", b.Column, b.Placeholder), []any{k.Before}
		}
		return fmt.Sprintf("%s < %s", b.Column, b.Placeholder), []any{k.Before}
	}
	return "", nil
}

// OrderClause returns the ORDER BY clause.
func (b *SQLBuilder) OrderClause() string {
	if b.Descending {
		return fmt.Sprintf("%s DESC", b.Column)
	}
	return fmt.Sprintf("%s ASC", b.Column)
}

// LimitClause returns the LIMIT clause.
func (b *SQLBuilder) LimitClause(k Keyset) string {
	// Fetch one extra to check if there are more
	return fmt.Sprintf("LIMIT %d", k.Limit+1)
}

// ProcessResults checks if there are more results and trims the extra item.
func ProcessResults[T any](data []T, limit int) ([]T, bool) {
	if len(data) > limit {
		return data[:limit], true
	}
	return data, false
}
