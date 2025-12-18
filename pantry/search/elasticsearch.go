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

// ElasticsearchConfig configures the Elasticsearch client.
type ElasticsearchConfig struct {
	// Addresses is a list of Elasticsearch node addresses.
	Addresses []string

	// Username for basic authentication.
	Username string

	// Password for basic authentication.
	Password string

	// APIKey for API key authentication.
	APIKey string

	// CloudID for Elastic Cloud deployments.
	CloudID string

	// CACert is the PEM-encoded CA certificate for TLS verification.
	CACert []byte

	// InsecureSkipVerify skips TLS certificate verification.
	InsecureSkipVerify bool

	// HTTPClient is the HTTP client to use.
	// If nil, a default client is created.
	HTTPClient *http.Client

	// MaxRetries is the maximum number of retries for failed requests.
	MaxRetries int

	// RetryBackoff is the initial backoff duration between retries.
	RetryBackoff time.Duration

	// Timeout is the timeout for requests.
	Timeout time.Duration

	// Headers are additional headers to include in all requests.
	Headers map[string]string
}

// Elasticsearch is a client for Elasticsearch and OpenSearch.
type Elasticsearch struct {
	addresses    []string
	username     string
	password     string
	apiKey       string
	httpClient   *http.Client
	maxRetries   int
	retryBackoff time.Duration
	headers      map[string]string
	currentNode  int
}

// NewElasticsearch creates a new Elasticsearch client.
func NewElasticsearch(cfg ElasticsearchConfig) (*Elasticsearch, error) {
	if len(cfg.Addresses) == 0 {
		cfg.Addresses = []string{"http://localhost:9200"}
	}

	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}

	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 100 * time.Millisecond
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

	// Normalize addresses
	addresses := make([]string, len(cfg.Addresses))
	for i, addr := range cfg.Addresses {
		addresses[i] = strings.TrimRight(addr, "/")
	}

	return &Elasticsearch{
		addresses:    addresses,
		username:     cfg.Username,
		password:     cfg.Password,
		apiKey:       cfg.APIKey,
		httpClient:   httpClient,
		maxRetries:   cfg.MaxRetries,
		retryBackoff: cfg.RetryBackoff,
		headers:      cfg.Headers,
	}, nil
}

// Backend returns the backend type identifier.
func (e *Elasticsearch) Backend() string {
	return "elasticsearch"
}

// Close closes the client connection.
func (e *Elasticsearch) Close() error {
	return nil
}

// Index indexes a document.
func (e *Elasticsearch) Index(ctx context.Context, index, id string, document any) error {
	return e.IndexWithOptions(ctx, index, id, document, nil)
}

// IndexWithOptions indexes a document with additional options.
func (e *Elasticsearch) IndexWithOptions(ctx context.Context, index, id string, document any, opts *IndexOptions) error {
	body, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("search: failed to marshal document: %w", err)
	}

	path := fmt.Sprintf("/%s/_doc/%s", index, id)
	query := make(map[string]string)

	if opts != nil {
		if opts.Refresh != "" {
			query["refresh"] = opts.Refresh
		}
		if opts.Routing != "" {
			query["routing"] = opts.Routing
		}
		if opts.Pipeline != "" {
			query["pipeline"] = opts.Pipeline
		}
		if opts.Version > 0 {
			query["version"] = fmt.Sprintf("%d", opts.Version)
		}
		if opts.VersionType != "" {
			query["version_type"] = opts.VersionType
		}
		if opts.OpType != "" {
			query["op_type"] = opts.OpType
		}
	}

	resp, err := e.request(ctx, http.MethodPut, path, query, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return e.parseError(resp)
	}

	return nil
}

// Get retrieves a document by ID.
func (e *Elasticsearch) Get(ctx context.Context, index, id string) (*Document, error) {
	path := fmt.Sprintf("/%s/_doc/%s", index, id)

	resp, err := e.request(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode >= 400 {
		return nil, e.parseError(resp)
	}

	var result esGetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("search: failed to decode response: %w", err)
	}

	if !result.Found {
		return nil, ErrNotFound
	}

	return &Document{
		ID:      result.ID,
		Index:   result.Index,
		Source:  result.Source,
		Version: result.Version,
	}, nil
}

// Delete removes a document by ID.
func (e *Elasticsearch) Delete(ctx context.Context, index, id string) error {
	path := fmt.Sprintf("/%s/_doc/%s", index, id)

	resp, err := e.request(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	if resp.StatusCode >= 400 {
		return e.parseError(resp)
	}

	return nil
}

// Search performs a search query.
func (e *Elasticsearch) Search(ctx context.Context, index string, query *QueryBuilder) (*SearchResult, error) {
	return e.SearchWithOptions(ctx, index, query, nil)
}

// SearchWithOptions performs a search with additional options.
func (e *Elasticsearch) SearchWithOptions(ctx context.Context, index string, query *QueryBuilder, opts *SearchOptions) (*SearchResult, error) {
	path := fmt.Sprintf("/%s/_search", index)
	queryParams := make(map[string]string)

	// Build request body
	body := make(map[string]any)

	if query != nil {
		body["query"] = query.Build()
	}

	if opts != nil {
		if opts.From > 0 {
			body["from"] = opts.From
		}
		if opts.Size > 0 {
			body["size"] = opts.Size
		}
		if len(opts.Sort) > 0 {
			sort := make([]map[string]any, len(opts.Sort))
			for i, s := range opts.Sort {
				sortSpec := map[string]any{
					"order": s.Order,
				}
				if s.Mode != "" {
					sortSpec["mode"] = s.Mode
				}
				sort[i] = map[string]any{s.Field: sortSpec}
			}
			body["sort"] = sort
		}
		if opts.Source != nil {
			if len(opts.Source.Includes) > 0 || len(opts.Source.Excludes) > 0 {
				source := make(map[string]any)
				if len(opts.Source.Includes) > 0 {
					source["includes"] = opts.Source.Includes
				}
				if len(opts.Source.Excludes) > 0 {
					source["excludes"] = opts.Source.Excludes
				}
				body["_source"] = source
			}
		}
		if opts.Highlight != nil {
			body["highlight"] = e.buildHighlight(opts.Highlight)
		}
		if opts.Aggregations != nil {
			body["aggs"] = opts.Aggregations
		}
		if opts.TrackTotalHits != nil {
			body["track_total_hits"] = opts.TrackTotalHits
		}
		if opts.MinScore > 0 {
			body["min_score"] = opts.MinScore
		}
		if opts.Explain {
			body["explain"] = true
		}
		if len(opts.SearchAfter) > 0 {
			body["search_after"] = opts.SearchAfter
		}
		if opts.Timeout > 0 {
			queryParams["timeout"] = fmt.Sprintf("%dms", opts.Timeout.Milliseconds())
		}
		if opts.Routing != "" {
			queryParams["routing"] = opts.Routing
		}
		if opts.Preference != "" {
			queryParams["preference"] = opts.Preference
		}
		if opts.Scroll > 0 {
			queryParams["scroll"] = fmt.Sprintf("%dms", opts.Scroll.Milliseconds())
		}
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("search: failed to marshal query: %w", err)
	}

	resp, err := e.request(ctx, http.MethodPost, path, queryParams, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, e.parseError(resp)
	}

	var result esSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("search: failed to decode response: %w", err)
	}

	return e.parseSearchResponse(&result), nil
}

// Bulk performs bulk operations.
func (e *Elasticsearch) Bulk(ctx context.Context, operations []BulkOperation) (*BulkResult, error) {
	var buf bytes.Buffer

	for _, op := range operations {
		// Write action line
		action := map[string]any{
			op.Action: map[string]any{
				"_index": op.Index,
				"_id":    op.ID,
			},
		}
		if op.Routing != "" {
			action[op.Action].(map[string]any)["routing"] = op.Routing
		}

		actionLine, err := json.Marshal(action)
		if err != nil {
			return nil, fmt.Errorf("search: failed to marshal bulk action: %w", err)
		}
		buf.Write(actionLine)
		buf.WriteByte('\n')

		// Write document line (except for delete)
		if op.Action != "delete" && op.Document != nil {
			var docLine []byte
			if op.Action == "update" {
				docLine, err = json.Marshal(map[string]any{"doc": op.Document})
			} else {
				docLine, err = json.Marshal(op.Document)
			}
			if err != nil {
				return nil, fmt.Errorf("search: failed to marshal bulk document: %w", err)
			}
			buf.Write(docLine)
			buf.WriteByte('\n')
		}
	}

	resp, err := e.request(ctx, http.MethodPost, "/_bulk", nil, buf.Bytes())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, e.parseError(resp)
	}

	var result esBulkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("search: failed to decode response: %w", err)
	}

	return e.parseBulkResponse(&result), nil
}

// CreateIndex creates a new index.
func (e *Elasticsearch) CreateIndex(ctx context.Context, index string, settings *IndexSettings) error {
	path := fmt.Sprintf("/%s", index)

	var body []byte
	if settings != nil {
		var err error
		body, err = json.Marshal(settings.ToMap())
		if err != nil {
			return fmt.Errorf("search: failed to marshal settings: %w", err)
		}
	}

	resp, err := e.request(ctx, http.MethodPut, path, nil, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return e.parseError(resp)
	}

	return nil
}

// DeleteIndex deletes an index.
func (e *Elasticsearch) DeleteIndex(ctx context.Context, index string) error {
	path := fmt.Sprintf("/%s", index)

	resp, err := e.request(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrIndexNotFound
	}

	if resp.StatusCode >= 400 {
		return e.parseError(resp)
	}

	return nil
}

// IndexExists checks if an index exists.
func (e *Elasticsearch) IndexExists(ctx context.Context, index string) (bool, error) {
	path := fmt.Sprintf("/%s", index)

	resp, err := e.request(ctx, http.MethodHead, path, nil, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// Refresh refreshes an index to make recent changes searchable.
func (e *Elasticsearch) Refresh(ctx context.Context, index string) error {
	path := fmt.Sprintf("/%s/_refresh", index)

	resp, err := e.request(ctx, http.MethodPost, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return e.parseError(resp)
	}

	return nil
}

// Count returns the number of documents matching a query.
func (e *Elasticsearch) Count(ctx context.Context, index string, query *QueryBuilder) (int64, error) {
	path := fmt.Sprintf("/%s/_count", index)

	var body []byte
	if query != nil {
		var err error
		body, err = json.Marshal(map[string]any{
			"query": query.Build(),
		})
		if err != nil {
			return 0, fmt.Errorf("search: failed to marshal query: %w", err)
		}
	}

	resp, err := e.request(ctx, http.MethodPost, path, nil, body)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, e.parseError(resp)
	}

	var result struct {
		Count int64 `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("search: failed to decode response: %w", err)
	}

	return result.Count, nil
}

// UpdateByQuery updates documents matching a query.
func (e *Elasticsearch) UpdateByQuery(ctx context.Context, index string, query *QueryBuilder, script string) (int64, error) {
	path := fmt.Sprintf("/%s/_update_by_query", index)

	body := map[string]any{
		"query": query.Build(),
		"script": map[string]any{
			"source": script,
		},
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return 0, fmt.Errorf("search: failed to marshal request: %w", err)
	}

	resp, err := e.request(ctx, http.MethodPost, path, nil, reqBody)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, e.parseError(resp)
	}

	var result struct {
		Updated int64 `json:"updated"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("search: failed to decode response: %w", err)
	}

	return result.Updated, nil
}

// DeleteByQuery deletes documents matching a query.
func (e *Elasticsearch) DeleteByQuery(ctx context.Context, index string, query *QueryBuilder) (int64, error) {
	path := fmt.Sprintf("/%s/_delete_by_query", index)

	body := map[string]any{
		"query": query.Build(),
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return 0, fmt.Errorf("search: failed to marshal request: %w", err)
	}

	resp, err := e.request(ctx, http.MethodPost, path, nil, reqBody)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, e.parseError(resp)
	}

	var result struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("search: failed to decode response: %w", err)
	}

	return result.Deleted, nil
}

// Scroll continues a scroll search.
func (e *Elasticsearch) Scroll(ctx context.Context, scrollID string, keepAlive time.Duration) (*SearchResult, error) {
	body := map[string]any{
		"scroll_id": scrollID,
		"scroll":    fmt.Sprintf("%dms", keepAlive.Milliseconds()),
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("search: failed to marshal request: %w", err)
	}

	resp, err := e.request(ctx, http.MethodPost, "/_search/scroll", nil, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, e.parseError(resp)
	}

	var result esSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("search: failed to decode response: %w", err)
	}

	return e.parseSearchResponse(&result), nil
}

// ClearScroll clears a scroll context.
func (e *Elasticsearch) ClearScroll(ctx context.Context, scrollID string) error {
	body := map[string]any{
		"scroll_id": scrollID,
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("search: failed to marshal request: %w", err)
	}

	resp, err := e.request(ctx, http.MethodDelete, "/_search/scroll", nil, reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// request makes an HTTP request to Elasticsearch.
func (e *Elasticsearch) request(ctx context.Context, method, path string, query map[string]string, body []byte) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		// Select node (round-robin)
		node := e.addresses[e.currentNode%len(e.addresses)]
		e.currentNode++

		// Build URL
		url := node + path
		if len(query) > 0 {
			params := make([]string, 0, len(query))
			for k, v := range query {
				params = append(params, fmt.Sprintf("%s=%s", k, v))
			}
			url += "?" + strings.Join(params, "&")
		}

		// Create request
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("search: failed to create request: %w", err)
		}

		// Set headers
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		if e.apiKey != "" {
			req.Header.Set("Authorization", "ApiKey "+e.apiKey)
		} else if e.username != "" {
			req.SetBasicAuth(e.username, e.password)
		}

		for k, v := range e.headers {
			req.Header.Set(k, v)
		}

		// Execute request
		resp, err := e.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < e.maxRetries {
				time.Sleep(e.retryBackoff * time.Duration(attempt+1))
				continue
			}
			return nil, fmt.Errorf("%w: %v", ErrConnectionError, err)
		}

		// Retry on certain status codes
		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusServiceUnavailable ||
			resp.StatusCode == http.StatusGatewayTimeout {
			resp.Body.Close()
			lastErr = fmt.Errorf("received status %d", resp.StatusCode)
			if attempt < e.maxRetries {
				time.Sleep(e.retryBackoff * time.Duration(attempt+1))
				continue
			}
		}

		return resp, nil
	}

	return nil, fmt.Errorf("%w: %v", ErrConnectionError, lastErr)
}

// parseError parses an error response from Elasticsearch.
func (e *Elasticsearch) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	var esErr struct {
		Error struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
		} `json:"error"`
	}

	if json.Unmarshal(body, &esErr) == nil && esErr.Error.Type != "" {
		switch esErr.Error.Type {
		case "index_not_found_exception":
			return ErrIndexNotFound
		case "document_missing_exception":
			return ErrNotFound
		case "version_conflict_engine_exception":
			return ErrConflict
		case "parsing_exception", "query_shard_exception":
			return fmt.Errorf("%w: %s", ErrInvalidQuery, esErr.Error.Reason)
		}
		return fmt.Errorf("search: %s: %s", esErr.Error.Type, esErr.Error.Reason)
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
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return ErrTimeout
	default:
		return fmt.Errorf("search: request failed with status %d: %s", resp.StatusCode, string(body))
	}
}

// buildHighlight converts HighlightConfig to Elasticsearch format.
func (e *Elasticsearch) buildHighlight(cfg *HighlightConfig) map[string]any {
	highlight := make(map[string]any)

	if len(cfg.PreTags) > 0 {
		highlight["pre_tags"] = cfg.PreTags
	}
	if len(cfg.PostTags) > 0 {
		highlight["post_tags"] = cfg.PostTags
	}
	if cfg.FragmentSize > 0 {
		highlight["fragment_size"] = cfg.FragmentSize
	}
	if cfg.NumFragments > 0 {
		highlight["number_of_fragments"] = cfg.NumFragments
	}

	if len(cfg.Fields) > 0 {
		fields := make(map[string]any)
		for name, field := range cfg.Fields {
			fieldCfg := make(map[string]any)
			if field != nil {
				if field.FragmentSize > 0 {
					fieldCfg["fragment_size"] = field.FragmentSize
				}
				if field.NumFragments > 0 {
					fieldCfg["number_of_fragments"] = field.NumFragments
				}
				if len(field.PreTags) > 0 {
					fieldCfg["pre_tags"] = field.PreTags
				}
				if len(field.PostTags) > 0 {
					fieldCfg["post_tags"] = field.PostTags
				}
			}
			fields[name] = fieldCfg
		}
		highlight["fields"] = fields
	}

	return highlight
}

// parseSearchResponse converts Elasticsearch response to SearchResult.
func (e *Elasticsearch) parseSearchResponse(resp *esSearchResponse) *SearchResult {
	result := &SearchResult{
		Total:         resp.Hits.Total.Value,
		TotalRelation: resp.Hits.Total.Relation,
		Took:          resp.Took,
		TimedOut:      resp.TimedOut,
		ScrollID:      resp.ScrollID,
		Aggregations:  resp.Aggregations,
	}

	result.Hits = make([]Document, len(resp.Hits.Hits))
	for i, hit := range resp.Hits.Hits {
		doc := Document{
			ID:     hit.ID,
			Index:  hit.Index,
			Source: hit.Source,
			Score:  hit.Score,
		}
		if hit.Highlight != nil {
			doc.Highlights = hit.Highlight
		}
		result.Hits[i] = doc
	}

	return result
}

// parseBulkResponse converts Elasticsearch bulk response to BulkResult.
func (e *Elasticsearch) parseBulkResponse(resp *esBulkResponse) *BulkResult {
	result := &BulkResult{
		Took:   resp.Took,
		Errors: resp.Errors,
		Items:  make([]BulkItemResult, 0, len(resp.Items)),
	}

	for _, item := range resp.Items {
		for action, data := range item {
			itemResult := BulkItemResult{
				Action: action,
				Index:  data.Index,
				ID:     data.ID,
				Status: data.Status,
			}
			if data.Error != nil {
				itemResult.Error = fmt.Sprintf("%s: %s", data.Error.Type, data.Error.Reason)
				result.ErrorCount++
			} else {
				result.SuccessCount++
			}
			result.Items = append(result.Items, itemResult)
		}
	}

	return result
}

// Elasticsearch response types

type esGetResponse struct {
	Index   string          `json:"_index"`
	ID      string          `json:"_id"`
	Version int64           `json:"_version"`
	Found   bool            `json:"found"`
	Source  json.RawMessage `json:"_source"`
}

type esSearchResponse struct {
	Took     int64  `json:"took"`
	TimedOut bool   `json:"timed_out"`
	ScrollID string `json:"_scroll_id"`
	Hits     struct {
		Total struct {
			Value    int64  `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		Hits []struct {
			Index     string              `json:"_index"`
			ID        string              `json:"_id"`
			Score     float64             `json:"_score"`
			Source    json.RawMessage     `json:"_source"`
			Highlight map[string][]string `json:"highlight"`
		} `json:"hits"`
	} `json:"hits"`
	Aggregations map[string]json.RawMessage `json:"aggregations"`
}

type esBulkResponse struct {
	Took   int64 `json:"took"`
	Errors bool  `json:"errors"`
	Items  []map[string]struct {
		Index  string `json:"_index"`
		ID     string `json:"_id"`
		Status int    `json:"status"`
		Error  *struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
		} `json:"error"`
	} `json:"items"`
}
