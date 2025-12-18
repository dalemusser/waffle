// Package storage provides a unified interface for file/blob storage across
// multiple backends including local filesystem, AWS S3, Google Cloud Storage,
// and Azure Blob Storage.
//
// Basic usage:
//
//	// Create a local storage backend
//	store, err := storage.NewLocal(storage.LocalConfig{
//	    BasePath: "/var/data/uploads",
//	})
//
//	// Upload a file
//	err = store.Put(ctx, "documents/report.pdf", reader, &storage.PutOptions{
//	    ContentType: "application/pdf",
//	})
//
//	// Download a file
//	reader, err := store.Get(ctx, "documents/report.pdf")
//
//	// Generate a presigned URL (for backends that support it)
//	url, err := store.PresignedURL(ctx, "documents/report.pdf", &storage.PresignOptions{
//	    Expires: 15 * time.Minute,
//	})
package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// Common errors returned by storage operations.
var (
	ErrNotFound            = errors.New("storage: object not found")
	ErrPermissionDenied    = errors.New("storage: permission denied")
	ErrAlreadyExists       = errors.New("storage: object already exists")
	ErrInvalidPath         = errors.New("storage: invalid path")
	ErrPresignNotSupported = errors.New("storage: presigned URLs not supported by this backend")
	ErrBucketNotFound      = errors.New("storage: bucket not found")
	ErrInvalidConfig       = errors.New("storage: invalid configuration")
)

// Store is the interface that all storage backends must implement.
type Store interface {
	// Put uploads an object to the store.
	// If opts is nil, defaults are used.
	Put(ctx context.Context, path string, r io.Reader, opts *PutOptions) error

	// PutBytes is a convenience method to upload bytes.
	PutBytes(ctx context.Context, path string, data []byte, opts *PutOptions) error

	// Get retrieves an object from the store.
	// The caller is responsible for closing the returned reader.
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// GetBytes is a convenience method to download an object as bytes.
	GetBytes(ctx context.Context, path string) ([]byte, error)

	// GetWithInfo retrieves an object along with its metadata.
	GetWithInfo(ctx context.Context, path string) (io.ReadCloser, *ObjectInfo, error)

	// Head returns metadata about an object without downloading it.
	Head(ctx context.Context, path string) (*ObjectInfo, error)

	// Delete removes an object from the store.
	Delete(ctx context.Context, path string) error

	// DeleteMany removes multiple objects from the store.
	// Returns the number of objects deleted and any error.
	DeleteMany(ctx context.Context, paths []string) (int, error)

	// Exists checks if an object exists.
	Exists(ctx context.Context, path string) (bool, error)

	// List returns objects matching the given prefix.
	List(ctx context.Context, prefix string, opts *ListOptions) (*ListResult, error)

	// Copy copies an object from src to dst within the same store.
	Copy(ctx context.Context, src, dst string) error

	// Move moves an object from src to dst within the same store.
	Move(ctx context.Context, src, dst string) error

	// PresignedURL generates a presigned URL for the object.
	// Returns ErrPresignNotSupported if the backend doesn't support presigned URLs.
	PresignedURL(ctx context.Context, path string, opts *PresignOptions) (string, error)

	// PresignedUploadURL generates a presigned URL for uploading an object.
	// Returns ErrPresignNotSupported if the backend doesn't support presigned URLs.
	PresignedUploadURL(ctx context.Context, path string, opts *PresignUploadOptions) (*PresignedUpload, error)

	// URL returns the public URL for an object (if publicly accessible).
	// This is not a presigned URL - it requires the object to be public.
	URL(path string) string

	// Backend returns the backend type identifier.
	Backend() string
}

// PutOptions configures object upload behavior.
type PutOptions struct {
	// ContentType is the MIME type of the object.
	// If empty, it will be detected from the file extension or content.
	ContentType string

	// ContentDisposition sets the Content-Disposition header.
	// Use "attachment; filename=..." to force download.
	ContentDisposition string

	// CacheControl sets the Cache-Control header.
	CacheControl string

	// Metadata is custom metadata to attach to the object.
	Metadata map[string]string

	// ACL sets the access control for the object.
	// Values depend on the backend (e.g., "public-read", "private").
	ACL string

	// ContentEncoding sets the Content-Encoding header.
	ContentEncoding string

	// IfNotExists only uploads if the object doesn't already exist.
	IfNotExists bool

	// ServerSideEncryption enables server-side encryption.
	// Values depend on the backend (e.g., "AES256", "aws:kms").
	ServerSideEncryption string

	// StorageClass sets the storage class for the object.
	// Values depend on the backend (e.g., "STANDARD", "GLACIER").
	StorageClass string
}

// ListOptions configures object listing behavior.
type ListOptions struct {
	// MaxKeys is the maximum number of objects to return.
	// Default: 1000
	MaxKeys int

	// Delimiter is used to group objects by prefix.
	// Typically "/" to list "directories".
	Delimiter string

	// ContinuationToken is used for pagination.
	ContinuationToken string

	// IncludeMetadata includes object metadata in results.
	// This may require additional API calls.
	IncludeMetadata bool
}

// ListResult contains the results of a List operation.
type ListResult struct {
	// Objects is the list of objects found.
	Objects []ObjectInfo

	// CommonPrefixes contains "directory" prefixes when using a delimiter.
	CommonPrefixes []string

	// IsTruncated is true if there are more results.
	IsTruncated bool

	// NextContinuationToken is used to fetch the next page.
	NextContinuationToken string
}

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	// Path is the full path/key of the object.
	Path string

	// Size is the size in bytes.
	Size int64

	// ContentType is the MIME type.
	ContentType string

	// LastModified is when the object was last modified.
	LastModified time.Time

	// ETag is the entity tag (usually MD5 hash).
	ETag string

	// Metadata is custom metadata attached to the object.
	Metadata map[string]string

	// StorageClass is the storage class of the object.
	StorageClass string
}

// PresignOptions configures presigned URL generation for downloads.
type PresignOptions struct {
	// Expires is how long the URL should be valid.
	// Default: 15 minutes
	Expires time.Duration

	// ContentType forces a specific Content-Type header.
	ContentType string

	// ContentDisposition forces a specific Content-Disposition header.
	ContentDisposition string

	// ResponseCacheControl sets the Cache-Control response header.
	ResponseCacheControl string
}

// PresignUploadOptions configures presigned URL generation for uploads.
type PresignUploadOptions struct {
	// Expires is how long the URL should be valid.
	// Default: 15 minutes
	Expires time.Duration

	// ContentType is the required Content-Type for the upload.
	ContentType string

	// MaxSize is the maximum allowed upload size in bytes.
	// Only supported by some backends.
	MaxSize int64

	// Metadata is custom metadata that will be attached to the uploaded object.
	Metadata map[string]string

	// ACL sets the access control for the uploaded object.
	ACL string
}

// PresignedUpload contains information for a presigned upload.
type PresignedUpload struct {
	// URL is the presigned URL for uploading.
	URL string

	// Method is the HTTP method to use (usually PUT or POST).
	Method string

	// Headers are required headers for the upload request.
	Headers map[string]string

	// FormFields are form fields for POST uploads (used by S3).
	FormFields map[string]string

	// Expires is when the presigned URL expires.
	Expires time.Time
}

// NormalizePath cleans and normalizes an object path.
func NormalizePath(path string) string {
	// Remove leading/trailing slashes
	path = strings.Trim(path, "/")
	// Clean the path
	path = filepath.Clean(path)
	// Convert backslashes to forward slashes (Windows compatibility)
	path = strings.ReplaceAll(path, "\\", "/")
	// Remove any double slashes
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	return path
}

// ValidatePath checks if a path is valid for storage operations.
func ValidatePath(path string) error {
	if path == "" {
		return ErrInvalidPath
	}
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return ErrInvalidPath
	}
	// Check for null bytes
	if strings.ContainsRune(path, 0) {
		return ErrInvalidPath
	}
	return nil
}

// DetectContentType detects the MIME type from filename or content.
func DetectContentType(path string, content []byte) string {
	// Try to detect from extension first
	ext := strings.ToLower(filepath.Ext(path))
	if ct := mimeTypes[ext]; ct != "" {
		return ct
	}

	// Fall back to content detection if we have content
	if len(content) > 0 {
		return http.DetectContentType(content)
	}

	return "application/octet-stream"
}

// mimeTypes maps file extensions to MIME types.
var mimeTypes = map[string]string{
	// Text
	".txt":  "text/plain",
	".html": "text/html",
	".htm":  "text/html",
	".css":  "text/css",
	".csv":  "text/csv",
	".xml":  "application/xml",
	".json": "application/json",
	".js":   "application/javascript",
	".mjs":  "application/javascript",
	".ts":   "application/typescript",
	".md":   "text/markdown",
	".yaml": "application/x-yaml",
	".yml":  "application/x-yaml",
	".toml": "application/toml",

	// Images
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".bmp":  "image/bmp",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".ico":  "image/x-icon",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
	".avif": "image/avif",
	".heic": "image/heic",
	".heif": "image/heif",

	// Video
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".avi":  "video/x-msvideo",
	".mov":  "video/quicktime",
	".wmv":  "video/x-ms-wmv",
	".flv":  "video/x-flv",
	".mkv":  "video/x-matroska",
	".m4v":  "video/x-m4v",

	// Audio
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".flac": "audio/flac",
	".aac":  "audio/aac",
	".m4a":  "audio/mp4",
	".wma":  "audio/x-ms-wma",

	// Documents
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".odt":  "application/vnd.oasis.opendocument.text",
	".ods":  "application/vnd.oasis.opendocument.spreadsheet",
	".odp":  "application/vnd.oasis.opendocument.presentation",
	".rtf":  "application/rtf",
	".epub": "application/epub+zip",

	// Archives
	".zip": "application/zip",
	".tar": "application/x-tar",
	".gz":  "application/gzip",
	".bz2": "application/x-bzip2",
	".xz":  "application/x-xz",
	".7z":  "application/x-7z-compressed",
	".rar": "application/vnd.rar",

	// Fonts
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".eot":   "application/vnd.ms-fontobject",

	// Other
	".wasm": "application/wasm",
	".exe":  "application/x-msdownload",
	".dll":  "application/x-msdownload",
	".dmg":  "application/x-apple-diskimage",
	".iso":  "application/x-iso9660-image",
	".apk":  "application/vnd.android.package-archive",
	".ipa":  "application/x-itunes-ipa",
}

// RegisterMimeType registers a custom MIME type for a file extension.
func RegisterMimeType(ext, mimeType string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	mimeTypes[strings.ToLower(ext)] = mimeType
}
