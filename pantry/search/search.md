# Search Package

The `search` package provides a unified interface for full-text search across multiple backends including Elasticsearch, OpenSearch, and Meilisearch.

## Features

- **Unified Client Interface**: Same API for all search backends
- **Query Builders**: Fluent API for building complex queries
- **Elasticsearch/OpenSearch Support**: Full-featured client with retries
- **Meilisearch Support**: Fast, typo-tolerant search
- **Bulk Operations**: Efficient batch indexing
- **Pagination**: Search-after and scroll support
- **Aggregations**: Terms, histogram, stats, and more
- **Highlighting**: Result highlighting with customizable tags

## Installation

The search package is part of the waffle pantry:

```go
import "github.com/dalemusser/waffle/pantry/search"
```

## Quick Start

### Elasticsearch

```go
// Create client
client, err := search.NewElasticsearch(search.ElasticsearchConfig{
    Addresses: []string{"http://localhost:9200"},
    Username:  "elastic",
    Password:  "changeme",
})
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Index a document
product := map[string]any{
    "name":     "Laptop",
    "price":    999.99,
    "category": "electronics",
}
err = client.Index(ctx, "products", "1", product)

// Search
results, err := client.Search(ctx, "products",
    search.Query().
        Must(search.Match("name", "laptop")).
        Filter(search.Range("price").Lte(1000)),
)

// Process results
for _, hit := range results.Hits {
    var p Product
    hit.Decode(&p)
    fmt.Printf("Found: %s ($%.2f)\n", p.Name, p.Price)
}
```

### Meilisearch

```go
// Create client
client, err := search.NewMeilisearch(search.MeilisearchConfig{
    Host:   "http://localhost:7700",
    APIKey: "masterKey",
})
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Create index with settings
err = client.CreateIndex(ctx, "products", nil)

// Configure searchable/filterable attributes
err = client.UpdateSettings(ctx, "products", &search.MeilisearchSettings{
    SearchableAttributes: []string{"name", "description"},
    FilterableAttributes: []string{"category", "price"},
    SortableAttributes:   []string{"price", "created_at"},
})

// Index and search work the same as Elasticsearch
err = client.Index(ctx, "products", "1", product)
results, err := client.Search(ctx, "products", search.Query().Must(search.Match("name", "laptop")))
```

## Client Interface

All search backends implement the `Client` interface:

```go
type Client interface {
    // Document operations
    Index(ctx context.Context, index, id string, document any) error
    IndexWithOptions(ctx context.Context, index, id string, document any, opts *IndexOptions) error
    Get(ctx context.Context, index, id string) (*Document, error)
    Delete(ctx context.Context, index, id string) error

    // Search operations
    Search(ctx context.Context, index string, query *QueryBuilder) (*SearchResult, error)
    SearchWithOptions(ctx context.Context, index string, query *QueryBuilder, opts *SearchOptions) (*SearchResult, error)
    Count(ctx context.Context, index string, query *QueryBuilder) (int64, error)

    // Bulk operations
    Bulk(ctx context.Context, operations []BulkOperation) (*BulkResult, error)

    // Index management
    CreateIndex(ctx context.Context, index string, settings *IndexSettings) error
    DeleteIndex(ctx context.Context, index string) error
    IndexExists(ctx context.Context, index string) (bool, error)
    Refresh(ctx context.Context, index string) error

    // Lifecycle
    Close() error
    Backend() string
}
```

## Query Building

The package provides a fluent query builder for constructing complex queries:

### Boolean Queries

```go
// Must (AND)
query := search.Query().Must(
    search.Match("title", "golang"),
    search.Match("content", "tutorial"),
)

// Should (OR)
query := search.Query().Should(
    search.Match("title", "golang"),
    search.Match("title", "go programming"),
).MinimumShouldMatch(1)

// Must Not (NOT)
query := search.Query().MustNot(
    search.Term("status", "draft"),
)

// Filter (no scoring)
query := search.Query().Filter(
    search.Term("published", true),
    search.Range("date").Gte("2024-01-01"),
)

// Combined
query := search.Query().
    Must(search.Match("content", "search")).
    Filter(search.Term("status", "published")).
    MustNot(search.Term("category", "spam"))
```

### Match Queries

```go
// Simple match
search.Match("title", "golang tutorial")

// Match with options
search.MatchWithOptions("title", "golang tutorial", search.MatchOptions{
    Operator:           "and",      // "and" or "or"
    Fuzziness:          "AUTO",     // "AUTO", "0", "1", "2"
    PrefixLength:       2,
    MaxExpansions:      50,
    MinimumShouldMatch: "75%",
})

// Match phrase
search.MatchPhrase("title", "golang tutorial")

// Multi-match across fields
search.MultiMatch([]string{"title", "content"}, "golang")

// Multi-match with options
search.MultiMatchWithOptions([]string{"title^2", "content"}, "golang", search.MultiMatchOptions{
    Type:     "best_fields",  // best_fields, most_fields, cross_fields, phrase, phrase_prefix
    Operator: "and",
})
```

### Term Queries

```go
// Exact term match
search.Term("status", "published")

// Multiple terms (OR)
search.Terms("category", "tech", "programming", "golang")

// Exists (field has value)
search.Exists("thumbnail")

// Prefix
search.Prefix("title", "go")

// Wildcard
search.Wildcard("email", "*@example.com")

// Regexp
search.Regexp("sku", "[A-Z]{3}-[0-9]{4}")

// IDs
search.IDs("1", "2", "3")
```

### Range Queries

```go
// Numeric range
search.Range("price").Gte(100).Lt(500)

// Date range
search.Range("created_at").
    Gte("2024-01-01").
    Lte("2024-12-31").
    Format("yyyy-MM-dd")

// With timezone
search.Range("timestamp").
    Gte("2024-01-01T00:00:00").
    TimeZone("+00:00")
```

### Other Queries

```go
// Fuzzy search
search.Fuzzy("name", "laptap")  // Finds "laptop"

// Match all
search.MatchAll()

// Nested query
search.Nested("comments",
    search.Query().Must(search.Match("comments.text", "great")),
)

// Raw query (pass through)
search.RawQuery(map[string]any{
    "geo_distance": map[string]any{
        "distance": "10km",
        "location": map[string]any{
            "lat": 40.73,
            "lon": -73.93,
        },
    },
})
```

## Search Options

```go
results, err := client.SearchWithOptions(ctx, "products", query, &search.SearchOptions{
    // Pagination
    From: 0,
    Size: 20,

    // Sorting
    Sort: []search.SortOption{
        search.SortDesc("_score"),
        search.SortAsc("name"),
    },

    // Source filtering
    Source: &search.SourceFilter{
        Includes: []string{"name", "price"},
        Excludes: []string{"internal_*"},
    },

    // Highlighting
    Highlight: &search.HighlightConfig{
        Fields: map[string]*search.HighlightField{
            "name":        {FragmentSize: 100},
            "description": {FragmentSize: 200, NumFragments: 3},
        },
        PreTags:  []string{"<em>"},
        PostTags: []string{"</em>"},
    },

    // Aggregations
    Aggregations: map[string]any{
        "categories": search.TermsAgg("category").Size(10).Build(),
        "price_stats": search.StatsAgg("price").Build(),
    },

    // Other options
    TrackTotalHits: true,
    MinScore:       0.5,
    Timeout:        10 * time.Second,
})
```

## Aggregations

```go
// Terms aggregation
agg := search.TermsAgg("category").
    Size(20).
    MinDocCount(5).
    OrderByCount("desc")

// Histogram aggregation
agg := search.HistogramAgg("price").
    Interval(100).
    MinDocCount(1)

// Date histogram
agg := search.DateHistogramAgg("created_at").
    CalendarInterval("month").
    Format("yyyy-MM")

// Stats aggregation
agg := search.StatsAgg("price")

// Nested aggregations
agg := search.TermsAgg("category").
    Size(10).
    SubAgg("avg_price", search.AvgAgg("price"))

// Use in search
opts := &search.SearchOptions{
    Aggregations: map[string]any{
        "categories": agg.Build(),
    },
}

// Parse aggregation results
type TermsBucket struct {
    Key      string `json:"key"`
    DocCount int64  `json:"doc_count"`
}
type TermsAggResult struct {
    Buckets []TermsBucket `json:"buckets"`
}

aggResult, err := search.DecodeAggregation[TermsAggResult](results, "categories")
for _, bucket := range aggResult.Buckets {
    fmt.Printf("%s: %d documents\n", bucket.Key, bucket.DocCount)
}
```

## Bulk Operations

```go
// Build bulk operations
ops := []search.BulkOperation{
    search.BulkIndex("products", "1", product1),
    search.BulkIndex("products", "2", product2),
    search.BulkCreate("products", "3", product3),  // Fails if exists
    search.BulkUpdate("products", "4", updates),
    search.BulkDelete("products", "5"),
}

// Execute
result, err := client.Bulk(ctx, ops)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Success: %d, Errors: %d\n", result.SuccessCount, result.ErrorCount)

// Check individual results
for _, item := range result.Items {
    if item.Error != "" {
        log.Printf("Failed %s %s/%s: %s", item.Action, item.Index, item.ID, item.Error)
    }
}
```

## Index Management

```go
// Create index with settings
err := client.CreateIndex(ctx, "products", &search.IndexSettings{
    NumberOfShards:   3,
    NumberOfReplicas: 1,
    Mappings: map[string]any{
        "properties": map[string]any{
            "name": map[string]any{
                "type": "text",
                "analyzer": "standard",
            },
            "price": map[string]any{
                "type": "float",
            },
            "category": map[string]any{
                "type": "keyword",
            },
        },
    },
})

// Check if index exists
exists, err := client.IndexExists(ctx, "products")

// Refresh index (make changes searchable)
err = client.Refresh(ctx, "products")

// Delete index
err = client.DeleteIndex(ctx, "products")
```

## Result Handling

```go
// Search result
results, err := client.Search(ctx, "products", query)

fmt.Printf("Total: %d (%s)\n", results.Total, results.TotalRelation)
fmt.Printf("Took: %dms\n", results.Took)

// Iterate hits
for _, hit := range results.Hits {
    fmt.Printf("ID: %s, Score: %.2f\n", hit.ID, hit.Score)

    // Decode into struct
    var product Product
    if err := hit.Decode(&product); err != nil {
        continue
    }

    // Access highlights
    if highlights, ok := hit.Highlights["name"]; ok {
        for _, h := range highlights {
            fmt.Printf("Highlight: %s\n", h)
        }
    }
}

// Decode all hits at once
products, err := search.DecodeHits[Product](results)
```

## Pagination

### Offset-based Pagination

```go
// Simple pagination helper
opts := search.SearchOptionsForPage(page, pageSize)  // page is 1-based

// Manual
opts := &search.SearchOptions{
    From: (page - 1) * pageSize,
    Size: pageSize,
}
```

### Search After (Deep Pagination)

```go
// First request
opts := &search.SearchOptions{
    Size: 100,
    Sort: []search.SortOption{
        search.SortDesc("created_at"),
        search.SortAsc("_id"),
    },
}
results, _ := client.SearchWithOptions(ctx, "products", query, opts)

// Subsequent requests
if len(results.Hits) > 0 {
    lastHit := results.Hits[len(results.Hits)-1]
    opts.SearchAfter = []any{lastHit.Source["created_at"], lastHit.ID}
}
```

### Scroll API (Elasticsearch)

```go
es := client.(*search.Elasticsearch)

// Initial search with scroll
opts := &search.SearchOptions{
    Size:   1000,
    Scroll: 5 * time.Minute,
}
results, _ := es.SearchWithOptions(ctx, "products", query, opts)

// Continue scrolling
for len(results.Hits) > 0 {
    // Process results...

    results, _ = es.Scroll(ctx, results.ScrollID, 5*time.Minute)
}

// Clean up
es.ClearScroll(ctx, results.ScrollID)
```

## Elasticsearch-Specific Features

```go
es := client.(*search.Elasticsearch)

// Update by query
updated, err := es.UpdateByQuery(ctx, "products",
    search.Query().Filter(search.Term("status", "draft")),
    "ctx._source.status = 'published'",
)

// Delete by query
deleted, err := es.DeleteByQuery(ctx, "products",
    search.Query().Filter(search.Range("created_at").Lt("2020-01-01")),
)
```

## Meilisearch-Specific Features

```go
ms := client.(*search.Meilisearch)

// Configure index settings
err := ms.UpdateSettings(ctx, "products", &search.MeilisearchSettings{
    SearchableAttributes: []string{"name", "description", "brand"},
    FilterableAttributes: []string{"category", "price", "in_stock"},
    SortableAttributes:   []string{"price", "rating", "created_at"},
    RankingRules: []string{
        "words",
        "typo",
        "proximity",
        "attribute",
        "sort",
        "exactness",
    },
    Synonyms: map[string][]string{
        "phone": {"smartphone", "mobile", "cell phone"},
    },
    StopWords: []string{"the", "a", "an"},
    TypoTolerance: &search.TypoToleranceSettings{
        Enabled: true,
        MinWordSizeForTypos: search.MinTypos{
            OneTypo:  4,
            TwoTypos: 8,
        },
    },
})

// Get current settings
settings, err := ms.GetSettings(ctx, "products")
```

## Configuration

### Elasticsearch Configuration

```go
client, err := search.NewElasticsearch(search.ElasticsearchConfig{
    // Connection
    Addresses: []string{
        "http://node1:9200",
        "http://node2:9200",
    },

    // Authentication
    Username: "elastic",
    Password: "changeme",
    // Or API key
    APIKey: "your-api-key",
    // Or Elastic Cloud
    CloudID: "deployment:abc123",

    // TLS
    CACert:             pemCertData,
    InsecureSkipVerify: false,

    // Timeouts and retries
    Timeout:      30 * time.Second,
    MaxRetries:   3,
    RetryBackoff: 100 * time.Millisecond,

    // Custom HTTP client
    HTTPClient: customClient,

    // Custom headers
    Headers: map[string]string{
        "X-Custom-Header": "value",
    },
})
```

### Meilisearch Configuration

```go
client, err := search.NewMeilisearch(search.MeilisearchConfig{
    Host:       "http://localhost:7700",
    APIKey:     "masterKey",
    Timeout:    30 * time.Second,
    HTTPClient: customClient,
})
```

## Error Handling

The package defines standard errors:

```go
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
```

Use `errors.Is` for error checking:

```go
doc, err := client.Get(ctx, "products", "123")
if errors.Is(err, search.ErrNotFound) {
    // Document doesn't exist
}
```

## Backend Compatibility

| Feature | Elasticsearch | Meilisearch |
|---------|--------------|-------------|
| Full-text search | ✅ | ✅ |
| Filters | ✅ | ✅ |
| Sorting | ✅ | ✅ |
| Pagination | ✅ | ✅ |
| Highlighting | ✅ | ✅ |
| Aggregations | ✅ | ❌ |
| Scroll API | ✅ | ❌ |
| Fuzzy search | ✅ | ✅ (built-in) |
| Typo tolerance | Manual | ✅ (built-in) |
| Nested queries | ✅ | ❌ |
| Geo queries | ✅ | ✅ |

## Best Practices

1. **Use filters for non-scoring queries**: Filters are cached and faster
2. **Batch index operations**: Use `Bulk` for multiple documents
3. **Refresh sparingly**: Don't refresh after every index operation
4. **Use search_after for deep pagination**: Avoid large `from` values
5. **Close clients**: Call `Close()` when done
6. **Handle errors**: Check for specific error types
7. **Set appropriate timeouts**: Configure based on your use case
