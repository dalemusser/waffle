# text

Unicode-aware text folding for case-insensitive search and prefix matching.

## When to Use

Use `text` when you need to:

- Normalize user input for case-insensitive search
- Build prefix queries that work across languages
- Store a "folded" version of text for indexed lookups
- Tokenize search input into normalized terms

## Import

```go
import "github.com/dalemusser/waffle/pantry/text"
```

## Quick Start

```go
// Normalize for storage/comparison
folded := text.Fold("Café")  // "cafe"

// Build a prefix range for database queries
lo, hi := text.PrefixRange("Mül")  // "mul", "mul\U0010FFFD"

// Tokenize search input
tokens := text.FoldTokens("José García")  // ["jose", "garcia"]
```

## API

### Fold

**Location:** `fold.go`

```go
func Fold(s string) string
```

Lowercases and strips combining diacritics from a string using Unicode normalization (NFD → remove combining marks → NFC). Returns an empty string for blank or whitespace-only input.

**Behavior:**
- Trims whitespace
- Lowercases using Unicode-aware rules
- Removes combining diacritical marks (accents, umlauts, etc.)
- Does NOT convert to ASCII — characters like "ø" or "ß" remain

**Examples:**

```go
text.Fold("Café")        // "cafe"
text.Fold("MÜNCHEN")     // "munchen"
text.Fold("José García") // "jose garcia"
text.Fold("Ångström")    // "angstrom"
text.Fold("  hello  ")   // "hello"
text.Fold("")            // ""
text.Fold("   ")         // ""
```

### FoldTokens

**Location:** `fold.go`

```go
func FoldTokens(s string) []string
```

Folds the input string and splits it into whitespace-separated tokens. Returns `nil` for empty input. Useful for building multi-term search queries.

**Examples:**

```go
text.FoldTokens("José García")    // ["jose", "garcia"]
text.FoldTokens("  hello world ") // ["hello", "world"]
text.FoldTokens("")               // nil
```

### PrefixRange

**Location:** `fold.go`

```go
func PrefixRange(q string) (lo, hi string)
```

Returns a half-open range `[lo, hi)` for prefix queries. The input is folded, and `hi` is computed by appending a high Unicode sentinel to ensure all strings starting with the prefix fall within the range.

**Examples:**

```go
lo, hi := text.PrefixRange("Mül")
// lo = "mul"
// hi = "mul\U0010FFFD"

// Use in MongoDB:
filter := bson.M{
    "name_folded": bson.M{
        "$gte": lo,
        "$lt":  hi,
    },
}
```

### HiFromFolded

**Location:** `fold.go`

```go
func HiFromFolded(folded string) string
```

Returns `folded + High`. Use when you've already computed `Fold(q)` and need the upper bound without re-folding.

**Example:**

```go
folded := text.Fold(userInput)
hi := text.HiFromFolded(folded)
```

### High

**Location:** `fold.go`

```go
const High = "\U0010FFFD"
```

The sentinel character used to form exclusive upper bounds for prefix ranges. U+10FFFD is the highest valid Unicode scalar value that's not a noncharacter, ensuring astral-plane characters (emoji, etc.) fall within prefix ranges.

## Patterns

### Storing Folded Text for Search

Store a folded version alongside the original for indexed lookups:

```go
type User struct {
    ID         primitive.ObjectID `bson:"_id"`
    Name       string             `bson:"name"`
    NameFolded string             `bson:"name_folded"` // Index this field
}

func (r *Repo) Create(ctx context.Context, name string) (*User, error) {
    user := &User{
        ID:         primitive.NewObjectID(),
        Name:       name,
        NameFolded: text.Fold(name),
    }
    _, err := r.coll.InsertOne(ctx, user)
    return user, err
}
```

### Prefix Search

```go
func (r *Repo) SearchByPrefix(ctx context.Context, query string) ([]User, error) {
    lo, hi := text.PrefixRange(query)
    if lo == "" {
        return nil, nil // Empty query
    }

    filter := bson.M{
        "name_folded": bson.M{
            "$gte": lo,
            "$lt":  hi,
        },
    }

    cur, err := r.coll.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cur.Close(ctx)

    var users []User
    return users, cur.All(ctx, &users)
}
```

### Multi-Term Search

```go
func (r *Repo) Search(ctx context.Context, query string) ([]User, error) {
    tokens := text.FoldTokens(query)
    if len(tokens) == 0 {
        return nil, nil
    }

    // All tokens must match as prefixes
    var conditions []bson.M
    for _, tok := range tokens {
        hi := text.HiFromFolded(tok)
        conditions = append(conditions, bson.M{
            "name_folded": bson.M{"$gte": tok, "$lt": hi},
        })
    }

    filter := bson.M{"$and": conditions}
    // ... execute query
}
```

## Performance

- **ASCII fast path**: Strings that are already ASCII lowercase skip Unicode transformation
- **Pooled transformers**: The NFD→NFC pipeline uses `sync.Pool` to avoid allocations
- **Index-friendly**: Folded strings work with standard B-tree indexes for prefix queries

## See Also

- [mongo](../mongo/mongo.md) — Keyset pagination utilities that pair well with folded search
