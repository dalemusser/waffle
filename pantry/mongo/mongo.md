# mongo

MongoDB utilities for connection management, keyset pagination, cursor encoding, error detection, and URI validation.

## When to Use

Use `mongo` when working with MongoDB in a WAFFLE application for:

- Establishing connections with proper timeouts
- Implementing efficient keyset (cursor-based) pagination
- Detecting duplicate key errors reliably
- Validating MongoDB connection strings

## Import

```go
import "github.com/dalemusser/waffle/pantry/mongo"
```

## API

### Connect

```go
func Connect(ctx context.Context, uri string, dbName string) (*mongo.Client, error)
```

Opens a MongoDB connection with a bounded 10-second timeout. Pings the server to verify connectivity before returning. The caller is responsible for disconnecting the client.

**Example:**

```go
func (a *App) Startup(ctx context.Context) error {
    client, err := mongo.Connect(ctx, a.Config.MongoURI, a.Config.MongoDB)
    if err != nil {
        return fmt.Errorf("mongo connect: %w", err)
    }
    a.MongoClient = client
    a.DB = client.Database(a.Config.MongoDB)
    return nil
}

func (a *App) Shutdown(ctx context.Context) error {
    if a.MongoClient != nil {
        return a.MongoClient.Disconnect(ctx)
    }
    return nil
}
```

### ValidateURI

```go
func ValidateURI(raw string) error
```

Validates a MongoDB connection string without requiring the Mongo driver. Safe to use in configuration validation before connecting.

**Validation rules:**
- Must not be empty
- Must not contain `\r` or `\n`
- Must have `mongodb://` or `mongodb+srv://` scheme
- Must have a non-empty host

**Example:**

```go
func loadConfig() (*Config, error) {
    cfg := &Config{
        MongoURI: os.Getenv("MONGO_URI"),
    }
    if err := mongo.ValidateURI(cfg.MongoURI); err != nil {
        return nil, fmt.Errorf("invalid MONGO_URI: %w", err)
    }
    return cfg, nil
}
```

### IsDup

```go
func IsDup(err error) bool
```

Reports whether an error is a MongoDB duplicate key error (E11000). Handles all error types returned by the Go driver: `WriteException`, `BulkWriteException`, `CommandError`, and falls back to string matching for edge cases.

**Example:**

```go
func createUser(ctx context.Context, coll *mongo.Collection, user *User) error {
    _, err := coll.InsertOne(ctx, user)
    if mongo.IsDup(err) {
        return ErrEmailAlreadyExists
    }
    return err
}
```

## Keyset Pagination

Keyset pagination (also called cursor-based pagination) is more efficient than offset-based pagination for large datasets. It uses the last item's sort key and ID to fetch the next page.

### KeysetWindow

```go
func KeysetWindow(field, dir, key string, id any) bson.M
```

Composes a MongoDB `$or` filter for stable keyset pagination on a field with `_id` as tiebreaker.

**Parameters:**
- `field` — the field being sorted on (e.g., `"name_folded"`)
- `dir` — `"lt"` for descending/previous, `"gt"` for ascending/next
- `key` — the cursor's sort field value
- `id` — the cursor's `_id` value

**Example:**

```go
func listUsers(ctx context.Context, coll *mongo.Collection, cursor string, limit int) ([]User, string, error) {
    filter := bson.M{}

    // Apply cursor if provided
    if c, ok := mongo.DecodeCursor(cursor); ok {
        filter = mongo.KeysetWindow("name_folded", "gt", c.CI, c.ID)
    }

    opts := options.Find().
        SetSort(bson.D{{Key: "name_folded", Value: 1}, {Key: "_id", Value: 1}}).
        SetLimit(int64(limit + 1)) // Fetch one extra to detect more pages

    cur, err := coll.Find(ctx, filter, opts)
    if err != nil {
        return nil, "", err
    }
    defer cur.Close(ctx)

    var users []User
    if err := cur.All(ctx, &users); err != nil {
        return nil, "", err
    }

    // Check if there are more results
    var nextCursor string
    if len(users) > limit {
        users = users[:limit]
        last := users[limit-1]
        nextCursor = mongo.EncodeCursor(last.NameFolded, last.ID)
    }

    return users, nextCursor, nil
}
```

### Cursor Encoding

```go
// Cursor holds the sort key and ID for pagination
type Cursor struct {
    CI string             `json:"ci"` // Sort key value (e.g., folded name)
    ID primitive.ObjectID `json:"id"` // Document _id for tiebreaking
}

func EncodeCursor(ci string, id primitive.ObjectID) string
func DecodeCursor(s string) (Cursor, bool)
```

Cursors are encoded as URL-safe base64 JSON for use in query parameters.

**Example:**

```go
// Encode a cursor for the next page link
cursor := mongo.EncodeCursor(lastItem.NameFolded, lastItem.ID)
nextURL := fmt.Sprintf("/users?cursor=%s", cursor)

// Decode a cursor from a request
if c, ok := mongo.DecodeCursor(r.URL.Query().Get("cursor")); ok {
    // Use c.CI and c.ID to build the query
}
```

## Patterns

### WAFFLE Integration

Typical setup in a WAFFLE application:

```go
// internal/app/app.go
type App struct {
    Config      *config.Config
    MongoClient *mongo.Client
    DB          *mongo.Database
}

func (a *App) Startup(ctx context.Context) error {
    client, err := pmongo.Connect(ctx, a.Config.MongoURI, a.Config.MongoDB)
    if err != nil {
        return fmt.Errorf("mongo: %w", err)
    }
    a.MongoClient = client
    a.DB = client.Database(a.Config.MongoDB)
    return nil
}

func (a *App) Shutdown(ctx context.Context) error {
    if a.MongoClient != nil {
        return a.MongoClient.Disconnect(ctx)
    }
    return nil
}
```

Note: Import the pantry package with an alias to avoid collision with the driver:
```go
import pmongo "github.com/dalemusser/waffle/pantry/mongo"
```

### Paginated List Handler

```go
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    cursor := r.URL.Query().Get("cursor")
    const pageSize = 25

    items, nextCursor, err := h.repo.List(ctx, cursor, pageSize)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    data := map[string]any{
        "Items":      items,
        "NextCursor": nextCursor,
        "HasMore":    nextCursor != "",
    }
    templates.Render(w, "items_list", data)
}
```

### Upsert with Duplicate Detection

```go
func (r *Repo) CreateOrUpdate(ctx context.Context, item *Item) error {
    _, err := r.coll.InsertOne(ctx, item)
    if pmongo.IsDup(err) {
        // Item already exists, update instead
        _, err = r.coll.ReplaceOne(ctx,
            bson.M{"_id": item.ID},
            item,
        )
    }
    return err
}
```

## See Also

- [MongoDB Go Driver](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo) — Official driver documentation
