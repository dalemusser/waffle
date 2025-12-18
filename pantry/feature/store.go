// feature/store.go
package feature

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store is the interface for feature flag persistence.
type Store interface {
	// Load retrieves a flag by key.
	Load(key string) (*Flag, error)

	// LoadAll retrieves all flags.
	LoadAll() ([]*Flag, error)

	// Save persists a flag.
	Save(flag *Flag) error

	// Delete removes a flag.
	Delete(key string) error
}

// MemoryStore stores flags in memory.
type MemoryStore struct {
	mu    sync.RWMutex
	flags map[string]*Flag
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		flags: make(map[string]*Flag),
	}
}

// Load retrieves a flag by key.
func (s *MemoryStore) Load(key string) (*Flag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flag, exists := s.flags[key]
	if !exists {
		return nil, ErrFlagNotFound
	}
	return flag, nil
}

// LoadAll retrieves all flags.
func (s *MemoryStore) LoadAll() ([]*Flag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flags := make([]*Flag, 0, len(s.flags))
	for _, f := range s.flags {
		flags = append(flags, f)
	}
	return flags, nil
}

// Save persists a flag.
func (s *MemoryStore) Save(flag *Flag) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flags[flag.Key] = flag
	return nil
}

// Delete removes a flag.
func (s *MemoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.flags, key)
	return nil
}

// JSONStore stores flags in a JSON file.
type JSONStore struct {
	mu   sync.RWMutex
	path string
}

// NewJSONStore creates a new JSON file store.
func NewJSONStore(path string) *JSONStore {
	return &JSONStore{path: path}
}

// Load retrieves a flag by key.
func (s *JSONStore) Load(key string) (*Flag, error) {
	flags, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	for _, f := range flags {
		if f.Key == key {
			return f, nil
		}
	}
	return nil, ErrFlagNotFound
}

// LoadAll retrieves all flags from the JSON file.
func (s *JSONStore) LoadAll() ([]*Flag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Flag{}, nil
		}
		return nil, err
	}

	var flags []*Flag
	if err := json.Unmarshal(data, &flags); err != nil {
		return nil, err
	}

	return flags, nil
}

// Save persists a flag to the JSON file.
func (s *JSONStore) Save(flag *Flag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load existing flags
	flags, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	// Update or add flag
	found := false
	for i, f := range flags {
		if f.Key == flag.Key {
			flags[i] = flag
			found = true
			break
		}
	}
	if !found {
		flags = append(flags, flag)
	}

	return s.saveUnsafe(flags)
}

// Delete removes a flag from the JSON file.
func (s *JSONStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	flags, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	// Filter out the flag
	filtered := make([]*Flag, 0, len(flags))
	for _, f := range flags {
		if f.Key != key {
			filtered = append(filtered, f)
		}
	}

	return s.saveUnsafe(filtered)
}

func (s *JSONStore) loadUnsafe() ([]*Flag, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Flag{}, nil
		}
		return nil, err
	}

	var flags []*Flag
	if err := json.Unmarshal(data, &flags); err != nil {
		return nil, err
	}

	return flags, nil
}

func (s *JSONStore) saveUnsafe(flags []*Flag) error {
	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(flags, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

// EnvStore reads flags from environment variables.
// Flags are read as FEATURE_{KEY}=true/false.
type EnvStore struct {
	prefix string
	flags  map[string]*Flag
}

// NewEnvStore creates a new environment variable store.
func NewEnvStore(prefix string) *EnvStore {
	if prefix == "" {
		prefix = "FEATURE_"
	}
	return &EnvStore{
		prefix: prefix,
		flags:  make(map[string]*Flag),
	}
}

// Load retrieves a flag by key from environment.
func (s *EnvStore) Load(key string) (*Flag, error) {
	envKey := s.prefix + key
	value := os.Getenv(envKey)

	if value == "" {
		return nil, ErrFlagNotFound
	}

	enabled := value == "true" || value == "1" || value == "yes" || value == "on"
	return &Flag{
		Key:     key,
		Enabled: enabled,
	}, nil
}

// LoadAll retrieves all flags from environment.
// Note: This requires pre-registered flag keys since env vars can't be enumerated by prefix easily.
func (s *EnvStore) LoadAll() ([]*Flag, error) {
	flags := make([]*Flag, 0, len(s.flags))
	for key := range s.flags {
		if flag, err := s.Load(key); err == nil {
			flags = append(flags, flag)
		}
	}
	return flags, nil
}

// Save is a no-op for EnvStore (env vars are read-only).
func (s *EnvStore) Save(flag *Flag) error {
	// Store key for LoadAll enumeration
	s.flags[flag.Key] = flag
	return nil
}

// Delete is a no-op for EnvStore.
func (s *EnvStore) Delete(key string) error {
	delete(s.flags, key)
	return nil
}

// Register registers a flag key for enumeration.
func (s *EnvStore) Register(key string) {
	s.flags[key] = &Flag{Key: key}
}

// CompositeStore combines multiple stores with fallback.
type CompositeStore struct {
	stores []Store
}

// NewCompositeStore creates a store that checks multiple stores in order.
func NewCompositeStore(stores ...Store) *CompositeStore {
	return &CompositeStore{stores: stores}
}

// Load retrieves a flag from the first store that has it.
func (s *CompositeStore) Load(key string) (*Flag, error) {
	for _, store := range s.stores {
		if flag, err := store.Load(key); err == nil {
			return flag, nil
		}
	}
	return nil, ErrFlagNotFound
}

// LoadAll retrieves all flags from all stores, later stores override earlier ones.
func (s *CompositeStore) LoadAll() ([]*Flag, error) {
	flagMap := make(map[string]*Flag)

	for _, store := range s.stores {
		flags, err := store.LoadAll()
		if err != nil {
			continue
		}
		for _, f := range flags {
			flagMap[f.Key] = f
		}
	}

	flags := make([]*Flag, 0, len(flagMap))
	for _, f := range flagMap {
		flags = append(flags, f)
	}
	return flags, nil
}

// Save persists to the first store only.
func (s *CompositeStore) Save(flag *Flag) error {
	if len(s.stores) == 0 {
		return nil
	}
	return s.stores[0].Save(flag)
}

// Delete removes from the first store only.
func (s *CompositeStore) Delete(key string) error {
	if len(s.stores) == 0 {
		return nil
	}
	return s.stores[0].Delete(key)
}

// MapStore creates a store from a static map (read-only).
type MapStore struct {
	flags map[string]*Flag
}

// NewMapStore creates a store from a map of flags.
func NewMapStore(flags map[string]*Flag) *MapStore {
	return &MapStore{flags: flags}
}

// NewMapStoreSimple creates a store from a simple key->enabled map.
func NewMapStoreSimple(flags map[string]bool) *MapStore {
	m := make(map[string]*Flag)
	for k, v := range flags {
		m[k] = &Flag{Key: k, Enabled: v}
	}
	return &MapStore{flags: m}
}

// Load retrieves a flag by key.
func (s *MapStore) Load(key string) (*Flag, error) {
	flag, exists := s.flags[key]
	if !exists {
		return nil, ErrFlagNotFound
	}
	return flag, nil
}

// LoadAll retrieves all flags.
func (s *MapStore) LoadAll() ([]*Flag, error) {
	flags := make([]*Flag, 0, len(s.flags))
	for _, f := range s.flags {
		flags = append(flags, f)
	}
	return flags, nil
}

// Save is a no-op for MapStore (read-only).
func (s *MapStore) Save(flag *Flag) error {
	return nil
}

// Delete is a no-op for MapStore (read-only).
func (s *MapStore) Delete(key string) error {
	return nil
}
