# Storage Package

Unified file/blob storage interface supporting local filesystem, AWS S3, Google Cloud Storage, and Azure Blob Storage.

## Installation

```go
import "github.com/dalemusser/waffle/pantry/storage"
```

## Quick Start

```go
// Local filesystem
store, err := storage.NewLocal(storage.LocalConfig{
    BasePath: "/var/data/uploads",
})

// AWS S3
store, err := storage.NewS3(ctx, storage.S3Config{
    Bucket: "my-bucket",
    Region: "us-east-1",
})

// Google Cloud Storage
store, err := storage.NewGCS(ctx, storage.GCSConfig{
    Bucket: "my-bucket",
})

// Azure Blob Storage
store, err := storage.NewAzure(ctx, storage.AzureConfig{
    AccountName: "myaccount",
    AccountKey:  "...",
    Container:   "my-container",
})

// In-memory (for testing)
store := storage.NewMemory(storage.MemoryConfig{})
```

## Basic Operations

### Upload

```go
// From io.Reader
file, _ := os.Open("document.pdf")
defer file.Close()
err := store.Put(ctx, "documents/report.pdf", file, &storage.PutOptions{
    ContentType: "application/pdf",
})

// From bytes
data := []byte("Hello, World!")
err := store.PutBytes(ctx, "hello.txt", data, nil)

// With metadata
err := store.Put(ctx, "image.jpg", reader, &storage.PutOptions{
    ContentType: "image/jpeg",
    Metadata: map[string]string{
        "author": "john",
        "version": "1.0",
    },
    CacheControl: "max-age=31536000",
})
```

### Download

```go
// As io.ReadCloser (for streaming)
reader, err := store.Get(ctx, "documents/report.pdf")
if err != nil {
    if errors.Is(err, storage.ErrNotFound) {
        // Handle not found
    }
    return err
}
defer reader.Close()
io.Copy(dst, reader)

// As bytes (for small files)
data, err := store.GetBytes(ctx, "hello.txt")

// With metadata
reader, info, err := store.GetWithInfo(ctx, "image.jpg")
fmt.Printf("Size: %d, Type: %s\n", info.Size, info.ContentType)
```

### Delete

```go
// Single object
err := store.Delete(ctx, "old-file.txt")

// Multiple objects
deleted, err := store.DeleteMany(ctx, []string{
    "temp/file1.txt",
    "temp/file2.txt",
    "temp/file3.txt",
})
```

### Check Existence

```go
exists, err := store.Exists(ctx, "documents/report.pdf")
```

### Get Metadata

```go
info, err := store.Head(ctx, "documents/report.pdf")
fmt.Printf("Size: %d bytes\n", info.Size)
fmt.Printf("Content-Type: %s\n", info.ContentType)
fmt.Printf("Last Modified: %s\n", info.LastModified)
```

### List Objects

```go
// List all objects with prefix
result, err := store.List(ctx, "documents/", nil)
for _, obj := range result.Objects {
    fmt.Printf("%s (%d bytes)\n", obj.Path, obj.Size)
}

// List with delimiter (directory-style)
result, err := store.List(ctx, "uploads/", &storage.ListOptions{
    Delimiter: "/",
    MaxKeys:   100,
})
// result.Objects contains files
// result.CommonPrefixes contains "directories"

// Pagination
result, err := store.List(ctx, "", &storage.ListOptions{MaxKeys: 100})
for result.IsTruncated {
    result, err = store.List(ctx, "", &storage.ListOptions{
        MaxKeys:           100,
        ContinuationToken: result.NextContinuationToken,
    })
}
```

### Copy and Move

```go
// Copy
err := store.Copy(ctx, "source/file.txt", "dest/file.txt")

// Move (copy + delete)
err := store.Move(ctx, "old/path.txt", "new/path.txt")
```

## Presigned URLs

Generate time-limited URLs for direct client access without exposing credentials.

### Download URL

```go
url, err := store.PresignedURL(ctx, "private/document.pdf", &storage.PresignOptions{
    Expires:            15 * time.Minute,
    ContentDisposition: "attachment; filename=report.pdf",
})
// Client can download directly from this URL
```

### Upload URL

```go
upload, err := store.PresignedUploadURL(ctx, "uploads/user-file.jpg", &storage.PresignUploadOptions{
    Expires:     15 * time.Minute,
    ContentType: "image/jpeg",
})

// Return to client for direct upload
// upload.URL - the presigned URL
// upload.Method - HTTP method (PUT)
// upload.Headers - required headers
```

## Backend Configuration

### Local Filesystem

```go
store, err := storage.NewLocal(storage.LocalConfig{
    // Required: root directory for all files
    BasePath: "/var/data/uploads",

    // Optional: base URL for generating public URLs
    BaseURL: "https://cdn.example.com/uploads",

    // Optional: auto-create directories (default: true)
    CreateDirs: &createDirs,

    // Optional: file permissions (default: 0644)
    Permissions: 0644,

    // Optional: directory permissions (default: 0755)
    DirPermissions: 0755,
})
```

### AWS S3

```go
store, err := storage.NewS3(ctx, storage.S3Config{
    // Required
    Bucket: "my-bucket",

    // Optional: uses AWS_REGION env var if not set
    Region: "us-east-1",

    // Optional: uses default credential chain if not set
    AccessKeyID:     "AKIA...",
    SecretAccessKey: "...",

    // Optional: for S3-compatible services (MinIO, DigitalOcean Spaces, etc.)
    Endpoint:     "http://localhost:9000",
    UsePathStyle: true,

    // Optional: prefix for all object keys
    Prefix: "app-data",

    // Optional: default ACL for new objects
    DefaultACL: "private", // or "public-read"

    // Optional: server-side encryption
    ServerSideEncryption: "AES256",

    // Optional: storage class
    StorageClass: "STANDARD", // or "GLACIER", "INTELLIGENT_TIERING"
})
```

### Google Cloud Storage

```go
store, err := storage.NewGCS(ctx, storage.GCSConfig{
    // Required
    Bucket: "my-bucket",

    // Optional: uses Application Default Credentials if not set
    CredentialsFile: "/path/to/service-account.json",
    // or
    CredentialsJSON: []byte(`{...}`),

    // Optional
    Prefix:       "app-data",
    DefaultACL:   "private",
    StorageClass: "STANDARD", // or "NEARLINE", "COLDLINE", "ARCHIVE"
})
defer store.Close()
```

### Azure Blob Storage

```go
store, err := storage.NewAzure(ctx, storage.AzureConfig{
    // Required
    Container: "my-container",

    // Authentication option 1: account key
    AccountName: "myaccount",
    AccountKey:  "...",

    // Authentication option 2: connection string
    ConnectionString: "DefaultEndpointsProtocol=https;AccountName=...",

    // Authentication option 3: DefaultAzureCredential (for managed identity)
    AccountName: "myaccount", // only account name, no key

    // Optional
    Prefix:            "app-data",
    DefaultAccessTier: "Hot", // or "Cool", "Archive"
})
```

### Memory (Testing)

```go
store := storage.NewMemory(storage.MemoryConfig{
    BaseURL: "http://test.local",
})

// Additional methods for testing
store.Clear()           // Remove all objects
count := store.Count()  // Get object count
```

## Content-Type Detection

Content types are automatically detected from file extensions:

```go
// Automatic detection
err := store.Put(ctx, "image.jpg", reader, nil)
// ContentType will be "image/jpeg"

// Manual override
err := store.Put(ctx, "data.bin", reader, &storage.PutOptions{
    ContentType: "application/octet-stream",
})

// Register custom MIME types
storage.RegisterMimeType(".custom", "application/x-custom")
```

## Error Handling

```go
err := store.Delete(ctx, "file.txt")

switch {
case errors.Is(err, storage.ErrNotFound):
    // Object doesn't exist
case errors.Is(err, storage.ErrPermissionDenied):
    // Access denied
case errors.Is(err, storage.ErrAlreadyExists):
    // Object already exists (when using IfNotExists)
case errors.Is(err, storage.ErrPresignNotSupported):
    // Backend doesn't support presigned URLs
case errors.Is(err, storage.ErrBucketNotFound):
    // Bucket/container doesn't exist
case errors.Is(err, storage.ErrInvalidPath):
    // Invalid object path
default:
    // Other error
}
```

## HTTP Handler Example

```go
func uploadHandler(store storage.Store) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        file, header, err := r.FormFile("file")
        if err != nil {
            http.Error(w, "No file provided", http.StatusBadRequest)
            return
        }
        defer file.Close()

        path := fmt.Sprintf("uploads/%d-%s", time.Now().Unix(), header.Filename)

        err = store.Put(r.Context(), path, file, &storage.PutOptions{
            ContentType: header.Header.Get("Content-Type"),
        })
        if err != nil {
            http.Error(w, "Upload failed", http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]string{
            "path": path,
            "url":  store.URL(path),
        })
    }
}

func downloadHandler(store storage.Store) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        path := r.URL.Query().Get("path")

        reader, info, err := store.GetWithInfo(r.Context(), path)
        if err != nil {
            if errors.Is(err, storage.ErrNotFound) {
                http.NotFound(w, r)
                return
            }
            http.Error(w, "Download failed", http.StatusInternalServerError)
            return
        }
        defer reader.Close()

        w.Header().Set("Content-Type", info.ContentType)
        w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
        io.Copy(w, reader)
    }
}
```

## Presigned Upload Flow

```go
// 1. Server generates presigned upload URL
func getUploadURL(store storage.Store) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        filename := r.URL.Query().Get("filename")
        contentType := r.URL.Query().Get("type")

        path := fmt.Sprintf("uploads/%s/%s", uuid.New(), filename)

        upload, err := store.PresignedUploadURL(r.Context(), path, &storage.PresignUploadOptions{
            Expires:     15 * time.Minute,
            ContentType: contentType,
        })
        if err != nil {
            http.Error(w, "Failed to generate URL", http.StatusInternalServerError)
            return
        }

        json.NewEncoder(w).Encode(map[string]any{
            "uploadUrl": upload.URL,
            "method":    upload.Method,
            "headers":   upload.Headers,
            "path":      path,
        })
    }
}

// 2. Client uploads directly to cloud storage
// fetch(uploadUrl, { method: 'PUT', headers, body: file })

// 3. Client confirms upload complete
// Server can verify with store.Head(ctx, path)
```

## Interface

All backends implement the `Store` interface:

```go
type Store interface {
    Put(ctx context.Context, path string, r io.Reader, opts *PutOptions) error
    PutBytes(ctx context.Context, path string, data []byte, opts *PutOptions) error
    Get(ctx context.Context, path string) (io.ReadCloser, error)
    GetBytes(ctx context.Context, path string) ([]byte, error)
    GetWithInfo(ctx context.Context, path string) (io.ReadCloser, *ObjectInfo, error)
    Head(ctx context.Context, path string) (*ObjectInfo, error)
    Delete(ctx context.Context, path string) error
    DeleteMany(ctx context.Context, paths []string) (int, error)
    Exists(ctx context.Context, path string) (bool, error)
    List(ctx context.Context, prefix string, opts *ListOptions) (*ListResult, error)
    Copy(ctx context.Context, src, dst string) error
    Move(ctx context.Context, src, dst string) error
    PresignedURL(ctx context.Context, path string, opts *PresignOptions) (string, error)
    PresignedUploadURL(ctx context.Context, path string, opts *PresignUploadOptions) (*PresignedUpload, error)
    URL(path string) string
    Backend() string
}
```
