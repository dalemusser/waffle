// cache/redis.go
package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis implements a Redis-backed cache.
type Redis struct {
	client    redis.UniversalClient
	keyPrefix string
}

// RedisConfig configures the Redis cache.
type RedisConfig struct {
	// Client is an existing Redis client.
	// If provided, other connection options are ignored.
	Client redis.UniversalClient

	// Address is the Redis server address (e.g., "localhost:6379").
	Address string

	// Password for Redis authentication.
	Password string

	// DB is the database number to use.
	DB int

	// KeyPrefix is prepended to all keys.
	KeyPrefix string

	// PoolSize is the maximum number of connections.
	// Default: 10.
	PoolSize int

	// DialTimeout is the timeout for establishing connections.
	// Default: 5 seconds.
	DialTimeout time.Duration

	// ReadTimeout is the timeout for read operations.
	// Default: 3 seconds.
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for write operations.
	// Default: 3 seconds.
	WriteTimeout time.Duration
}

// NewRedis creates a new Redis cache with an existing client.
func NewRedis(client redis.UniversalClient) *Redis {
	return &Redis{
		client: client,
	}
}

// NewRedisWithConfig creates a Redis cache with custom configuration.
func NewRedisWithConfig(cfg RedisConfig) (*Redis, error) {
	var client redis.UniversalClient

	if cfg.Client != nil {
		client = cfg.Client
	} else {
		if cfg.Address == "" {
			return nil, errors.New("cache: redis address required")
		}

		opts := &redis.Options{
			Addr:     cfg.Address,
			Password: cfg.Password,
			DB:       cfg.DB,
		}

		if cfg.PoolSize > 0 {
			opts.PoolSize = cfg.PoolSize
		} else {
			opts.PoolSize = 10
		}

		if cfg.DialTimeout > 0 {
			opts.DialTimeout = cfg.DialTimeout
		} else {
			opts.DialTimeout = 5 * time.Second
		}

		if cfg.ReadTimeout > 0 {
			opts.ReadTimeout = cfg.ReadTimeout
		} else {
			opts.ReadTimeout = 3 * time.Second
		}

		if cfg.WriteTimeout > 0 {
			opts.WriteTimeout = cfg.WriteTimeout
		} else {
			opts.WriteTimeout = 3 * time.Second
		}

		client = redis.NewClient(opts)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Redis{
		client:    client,
		keyPrefix: cfg.KeyPrefix,
	}, nil
}

// Connect creates a Redis cache with simple connection parameters.
func Connect(addr, password string, db int) (*Redis, error) {
	return NewRedisWithConfig(RedisConfig{
		Address:  addr,
		Password: password,
		DB:       db,
	})
}

// prefixKey adds the key prefix.
func (r *Redis) prefixKey(key string) string {
	if r.keyPrefix == "" {
		return key
	}
	return r.keyPrefix + key
}

// Get retrieves a value by key.
func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := r.client.Get(ctx, r.prefixKey(key)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return result, nil
}

// Set stores a value with the given TTL.
func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, r.prefixKey(key), value, ttl).Err()
}

// Delete removes a key from the cache.
func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.prefixKey(key)).Err()
}

// Exists checks if a key exists.
func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, r.prefixKey(key)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Clear removes all entries with the key prefix.
// Warning: Uses SCAN which may be slow on large datasets.
func (r *Redis) Clear(ctx context.Context) error {
	if r.keyPrefix == "" {
		return r.client.FlushDB(ctx).Err()
	}

	// Scan and delete keys with prefix
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, r.keyPrefix+"*", 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// Close closes the Redis connection.
func (r *Redis) Close() error {
	return r.client.Close()
}

// GetMulti retrieves multiple values at once.
func (r *Redis) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Prefix all keys
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = r.prefixKey(key)
	}

	values, err := r.client.MGet(ctx, prefixedKeys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte, len(keys))
	for i, val := range values {
		if val == nil {
			continue
		}
		if str, ok := val.(string); ok {
			result[keys[i]] = []byte(str)
		}
	}

	return result, nil
}

// SetMulti stores multiple values at once.
func (r *Redis) SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()

	for key, value := range items {
		pipe.Set(ctx, r.prefixKey(key), value, ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// DeleteMulti removes multiple keys at once.
func (r *Redis) DeleteMulti(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = r.prefixKey(key)
	}

	return r.client.Del(ctx, prefixedKeys...).Err()
}

// SetNX sets a value only if the key doesn't exist.
// Returns true if the key was set.
func (r *Redis) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, r.prefixKey(key), value, ttl).Result()
}

// GetSet sets a value and returns the old value.
func (r *Redis) GetSet(ctx context.Context, key string, value []byte) ([]byte, error) {
	result, err := r.client.GetSet(ctx, r.prefixKey(key), value).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return result, nil
}

// Incr increments a numeric value and returns the new value.
func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, r.prefixKey(key)).Result()
}

// IncrBy increments a numeric value by delta and returns the new value.
func (r *Redis) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	return r.client.IncrBy(ctx, r.prefixKey(key), delta).Result()
}

// Decr decrements a numeric value and returns the new value.
func (r *Redis) Decr(ctx context.Context, key string) (int64, error) {
	return r.client.Decr(ctx, r.prefixKey(key)).Result()
}

// TTL returns the remaining TTL for a key.
// Returns -1 if the key has no TTL, -2 if the key doesn't exist.
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, r.prefixKey(key)).Result()
}

// Expire sets a TTL on an existing key.
func (r *Redis) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, r.prefixKey(key), ttl).Err()
}

// Client returns the underlying Redis client for advanced operations.
func (r *Redis) Client() redis.UniversalClient {
	return r.client
}
