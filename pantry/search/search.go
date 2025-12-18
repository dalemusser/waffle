// Package search provides a unified interface for full-text search across
// multiple backends including Elasticsearch, OpenSearch, and Meilisearch.
//
// Basic usage:
//
//	// Create an Elasticsearch client
//	client, err := search.NewElasticsearch(search.ElasticsearchConfig{
//	    Addresses: []string{"http://localhost:9200"},
//	})
//
//	// Index a document
//	err = client.Index(ctx, "products", "1", product)
//
//	// Search with query builder
//	results, err := client.Search(ctx, "products",
//	    search.Query().
//	        Must(search.Match("name", "laptop")).
//	        Filter(search.Range("price").Lte(1000)),
//	)
//
//	// Iterate results
//	for _, hit := range results.Hits {
//	    var product Product
//	    hit.Decode(&product)
//	}
package search

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Common errors returned by search operations.
var (
	ErrNotFound        = errors.New("search: document not found")
	ErrIndexNotFound   = errors.New("search: index not found")
	ErrInvalidQuery    = errors.New("search: invalid query")
	ErrConnectionError = errors.New("search: connection error")
	ErrTimeout         = errors.New("search: operation timed out")
	ErrConflict        = errors.New("search: version conflict")
	ErrBadRequest      = errors.New("search: bad request")
	ErrUnauthorized    = errors.New("search: unauthorized")
	ErrForbidden       = errors.New("search: forbidden")
)

// Client is the interface that all search backends must implement.
type Client interface {
	// Index indexes a document.
	Index(ctx context.Context, index, id string, document any) error

	// IndexWithOptions indexes a document with additional options.
	IndexWithOptions(ctx context.Context, index, id string, document any, opts *IndexOptions) error

	// Get retrieves a document by ID.
	Get(ctx context.Context, index, id string) (*Document, error)

	// Delete removes a document by ID.
	Delete(ctx context.Context, index, id string) error

	// Search performs a search query.
	Search(ctx context.Context, index string, query *QueryBuilder) (*SearchResult, error)

	// SearchWithOptions performs a search with additional options.
	SearchWithOptions(ctx context.Context, index string, query *QueryBuilder, opts *SearchOptions) (*SearchResult, error)

	// Bulk performs bulk operations.
	Bulk(ctx context.Context, operations []BulkOperation) (*BulkResult, error)

	// CreateIndex creates a new index.
	CreateIndex(ctx context.Context, index string, settings *IndexSettings) error

	// DeleteIndex deletes an index.
	DeleteIndex(ctx context.Context, index string) error

	// IndexExists checks if an index exists.
	IndexExists(ctx context.Context, index string) (bool, error)

	// Refresh refreshes an index to make recent changes searchable.
	Refresh(ctx context.Context, index string) error

	// Count returns the number of documents matching a query.
	Count(ctx context.Context, index string, query *QueryBuilder) (int64, error)

	// Close closes the client connection.
	Close() error

	// Backend returns the backend type identifier.
	Backend() string
}

// Document represents a retrieved document.
type Document struct {
	// ID is the document identifier.
	ID string

	// Index is the index name.
	Index string

	// Source is the raw document source.
	Source json.RawMessage

	// Version is the document version (if available).
	Version int64

	// Score is the relevance score (if from search).
	Score float64

	// Highlights contains highlighted snippets.
	Highlights map[string][]string
}

// Decode decodes the document source into the provided value.
func (d *Document) Decode(v any) error {
	return json.Unmarshal(d.Source, v)
}

// SearchResult contains the results of a search query.
type SearchResult struct {
	// Total is the total number of matching documents.
	Total int64

	// TotalRelation indicates if Total is exact ("eq") or a lower bound ("gte").
	TotalRelation string

	// Hits contains the matching documents.
	Hits []Document

	// Aggregations contains aggregation results.
	Aggregations map[string]json.RawMessage

	// Took is how long the search took in milliseconds.
	Took int64

	// TimedOut indicates if the search timed out.
	TimedOut bool

	// ScrollID is used for scroll/pagination (if applicable).
	ScrollID string
}

// IndexOptions configures document indexing.
type IndexOptions struct {
	// Refresh controls when changes become visible.
	// Values: "true", "false", "wait_for"
	Refresh string

	// Routing sets custom routing.
	Routing string

	// Version sets the expected version for optimistic concurrency.
	Version int64

	// VersionType sets the version type.
	VersionType string

	// Pipeline sets the ingest pipeline.
	Pipeline string

	// OpType sets the operation type ("index" or "create").
	OpType string
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	// From is the starting offset (for pagination).
	From int

	// Size is the maximum number of hits to return.
	Size int

	// Sort specifies sort order.
	Sort []SortOption

	// Source controls which fields to return.
	Source *SourceFilter

	// Highlight configures result highlighting.
	Highlight *HighlightConfig

	// Aggregations defines aggregations to compute.
	Aggregations map[string]any

	// TrackTotalHits controls total hit tracking.
	// true = exact count, false = no count, int = count up to N
	TrackTotalHits any

	// Timeout sets the search timeout.
	Timeout time.Duration

	// Routing sets custom routing.
	Routing string

	// Preference sets search preference.
	Preference string

	// SearchAfter is used for search_after pagination.
	SearchAfter []any

	// Scroll sets the scroll keep-alive time.
	Scroll time.Duration

	// MinScore sets the minimum score threshold.
	MinScore float64

	// Explain includes score explanations.
	Explain bool
}

// SortOption defines a sort field and order.
type SortOption struct {
	Field string
	Order string // "asc" or "desc"
	Mode  string // "min", "max", "avg", "sum", "median"
}

// SourceFilter controls which fields are returned.
type SourceFilter struct {
	Includes []string
	Excludes []string
}

// HighlightConfig configures result highlighting.
type HighlightConfig struct {
	Fields       map[string]*HighlightField
	PreTags      []string
	PostTags     []string
	FragmentSize int
	NumFragments int
}

// HighlightField configures highlighting for a specific field.
type HighlightField struct {
	FragmentSize int
	NumFragments int
	PreTags      []string
	PostTags     []string
}

// BulkOperation represents a bulk operation.
type BulkOperation struct {
	// Action is the operation type: "index", "create", "update", "delete"
	Action string

	// Index is the target index.
	Index string

	// ID is the document ID.
	ID string

	// Document is the document to index/update.
	Document any

	// Routing sets custom routing.
	Routing string
}

// BulkResult contains the results of a bulk operation.
type BulkResult struct {
	// Took is how long the operation took in milliseconds.
	Took int64

	// Errors indicates if any errors occurred.
	Errors bool

	// Items contains individual operation results.
	Items []BulkItemResult

	// SuccessCount is the number of successful operations.
	SuccessCount int

	// ErrorCount is the number of failed operations.
	ErrorCount int
}

// BulkItemResult contains the result of a single bulk item.
type BulkItemResult struct {
	Action string
	Index  string
	ID     string
	Status int
	Error  string
}

// IndexSettings configures an index.
type IndexSettings struct {
	// Settings contains index settings.
	Settings map[string]any

	// Mappings defines field mappings.
	Mappings map[string]any

	// Aliases defines index aliases.
	Aliases map[string]any

	// NumberOfShards sets the number of primary shards.
	NumberOfShards int

	// NumberOfReplicas sets the number of replica shards.
	NumberOfReplicas int
}

// ToMap converts IndexSettings to a map for API requests.
func (s *IndexSettings) ToMap() map[string]any {
	result := make(map[string]any)

	settings := make(map[string]any)
	if s.Settings != nil {
		for k, v := range s.Settings {
			settings[k] = v
		}
	}
	if s.NumberOfShards > 0 {
		settings["number_of_shards"] = s.NumberOfShards
	}
	if s.NumberOfReplicas >= 0 {
		settings["number_of_replicas"] = s.NumberOfReplicas
	}
	if len(settings) > 0 {
		result["settings"] = settings
	}

	if s.Mappings != nil {
		result["mappings"] = s.Mappings
	}

	if s.Aliases != nil {
		result["aliases"] = s.Aliases
	}

	return result
}

// Pagination helpers

// Page calculates From based on page number and size.
func Page(page, size int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * size
}

// SearchOptionsForPage creates SearchOptions for a specific page.
func SearchOptionsForPage(page, size int) *SearchOptions {
	return &SearchOptions{
		From: Page(page, size),
		Size: size,
	}
}

// Sort helpers

// SortAsc creates an ascending sort option.
func SortAsc(field string) SortOption {
	return SortOption{Field: field, Order: "asc"}
}

// SortDesc creates a descending sort option.
func SortDesc(field string) SortOption {
	return SortOption{Field: field, Order: "desc"}
}

// SortByScore sorts by relevance score descending.
func SortByScore() SortOption {
	return SortOption{Field: "_score", Order: "desc"}
}

// Bulk operation helpers

// BulkIndex creates an index bulk operation.
func BulkIndex(index, id string, doc any) BulkOperation {
	return BulkOperation{
		Action:   "index",
		Index:    index,
		ID:       id,
		Document: doc,
	}
}

// BulkCreate creates a create bulk operation (fails if exists).
func BulkCreate(index, id string, doc any) BulkOperation {
	return BulkOperation{
		Action:   "create",
		Index:    index,
		ID:       id,
		Document: doc,
	}
}

// BulkUpdate creates an update bulk operation.
func BulkUpdate(index, id string, doc any) BulkOperation {
	return BulkOperation{
		Action:   "update",
		Index:    index,
		ID:       id,
		Document: doc,
	}
}

// BulkDelete creates a delete bulk operation.
func BulkDelete(index, id string) BulkOperation {
	return BulkOperation{
		Action: "delete",
		Index:  index,
		ID:     id,
	}
}

// DecodeHits decodes all hits into a slice of the given type.
func DecodeHits[T any](result *SearchResult) ([]T, error) {
	items := make([]T, 0, len(result.Hits))
	for _, hit := range result.Hits {
		var item T
		if err := hit.Decode(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// DecodeAggregation decodes an aggregation result.
func DecodeAggregation[T any](result *SearchResult, name string) (*T, error) {
	data, ok := result.Aggregations[name]
	if !ok {
		return nil, nil
	}
	var agg T
	if err := json.Unmarshal(data, &agg); err != nil {
		return nil, err
	}
	return &agg, nil
}
