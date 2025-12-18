package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MeilisearchConfig configures the Meilisearch client.
type MeilisearchConfig struct {
	// Host is the Meilisearch server URL.
	Host string

	// APIKey is the API key for authentication.
	APIKey string

	// HTTPClient is the HTTP client to use.
	// If nil, a default client is created.
	HTTPClient *http.Client

	// Timeout is the timeout for requests.
	Timeout time.Duration
}

// Meilisearch is a client for Meilisearch.
type Meilisearch struct {
	host       string
	apiKey     string
	httpClient *http.Client
}

// NewMeilisearch creates a new Meilisearch client.
func NewMeilisearch(cfg MeilisearchConfig) (*Meilisearch, error) {
	if cfg.Host == "" {
		cfg.Host = "http://localhost:7700"
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	}

	return &Meilisearch{
		host:       strings.TrimRight(cfg.Host, "/"),
		apiKey:     cfg.APIKey,
		httpClient: httpClient,
	}, nil
}

// Backend returns the backend type identifier.
func (m *Meilisearch) Backend() string {
	return "meilisearch"
}

// Close closes the client connection.
func (m *Meilisearch) Close() error {
	return nil
}

// Index indexes a document.
func (m *Meilisearch) Index(ctx context.Context, index, id string, document any) error {
	return m.IndexWithOptions(ctx, index, id, document, nil)
}

// IndexWithOptions indexes a document with additional options.
func (m *Meilisearch) IndexWithOptions(ctx context.Context, index, id string, document any, opts *IndexOptions) error {
	// Meilisearch expects documents with an ID field
	doc := m.ensureID(document, id)

	body, err := json.Marshal([]any{doc})
	if err != nil {
		return fmt.Errorf("search: failed to marshal document: %w", err)
	}

	path := fmt.Sprintf("/indexes/%s/documents", index)

	resp, err := m.request(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return m.parseError(resp)
	}

	// Meilisearch returns a task, optionally wait for it
	if opts != nil && opts.Refresh == "true" {
		var task msTask
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			return nil // Document was accepted, task parsing is optional
		}
		return m.waitForTask(ctx, task.TaskUID)
	}

	return nil
}

// ensureID ensures the document has an ID field.
func (m *Meilisearch) ensureID(document any, id string) map[string]any {
	// Convert document to map
	data, err := json.Marshal(document)
	if err != nil {
		return map[string]any{"id": id}
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return map[string]any{"id": id}
	}

	// Set ID if not present
	if _, ok := doc["id"]; !ok {
		doc["id"] = id
	}

	return doc
}

// Get retrieves a document by ID.
func (m *Meilisearch) Get(ctx context.Context, index, id string) (*Document, error) {
	path := fmt.Sprintf("/indexes/%s/documents/%s", index, id)

	resp, err := m.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode >= 400 {
		return nil, m.parseError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("search: failed to read response: %w", err)
	}

	return &Document{
		ID:     id,
		Index:  index,
		Source: body,
	}, nil
}

// Delete removes a document by ID.
func (m *Meilisearch) Delete(ctx context.Context, index, id string) error {
	path := fmt.Sprintf("/indexes/%s/documents/%s", index, id)

	resp, err := m.request(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	if resp.StatusCode >= 400 {
		return m.parseError(resp)
	}

	return nil
}

// Search performs a search query.
func (m *Meilisearch) Search(ctx context.Context, index string, query *QueryBuilder) (*SearchResult, error) {
	return m.SearchWithOptions(ctx, index, query, nil)
}

// SearchWithOptions performs a search with additional options.
func (m *Meilisearch) SearchWithOptions(ctx context.Context, index string, query *QueryBuilder, opts *SearchOptions) (*SearchResult, error) {
	path := fmt.Sprintf("/indexes/%s/search", index)

	// Build request body
	body := make(map[string]any)

	// Extract search term from query
	if query != nil {
		body["q"] = m.extractSearchTerm(query)

		// Convert filters
		if filter := m.convertFilters(query); filter != "" {
			body["filter"] = filter
		}
	}

	if opts != nil {
		if opts.From > 0 {
			body["offset"] = opts.From
		}
		if opts.Size > 0 {
			body["limit"] = opts.Size
		}
		if len(opts.Sort) > 0 {
			sort := make([]string, len(opts.Sort))
			for i, s := range opts.Sort {
				if s.Order == "desc" {
					sort[i] = s.Field + ":desc"
				} else {
					sort[i] = s.Field + ":asc"
				}
			}
			body["sort"] = sort
		}
		if opts.Source != nil && len(opts.Source.Includes) > 0 {
			body["attributesToRetrieve"] = opts.Source.Includes
		}
		if opts.Highlight != nil && len(opts.Highlight.Fields) > 0 {
			fields := make([]string, 0, len(opts.Highlight.Fields))
			for field := range opts.Highlight.Fields {
				fields = append(fields, field)
			}
			body["attributesToHighlight"] = fields
			if len(opts.Highlight.PreTags) > 0 {
				body["highlightPreTag"] = opts.Highlight.PreTags[0]
			}
			if len(opts.Highlight.PostTags) > 0 {
				body["highlightPostTag"] = opts.Highlight.PostTags[0]
			}
		}
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("search: failed to marshal query: %w", err)
	}

	resp, err := m.request(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, m.parseError(resp)
	}

	var result msSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("search: failed to decode response: %w", err)
	}

	return m.parseSearchResponse(&result, index), nil
}

// extractSearchTerm extracts the search term from a QueryBuilder.
func (m *Meilisearch) extractSearchTerm(query *QueryBuilder) string {
	// Look for match queries to extract the search term
	for _, clause := range query.must {
		if clauseMap, ok := clause.(map[string]any); ok {
			if term := m.extractTermFromClause(clauseMap); term != "" {
				return term
			}
		}
	}
	for _, clause := range query.should {
		if clauseMap, ok := clause.(map[string]any); ok {
			if term := m.extractTermFromClause(clauseMap); term != "" {
				return term
			}
		}
	}
	return ""
}

// extractTermFromClause extracts search term from a query clause.
func (m *Meilisearch) extractTermFromClause(clause map[string]any) string {
	if match, ok := clause["match"]; ok {
		if matchMap, ok := match.(map[string]any); ok {
			for _, v := range matchMap {
				if vm, ok := v.(map[string]any); ok {
					if query, ok := vm["query"].(string); ok {
						return query
					}
				}
			}
		}
	}
	if multiMatch, ok := clause["multi_match"]; ok {
		if mm, ok := multiMatch.(map[string]any); ok {
			if query, ok := mm["query"].(string); ok {
				return query
			}
		}
	}
	if queryString, ok := clause["query_string"]; ok {
		if qs, ok := queryString.(map[string]any); ok {
			if query, ok := qs["query"].(string); ok {
				return query
			}
		}
	}
	return ""
}

// convertFilters converts QueryBuilder filters to Meilisearch filter syntax.
func (m *Meilisearch) convertFilters(query *QueryBuilder) string {
	var filters []string

	for _, clause := range query.filter {
		if clauseMap, ok := clause.(map[string]any); ok {
			if filter := m.convertClauseToFilter(clauseMap); filter != "" {
				filters = append(filters, filter)
			}
		}
	}

	for _, clause := range query.must {
		if clauseMap, ok := clause.(map[string]any); ok {
			if filter := m.convertClauseToFilter(clauseMap); filter != "" {
				filters = append(filters, filter)
			}
		}
	}

	if len(filters) == 0 {
		return ""
	}

	return strings.Join(filters, " AND ")
}

// convertClauseToFilter converts a single clause to Meilisearch filter.
func (m *Meilisearch) convertClauseToFilter(clause map[string]any) string {
	// Term query
	if term, ok := clause["term"]; ok {
		if termMap, ok := term.(map[string]any); ok {
			for field, value := range termMap {
				return fmt.Sprintf("%s = %v", field, m.formatValue(value))
			}
		}
	}

	// Terms query
	if terms, ok := clause["terms"]; ok {
		if termsMap, ok := terms.(map[string]any); ok {
			for field, values := range termsMap {
				if arr, ok := values.([]any); ok {
					vals := make([]string, len(arr))
					for i, v := range arr {
						vals[i] = fmt.Sprintf("%s = %v", field, m.formatValue(v))
					}
					return "(" + strings.Join(vals, " OR ") + ")"
				}
			}
		}
	}

	// Range query
	if rangeQ, ok := clause["range"]; ok {
		if rangeMap, ok := rangeQ.(map[string]any); ok {
			for field, conditions := range rangeMap {
				if condMap, ok := conditions.(map[string]any); ok {
					var parts []string
					if v, ok := condMap["gte"]; ok {
						parts = append(parts, fmt.Sprintf("%s >= %v", field, v))
					}
					if v, ok := condMap["gt"]; ok {
						parts = append(parts, fmt.Sprintf("%s > %v", field, v))
					}
					if v, ok := condMap["lte"]; ok {
						parts = append(parts, fmt.Sprintf("%s <= %v", field, v))
					}
					if v, ok := condMap["lt"]; ok {
						parts = append(parts, fmt.Sprintf("%s < %v", field, v))
					}
					if len(parts) > 0 {
						return strings.Join(parts, " AND ")
					}
				}
			}
		}
	}

	return ""
}

// formatValue formats a value for Meilisearch filter syntax.
func (m *Meilisearch) formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// Bulk performs bulk operations.
func (m *Meilisearch) Bulk(ctx context.Context, operations []BulkOperation) (*BulkResult, error) {
	// Group operations by index and action
	indexOps := make(map[string]struct {
		toIndex  []any
		toDelete []string
	})

	for _, op := range operations {
		ops := indexOps[op.Index]
		switch op.Action {
		case "index", "create":
			doc := m.ensureID(op.Document, op.ID)
			ops.toIndex = append(ops.toIndex, doc)
		case "delete":
			ops.toDelete = append(ops.toDelete, op.ID)
		}
		indexOps[op.Index] = ops
	}

	result := &BulkResult{
		Items: make([]BulkItemResult, 0, len(operations)),
	}

	// Process each index
	for index, ops := range indexOps {
		// Index documents
		if len(ops.toIndex) > 0 {
			body, err := json.Marshal(ops.toIndex)
			if err != nil {
				continue
			}

			path := fmt.Sprintf("/indexes/%s/documents", index)
			resp, err := m.request(ctx, http.MethodPost, path, body)
			if err != nil {
				for _, doc := range ops.toIndex {
					if docMap, ok := doc.(map[string]any); ok {
						result.Items = append(result.Items, BulkItemResult{
							Action: "index",
							Index:  index,
							ID:     fmt.Sprintf("%v", docMap["id"]),
							Status: 500,
							Error:  err.Error(),
						})
						result.ErrorCount++
					}
				}
				continue
			}
			resp.Body.Close()

			for _, doc := range ops.toIndex {
				if docMap, ok := doc.(map[string]any); ok {
					result.Items = append(result.Items, BulkItemResult{
						Action: "index",
						Index:  index,
						ID:     fmt.Sprintf("%v", docMap["id"]),
						Status: 202,
					})
					result.SuccessCount++
				}
			}
		}

		// Delete documents
		if len(ops.toDelete) > 0 {
			body, err := json.Marshal(ops.toDelete)
			if err != nil {
				continue
			}

			path := fmt.Sprintf("/indexes/%s/documents/delete-batch", index)
			resp, err := m.request(ctx, http.MethodPost, path, body)
			if err != nil {
				for _, id := range ops.toDelete {
					result.Items = append(result.Items, BulkItemResult{
						Action: "delete",
						Index:  index,
						ID:     id,
						Status: 500,
						Error:  err.Error(),
					})
					result.ErrorCount++
				}
				continue
			}
			resp.Body.Close()

			for _, id := range ops.toDelete {
				result.Items = append(result.Items, BulkItemResult{
					Action: "delete",
					Index:  index,
					ID:     id,
					Status: 202,
				})
				result.SuccessCount++
			}
		}
	}

	result.Errors = result.ErrorCount > 0
	return result, nil
}

// CreateIndex creates a new index.
func (m *Meilisearch) CreateIndex(ctx context.Context, index string, settings *IndexSettings) error {
	// Create index
	body := map[string]any{
		"uid": index,
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("search: failed to marshal request: %w", err)
	}

	resp, err := m.request(ctx, http.MethodPost, "/indexes", reqBody)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusConflict {
		return m.parseError(resp)
	}

	// Apply settings if provided
	if settings != nil && settings.Settings != nil {
		settingsBody, err := json.Marshal(settings.Settings)
		if err != nil {
			return fmt.Errorf("search: failed to marshal settings: %w", err)
		}

		path := fmt.Sprintf("/indexes/%s/settings", index)
		resp, err := m.request(ctx, http.MethodPatch, path, settingsBody)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			return m.parseError(resp)
		}
	}

	return nil
}

// DeleteIndex deletes an index.
func (m *Meilisearch) DeleteIndex(ctx context.Context, index string) error {
	path := fmt.Sprintf("/indexes/%s", index)

	resp, err := m.request(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrIndexNotFound
	}

	if resp.StatusCode >= 400 {
		return m.parseError(resp)
	}

	return nil
}

// IndexExists checks if an index exists.
func (m *Meilisearch) IndexExists(ctx context.Context, index string) (bool, error) {
	path := fmt.Sprintf("/indexes/%s", index)

	resp, err := m.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// Refresh refreshes an index. For Meilisearch, this waits for pending tasks.
func (m *Meilisearch) Refresh(ctx context.Context, index string) error {
	// Get pending tasks for this index
	path := fmt.Sprintf("/tasks?indexUids=%s&statuses=enqueued,processing", index)

	resp, err := m.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Results []msTask `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil // No tasks to wait for
	}

	// Wait for each task
	for _, task := range result.Results {
		if err := m.waitForTask(ctx, task.TaskUID); err != nil {
			return err
		}
	}

	return nil
}

// Count returns the number of documents matching a query.
func (m *Meilisearch) Count(ctx context.Context, index string, query *QueryBuilder) (int64, error) {
	// Meilisearch doesn't have a dedicated count endpoint
	// We do a search with limit 0 and get estimatedTotalHits
	opts := &SearchOptions{Size: 0}
	result, err := m.SearchWithOptions(ctx, index, query, opts)
	if err != nil {
		return 0, err
	}
	return result.Total, nil
}

// waitForTask waits for a task to complete.
func (m *Meilisearch) waitForTask(ctx context.Context, taskUID int64) error {
	path := fmt.Sprintf("/tasks/%d", taskUID)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := m.request(ctx, http.MethodGet, path, nil)
		if err != nil {
			return err
		}

		var task msTask
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()

		switch task.Status {
		case "succeeded":
			return nil
		case "failed":
			return fmt.Errorf("search: task failed: %s", task.Error.Message)
		case "canceled":
			return fmt.Errorf("search: task was canceled")
		}

		// Wait before polling again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// request makes an HTTP request to Meilisearch.
func (m *Meilisearch) request(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	url := m.host + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("search: failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if m.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionError, err)
	}

	return resp, nil
}

// parseError parses an error response from Meilisearch.
func (m *Meilisearch) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	var msErr struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Type    string `json:"type"`
	}

	if json.Unmarshal(body, &msErr) == nil && msErr.Code != "" {
		switch msErr.Code {
		case "index_not_found":
			return ErrIndexNotFound
		case "document_not_found":
			return ErrNotFound
		case "invalid_api_key", "missing_authorization_header":
			return ErrUnauthorized
		}
		return fmt.Errorf("search: %s: %s", msErr.Code, msErr.Message)
	}

	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrBadRequest, string(body))
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	default:
		return fmt.Errorf("search: request failed with status %d: %s", resp.StatusCode, string(body))
	}
}

// parseSearchResponse converts Meilisearch response to SearchResult.
func (m *Meilisearch) parseSearchResponse(resp *msSearchResponse, index string) *SearchResult {
	result := &SearchResult{
		Total:         resp.EstimatedTotalHits,
		TotalRelation: "eq",
		Took:          resp.ProcessingTimeMs,
	}

	if resp.TotalHits > 0 {
		result.Total = resp.TotalHits
	}

	result.Hits = make([]Document, len(resp.Hits))
	for i, hit := range resp.Hits {
		// Extract ID from hit
		var id string
		if idVal, ok := hit["id"]; ok {
			id = fmt.Sprintf("%v", idVal)
		}

		source, _ := json.Marshal(hit)

		doc := Document{
			ID:     id,
			Index:  index,
			Source: source,
		}

		// Extract highlights
		if formatted, ok := hit["_formatted"].(map[string]any); ok {
			doc.Highlights = make(map[string][]string)
			for field, value := range formatted {
				if field != "id" && field != "_formatted" {
					doc.Highlights[field] = []string{fmt.Sprintf("%v", value)}
				}
			}
		}

		result.Hits[i] = doc
	}

	return result
}

// Meilisearch response types

type msSearchResponse struct {
	Hits               []map[string]any `json:"hits"`
	EstimatedTotalHits int64            `json:"estimatedTotalHits"`
	TotalHits          int64            `json:"totalHits"`
	ProcessingTimeMs   int64            `json:"processingTimeMs"`
	Query              string           `json:"query"`
	Offset             int              `json:"offset"`
	Limit              int              `json:"limit"`
}

type msTask struct {
	TaskUID int64  `json:"taskUid"`
	Status  string `json:"status"`
	Error   struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

// MeilisearchSettings represents Meilisearch index settings.
type MeilisearchSettings struct {
	// SearchableAttributes is the list of searchable attributes.
	SearchableAttributes []string `json:"searchableAttributes,omitempty"`

	// FilterableAttributes is the list of filterable attributes.
	FilterableAttributes []string `json:"filterableAttributes,omitempty"`

	// SortableAttributes is the list of sortable attributes.
	SortableAttributes []string `json:"sortableAttributes,omitempty"`

	// RankingRules defines the ranking rules.
	RankingRules []string `json:"rankingRules,omitempty"`

	// StopWords is the list of stop words.
	StopWords []string `json:"stopWords,omitempty"`

	// Synonyms defines synonyms.
	Synonyms map[string][]string `json:"synonyms,omitempty"`

	// DistinctAttribute is the attribute for deduplication.
	DistinctAttribute string `json:"distinctAttribute,omitempty"`

	// TypoTolerance configures typo tolerance.
	TypoTolerance *TypoToleranceSettings `json:"typoTolerance,omitempty"`
}

// TypoToleranceSettings configures typo tolerance.
type TypoToleranceSettings struct {
	Enabled             bool     `json:"enabled"`
	MinWordSizeForTypos MinTypos `json:"minWordSizeForTypos,omitempty"`
	DisableOnWords      []string `json:"disableOnWords,omitempty"`
	DisableOnAttributes []string `json:"disableOnAttributes,omitempty"`
}

// MinTypos configures minimum word size for typos.
type MinTypos struct {
	OneTypo  int `json:"oneTypo,omitempty"`
	TwoTypos int `json:"twoTypos,omitempty"`
}

// UpdateSettings updates the settings for an index.
func (m *Meilisearch) UpdateSettings(ctx context.Context, index string, settings *MeilisearchSettings) error {
	body, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("search: failed to marshal settings: %w", err)
	}

	path := fmt.Sprintf("/indexes/%s/settings", index)
	resp, err := m.request(ctx, http.MethodPatch, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return m.parseError(resp)
	}

	return nil
}

// GetSettings retrieves the settings for an index.
func (m *Meilisearch) GetSettings(ctx context.Context, index string) (*MeilisearchSettings, error) {
	path := fmt.Sprintf("/indexes/%s/settings", index)

	resp, err := m.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, m.parseError(resp)
	}

	var settings MeilisearchSettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("search: failed to decode settings: %w", err)
	}

	return &settings, nil
}
