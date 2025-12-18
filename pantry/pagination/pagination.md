# pagination

Standard pagination helpers for WAFFLE APIs.

## Overview

The `pagination` package provides:
- **Offset pagination** — Traditional page/per_page pagination
- **Cursor pagination** — Opaque cursor-based pagination
- **Keyset pagination** — Efficient seek-based pagination for large datasets
- **Response helpers** — Links, headers, and result wrappers

## Import

```go
import "github.com/dalemusser/waffle/pagination"
```

---

## Quick Start

```go
func listUsers(w http.ResponseWriter, r *http.Request) {
    // Get pagination from request
    p := pagination.FromRequest(r)  // ?page=1&per_page=20

    // Query database
    users, total := db.ListUsers(p.Offset(), p.Limit())
    p.SetTotal(total)

    // Return paginated result
    result := pagination.NewResult(users, p)
    json.NewEncoder(w).Encode(result)
}
```

---

## Offset Pagination

Traditional page-based pagination. Simple but can be slow for large offsets.

**Location:** `pagination.go`

### Parsing from Request

```go
// Default: ?page=1&per_page=20 or ?page=1&limit=20
p := pagination.FromRequest(r)

// With custom defaults
p := pagination.FromRequestWithDefaults(r, 50, 200)  // default 50, max 200

// Manual creation
p := pagination.New(1, 20)  // page 1, 20 per page
p := pagination.Default()   // page 1, 20 per page
```

### Using with Database

```go
func listItems(ctx context.Context, p pagination.Page) ([]Item, int) {
    // Get total count
    var total int
    db.QueryRow("SELECT COUNT(*) FROM items").Scan(&total)

    // Get page of data
    rows, _ := db.Query(
        "SELECT * FROM items ORDER BY id LIMIT $1 OFFSET $2",
        p.Limit(),
        p.Offset(),
    )

    var items []Item
    for rows.Next() {
        var item Item
        rows.Scan(&item.ID, &item.Name)
        items = append(items, item)
    }

    return items, total
}
```

### Page Methods

```go
p := pagination.FromRequest(r)

// Database helpers
offset := p.Offset()  // (page-1) * per_page
limit := p.Limit()    // per_page

// After setting total
p.SetTotal(100)

// Navigation
hasNext := p.HasNext()
hasPrev := p.HasPrev()
nextPage := p.Next()
prevPage := p.Prev()

// Metadata
totalPages := p.TotalPages
```

### Result Types

```go
// Basic result
type Result[T any] struct {
    Data       []T  `json:"data"`
    Page       int  `json:"page"`
    PerPage    int  `json:"per_page"`
    Total      int  `json:"total"`
    TotalPages int  `json:"total_pages"`
    HasNext    bool `json:"has_next"`
    HasPrev    bool `json:"has_prev"`
}

// Usage
items, total := listItems(ctx, p)
p.SetTotal(total)
result := pagination.NewResult(items, p)

// Result with links
result := pagination.NewResultWithLinks(items, p, "/api/items")
// Includes: self, first, last, next, prev URLs
```

### Link Headers (RFC 5988)

```go
func handler(w http.ResponseWriter, r *http.Request) {
    p := pagination.FromRequest(r)
    items, total := listItems(ctx, p)
    p.SetTotal(total)

    // Set Link header and X-Total-Count
    pagination.SetLinkHeader(w, p, r.URL.Path)
    // Link: </api/items?page=1&per_page=20>; rel="first", ...
    // X-Total-Count: 100
    // X-Total-Pages: 5
    // X-Page: 1
    // X-Per-Page: 20

    json.NewEncoder(w).Encode(items)
}
```

---

## Cursor Pagination

Uses opaque cursors instead of page numbers. Better for real-time data where items may be added/removed.

### Parsing from Request

```go
// ?cursor=xxx&limit=20
c := pagination.CursorFromRequest(r)

cursor := c.Cursor    // opaque cursor string
limit := c.Limit      // number of items
isFirst := c.IsFirst() // true if no cursor (first page)
```

### Creating Cursors

```go
// Encode any data as cursor
type MyCursor struct {
    LastID    string
    Timestamp int64
}

cursor, _ := pagination.EncodeCursor(MyCursor{
    LastID:    "123",
    Timestamp: time.Now().Unix(),
})

// Decode cursor
var decoded MyCursor
pagination.DecodeCursor(cursor, &decoded)
```

### Built-in Cursor Types

```go
// ID-based cursor
cursor := pagination.EncodeIDCursor("item-123", "next")
id, direction, _ := pagination.DecodeIDCursor(cursor)

// Time + ID cursor (stable ordering)
cursor := pagination.EncodeTimeCursor(timestamp, "item-123", "next")
ts, id, dir, _ := pagination.DecodeTimeCursor(cursor)
```

### Cursor Result

```go
func listItems(r *http.Request) CursorResult[Item] {
    c := pagination.CursorFromRequest(r)

    // Decode cursor if present
    var lastID string
    if !c.IsFirst() {
        lastID, _, _ = pagination.DecodeIDCursor(c.Cursor)
    }

    // Query with cursor
    items := db.ListItemsAfter(lastID, c.Limit+1)

    // Check if there are more
    hasMore := len(items) > c.Limit
    if hasMore {
        items = items[:c.Limit]
    }

    // Create next cursor
    var nextCursor string
    if hasMore && len(items) > 0 {
        lastItem := items[len(items)-1]
        nextCursor = pagination.EncodeIDCursor(lastItem.ID, "next")
    }

    return pagination.NewCursorResult(items, nextCursor, "", hasMore)
}
```

---

## Keyset Pagination

Also known as "seek method". Most efficient for large datasets - doesn't slow down at high page numbers.

**Location:** `keyset.go`

### Parsing from Request

```go
// ?limit=20&after=xxx or ?limit=20&before=xxx
k := pagination.KeysetFromRequest(r)

limit := k.Limit
after := k.After      // pagination forward
before := k.Before    // pagination backward
isFirst := k.IsFirst()
isForward := k.IsForward()
```

### Using with SQL

```go
func listItems(ctx context.Context, k pagination.Keyset) ([]Item, bool) {
    builder := pagination.NewSQLBuilder("id", false) // ascending by id

    // Build query
    where, args := builder.WhereClause(k)
    order := builder.OrderClause()
    limit := builder.LimitClause(k)

    query := "SELECT * FROM items"
    if where != "" {
        query += " WHERE " + where
    }
    query += " ORDER BY " + order + " " + limit

    rows, _ := db.Query(query, args...)

    var items []Item
    for rows.Next() {
        var item Item
        rows.Scan(&item.ID, &item.Name)
        items = append(items, item)
    }

    // Process results (trim extra item, check hasMore)
    return pagination.ProcessResults(items, k.Limit)
}
```

### Keyset with Postgres

```go
func listItems(ctx context.Context, k pagination.Keyset) ([]Item, bool) {
    builder := pagination.NewSQLBuilder("created_at", true). // descending
        WithPlaceholder("$1")

    where, args := builder.WhereClause(k)

    query := `
        SELECT id, name, created_at FROM items
        WHERE deleted_at IS NULL
    `
    if where != "" {
        query += " AND " + where
    }
    query += " ORDER BY " + builder.OrderClause()
    query += " " + builder.LimitClause(k)

    // ... execute query
}
```

### Keyset Result

```go
func handler(w http.ResponseWriter, r *http.Request) {
    k := pagination.KeysetFromRequest(r)
    items, hasMore := listItems(ctx, k)

    var firstID, lastID string
    if len(items) > 0 {
        firstID = items[0].ID
        lastID = items[len(items)-1].ID
    }

    // With links
    result := pagination.NewKeysetResultWithLinks(items, k, hasMore, firstID, lastID, "/api/items")
    json.NewEncoder(w).Encode(result)

    // Or set Link header
    pagination.SetKeysetLinkHeader(w, k, hasMore, firstID, lastID, "/api/items")
}
```

---

## Choosing a Pagination Strategy

| Strategy | Best For | Pros | Cons |
|----------|----------|------|------|
| **Offset** | Small datasets, admin UIs | Simple, jump to any page | Slow for large offsets |
| **Cursor** | Real-time feeds, infinite scroll | Stable with changing data | Can't jump to page N |
| **Keyset** | Large datasets, logs | Fast at any position | Requires sortable column |

### When to Use Each

**Offset Pagination:**
- Admin dashboards with < 10k items
- When users need to jump to specific pages
- When dataset changes rarely

**Cursor Pagination:**
- Social feeds, activity streams
- Real-time data with frequent inserts
- Mobile infinite scroll

**Keyset Pagination:**
- Large datasets (millions of rows)
- Log/audit data ordered by timestamp
- When performance at high pages matters

---

## WAFFLE Integration

### Complete Offset Example

```go
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    p := pagination.FromRequest(r)

    // Query
    users, err := db.ListUsers(r.Context(), p.Offset(), p.Limit())
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // Get total
    total, _ := db.CountUsers(r.Context())
    p.SetTotal(total)

    // Set headers
    pagination.SetLinkHeader(w, p, r.URL.Path)

    // Return result
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(pagination.NewResult(users, p))
}
```

### Complete Cursor Example

```go
func listNotificationsHandler(w http.ResponseWriter, r *http.Request) {
    userID := auth.UserID(r.Context())
    c := pagination.CursorFromRequest(r)

    // Decode cursor
    var afterTime int64
    var afterID string
    if !c.IsFirst() {
        afterTime, afterID, _, _ = pagination.DecodeTimeCursor(c.Cursor)
    }

    // Query one extra to check for more
    notifications, err := db.ListNotifications(r.Context(), userID, afterTime, afterID, c.Limit+1)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // Check for more
    hasMore := len(notifications) > c.Limit
    if hasMore {
        notifications = notifications[:c.Limit]
    }

    // Build cursors
    var nextCursor string
    if hasMore && len(notifications) > 0 {
        last := notifications[len(notifications)-1]
        nextCursor = pagination.EncodeTimeCursor(last.CreatedAt.Unix(), last.ID, "next")
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(pagination.NewCursorResult(notifications, nextCursor, "", hasMore))
}
```

### Complete Keyset Example

```go
func listAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
    k := pagination.KeysetFromRequest(r)
    builder := pagination.NewSQLBuilder("timestamp", true).WithPlaceholder("$1")

    // Build query
    query := "SELECT id, action, timestamp FROM audit_logs"
    where, args := builder.WhereClause(k)
    if where != "" {
        query += " WHERE " + where
    }
    query += " ORDER BY " + builder.OrderClause()
    query += " " + builder.LimitClause(k)

    rows, err := db.Query(query, args...)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    defer rows.Close()

    var logs []AuditLog
    for rows.Next() {
        var log AuditLog
        rows.Scan(&log.ID, &log.Action, &log.Timestamp)
        logs = append(logs, log)
    }

    // Process results
    logs, hasMore := pagination.ProcessResults(logs, k.Limit)

    var firstID, lastID string
    if len(logs) > 0 {
        firstID = logs[0].ID
        lastID = logs[len(logs)-1].ID
    }

    pagination.SetKeysetLinkHeader(w, k, hasMore, firstID, lastID, r.URL.Path)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(pagination.NewKeysetResult(logs, hasMore, firstID, lastID))
}
```

---

## Constants

```go
pagination.DefaultPage    // 1
pagination.DefaultPerPage // 20
pagination.MaxPerPage     // 100
```

---

## See Also

- [httputil](../httputil/httputil.md) — HTTP utilities
- [db/postgres](../db/postgres/postgres.md) — PostgreSQL helpers
