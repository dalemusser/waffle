// db/sqlite/sqlite.go
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Connect opens a SQLite database with sensible defaults for web applications.
// It enables WAL mode for better concurrency, foreign keys, and sets a busy timeout.
//
// The returned *sql.DB is a pool of connections. For SQLite, this is typically
// limited to a single writer, but multiple readers are supported in WAL mode.
//
// The caller is responsible for calling db.Close() when done.
//
// Path can be:
//   - A file path: "./data.db", "/var/lib/myapp/data.db"
//   - ":memory:" for an in-memory database (data lost on close)
//   - "file::memory:?cache=shared" for a shared in-memory database
func Connect(path string, timeout time.Duration) (*sql.DB, error) {
	return ConnectWithOptions(path, DefaultOptions(), timeout)
}

// ConnectWithOptions opens a SQLite database with custom options.
//
// The caller is responsible for calling db.Close() when done.
func ConnectWithOptions(path string, opts Options, timeout time.Duration) (*sql.DB, error) {
	// Build DSN with options
	dsn := buildDSN(path, opts)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// SQLite performs best with limited connections due to file locking
	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	if opts.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(opts.ConnMaxLifetime)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	// Apply pragmas that can't be set in DSN
	if err := applyPragmas(ctx, db, opts); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// Options configures SQLite database behavior.
type Options struct {
	// WALMode enables Write-Ahead Logging for better concurrent read performance.
	// Default: true (recommended for web applications)
	WALMode bool

	// ForeignKeys enables foreign key constraint enforcement.
	// Default: true
	ForeignKeys bool

	// BusyTimeout sets how long to wait when the database is locked (milliseconds).
	// Default: 5000 (5 seconds)
	BusyTimeout int

	// CacheSize sets the page cache size in KiB (negative) or pages (positive).
	// Default: -64000 (64MB)
	CacheSize int

	// Synchronous sets the synchronous mode for durability vs performance.
	// Options: "OFF", "NORMAL", "FULL", "EXTRA"
	// Default: "NORMAL" (good balance for WAL mode)
	Synchronous string

	// JournalMode overrides WALMode setting. Usually leave empty.
	// Options: "DELETE", "TRUNCATE", "PERSIST", "MEMORY", "WAL", "OFF"
	JournalMode string

	// MaxOpenConns limits concurrent connections. SQLite handles this internally,
	// but limiting helps avoid "database is locked" errors.
	// Default: 1 (recommended for most cases)
	MaxOpenConns int

	// MaxIdleConns sets idle connection pool size.
	// Default: 1
	MaxIdleConns int

	// ConnMaxLifetime sets maximum connection reuse time.
	// Default: 0 (no limit)
	ConnMaxLifetime time.Duration
}

// DefaultOptions returns sensible defaults for web applications.
//
// Settings:
//   - WAL mode enabled (better concurrency)
//   - Foreign keys enabled
//   - 5 second busy timeout
//   - 64MB cache
//   - NORMAL synchronous mode
//   - Single connection (avoids lock contention)
func DefaultOptions() Options {
	return Options{
		WALMode:      true,
		ForeignKeys:  true,
		BusyTimeout:  5000,
		CacheSize:    -64000, // 64MB in KiB
		Synchronous:  "NORMAL",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}
}

// ReadOnlyOptions returns options optimized for read-only access.
// Useful for read replicas or when you only need to query data.
func ReadOnlyOptions() Options {
	opts := DefaultOptions()
	opts.MaxOpenConns = 4 // Multiple readers are fine
	opts.MaxIdleConns = 2
	return opts
}

// InMemoryOptions returns options for in-memory databases.
// Data is lost when the connection closes.
func InMemoryOptions() Options {
	opts := DefaultOptions()
	opts.WALMode = false      // Not applicable for in-memory
	opts.Synchronous = "OFF"  // No durability needed
	opts.MaxOpenConns = 1     // Must be 1 for in-memory
	return opts
}

// buildDSN constructs the SQLite connection string.
func buildDSN(path string, opts Options) string {
	// Basic path
	dsn := path

	// Add query parameters
	params := make([]string, 0)

	if opts.BusyTimeout > 0 {
		params = append(params, fmt.Sprintf("_busy_timeout=%d", opts.BusyTimeout))
	}

	if opts.ForeignKeys {
		params = append(params, "_foreign_keys=on")
	}

	if len(params) > 0 {
		dsn += "?"
		for i, p := range params {
			if i > 0 {
				dsn += "&"
			}
			dsn += p
		}
	}

	return dsn
}

// applyPragmas sets SQLite pragmas that must be run as SQL statements.
func applyPragmas(ctx context.Context, db *sql.DB, opts Options) error {
	// Determine journal mode
	journalMode := opts.JournalMode
	if journalMode == "" && opts.WALMode {
		journalMode = "WAL"
	}

	if journalMode != "" {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA journal_mode=%s", journalMode)); err != nil {
			return fmt.Errorf("set journal_mode: %w", err)
		}
	}

	if opts.Synchronous != "" {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA synchronous=%s", opts.Synchronous)); err != nil {
			return fmt.Errorf("set synchronous: %w", err)
		}
	}

	if opts.CacheSize != 0 {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA cache_size=%d", opts.CacheSize)); err != nil {
			return fmt.Errorf("set cache_size: %w", err)
		}
	}

	return nil
}

// InMemory opens a shared in-memory SQLite database.
// Multiple connections can access the same in-memory database.
// Data is lost when all connections close.
//
// The caller is responsible for calling db.Close() when done.
func InMemory(timeout time.Duration) (*sql.DB, error) {
	return ConnectWithOptions("file::memory:?cache=shared", InMemoryOptions(), timeout)
}

// HealthCheck returns a health check function compatible with the health package.
//
// Example:
//
//	health.Mount(r, map[string]health.Check{
//	    "sqlite": sqlite.HealthCheck(db),
//	}, logger)
func HealthCheck(db *sql.DB) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return db.PingContext(ctx)
	}
}
