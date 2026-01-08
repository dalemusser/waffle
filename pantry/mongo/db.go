// pantry/mongo/db.go
package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const mongoConnectTimeout = 10 * time.Second

// PoolConfig holds connection pool settings for MongoDB.
// Use DefaultPoolConfig() for sensible defaults.
type PoolConfig struct {
	// MaxPoolSize is the maximum number of connections in the pool.
	// Default: 100 (MongoDB driver default)
	MaxPoolSize uint64

	// MinPoolSize is the minimum number of connections to keep open.
	// Default: 0 (MongoDB driver default)
	MinPoolSize uint64

	// MaxConnIdleTime is how long a connection can be idle before being closed.
	// Default: 0 (no limit, MongoDB driver default)
	MaxConnIdleTime time.Duration

	// ConnectTimeout is the timeout for establishing a connection.
	// Default: 10 seconds
	ConnectTimeout time.Duration

	// ServerSelectionTimeout is the timeout for selecting a server.
	// Default: 30 seconds (MongoDB driver default)
	ServerSelectionTimeout time.Duration
}

// DefaultPoolConfig returns sensible pool defaults for production use.
// These are more conservative than MongoDB driver defaults.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxPoolSize:            100,
		MinPoolSize:            10,
		MaxConnIdleTime:        5 * time.Minute,
		ConnectTimeout:         10 * time.Second,
		ServerSelectionTimeout: 10 * time.Second,
	}
}

// HighTrafficPoolConfig returns pool settings optimized for high-traffic apps.
// Use for applications expecting 100+ concurrent database operations.
func HighTrafficPoolConfig() PoolConfig {
	return PoolConfig{
		MaxPoolSize:            300,
		MinPoolSize:            50,
		MaxConnIdleTime:        5 * time.Minute,
		ConnectTimeout:         10 * time.Second,
		ServerSelectionTimeout: 10 * time.Second,
	}
}

// Connect opens a Mongo connection with a bounded timeout derived from the
// provided parent context. Uses MongoDB driver default pool settings.
// The returned client must be disconnected by the caller.
func Connect(ctx context.Context, uri string, dbName string) (*mongo.Client, error) {
	// Derive a timeout from the parent context so connection attempts
	// do not hang indefinitely.
	ctx, cancel := context.WithTimeout(ctx, mongoConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

// ConnectWithPool opens a Mongo connection with custom pool settings.
// Use DefaultPoolConfig() or HighTrafficPoolConfig() as a starting point.
// The returned client must be disconnected by the caller.
//
// Example:
//
//	cfg := mongo.DefaultPoolConfig()
//	cfg.MaxPoolSize = 200  // Override specific settings
//	client, err := mongo.ConnectWithPool(ctx, uri, dbName, cfg)
func ConnectWithPool(ctx context.Context, uri string, dbName string, pool PoolConfig) (*mongo.Client, error) {
	connectTimeout := pool.ConnectTimeout
	if connectTimeout == 0 {
		connectTimeout = mongoConnectTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)

	if pool.MaxPoolSize > 0 {
		clientOpts.SetMaxPoolSize(pool.MaxPoolSize)
	}
	if pool.MinPoolSize > 0 {
		clientOpts.SetMinPoolSize(pool.MinPoolSize)
	}
	if pool.MaxConnIdleTime > 0 {
		clientOpts.SetMaxConnIdleTime(pool.MaxConnIdleTime)
	}
	if pool.ConnectTimeout > 0 {
		clientOpts.SetConnectTimeout(pool.ConnectTimeout)
	}
	if pool.ServerSelectionTimeout > 0 {
		clientOpts.SetServerSelectionTimeout(pool.ServerSelectionTimeout)
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}
