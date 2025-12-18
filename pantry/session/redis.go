// session/redis.go
package session

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Redis-backed session storage.
type RedisStore struct {
	client    redis.UniversalClient
	keyPrefix string
}

// RedisStoreConfig configures the Redis store.
type RedisStoreConfig struct {
	// Client is an existing Redis client.
	// If provided, other connection options are ignored.
	Client redis.UniversalClient

	// Address is the Redis server address.
	Address string

	// Password for Redis authentication.
	Password string

	// DB is the database number.
	DB int

	// KeyPrefix is prepended to session keys.
	// Default: "session:".
	KeyPrefix string

	// PoolSize is the connection pool size.
	// Default: 10.
	PoolSize int
}

// NewRedisStore creates a Redis store with an existing client.
func NewRedisStore(client redis.UniversalClient) *RedisStore {
	return &RedisStore{
		client:    client,
		keyPrefix: "session:",
	}
}

// NewRedisStoreWithConfig creates a Redis store with custom configuration.
func NewRedisStoreWithConfig(cfg RedisStoreConfig) (*RedisStore, error) {
	var client redis.UniversalClient

	if cfg.Client != nil {
		client = cfg.Client
	} else {
		if cfg.Address == "" {
			return nil, errors.New("session: redis address required")
		}

		poolSize := cfg.PoolSize
		if poolSize == 0 {
			poolSize = 10
		}

		client = redis.NewClient(&redis.Options{
			Addr:     cfg.Address,
			Password: cfg.Password,
			DB:       cfg.DB,
			PoolSize: poolSize,
		})
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	keyPrefix := cfg.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "session:"
	}

	return &RedisStore{
		client:    client,
		keyPrefix: keyPrefix,
	}, nil
}

// ConnectRedis creates a Redis store with simple connection parameters.
func ConnectRedis(addr, password string, db int) (*RedisStore, error) {
	return NewRedisStoreWithConfig(RedisStoreConfig{
		Address:  addr,
		Password: password,
		DB:       db,
	})
}

// key returns the full Redis key for a session ID.
func (s *RedisStore) key(id string) string {
	return s.keyPrefix + id
}

// Load retrieves session data by ID.
func (s *RedisStore) Load(ctx context.Context, id string) (*SessionData, error) {
	data, err := s.client.Get(ctx, s.key(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrExpired
	}

	return &session, nil
}

// Save stores session data.
func (s *RedisStore) Save(ctx context.Context, data *SessionData) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Calculate TTL from expiration time
	ttl := time.Until(data.ExpiresAt)
	if ttl <= 0 {
		// Already expired, don't save
		return nil
	}

	return s.client.Set(ctx, s.key(data.ID), bytes, ttl).Err()
}

// Delete removes a session by ID.
func (s *RedisStore) Delete(ctx context.Context, id string) error {
	return s.client.Del(ctx, s.key(id)).Err()
}

// Close closes the Redis connection.
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Client returns the underlying Redis client.
func (s *RedisStore) Client() redis.UniversalClient {
	return s.client
}
