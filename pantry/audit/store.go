// audit/store.go
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Store is the interface for audit event storage.
type Store interface {
	// Store saves an audit event.
	Store(ctx context.Context, event *Event) error

	// Query retrieves events matching the criteria.
	Query(ctx context.Context, query *Query) (*QueryResult, error)

	// Close closes the store.
	Close() error
}

// MemoryStore stores events in memory.
// Useful for testing and development.
type MemoryStore struct {
	mu     sync.RWMutex
	events []*Event
	maxLen int
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore(maxEvents int) *MemoryStore {
	if maxEvents <= 0 {
		maxEvents = 10000
	}
	return &MemoryStore{
		events: make([]*Event, 0, maxEvents),
		maxLen: maxEvents,
	}
}

// Store saves an event to memory.
func (s *MemoryStore) Store(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove oldest if at capacity
	if len(s.events) >= s.maxLen {
		s.events = s.events[1:]
	}

	s.events = append(s.events, event)
	return nil
}

// Query retrieves events from memory.
func (s *MemoryStore) Query(ctx context.Context, query *Query) (*QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter events
	var filtered []*Event
	for _, e := range s.events {
		if matchesQuery(e, query) {
			filtered = append(filtered, e)
		}
	}

	// Sort
	sortEvents(filtered, query.OrderBy, query.OrderDesc)

	// Paginate
	total := len(filtered)
	start := query.Offset
	if start > total {
		start = total
	}

	end := total
	if query.Limit > 0 {
		end = start + query.Limit
		if end > total {
			end = total
		}
	}

	return &QueryResult{
		Events:  filtered[start:end],
		Total:   total,
		HasMore: end < total,
	}, nil
}

// Close closes the memory store.
func (s *MemoryStore) Close() error {
	return nil
}

// Events returns all stored events (for testing).
func (s *MemoryStore) Events() []*Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]*Event{}, s.events...)
}

// Clear removes all events (for testing).
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = s.events[:0]
}

// FileStore stores events to a file (JSON lines format).
type FileStore struct {
	mu       sync.Mutex
	path     string
	file     *os.File
	encoder  *json.Encoder
	maxSize  int64
	rotation int
}

// FileStoreConfig configures the file store.
type FileStoreConfig struct {
	// Path is the file path.
	Path string

	// MaxSize is the maximum file size before rotation (in bytes).
	// Default: 100MB
	MaxSize int64

	// Rotation is the number of rotated files to keep.
	// Default: 5
	Rotation int
}

// NewFileStore creates a new file-based store.
func NewFileStore(cfg FileStoreConfig) (*FileStore, error) {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 100 * 1024 * 1024 // 100MB
	}
	if cfg.Rotation <= 0 {
		cfg.Rotation = 5
	}

	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("audit: failed to create directory: %w", err)
	}

	// Open file for appending
	file, err := os.OpenFile(cfg.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("audit: failed to open file: %w", err)
	}

	return &FileStore{
		path:     cfg.Path,
		file:     file,
		encoder:  json.NewEncoder(file),
		maxSize:  cfg.MaxSize,
		rotation: cfg.Rotation,
	}, nil
}

// Store saves an event to the file.
func (s *FileStore) Store(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if rotation needed
	if err := s.checkRotation(); err != nil {
		return err
	}

	return s.encoder.Encode(event)
}

// checkRotation rotates the file if needed.
func (s *FileStore) checkRotation() error {
	info, err := s.file.Stat()
	if err != nil {
		return err
	}

	if info.Size() < s.maxSize {
		return nil
	}

	// Close current file
	s.file.Close()

	// Rotate files
	for i := s.rotation - 1; i > 0; i-- {
		old := fmt.Sprintf("%s.%d", s.path, i)
		new := fmt.Sprintf("%s.%d", s.path, i+1)
		os.Rename(old, new)
	}

	// Move current to .1
	os.Rename(s.path, s.path+".1")

	// Open new file
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	s.file = file
	s.encoder = json.NewEncoder(file)
	return nil
}

// Query retrieves events from the file.
func (s *FileStore) Query(ctx context.Context, query *Query) (*QueryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read all events from file and rotated files
	var allEvents []*Event

	// Read rotated files first (oldest)
	for i := s.rotation; i > 0; i-- {
		rotatedPath := fmt.Sprintf("%s.%d", s.path, i)
		events, _ := s.readFile(rotatedPath)
		allEvents = append(allEvents, events...)
	}

	// Read current file
	events, _ := s.readFile(s.path)
	allEvents = append(allEvents, events...)

	// Filter
	var filtered []*Event
	for _, e := range allEvents {
		if matchesQuery(e, query) {
			filtered = append(filtered, e)
		}
	}

	// Sort
	sortEvents(filtered, query.OrderBy, query.OrderDesc)

	// Paginate
	total := len(filtered)
	start := query.Offset
	if start > total {
		start = total
	}

	end := total
	if query.Limit > 0 {
		end = start + query.Limit
		if end > total {
			end = total
		}
	}

	return &QueryResult{
		Events:  filtered[start:end],
		Total:   total,
		HasMore: end < total,
	}, nil
}

// readFile reads events from a single file.
func (s *FileStore) readFile(path string) ([]*Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var events []*Event
	decoder := json.NewDecoder(file)

	for {
		var event Event
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			continue // Skip malformed lines
		}
		events = append(events, &event)
	}

	return events, nil
}

// Close closes the file store.
func (s *FileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.file.Close()
}

// WriterStore writes events to an io.Writer.
type WriterStore struct {
	mu      sync.Mutex
	writer  io.Writer
	encoder *json.Encoder
}

// NewWriterStore creates a store that writes to an io.Writer.
func NewWriterStore(w io.Writer) *WriterStore {
	return &WriterStore{
		writer:  w,
		encoder: json.NewEncoder(w),
	}
}

// Store writes an event.
func (s *WriterStore) Store(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.encoder.Encode(event)
}

// Query is not supported for WriterStore.
func (s *WriterStore) Query(ctx context.Context, query *Query) (*QueryResult, error) {
	return nil, ErrQueryNotSupported
}

// Close is a no-op for WriterStore.
func (s *WriterStore) Close() error {
	return nil
}

// MultiStore writes to multiple stores.
type MultiStore struct {
	stores []Store
}

// NewMultiStore creates a store that writes to multiple backends.
func NewMultiStore(stores ...Store) *MultiStore {
	return &MultiStore{stores: stores}
}

// Store writes to all stores.
func (s *MultiStore) Store(ctx context.Context, event *Event) error {
	var lastErr error
	for _, store := range s.stores {
		if err := store.Store(ctx, event); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Query queries the first store that supports it.
func (s *MultiStore) Query(ctx context.Context, query *Query) (*QueryResult, error) {
	for _, store := range s.stores {
		result, err := store.Query(ctx, query)
		if err == nil {
			return result, nil
		}
		if err != ErrQueryNotSupported {
			return nil, err
		}
	}
	return nil, ErrQueryNotSupported
}

// Close closes all stores.
func (s *MultiStore) Close() error {
	var lastErr error
	for _, store := range s.stores {
		if err := store.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// FilterStore filters events before storing.
type FilterStore struct {
	store  Store
	filter func(*Event) bool
}

// NewFilterStore creates a store that filters events.
func NewFilterStore(store Store, filter func(*Event) bool) *FilterStore {
	return &FilterStore{
		store:  store,
		filter: filter,
	}
}

// Store stores the event if it passes the filter.
func (s *FilterStore) Store(ctx context.Context, event *Event) error {
	if s.filter != nil && !s.filter(event) {
		return nil // Skip filtered events
	}
	return s.store.Store(ctx, event)
}

// Query delegates to the underlying store.
func (s *FilterStore) Query(ctx context.Context, query *Query) (*QueryResult, error) {
	return s.store.Query(ctx, query)
}

// Close closes the underlying store.
func (s *FilterStore) Close() error {
	return s.store.Close()
}

// TransformStore transforms events before storing.
type TransformStore struct {
	store     Store
	transform func(*Event) *Event
}

// NewTransformStore creates a store that transforms events.
func NewTransformStore(store Store, transform func(*Event) *Event) *TransformStore {
	return &TransformStore{
		store:     store,
		transform: transform,
	}
}

// Store transforms and stores the event.
func (s *TransformStore) Store(ctx context.Context, event *Event) error {
	if s.transform != nil {
		event = s.transform(event)
	}
	return s.store.Store(ctx, event)
}

// Query delegates to the underlying store.
func (s *TransformStore) Query(ctx context.Context, query *Query) (*QueryResult, error) {
	return s.store.Query(ctx, query)
}

// Close closes the underlying store.
func (s *TransformStore) Close() error {
	return s.store.Close()
}

// NullStore discards all events.
type NullStore struct{}

// NewNullStore creates a store that discards events.
func NewNullStore() *NullStore {
	return &NullStore{}
}

// Store discards the event.
func (s *NullStore) Store(ctx context.Context, event *Event) error {
	return nil
}

// Query returns empty results.
func (s *NullStore) Query(ctx context.Context, query *Query) (*QueryResult, error) {
	return &QueryResult{Events: []*Event{}}, nil
}

// Close is a no-op.
func (s *NullStore) Close() error {
	return nil
}

// Helper functions for query matching.

func matchesQuery(event *Event, query *Query) bool {
	// Actor ID
	if query.ActorID != "" {
		if event.Actor == nil || event.Actor.ID != query.ActorID {
			return false
		}
	}

	// Actor Type
	if query.ActorType != "" {
		if event.Actor == nil || event.Actor.Type != query.ActorType {
			return false
		}
	}

	// Resource ID
	if query.ResourceID != "" {
		if event.Resource == nil || event.Resource.ID != query.ResourceID {
			return false
		}
	}

	// Resource Type
	if query.ResourceType != "" {
		if event.Resource == nil || event.Resource.Type != query.ResourceType {
			return false
		}
	}

	// Action (supports wildcards)
	if query.Action != "" {
		if !matchAction(event.Action, query.Action) {
			return false
		}
	}

	// Actions list
	if len(query.Actions) > 0 {
		found := false
		for _, action := range query.Actions {
			if matchAction(event.Action, action) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Outcome
	if query.Outcome != "" && event.Outcome != query.Outcome {
		return false
	}

	// Tags (all must match)
	if len(query.Tags) > 0 {
		for _, tag := range query.Tags {
			found := false
			for _, eventTag := range event.Tags {
				if eventTag == tag {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Time range
	if !query.From.IsZero() && event.Timestamp.Before(query.From) {
		return false
	}
	if !query.To.IsZero() && event.Timestamp.After(query.To) {
		return false
	}

	return true
}

func matchAction(action, pattern string) bool {
	if pattern == action {
		return true
	}

	// Support wildcards (e.g., "user.*", "*.create")
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			if parts[0] == "" {
				return strings.HasSuffix(action, parts[1])
			}
			if parts[1] == "" {
				return strings.HasPrefix(action, parts[0])
			}
			return strings.HasPrefix(action, parts[0]) && strings.HasSuffix(action, parts[1])
		}
	}

	return false
}

func sortEvents(events []*Event, orderBy string, desc bool) {
	if orderBy == "" {
		orderBy = "timestamp"
	}

	sort.Slice(events, func(i, j int) bool {
		var cmp int
		switch orderBy {
		case "timestamp":
			if events[i].Timestamp.Before(events[j].Timestamp) {
				cmp = -1
			} else if events[i].Timestamp.After(events[j].Timestamp) {
				cmp = 1
			}
		case "action":
			cmp = strings.Compare(events[i].Action, events[j].Action)
		case "actor":
			var ai, aj string
			if events[i].Actor != nil {
				ai = events[i].Actor.ID
			}
			if events[j].Actor != nil {
				aj = events[j].Actor.ID
			}
			cmp = strings.Compare(ai, aj)
		case "resource":
			var ri, rj string
			if events[i].Resource != nil {
				ri = events[i].Resource.ID
			}
			if events[j].Resource != nil {
				rj = events[j].Resource.ID
			}
			cmp = strings.Compare(ri, rj)
		default:
			cmp = 0
		}

		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}

// ChannelStore sends events to a channel.
// Useful for streaming events to other systems.
type ChannelStore struct {
	ch chan<- *Event
}

// NewChannelStore creates a store that sends events to a channel.
func NewChannelStore(ch chan<- *Event) *ChannelStore {
	return &ChannelStore{ch: ch}
}

// Store sends the event to the channel.
func (s *ChannelStore) Store(ctx context.Context, event *Event) error {
	select {
	case s.ch <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrChannelFull
	}
}

// Query is not supported.
func (s *ChannelStore) Query(ctx context.Context, query *Query) (*QueryResult, error) {
	return nil, ErrQueryNotSupported
}

// Close is a no-op (channel should be closed by the creator).
func (s *ChannelStore) Close() error {
	return nil
}

// BatchStore batches events before writing.
type BatchStore struct {
	mu         sync.Mutex
	store      Store
	batch      []*Event
	batchSize  int
	flushAfter time.Duration
	timer      *time.Timer
	done       chan struct{}
}

// NewBatchStore creates a store that batches events.
func NewBatchStore(store Store, batchSize int, flushAfter time.Duration) *BatchStore {
	if batchSize <= 0 {
		batchSize = 100
	}
	if flushAfter <= 0 {
		flushAfter = time.Second
	}

	s := &BatchStore{
		store:      store,
		batch:      make([]*Event, 0, batchSize),
		batchSize:  batchSize,
		flushAfter: flushAfter,
		done:       make(chan struct{}),
	}

	return s
}

// Store adds an event to the batch.
func (s *BatchStore) Store(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.batch = append(s.batch, event)

	// Start timer on first event
	if len(s.batch) == 1 && s.flushAfter > 0 {
		s.timer = time.AfterFunc(s.flushAfter, func() {
			s.Flush(context.Background())
		})
	}

	// Flush if batch is full
	if len(s.batch) >= s.batchSize {
		return s.flushLocked(ctx)
	}

	return nil
}

// Flush writes all pending events.
func (s *BatchStore) Flush(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flushLocked(ctx)
}

func (s *BatchStore) flushLocked(ctx context.Context) error {
	if len(s.batch) == 0 {
		return nil
	}

	// Stop timer
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}

	// Store all events
	var lastErr error
	for _, event := range s.batch {
		if err := s.store.Store(ctx, event); err != nil {
			lastErr = err
		}
	}

	// Clear batch
	s.batch = s.batch[:0]

	return lastErr
}

// Query delegates to the underlying store.
func (s *BatchStore) Query(ctx context.Context, query *Query) (*QueryResult, error) {
	return s.store.Query(ctx, query)
}

// Close flushes pending events and closes the store.
func (s *BatchStore) Close() error {
	s.Flush(context.Background())
	close(s.done)
	return s.store.Close()
}
