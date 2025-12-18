// cache/cache.go
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Cache defines the interface for cache implementations.
type Cache interface {
	// Get retrieves a value by key.
	// Returns ErrNotFound if the key doesn't exist or has expired.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with the given TTL.
	// If ttl is 0, the value never expires.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a key from the cache.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists.
	Exists(ctx context.Context, key string) (bool, error)

	// Clear removes all entries from the cache.
	Clear(ctx context.Context) error

	// Close releases any resources held by the cache.
	Close() error
}

// Common errors
var (
	ErrNotFound = errors.New("cache: key not found")
	ErrClosed   = errors.New("cache: cache is closed")
)

// GetJSON retrieves and unmarshals a JSON value.
func GetJSON[T any](ctx context.Context, c Cache, key string) (T, error) {
	var result T
	data, err := c.Get(ctx, key)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}
	return result, nil
}

// SetJSON marshals and stores a value as JSON.
func SetJSON(ctx context.Context, c Cache, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, data, ttl)
}

// GetOrSet retrieves a value, or computes and stores it if not found.
func GetOrSet(ctx context.Context, c Cache, key string, ttl time.Duration, compute func() ([]byte, error)) ([]byte, error) {
	// Try to get existing value
	data, err := c.Get(ctx, key)
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	// Compute new value
	data, err = compute()
	if err != nil {
		return nil, err
	}

	// Store it
	if err := c.Set(ctx, key, data, ttl); err != nil {
		return nil, err
	}

	return data, nil
}

// GetOrSetJSON retrieves a JSON value, or computes and stores it if not found.
func GetOrSetJSON[T any](ctx context.Context, c Cache, key string, ttl time.Duration, compute func() (T, error)) (T, error) {
	var result T

	// Try to get existing value
	data, err := c.Get(ctx, key)
	if err == nil {
		if err := json.Unmarshal(data, &result); err != nil {
			return result, err
		}
		return result, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return result, err
	}

	// Compute new value
	result, err = compute()
	if err != nil {
		return result, err
	}

	// Store it
	data, err = json.Marshal(result)
	if err != nil {
		return result, err
	}
	if err := c.Set(ctx, key, data, ttl); err != nil {
		return result, err
	}

	return result, nil
}

// Multi-key operations (optional interface)

// MultiGetter supports batch get operations.
type MultiGetter interface {
	GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)
}

// MultiSetter supports batch set operations.
type MultiSetter interface {
	SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) error
}

// MultiDeleter supports batch delete operations.
type MultiDeleter interface {
	DeleteMulti(ctx context.Context, keys []string) error
}
