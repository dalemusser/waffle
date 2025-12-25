package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LocalConfig configures the local filesystem storage backend.
type LocalConfig struct {
	// BasePath is the root directory for storage.
	// All paths are relative to this directory.
	BasePath string

	// BaseURL is the base URL for generating public URLs.
	// If empty, URL() returns an empty string.
	BaseURL string

	// CreateDirs automatically creates directories as needed.
	// Default: true
	CreateDirs *bool

	// Permissions sets the file permissions for new files.
	// Default: 0644
	Permissions os.FileMode

	// DirPermissions sets the directory permissions for new directories.
	// Default: 0755
	DirPermissions os.FileMode
}

// Local is a storage backend that uses the local filesystem.
type Local struct {
	basePath       string
	baseURL        string
	createDirs     bool
	permissions    os.FileMode
	dirPermissions os.FileMode
}

// NewLocal creates a new local filesystem storage backend.
func NewLocal(cfg LocalConfig) (*Local, error) {
	if cfg.BasePath == "" {
		return nil, fmt.Errorf("%w: BasePath is required", ErrInvalidConfig)
	}

	// Resolve to absolute path
	basePath, err := filepath.Abs(cfg.BasePath)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to resolve path: %w", err)
	}

	createDirs := true
	if cfg.CreateDirs != nil {
		createDirs = *cfg.CreateDirs
	}

	permissions := cfg.Permissions
	if permissions == 0 {
		permissions = 0644
	}

	dirPermissions := cfg.DirPermissions
	if dirPermissions == 0 {
		dirPermissions = 0755
	}

	// Create base directory if it doesn't exist
	if createDirs {
		if err := os.MkdirAll(basePath, dirPermissions); err != nil {
			return nil, fmt.Errorf("storage: failed to create base directory: %w", err)
		}
	}

	// Verify base path exists and is a directory
	info, err := os.Stat(basePath)
	if err != nil {
		return nil, fmt.Errorf("storage: base path does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: BasePath must be a directory", ErrInvalidConfig)
	}

	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	return &Local{
		basePath:       basePath,
		baseURL:        baseURL,
		createDirs:     createDirs,
		permissions:    permissions,
		dirPermissions: dirPermissions,
	}, nil
}

// Backend returns the backend type identifier.
func (l *Local) Backend() string {
	return "local"
}

// fullPath returns the full filesystem path for an object.
func (l *Local) fullPath(path string) (string, error) {
	path = NormalizePath(path)
	if err := ValidatePath(path); err != nil {
		return "", err
	}

	full := filepath.Join(l.basePath, path)

	// Security: ensure the path is still within basePath after joining
	if !strings.HasPrefix(full, l.basePath) {
		return "", ErrInvalidPath
	}

	return full, nil
}

// Put uploads an object to local storage.
func (l *Local) Put(ctx context.Context, path string, r io.Reader, opts *PutOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	fullPath, err := l.fullPath(path)
	if err != nil {
		return err
	}

	// Check if object exists and IfNotExists is set
	if opts != nil && opts.IfNotExists {
		if _, err := os.Stat(fullPath); err == nil {
			return ErrAlreadyExists
		}
	}

	// Create parent directories if needed
	if l.createDirs {
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, l.dirPermissions); err != nil {
			return fmt.Errorf("storage: failed to create directory: %w", err)
		}
	}

	// Create temp file in the same directory for atomic write
	dir := filepath.Dir(fullPath)
	tmpFile, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return fmt.Errorf("storage: failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on any error
	success := false
	defer func() {
		if !success {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Copy data to temp file
	if _, err := io.Copy(tmpFile, r); err != nil {
		return fmt.Errorf("storage: failed to write file: %w", err)
	}

	// Sync to disk
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("storage: failed to sync file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("storage: failed to close file: %w", err)
	}

	// Set permissions
	if err := os.Chmod(tmpPath, l.permissions); err != nil {
		return fmt.Errorf("storage: failed to set permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, fullPath); err != nil {
		return fmt.Errorf("storage: failed to rename file: %w", err)
	}

	success = true
	return nil
}

// PutBytes uploads bytes to local storage.
func (l *Local) PutBytes(ctx context.Context, path string, data []byte, opts *PutOptions) error {
	return l.Put(ctx, path, bytes.NewReader(data), opts)
}

// Get retrieves an object from local storage.
func (l *Local) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	fullPath, err := l.fullPath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		if os.IsPermission(err) {
			return nil, ErrPermissionDenied
		}
		return nil, fmt.Errorf("storage: failed to open file: %w", err)
	}

	return file, nil
}

// GetBytes retrieves an object as bytes.
func (l *Local) GetBytes(ctx context.Context, path string) ([]byte, error) {
	rc, err := l.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// GetWithInfo retrieves an object along with its metadata.
func (l *Local) GetWithInfo(ctx context.Context, path string) (io.ReadCloser, *ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	fullPath, err := l.fullPath(path)
	if err != nil {
		return nil, nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrNotFound
		}
		if os.IsPermission(err) {
			return nil, nil, ErrPermissionDenied
		}
		return nil, nil, fmt.Errorf("storage: failed to open file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("storage: failed to stat file: %w", err)
	}

	objInfo := l.fileInfoToObjectInfo(path, info)
	return file, objInfo, nil
}

// Head returns metadata about an object without downloading it.
func (l *Local) Head(ctx context.Context, path string) (*ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	fullPath, err := l.fullPath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		if os.IsPermission(err) {
			return nil, ErrPermissionDenied
		}
		return nil, fmt.Errorf("storage: failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, ErrNotFound
	}

	return l.fileInfoToObjectInfo(path, info), nil
}

// fileInfoToObjectInfo converts os.FileInfo to ObjectInfo.
func (l *Local) fileInfoToObjectInfo(path string, info os.FileInfo) *ObjectInfo {
	return &ObjectInfo{
		Path:         NormalizePath(path),
		Size:         info.Size(),
		ContentType:  DetectContentType(path, nil),
		LastModified: info.ModTime(),
		ETag:         fmt.Sprintf(`"%x"`, info.ModTime().UnixNano()),
	}
}

// Delete removes an object from local storage.
func (l *Local) Delete(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	fullPath, err := l.fullPath(path)
	if err != nil {
		return err
	}

	err = os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		if os.IsPermission(err) {
			return ErrPermissionDenied
		}
		return fmt.Errorf("storage: failed to delete file: %w", err)
	}

	return nil
}

// DeleteMany removes multiple objects from local storage.
func (l *Local) DeleteMany(ctx context.Context, paths []string) (int, error) {
	deleted := 0
	var lastErr error

	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}

		if err := l.Delete(ctx, path); err != nil {
			if err != ErrNotFound {
				lastErr = err
			}
		} else {
			deleted++
		}
	}

	return deleted, lastErr
}

// Exists checks if an object exists in local storage.
func (l *Local) Exists(ctx context.Context, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	fullPath, err := l.fullPath(path)
	if err != nil {
		return false, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("storage: failed to check existence: %w", err)
	}

	return !info.IsDir(), nil
}

// List returns objects matching the given prefix.
func (l *Local) List(ctx context.Context, prefix string, opts *ListOptions) (*ListResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix = NormalizePath(prefix)

	searchPath := l.basePath
	if prefix != "" && prefix != "." {
		searchPath = filepath.Join(l.basePath, prefix)
	}

	if opts == nil {
		opts = &ListOptions{}
	}

	maxKeys := opts.MaxKeys
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	delimiter := opts.Delimiter

	result := &ListResult{
		Objects:        make([]ObjectInfo, 0),
		CommonPrefixes: make([]string, 0),
	}

	// Track seen prefixes for deduplication
	seenPrefixes := make(map[string]bool)

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// Skip base path
		if path == l.basePath {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(l.basePath, path)
		if err != nil {
			return err
		}
		relPath = strings.ReplaceAll(relPath, string(filepath.Separator), "/")

		// Skip if doesn't match prefix
		if prefix != "" && prefix != "." && !strings.HasPrefix(relPath, prefix) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Handle delimiter (directory grouping)
		if delimiter != "" {
			// Get the part after the prefix
			suffix := relPath
			if prefix != "" && prefix != "." {
				suffix = strings.TrimPrefix(relPath, prefix)
				suffix = strings.TrimPrefix(suffix, "/")
			}

			// Check if there's a delimiter in the suffix
			if idx := strings.Index(suffix, delimiter); idx >= 0 {
				// This is a "directory" - add to common prefixes
				commonPrefix := relPath[:len(relPath)-len(suffix)+idx+1]
				if !seenPrefixes[commonPrefix] {
					seenPrefixes[commonPrefix] = true
					result.CommonPrefixes = append(result.CommonPrefixes, commonPrefix)
				}
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Skip directories (we only list files)
		if info.IsDir() {
			return nil
		}

		// Check max keys
		if len(result.Objects) >= maxKeys {
			result.IsTruncated = true
			result.NextContinuationToken = relPath
			return filepath.SkipAll
		}

		result.Objects = append(result.Objects, *l.fileInfoToObjectInfo(relPath, info))
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, fmt.Errorf("storage: failed to list files: %w", err)
	}

	// Sort results for consistent ordering
	sort.Slice(result.Objects, func(i, j int) bool {
		return result.Objects[i].Path < result.Objects[j].Path
	})
	sort.Strings(result.CommonPrefixes)

	return result, nil
}

// Copy copies an object from src to dst.
func (l *Local) Copy(ctx context.Context, src, dst string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	srcPath, err := l.fullPath(src)
	if err != nil {
		return err
	}

	dstPath, err := l.fullPath(dst)
	if err != nil {
		return err
	}

	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("storage: failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create parent directories if needed
	if l.createDirs {
		dir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dir, l.dirPermissions); err != nil {
			return fmt.Errorf("storage: failed to create directory: %w", err)
		}
	}

	// Create destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("storage: failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("storage: failed to copy file: %w", err)
	}

	// Set permissions
	if err := os.Chmod(dstPath, l.permissions); err != nil {
		return fmt.Errorf("storage: failed to set permissions: %w", err)
	}

	return nil
}

// Move moves an object from src to dst.
func (l *Local) Move(ctx context.Context, src, dst string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	srcPath, err := l.fullPath(src)
	if err != nil {
		return err
	}

	dstPath, err := l.fullPath(dst)
	if err != nil {
		return err
	}

	// Create parent directories if needed
	if l.createDirs {
		dir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dir, l.dirPermissions); err != nil {
			return fmt.Errorf("storage: failed to create directory: %w", err)
		}
	}

	// Try rename first (atomic if on same filesystem)
	if err := os.Rename(srcPath, dstPath); err == nil {
		return nil
	}

	// Fall back to copy + delete for cross-filesystem moves
	if err := l.Copy(ctx, src, dst); err != nil {
		return err
	}

	return os.Remove(srcPath)
}

// PresignedURL is not supported for local storage.
func (l *Local) PresignedURL(ctx context.Context, path string, opts *PresignOptions) (string, error) {
	return "", ErrPresignNotSupported
}

// PresignedUploadURL is not supported for local storage.
func (l *Local) PresignedUploadURL(ctx context.Context, path string, opts *PresignUploadOptions) (*PresignedUpload, error) {
	return nil, ErrPresignNotSupported
}

// URL returns the public URL for an object.
func (l *Local) URL(path string) string {
	if l.baseURL == "" {
		return ""
	}
	path = NormalizePath(path)
	return l.baseURL + "/" + path
}

// GetFullPath returns the full filesystem path for an object.
// This is useful for serving files directly via http.ServeFile.
func (l *Local) GetFullPath(path string) (string, error) {
	return l.fullPath(path)
}

// ComputeETag computes an ETag (MD5 hash) for file content.
func ComputeETag(r io.Reader) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%s"`, hex.EncodeToString(h.Sum(nil))), nil
}
