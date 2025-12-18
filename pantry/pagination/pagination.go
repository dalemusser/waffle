// pagination/pagination.go
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Default pagination values.
const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// Page represents offset-based pagination parameters.
type Page struct {
	// Page number (1-indexed)
	Page int `json:"page"`

	// Items per page
	PerPage int `json:"per_page"`

	// Total items (set after query)
	Total int `json:"total,omitempty"`

	// Total pages (calculated from Total and PerPage)
	TotalPages int `json:"total_pages,omitempty"`
}

// Offset returns the offset for database queries.
func (p Page) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

// Limit returns the limit for database queries.
func (p Page) Limit() int {
	return p.PerPage
}

// HasNext returns true if there are more pages.
func (p Page) HasNext() bool {
	if p.Total == 0 {
		return false
	}
	return p.Page < p.TotalPages
}

// HasPrev returns true if there are previous pages.
func (p Page) HasPrev() bool {
	return p.Page > 1
}

// Next returns the next page number.
func (p Page) Next() int {
	if !p.HasNext() {
		return p.Page
	}
	return p.Page + 1
}

// Prev returns the previous page number.
func (p Page) Prev() int {
	if !p.HasPrev() {
		return p.Page
	}
	return p.Page - 1
}

// SetTotal sets the total count and calculates total pages.
func (p *Page) SetTotal(total int) {
	p.Total = total
	if p.PerPage > 0 {
		p.TotalPages = (total + p.PerPage - 1) / p.PerPage
	}
}

// New creates pagination from page number and per-page count.
func New(page, perPage int) Page {
	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	return Page{
		Page:    page,
		PerPage: perPage,
	}
}

// Default returns default pagination.
func Default() Page {
	return New(DefaultPage, DefaultPerPage)
}

// FromRequest extracts pagination from HTTP request query parameters.
// Supports: ?page=1&per_page=20 or ?page=1&limit=20
func FromRequest(r *http.Request) Page {
	q := r.URL.Query()

	page := parseInt(q.Get("page"), DefaultPage)
	perPage := parseInt(q.Get("per_page"), 0)
	if perPage == 0 {
		perPage = parseInt(q.Get("limit"), DefaultPerPage)
	}

	return New(page, perPage)
}

// FromRequestWithDefaults extracts pagination with custom defaults.
func FromRequestWithDefaults(r *http.Request, defaultPerPage, maxPerPage int) Page {
	q := r.URL.Query()

	page := parseInt(q.Get("page"), DefaultPage)
	perPage := parseInt(q.Get("per_page"), 0)
	if perPage == 0 {
		perPage = parseInt(q.Get("limit"), defaultPerPage)
	}

	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	return New(page, perPage)
}

// parseInt parses a string to int with a default value.
func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return def
	}
	return n
}

// Result wraps paginated data with metadata.
type Result[T any] struct {
	Data       []T  `json:"data"`
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// NewResult creates a paginated result.
func NewResult[T any](data []T, p Page) Result[T] {
	return Result[T]{
		Data:       data,
		Page:       p.Page,
		PerPage:    p.PerPage,
		Total:      p.Total,
		TotalPages: p.TotalPages,
		HasNext:    p.HasNext(),
		HasPrev:    p.HasPrev(),
	}
}

// Links contains pagination link URLs.
type Links struct {
	Self  string `json:"self,omitempty"`
	First string `json:"first,omitempty"`
	Last  string `json:"last,omitempty"`
	Next  string `json:"next,omitempty"`
	Prev  string `json:"prev,omitempty"`
}

// ResultWithLinks wraps paginated data with links.
type ResultWithLinks[T any] struct {
	Data       []T   `json:"data"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int   `json:"total"`
	TotalPages int   `json:"total_pages"`
	Links      Links `json:"links"`
}

// NewResultWithLinks creates a paginated result with navigation links.
func NewResultWithLinks[T any](data []T, p Page, baseURL string) ResultWithLinks[T] {
	links := Links{
		Self:  buildURL(baseURL, p.Page, p.PerPage),
		First: buildURL(baseURL, 1, p.PerPage),
	}

	if p.TotalPages > 0 {
		links.Last = buildURL(baseURL, p.TotalPages, p.PerPage)
	}
	if p.HasNext() {
		links.Next = buildURL(baseURL, p.Next(), p.PerPage)
	}
	if p.HasPrev() {
		links.Prev = buildURL(baseURL, p.Prev(), p.PerPage)
	}

	return ResultWithLinks[T]{
		Data:       data,
		Page:       p.Page,
		PerPage:    p.PerPage,
		Total:      p.Total,
		TotalPages: p.TotalPages,
		Links:      links,
	}
}

// buildURL builds a pagination URL.
func buildURL(baseURL string, page, perPage int) string {
	sep := "?"
	if strings.Contains(baseURL, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%spage=%d&per_page=%d", baseURL, sep, page, perPage)
}

// SetLinkHeader sets the Link header for pagination (RFC 5988).
func SetLinkHeader(w http.ResponseWriter, p Page, baseURL string) {
	var links []string

	// First
	links = append(links, fmt.Sprintf(`<%s>; rel="first"`, buildURL(baseURL, 1, p.PerPage)))

	// Prev
	if p.HasPrev() {
		links = append(links, fmt.Sprintf(`<%s>; rel="prev"`, buildURL(baseURL, p.Prev(), p.PerPage)))
	}

	// Next
	if p.HasNext() {
		links = append(links, fmt.Sprintf(`<%s>; rel="next"`, buildURL(baseURL, p.Next(), p.PerPage)))
	}

	// Last
	if p.TotalPages > 0 {
		links = append(links, fmt.Sprintf(`<%s>; rel="last"`, buildURL(baseURL, p.TotalPages, p.PerPage)))
	}

	w.Header().Set("Link", strings.Join(links, ", "))
	w.Header().Set("X-Total-Count", strconv.Itoa(p.Total))
	w.Header().Set("X-Total-Pages", strconv.Itoa(p.TotalPages))
	w.Header().Set("X-Page", strconv.Itoa(p.Page))
	w.Header().Set("X-Per-Page", strconv.Itoa(p.PerPage))
}

// Cursor represents cursor-based pagination parameters.
type Cursor struct {
	// Cursor is the opaque cursor string
	Cursor string `json:"cursor,omitempty"`

	// Limit is the number of items to return
	Limit int `json:"limit"`

	// Direction is "next" or "prev"
	Direction string `json:"direction,omitempty"`
}

// CursorFromRequest extracts cursor pagination from request.
// Supports: ?cursor=xxx&limit=20
func CursorFromRequest(r *http.Request) Cursor {
	q := r.URL.Query()

	limit := parseInt(q.Get("limit"), DefaultPerPage)
	if limit > MaxPerPage {
		limit = MaxPerPage
	}

	return Cursor{
		Cursor:    q.Get("cursor"),
		Limit:     limit,
		Direction: q.Get("direction"),
	}
}

// IsFirst returns true if this is the first page (no cursor).
func (c Cursor) IsFirst() bool {
	return c.Cursor == ""
}

// CursorResult wraps cursor-paginated data.
type CursorResult[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// NewCursorResult creates a cursor-paginated result.
func NewCursorResult[T any](data []T, nextCursor, prevCursor string, hasMore bool) CursorResult[T] {
	return CursorResult[T]{
		Data:       data,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasMore:    hasMore,
	}
}

// EncodeCursor encodes cursor data to a string.
func EncodeCursor(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// DecodeCursor decodes a cursor string to data.
func DecodeCursor(cursor string, dest any) error {
	if cursor == "" {
		return nil
	}
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

// IDCursor is a simple cursor using an ID field.
type IDCursor struct {
	LastID    string `json:"id"`
	Direction string `json:"dir,omitempty"`
}

// EncodeIDCursor creates a cursor from an ID.
func EncodeIDCursor(id string, direction string) string {
	cursor, _ := EncodeCursor(IDCursor{LastID: id, Direction: direction})
	return cursor
}

// DecodeIDCursor decodes an ID cursor.
func DecodeIDCursor(cursor string) (id string, direction string, err error) {
	var c IDCursor
	if err := DecodeCursor(cursor, &c); err != nil {
		return "", "", err
	}
	return c.LastID, c.Direction, nil
}

// TimeCursor is a cursor using timestamp and ID for stable ordering.
type TimeCursor struct {
	Timestamp int64  `json:"ts"`
	ID        string `json:"id"`
	Direction string `json:"dir,omitempty"`
}

// EncodeTimeCursor creates a cursor from timestamp and ID.
func EncodeTimeCursor(timestamp int64, id string, direction string) string {
	cursor, _ := EncodeCursor(TimeCursor{Timestamp: timestamp, ID: id, Direction: direction})
	return cursor
}

// DecodeTimeCursor decodes a time cursor.
func DecodeTimeCursor(cursor string) (timestamp int64, id string, direction string, err error) {
	var c TimeCursor
	if err := DecodeCursor(cursor, &c); err != nil {
		return 0, "", "", err
	}
	return c.Timestamp, c.ID, c.Direction, nil
}
