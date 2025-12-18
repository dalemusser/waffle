// db/postgres/postgres.go
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a single PostgreSQL connection using the given connection string
// and timeout. It performs a Ping to ensure the connection is usable before returning.
//
// For production use with multiple concurrent requests, use ConnectPool instead.
//
// The caller is responsible for calling conn.Close(ctx) when done.
//
// Connection string examples:
//
//	"postgres://user:pass@localhost:5432/dbname"
//	"postgres://user:pass@localhost:5432/dbname?sslmode=disable"
//	"host=localhost port=5432 user=user password=pass dbname=dbname sslmode=disable"
func Connect(connString string, timeout time.Duration) (*pgx.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, err
	}

	return conn, nil
}

// ConnectPool opens a PostgreSQL connection pool using the given connection string
// and timeout. It performs a Ping to ensure the pool is usable before returning.
//
// This is the recommended approach for production applications handling multiple
// concurrent requests. The pool automatically manages connection lifecycle,
// handles reconnection, and provides connection reuse.
//
// The caller is responsible for calling pool.Close() when done.
//
// Connection string examples:
//
//	"postgres://user:pass@localhost:5432/dbname"
//	"postgres://user:pass@localhost:5432/dbname?pool_max_conns=10"
func ConnectPool(connString string, timeout time.Duration) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// ConnectPoolWithConfig opens a PostgreSQL connection pool using the provided
// configuration. Use this when you need fine-grained control over pool settings.
//
// The caller is responsible for calling pool.Close() when done.
//
// Example:
//
//	config, _ := pgxpool.ParseConfig(connString)
//	config.MaxConns = 20
//	config.MinConns = 5
//	config.MaxConnLifetime = time.Hour
//	pool, err := postgres.ConnectPoolWithConfig(config, 10*time.Second)
func ConnectPoolWithConfig(config *pgxpool.Config, timeout time.Duration) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// ParseConfig parses a connection string into a pool configuration that can be
// modified before connecting. This is a convenience wrapper around pgxpool.ParseConfig.
//
// Example:
//
//	config, err := postgres.ParseConfig("postgres://localhost/mydb")
//	if err != nil {
//	    return err
//	}
//	config.MaxConns = 20
//	pool, err := postgres.ConnectPoolWithConfig(config, 10*time.Second)
func ParseConfig(connString string) (*pgxpool.Config, error) {
	return pgxpool.ParseConfig(connString)
}
