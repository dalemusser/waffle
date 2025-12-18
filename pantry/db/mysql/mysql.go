// db/mysql/mysql.go
package mysql

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Connect opens a MySQL connection pool using the given DSN and timeout.
// It performs a Ping to ensure the connection is usable before returning.
//
// The returned *sql.DB is a pool of connections, not a single connection.
// It is safe for concurrent use and should be reused throughout the application.
//
// The caller is responsible for calling db.Close() when done.
//
// DSN format:
//
//	user:password@tcp(host:port)/dbname
//	user:password@tcp(host:port)/dbname?parseTime=true
//	user:password@tcp(host:port)/dbname?parseTime=true&loc=Local
func Connect(dsn string, timeout time.Duration) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// ConnectWithConfig opens a MySQL connection pool with custom pool settings.
// It performs a Ping to ensure the connection is usable before returning.
//
// The caller is responsible for calling db.Close() when done.
//
// Example:
//
//	db, err := mysql.ConnectWithConfig("user:pass@tcp(localhost)/mydb", mysql.PoolConfig{
//	    MaxOpenConns:    25,
//	    MaxIdleConns:    5,
//	    ConnMaxLifetime: time.Hour,
//	    ConnMaxIdleTime: 10 * time.Minute,
//	}, 10*time.Second)
func ConnectWithConfig(dsn string, config PoolConfig, timeout time.Duration) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Apply pool configuration
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// PoolConfig holds connection pool settings for MySQL.
type PoolConfig struct {
	// MaxOpenConns sets the maximum number of open connections to the database.
	// Default (0) means unlimited.
	MaxOpenConns int

	// MaxIdleConns sets the maximum number of connections in the idle pool.
	// Default is 2.
	MaxIdleConns int

	// ConnMaxLifetime sets the maximum amount of time a connection may be reused.
	// Default (0) means connections are not closed due to age.
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime sets the maximum amount of time a connection may be idle.
	// Default (0) means connections are not closed due to idle time.
	ConnMaxIdleTime time.Duration
}

// DefaultPoolConfig returns sensible defaults for production use.
//
//	MaxOpenConns:    25
//	MaxIdleConns:    5
//	ConnMaxLifetime: 5 minutes
//	ConnMaxIdleTime: 5 minutes
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}
