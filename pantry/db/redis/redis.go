// db/redis/redis.go
package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client is an alias for the go-redis client, re-exported for convenience.
type Client = redis.Client

// Options is an alias for redis.Options, re-exported for convenience.
type Options = redis.Options

// Connect opens a Redis connection using the given address and timeout.
// It performs a Ping to ensure the connection is usable before returning.
//
// The caller is responsible for calling client.Close() when done.
//
// Address format: "host:port" (e.g., "localhost:6379")
func Connect(addr string, timeout time.Duration) (*Client, error) {
	return ConnectWithOptions(&Options{
		Addr: addr,
	}, timeout)
}

// ConnectWithPassword opens a Redis connection with authentication.
// It performs a Ping to ensure the connection is usable before returning.
//
// The caller is responsible for calling client.Close() when done.
func ConnectWithPassword(addr, password string, db int, timeout time.Duration) (*Client, error) {
	return ConnectWithOptions(&Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	}, timeout)
}

// ConnectWithOptions opens a Redis connection with full configuration control.
// It performs a Ping to ensure the connection is usable before returning.
//
// The caller is responsible for calling client.Close() when done.
//
// Example:
//
//	client, err := redis.ConnectWithOptions(&redis.Options{
//	    Addr:         "localhost:6379",
//	    Password:     "secret",
//	    DB:           0,
//	    PoolSize:     10,
//	    MinIdleConns: 2,
//	}, 10*time.Second)
func ConnectWithOptions(opts *Options, timeout time.Duration) (*Client, error) {
	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

// ConnectURL opens a Redis connection using a URL.
// It performs a Ping to ensure the connection is usable before returning.
//
// The caller is responsible for calling client.Close() when done.
//
// URL formats:
//
//	redis://localhost:6379
//	redis://:password@localhost:6379/0
//	rediss://localhost:6379 (TLS)
func ConnectURL(url string, timeout time.Duration) (*Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	return ConnectWithOptions(opts, timeout)
}

// ConnectCluster opens a Redis Cluster connection.
// It performs a Ping to ensure the connection is usable before returning.
//
// The caller is responsible for calling client.Close() when done.
//
// Example:
//
//	client, err := redis.ConnectCluster([]string{
//	    "node1:6379",
//	    "node2:6379",
//	    "node3:6379",
//	}, "", 10*time.Second)
func ConnectCluster(addrs []string, password string, timeout time.Duration) (*redis.ClusterClient, error) {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		Password: password,
	})

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

// ConnectSentinel opens a Redis connection via Sentinel for high availability.
// It performs a Ping to ensure the connection is usable before returning.
//
// The caller is responsible for calling client.Close() when done.
//
// Example:
//
//	client, err := redis.ConnectSentinel("mymaster", []string{
//	    "sentinel1:26379",
//	    "sentinel2:26379",
//	    "sentinel3:26379",
//	}, "", 10*time.Second)
func ConnectSentinel(masterName string, sentinelAddrs []string, password string, timeout time.Duration) (*Client, error) {
	client := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    masterName,
		SentinelAddrs: sentinelAddrs,
		Password:      password,
	})

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

// HealthCheck returns a health check function compatible with the health package.
//
// Example:
//
//	health.Mount(r, map[string]health.Check{
//	    "redis": redis.HealthCheck(redisClient),
//	}, logger)
func HealthCheck(client *Client) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return client.Ping(ctx).Err()
	}
}

// ClusterHealthCheck returns a health check function for Redis Cluster.
func ClusterHealthCheck(client *redis.ClusterClient) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return client.Ping(ctx).Err()
	}
}
