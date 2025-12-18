package storage

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryConfig configures the in-memory storage backend.
type MemoryConfig struct {
	// BaseURL is the base URL for generating URLs.
	BaseURL string
}

// Memory is an in-memory storage backend, primarily for testing.
type Memory struct {
	mu      sync.RWMutex
	objects map[string]*memoryObject
	baseURL string
}

type memoryObject struct {
	data        []byte
	contentType string
	metadata    map[string]string
	modTime     time.Time
}

// NewMemory creates a new in-memory storage backend.
func NewMemory(cfg MemoryConfig) *Memory {
	return &Memory{
		objects: make(map[string]*memoryObject),
		baseURL: cfg.BaseURL,
	}
}

// Backend returns the backend type identifier.
func (m *Memory) Backend() string {
	return "memory"
}

// Put uploads an object to memory.
func (m *Memory) Put(ctx context.Context, path string, r io.Reader, opts *PutOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path = NormalizePath(path)
	if err := ValidatePath(path); err != nil {
		return err
	}

	if opts == nil {
		opts = &PutOptions{}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if object exists for IfNotExists
	if opts.IfNotExists {
		if _, exists := m.objects[path]; exists {
			return ErrAlreadyExists
		}
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	contentType := opts.ContentType
	if contentType == "" {
		contentType = DetectContentType(path, data)
	}

	var metadata map[string]string
	if len(opts.Metadata) > 0 {
		metadata = make(map[string]string, len(opts.Metadata))
		for k, v := range opts.Metadata {
			metadata[k] = v
		}
	}

	m.objects[path] = &memoryObject{
		data:        data,
		contentType: contentType,
		metadata:    metadata,
		modTime:     time.Now(),
	}

	return nil
}

// PutBytes uploads bytes to memory.
func (m *Memory) PutBytes(ctx context.Context, path string, data []byte, opts *PutOptions) error {
	return m.Put(ctx, path, bytes.NewReader(data), opts)
}

// Get retrieves an object from memory.
func (m *Memory) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	path = NormalizePath(path)

	m.mu.RLock()
	obj, exists := m.objects[path]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrNotFound
	}

	return io.NopCloser(bytes.NewReader(obj.data)), nil
}

// GetBytes retrieves an object as bytes.
func (m *Memory) GetBytes(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	path = NormalizePath(path)

	m.mu.RLock()
	obj, exists := m.objects[path]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrNotFound
	}

	// Return a copy to prevent modification
	data := make([]byte, len(obj.data))
	copy(data, obj.data)
	return data, nil
}

// GetWithInfo retrieves an object along with its metadata.
func (m *Memory) GetWithInfo(ctx context.Context, path string) (io.ReadCloser, *ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	path = NormalizePath(path)

	m.mu.RLock()
	obj, exists := m.objects[path]
	m.mu.RUnlock()

	if !exists {
		return nil, nil, ErrNotFound
	}

	info := &ObjectInfo{
		Path:         path,
		Size:         int64(len(obj.data)),
		ContentType:  obj.contentType,
		LastModified: obj.modTime,
		Metadata:     obj.metadata,
	}

	return io.NopCloser(bytes.NewReader(obj.data)), info, nil
}

// Head returns metadata about an object without downloading it.
func (m *Memory) Head(ctx context.Context, path string) (*ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	path = NormalizePath(path)

	m.mu.RLock()
	obj, exists := m.objects[path]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrNotFound
	}

	return &ObjectInfo{
		Path:         path,
		Size:         int64(len(obj.data)),
		ContentType:  obj.contentType,
		LastModified: obj.modTime,
		Metadata:     obj.metadata,
	}, nil
}

// Delete removes an object from memory.
func (m *Memory) Delete(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path = NormalizePath(path)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.objects[path]; !exists {
		return ErrNotFound
	}

	delete(m.objects, path)
	return nil
}

// DeleteMany removes multiple objects from memory.
func (m *Memory) DeleteMany(ctx context.Context, paths []string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	deleted := 0
	for _, path := range paths {
		path = NormalizePath(path)
		if _, exists := m.objects[path]; exists {
			delete(m.objects, path)
			deleted++
		}
	}

	return deleted, nil
}

// Exists checks if an object exists in memory.
func (m *Memory) Exists(ctx context.Context, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	path = NormalizePath(path)

	m.mu.RLock()
	_, exists := m.objects[path]
	m.mu.RUnlock()

	return exists, nil
}

// List returns objects matching the given prefix.
func (m *Memory) List(ctx context.Context, prefix string, opts *ListOptions) (*ListResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &ListOptions{}
	}

	prefix = NormalizePath(prefix)
	if prefix == "." {
		prefix = ""
	}

	maxKeys := opts.MaxKeys
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	result := &ListResult{
		Objects:        make([]ObjectInfo, 0),
		CommonPrefixes: make([]string, 0),
	}

	seenPrefixes := make(map[string]bool)

	// Collect and sort keys for consistent ordering
	keys := make([]string, 0, len(m.objects))
	for k := range m.objects {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, path := range keys {
		obj := m.objects[path]

		// Handle delimiter
		if opts.Delimiter != "" {
			suffix := path
			if prefix != "" {
				suffix = strings.TrimPrefix(path, prefix)
				suffix = strings.TrimPrefix(suffix, "/")
			}

			if idx := strings.Index(suffix, opts.Delimiter); idx >= 0 {
				commonPrefix := path[:len(path)-len(suffix)+idx+1]
				if !seenPrefixes[commonPrefix] {
					seenPrefixes[commonPrefix] = true
					result.CommonPrefixes = append(result.CommonPrefixes, commonPrefix)
				}
				continue
			}
		}

		if len(result.Objects) >= maxKeys {
			result.IsTruncated = true
			result.NextContinuationToken = path
			break
		}

		result.Objects = append(result.Objects, ObjectInfo{
			Path:         path,
			Size:         int64(len(obj.data)),
			ContentType:  obj.contentType,
			LastModified: obj.modTime,
			Metadata:     obj.metadata,
		})
	}

	return result, nil
}

// Copy copies an object from src to dst.
func (m *Memory) Copy(ctx context.Context, src, dst string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	src = NormalizePath(src)
	dst = NormalizePath(dst)

	m.mu.Lock()
	defer m.mu.Unlock()

	srcObj, exists := m.objects[src]
	if !exists {
		return ErrNotFound
	}

	// Deep copy
	data := make([]byte, len(srcObj.data))
	copy(data, srcObj.data)

	var metadata map[string]string
	if srcObj.metadata != nil {
		metadata = make(map[string]string, len(srcObj.metadata))
		for k, v := range srcObj.metadata {
			metadata[k] = v
		}
	}

	m.objects[dst] = &memoryObject{
		data:        data,
		contentType: srcObj.contentType,
		metadata:    metadata,
		modTime:     time.Now(),
	}

	return nil
}

// Move moves an object from src to dst.
func (m *Memory) Move(ctx context.Context, src, dst string) error {
	if err := m.Copy(ctx, src, dst); err != nil {
		return err
	}

	src = NormalizePath(src)

	m.mu.Lock()
	delete(m.objects, src)
	m.mu.Unlock()

	return nil
}

// PresignedURL is not supported for memory storage.
func (m *Memory) PresignedURL(ctx context.Context, path string, opts *PresignOptions) (string, error) {
	return "", ErrPresignNotSupported
}

// PresignedUploadURL is not supported for memory storage.
func (m *Memory) PresignedUploadURL(ctx context.Context, path string, opts *PresignUploadOptions) (*PresignedUpload, error) {
	return nil, ErrPresignNotSupported
}

// URL returns the URL for an object.
func (m *Memory) URL(path string) string {
	if m.baseURL == "" {
		return ""
	}
	path = NormalizePath(path)
	return m.baseURL + "/" + path
}

// Clear removes all objects from memory.
func (m *Memory) Clear() {
	m.mu.Lock()
	m.objects = make(map[string]*memoryObject)
	m.mu.Unlock()
}

// Count returns the number of objects in memory.
func (m *Memory) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.objects)
}
