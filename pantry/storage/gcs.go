package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GCSConfig configures the Google Cloud Storage backend.
type GCSConfig struct {
	// Bucket is the GCS bucket name (required).
	Bucket string

	// CredentialsFile is the path to the service account JSON file.
	// If empty, uses Application Default Credentials.
	CredentialsFile string

	// CredentialsJSON is the service account JSON content.
	// If empty, uses CredentialsFile or Application Default Credentials.
	CredentialsJSON []byte

	// ProjectID is the Google Cloud project ID.
	// Required for some operations if not using service account credentials.
	ProjectID string

	// Prefix is a path prefix for all objects.
	Prefix string

	// BaseURL is the base URL for generating public URLs.
	// If empty, uses the default GCS URL format.
	BaseURL string

	// DefaultACL is the default ACL for new objects.
	// Common values: "private", "publicRead", "bucketOwnerFullControl"
	DefaultACL string

	// StorageClass sets the default storage class.
	// Common values: "STANDARD", "NEARLINE", "COLDLINE", "ARCHIVE"
	StorageClass string
}

// GCS is a storage backend that uses Google Cloud Storage.
type GCS struct {
	client       *storage.Client
	bucket       *storage.BucketHandle
	bucketName   string
	prefix       string
	baseURL      string
	defaultACL   string
	storageClass string
}

// NewGCS creates a new Google Cloud Storage backend.
func NewGCS(ctx context.Context, cfg GCSConfig) (*GCS, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("%w: Bucket is required", ErrInvalidConfig)
	}

	var opts []option.ClientOption

	if len(cfg.CredentialsJSON) > 0 {
		opts = append(opts, option.WithCredentialsJSON(cfg.CredentialsJSON))
	} else if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to create GCS client: %w", err)
	}

	bucket := client.Bucket(cfg.Bucket)

	// Determine base URL
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://storage.googleapis.com/%s", cfg.Bucket)
	}

	return &GCS{
		client:       client,
		bucket:       bucket,
		bucketName:   cfg.Bucket,
		prefix:       NormalizePath(cfg.Prefix),
		baseURL:      baseURL,
		defaultACL:   cfg.DefaultACL,
		storageClass: cfg.StorageClass,
	}, nil
}

// Close closes the GCS client.
func (g *GCS) Close() error {
	return g.client.Close()
}

// Backend returns the backend type identifier.
func (g *GCS) Backend() string {
	return "gcs"
}

// fullKey returns the full GCS object name.
func (g *GCS) fullKey(path string) (string, error) {
	path = NormalizePath(path)
	if err := ValidatePath(path); err != nil {
		return "", err
	}

	if g.prefix != "" {
		return g.prefix + "/" + path, nil
	}
	return path, nil
}

// Put uploads an object to GCS.
func (g *GCS) Put(ctx context.Context, path string, r io.Reader, opts *PutOptions) error {
	key, err := g.fullKey(path)
	if err != nil {
		return err
	}

	if opts == nil {
		opts = &PutOptions{}
	}

	// Check if object exists for IfNotExists
	if opts.IfNotExists {
		exists, err := g.Exists(ctx, path)
		if err != nil {
			return err
		}
		if exists {
			return ErrAlreadyExists
		}
	}

	obj := g.bucket.Object(key)
	writer := obj.NewWriter(ctx)

	// Set content type
	if opts.ContentType != "" {
		writer.ContentType = opts.ContentType
	} else {
		writer.ContentType = DetectContentType(path, nil)
	}

	if opts.ContentDisposition != "" {
		writer.ContentDisposition = opts.ContentDisposition
	}

	if opts.CacheControl != "" {
		writer.CacheControl = opts.CacheControl
	}

	if opts.ContentEncoding != "" {
		writer.ContentEncoding = opts.ContentEncoding
	}

	// Set storage class
	sc := opts.StorageClass
	if sc == "" {
		sc = g.storageClass
	}
	if sc != "" {
		writer.StorageClass = sc
	}

	// Set metadata
	if len(opts.Metadata) > 0 {
		writer.Metadata = opts.Metadata
	}

	// Copy data
	if _, err := io.Copy(writer, r); err != nil {
		writer.Close()
		return fmt.Errorf("storage: failed to write object: %w", err)
	}

	if err := writer.Close(); err != nil {
		return g.translateError(err)
	}

	// Set ACL if specified
	acl := opts.ACL
	if acl == "" {
		acl = g.defaultACL
	}
	if acl != "" {
		if err := g.setACL(ctx, obj, acl); err != nil {
			return err
		}
	}

	return nil
}

// setACL sets the ACL for an object.
func (g *GCS) setACL(ctx context.Context, obj *storage.ObjectHandle, acl string) error {
	var predefinedACL storage.ACLRule

	switch acl {
	case "private":
		// Default, no action needed
		return nil
	case "publicRead", "public-read":
		predefinedACL = storage.ACLRule{
			Entity: storage.AllUsers,
			Role:   storage.RoleReader,
		}
	case "bucketOwnerFullControl", "bucket-owner-full-control":
		// This is typically the default
		return nil
	default:
		return fmt.Errorf("storage: unknown ACL: %s", acl)
	}

	return obj.ACL().Set(ctx, predefinedACL.Entity, predefinedACL.Role)
}

// PutBytes uploads bytes to GCS.
func (g *GCS) PutBytes(ctx context.Context, path string, data []byte, opts *PutOptions) error {
	return g.Put(ctx, path, bytes.NewReader(data), opts)
}

// Get retrieves an object from GCS.
func (g *GCS) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	key, err := g.fullKey(path)
	if err != nil {
		return nil, err
	}

	reader, err := g.bucket.Object(key).NewReader(ctx)
	if err != nil {
		return nil, g.translateError(err)
	}

	return reader, nil
}

// GetBytes retrieves an object as bytes.
func (g *GCS) GetBytes(ctx context.Context, path string) ([]byte, error) {
	rc, err := g.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// GetWithInfo retrieves an object along with its metadata.
func (g *GCS) GetWithInfo(ctx context.Context, path string) (io.ReadCloser, *ObjectInfo, error) {
	key, err := g.fullKey(path)
	if err != nil {
		return nil, nil, err
	}

	obj := g.bucket.Object(key)

	// Get attributes first
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, nil, g.translateError(err)
	}

	// Get reader
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, nil, g.translateError(err)
	}

	info := g.attrsToObjectInfo(path, attrs)
	return reader, info, nil
}

// Head returns metadata about an object without downloading it.
func (g *GCS) Head(ctx context.Context, path string) (*ObjectInfo, error) {
	key, err := g.fullKey(path)
	if err != nil {
		return nil, err
	}

	attrs, err := g.bucket.Object(key).Attrs(ctx)
	if err != nil {
		return nil, g.translateError(err)
	}

	return g.attrsToObjectInfo(path, attrs), nil
}

// attrsToObjectInfo converts GCS ObjectAttrs to ObjectInfo.
func (g *GCS) attrsToObjectInfo(path string, attrs *storage.ObjectAttrs) *ObjectInfo {
	return &ObjectInfo{
		Path:         path,
		Size:         attrs.Size,
		ContentType:  attrs.ContentType,
		LastModified: attrs.Updated,
		ETag:         attrs.Etag,
		Metadata:     attrs.Metadata,
		StorageClass: attrs.StorageClass,
	}
}

// Delete removes an object from GCS.
func (g *GCS) Delete(ctx context.Context, path string) error {
	key, err := g.fullKey(path)
	if err != nil {
		return err
	}

	err = g.bucket.Object(key).Delete(ctx)
	if err != nil {
		return g.translateError(err)
	}

	return nil
}

// DeleteMany removes multiple objects from GCS.
func (g *GCS) DeleteMany(ctx context.Context, paths []string) (int, error) {
	deleted := 0
	var lastErr error

	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}

		if err := g.Delete(ctx, path); err != nil {
			if !errors.Is(err, ErrNotFound) {
				lastErr = err
			}
		} else {
			deleted++
		}
	}

	return deleted, lastErr
}

// Exists checks if an object exists in GCS.
func (g *GCS) Exists(ctx context.Context, path string) (bool, error) {
	_, err := g.Head(ctx, path)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// List returns objects matching the given prefix.
func (g *GCS) List(ctx context.Context, prefix string, opts *ListOptions) (*ListResult, error) {
	if opts == nil {
		opts = &ListOptions{}
	}

	maxKeys := opts.MaxKeys
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	// Build full prefix
	fullPrefix := g.prefix
	prefix = NormalizePath(prefix)
	if prefix != "" && prefix != "." {
		if fullPrefix != "" {
			fullPrefix = fullPrefix + "/" + prefix
		} else {
			fullPrefix = prefix
		}
	}

	query := &storage.Query{
		Prefix: fullPrefix,
	}

	if opts.Delimiter != "" {
		query.Delimiter = opts.Delimiter
	}

	result := &ListResult{
		Objects:        make([]ObjectInfo, 0),
		CommonPrefixes: make([]string, 0),
	}

	it := g.bucket.Objects(ctx, query)
	count := 0

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, g.translateError(err)
		}

		// Handle common prefixes (directories)
		if attrs.Prefix != "" {
			cp := attrs.Prefix
			if g.prefix != "" && len(cp) > len(g.prefix) {
				cp = cp[len(g.prefix)+1:]
			}
			result.CommonPrefixes = append(result.CommonPrefixes, cp)
			continue
		}

		if count >= maxKeys {
			result.IsTruncated = true
			result.NextContinuationToken = attrs.Name
			break
		}

		// Remove prefix from key
		path := attrs.Name
		if g.prefix != "" && len(attrs.Name) > len(g.prefix) {
			path = attrs.Name[len(g.prefix)+1:]
		}

		result.Objects = append(result.Objects, ObjectInfo{
			Path:         path,
			Size:         attrs.Size,
			ContentType:  attrs.ContentType,
			LastModified: attrs.Updated,
			ETag:         attrs.Etag,
			Metadata:     attrs.Metadata,
			StorageClass: attrs.StorageClass,
		})
		count++
	}

	return result, nil
}

// Copy copies an object from src to dst within GCS.
func (g *GCS) Copy(ctx context.Context, src, dst string) error {
	srcKey, err := g.fullKey(src)
	if err != nil {
		return err
	}

	dstKey, err := g.fullKey(dst)
	if err != nil {
		return err
	}

	srcObj := g.bucket.Object(srcKey)
	dstObj := g.bucket.Object(dstKey)

	_, err = dstObj.CopierFrom(srcObj).Run(ctx)
	if err != nil {
		return g.translateError(err)
	}

	return nil
}

// Move moves an object from src to dst within GCS.
func (g *GCS) Move(ctx context.Context, src, dst string) error {
	if err := g.Copy(ctx, src, dst); err != nil {
		return err
	}
	return g.Delete(ctx, src)
}

// PresignedURL generates a presigned URL for downloading an object.
func (g *GCS) PresignedURL(ctx context.Context, path string, opts *PresignOptions) (string, error) {
	key, err := g.fullKey(path)
	if err != nil {
		return "", err
	}

	if opts == nil {
		opts = &PresignOptions{}
	}

	expires := opts.Expires
	if expires == 0 {
		expires = 15 * time.Minute
	}

	signOpts := &storage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(expires),
	}

	if opts.ContentType != "" {
		signOpts.ContentType = opts.ContentType
	}

	url, err := g.bucket.SignedURL(key, signOpts)
	if err != nil {
		return "", fmt.Errorf("storage: failed to sign URL: %w", err)
	}

	return url, nil
}

// PresignedUploadURL generates a presigned URL for uploading an object.
func (g *GCS) PresignedUploadURL(ctx context.Context, path string, opts *PresignUploadOptions) (*PresignedUpload, error) {
	key, err := g.fullKey(path)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &PresignUploadOptions{}
	}

	expires := opts.Expires
	if expires == 0 {
		expires = 15 * time.Minute
	}

	signOpts := &storage.SignedURLOptions{
		Method:  "PUT",
		Expires: time.Now().Add(expires),
	}

	if opts.ContentType != "" {
		signOpts.ContentType = opts.ContentType
	}

	url, err := g.bucket.SignedURL(key, signOpts)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to sign upload URL: %w", err)
	}

	headers := make(map[string]string)
	if opts.ContentType != "" {
		headers["Content-Type"] = opts.ContentType
	}

	return &PresignedUpload{
		URL:     url,
		Method:  "PUT",
		Headers: headers,
		Expires: time.Now().Add(expires),
	}, nil
}

// URL returns the public URL for an object.
func (g *GCS) URL(path string) string {
	if g.baseURL == "" {
		return ""
	}

	path = NormalizePath(path)
	if g.prefix != "" {
		return g.baseURL + "/" + g.prefix + "/" + path
	}
	return g.baseURL + "/" + path
}

// translateError converts GCS errors to storage errors.
func (g *GCS) translateError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, storage.ErrObjectNotExist) {
		return ErrNotFound
	}

	if errors.Is(err, storage.ErrBucketNotExist) {
		return ErrBucketNotFound
	}

	return fmt.Errorf("storage: GCS error: %w", err)
}
