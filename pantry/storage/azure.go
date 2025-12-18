package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

// AzureConfig configures the Azure Blob Storage backend.
type AzureConfig struct {
	// AccountName is the Azure storage account name (required).
	AccountName string

	// AccountKey is the Azure storage account key.
	// If empty, uses DefaultAzureCredential.
	AccountKey string

	// ConnectionString is the full connection string (alternative to AccountName/AccountKey).
	ConnectionString string

	// Container is the blob container name (required).
	Container string

	// Endpoint is a custom endpoint URL (for emulators or sovereign clouds).
	// If empty, uses the default Azure Blob endpoint.
	Endpoint string

	// Prefix is a path prefix for all objects.
	Prefix string

	// BaseURL is the base URL for generating public URLs.
	// If empty, uses the default Azure Blob URL format.
	BaseURL string

	// DefaultAccessTier sets the default access tier.
	// Common values: "Hot", "Cool", "Archive"
	DefaultAccessTier string
}

// Azure is a storage backend that uses Azure Blob Storage.
type Azure struct {
	client            *azblob.Client
	containerClient   *container.Client
	accountName       string
	accountKey        string
	containerName     string
	prefix            string
	baseURL           string
	defaultAccessTier string
}

// NewAzure creates a new Azure Blob Storage backend.
func NewAzure(ctx context.Context, cfg AzureConfig) (*Azure, error) {
	if cfg.Container == "" {
		return nil, fmt.Errorf("%w: Container is required", ErrInvalidConfig)
	}

	var client *azblob.Client
	var err error
	var accountName, accountKey string

	if cfg.ConnectionString != "" {
		// Use connection string
		client, err = azblob.NewClientFromConnectionString(cfg.ConnectionString, nil)
		if err != nil {
			return nil, fmt.Errorf("storage: failed to create Azure client from connection string: %w", err)
		}
	} else if cfg.AccountName != "" && cfg.AccountKey != "" {
		// Use shared key credential
		accountName = cfg.AccountName
		accountKey = cfg.AccountKey

		cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
		if err != nil {
			return nil, fmt.Errorf("storage: failed to create Azure credential: %w", err)
		}

		endpoint := cfg.Endpoint
		if endpoint == "" {
			endpoint = fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
		}

		client, err = azblob.NewClientWithSharedKeyCredential(endpoint, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("storage: failed to create Azure client: %w", err)
		}
	} else if cfg.AccountName != "" {
		// Use DefaultAzureCredential
		accountName = cfg.AccountName

		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("storage: failed to create Azure credential: %w", err)
		}

		endpoint := cfg.Endpoint
		if endpoint == "" {
			endpoint = fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
		}

		client, err = azblob.NewClient(endpoint, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("storage: failed to create Azure client: %w", err)
		}
	} else {
		return nil, fmt.Errorf("%w: AccountName or ConnectionString is required", ErrInvalidConfig)
	}

	containerClient := client.ServiceClient().NewContainerClient(cfg.Container)

	// Determine base URL
	baseURL := cfg.BaseURL
	if baseURL == "" && accountName != "" {
		baseURL = fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, cfg.Container)
	}

	return &Azure{
		client:            client,
		containerClient:   containerClient,
		accountName:       accountName,
		accountKey:        accountKey,
		containerName:     cfg.Container,
		prefix:            NormalizePath(cfg.Prefix),
		baseURL:           baseURL,
		defaultAccessTier: cfg.DefaultAccessTier,
	}, nil
}

// Backend returns the backend type identifier.
func (a *Azure) Backend() string {
	return "azure"
}

// fullKey returns the full blob name.
func (a *Azure) fullKey(path string) (string, error) {
	path = NormalizePath(path)
	if err := ValidatePath(path); err != nil {
		return "", err
	}

	if a.prefix != "" {
		return a.prefix + "/" + path, nil
	}
	return path, nil
}

// Put uploads an object to Azure Blob Storage.
func (a *Azure) Put(ctx context.Context, path string, r io.Reader, opts *PutOptions) error {
	key, err := a.fullKey(path)
	if err != nil {
		return err
	}

	if opts == nil {
		opts = &PutOptions{}
	}

	// Check if object exists for IfNotExists
	if opts.IfNotExists {
		exists, err := a.Exists(ctx, path)
		if err != nil {
			return err
		}
		if exists {
			return ErrAlreadyExists
		}
	}

	// Detect content type
	contentType := opts.ContentType
	if contentType == "" {
		contentType = DetectContentType(path, nil)
	}

	blobClient := a.containerClient.NewBlockBlobClient(key)

	uploadOpts := &blockblob.UploadStreamOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: &contentType,
		},
	}

	if opts.ContentDisposition != "" {
		uploadOpts.HTTPHeaders.BlobContentDisposition = &opts.ContentDisposition
	}

	if opts.CacheControl != "" {
		uploadOpts.HTTPHeaders.BlobCacheControl = &opts.CacheControl
	}

	if opts.ContentEncoding != "" {
		uploadOpts.HTTPHeaders.BlobContentEncoding = &opts.ContentEncoding
	}

	// Set access tier
	tier := opts.StorageClass
	if tier == "" {
		tier = a.defaultAccessTier
	}
	if tier != "" {
		accessTier := blob.AccessTier(tier)
		uploadOpts.AccessTier = &accessTier
	}

	// Set metadata
	if len(opts.Metadata) > 0 {
		uploadOpts.Metadata = toAzureMetadata(opts.Metadata)
	}

	_, err = blobClient.UploadStream(ctx, r, uploadOpts)
	if err != nil {
		return a.translateError(err)
	}

	return nil
}

// PutBytes uploads bytes to Azure Blob Storage.
func (a *Azure) PutBytes(ctx context.Context, path string, data []byte, opts *PutOptions) error {
	return a.Put(ctx, path, bytes.NewReader(data), opts)
}

// Get retrieves an object from Azure Blob Storage.
func (a *Azure) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	key, err := a.fullKey(path)
	if err != nil {
		return nil, err
	}

	blobClient := a.containerClient.NewBlobClient(key)

	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, a.translateError(err)
	}

	return resp.Body, nil
}

// GetBytes retrieves an object as bytes.
func (a *Azure) GetBytes(ctx context.Context, path string) ([]byte, error) {
	rc, err := a.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// GetWithInfo retrieves an object along with its metadata.
func (a *Azure) GetWithInfo(ctx context.Context, path string) (io.ReadCloser, *ObjectInfo, error) {
	key, err := a.fullKey(path)
	if err != nil {
		return nil, nil, err
	}

	blobClient := a.containerClient.NewBlobClient(key)

	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, nil, a.translateError(err)
	}

	info := &ObjectInfo{
		Path:         path,
		Size:         *resp.ContentLength,
		ContentType:  *resp.ContentType,
		LastModified: *resp.LastModified,
		Metadata:     fromAzureMetadata(resp.Metadata),
	}

	if resp.ETag != nil {
		info.ETag = string(*resp.ETag)
	}

	return resp.Body, info, nil
}

// Head returns metadata about an object without downloading it.
func (a *Azure) Head(ctx context.Context, path string) (*ObjectInfo, error) {
	key, err := a.fullKey(path)
	if err != nil {
		return nil, err
	}

	blobClient := a.containerClient.NewBlobClient(key)

	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, a.translateError(err)
	}

	info := &ObjectInfo{
		Path:         path,
		Size:         *props.ContentLength,
		LastModified: *props.LastModified,
		Metadata:     fromAzureMetadata(props.Metadata),
	}

	if props.ContentType != nil {
		info.ContentType = *props.ContentType
	}

	if props.ETag != nil {
		info.ETag = string(*props.ETag)
	}

	return info, nil
}

// Delete removes an object from Azure Blob Storage.
func (a *Azure) Delete(ctx context.Context, path string) error {
	key, err := a.fullKey(path)
	if err != nil {
		return err
	}

	blobClient := a.containerClient.NewBlobClient(key)

	_, err = blobClient.Delete(ctx, nil)
	if err != nil {
		return a.translateError(err)
	}

	return nil
}

// DeleteMany removes multiple objects from Azure Blob Storage.
func (a *Azure) DeleteMany(ctx context.Context, paths []string) (int, error) {
	deleted := 0
	var lastErr error

	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}

		if err := a.Delete(ctx, path); err != nil {
			if !errors.Is(err, ErrNotFound) {
				lastErr = err
			}
		} else {
			deleted++
		}
	}

	return deleted, lastErr
}

// Exists checks if an object exists in Azure Blob Storage.
func (a *Azure) Exists(ctx context.Context, path string) (bool, error) {
	_, err := a.Head(ctx, path)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// List returns objects matching the given prefix.
func (a *Azure) List(ctx context.Context, prefix string, opts *ListOptions) (*ListResult, error) {
	if opts == nil {
		opts = &ListOptions{}
	}

	maxKeys := opts.MaxKeys
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	// Build full prefix
	fullPrefix := a.prefix
	prefix = NormalizePath(prefix)
	if prefix != "" && prefix != "." {
		if fullPrefix != "" {
			fullPrefix = fullPrefix + "/" + prefix
		} else {
			fullPrefix = prefix
		}
	}

	listOpts := &container.ListBlobsFlatOptions{
		MaxResults: toInt32Ptr(int32(maxKeys)),
	}

	if fullPrefix != "" {
		listOpts.Prefix = &fullPrefix
	}

	if opts.ContinuationToken != "" {
		listOpts.Marker = &opts.ContinuationToken
	}

	result := &ListResult{
		Objects:        make([]ObjectInfo, 0),
		CommonPrefixes: make([]string, 0),
	}

	// Use hierarchical listing if delimiter is specified
	if opts.Delimiter != "" {
		return a.listHierarchical(ctx, fullPrefix, opts)
	}

	pager := a.containerClient.NewListBlobsFlatPager(listOpts)

	if pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, a.translateError(err)
		}

		for _, item := range page.Segment.BlobItems {
			// Remove prefix from key
			path := *item.Name
			if a.prefix != "" && len(path) > len(a.prefix) {
				path = path[len(a.prefix)+1:]
			}

			info := ObjectInfo{
				Path: path,
			}

			if item.Properties != nil {
				if item.Properties.ContentLength != nil {
					info.Size = *item.Properties.ContentLength
				}
				if item.Properties.ContentType != nil {
					info.ContentType = *item.Properties.ContentType
				}
				if item.Properties.LastModified != nil {
					info.LastModified = *item.Properties.LastModified
				}
				if item.Properties.ETag != nil {
					info.ETag = string(*item.Properties.ETag)
				}
				if item.Properties.AccessTier != nil {
					info.StorageClass = string(*item.Properties.AccessTier)
				}
			}

			result.Objects = append(result.Objects, info)
		}

		if page.NextMarker != nil && *page.NextMarker != "" {
			result.IsTruncated = true
			result.NextContinuationToken = *page.NextMarker
		}
	}

	return result, nil
}

// listHierarchical lists blobs with delimiter support.
func (a *Azure) listHierarchical(ctx context.Context, prefix string, opts *ListOptions) (*ListResult, error) {
	maxKeys := opts.MaxKeys
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	listOpts := &container.ListBlobsHierarchyOptions{
		MaxResults: toInt32Ptr(int32(maxKeys)),
	}

	if prefix != "" {
		listOpts.Prefix = &prefix
	}

	if opts.ContinuationToken != "" {
		listOpts.Marker = &opts.ContinuationToken
	}

	result := &ListResult{
		Objects:        make([]ObjectInfo, 0),
		CommonPrefixes: make([]string, 0),
	}

	pager := a.containerClient.NewListBlobsHierarchyPager(opts.Delimiter, listOpts)

	if pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, a.translateError(err)
		}

		// Process blob prefixes (directories)
		for _, item := range page.Segment.BlobPrefixes {
			cp := *item.Name
			if a.prefix != "" && len(cp) > len(a.prefix) {
				cp = cp[len(a.prefix)+1:]
			}
			result.CommonPrefixes = append(result.CommonPrefixes, cp)
		}

		// Process blob items
		for _, item := range page.Segment.BlobItems {
			path := *item.Name
			if a.prefix != "" && len(path) > len(a.prefix) {
				path = path[len(a.prefix)+1:]
			}

			info := ObjectInfo{
				Path: path,
			}

			if item.Properties != nil {
				if item.Properties.ContentLength != nil {
					info.Size = *item.Properties.ContentLength
				}
				if item.Properties.ContentType != nil {
					info.ContentType = *item.Properties.ContentType
				}
				if item.Properties.LastModified != nil {
					info.LastModified = *item.Properties.LastModified
				}
				if item.Properties.ETag != nil {
					info.ETag = string(*item.Properties.ETag)
				}
				if item.Properties.AccessTier != nil {
					info.StorageClass = string(*item.Properties.AccessTier)
				}
			}

			result.Objects = append(result.Objects, info)
		}

		if page.NextMarker != nil && *page.NextMarker != "" {
			result.IsTruncated = true
			result.NextContinuationToken = *page.NextMarker
		}
	}

	return result, nil
}

// Copy copies an object from src to dst within Azure Blob Storage.
func (a *Azure) Copy(ctx context.Context, src, dst string) error {
	srcKey, err := a.fullKey(src)
	if err != nil {
		return err
	}

	dstKey, err := a.fullKey(dst)
	if err != nil {
		return err
	}

	srcBlobClient := a.containerClient.NewBlobClient(srcKey)
	dstBlobClient := a.containerClient.NewBlobClient(dstKey)

	_, err = dstBlobClient.CopyFromURL(ctx, srcBlobClient.URL(), nil)
	if err != nil {
		// Try async copy if sync copy fails
		_, err = dstBlobClient.StartCopyFromURL(ctx, srcBlobClient.URL(), nil)
		if err != nil {
			return a.translateError(err)
		}
	}

	return nil
}

// Move moves an object from src to dst within Azure Blob Storage.
func (a *Azure) Move(ctx context.Context, src, dst string) error {
	if err := a.Copy(ctx, src, dst); err != nil {
		return err
	}
	return a.Delete(ctx, src)
}

// PresignedURL generates a presigned URL for downloading an object.
func (a *Azure) PresignedURL(ctx context.Context, path string, opts *PresignOptions) (string, error) {
	if a.accountKey == "" {
		return "", fmt.Errorf("storage: presigned URLs require account key authentication")
	}

	key, err := a.fullKey(path)
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

	cred, err := azblob.NewSharedKeyCredential(a.accountName, a.accountKey)
	if err != nil {
		return "", fmt.Errorf("storage: failed to create credential: %w", err)
	}

	blobClient := a.containerClient.NewBlobClient(key)

	// Create SAS query parameters
	sasPermissions := sas.BlobPermissions{Read: true}

	startTime := time.Now().UTC().Add(-10 * time.Minute)
	expiryTime := time.Now().UTC().Add(expires)

	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     startTime,
		ExpiryTime:    expiryTime,
		Permissions:   sasPermissions.String(),
		ContainerName: a.containerName,
		BlobName:      key,
	}.SignWithSharedKey(cred)
	if err != nil {
		return "", fmt.Errorf("storage: failed to sign URL: %w", err)
	}

	return blobClient.URL() + "?" + sasQueryParams.Encode(), nil
}

// PresignedUploadURL generates a presigned URL for uploading an object.
func (a *Azure) PresignedUploadURL(ctx context.Context, path string, opts *PresignUploadOptions) (*PresignedUpload, error) {
	if a.accountKey == "" {
		return nil, fmt.Errorf("storage: presigned URLs require account key authentication")
	}

	key, err := a.fullKey(path)
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

	cred, err := azblob.NewSharedKeyCredential(a.accountName, a.accountKey)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to create credential: %w", err)
	}

	blobClient := a.containerClient.NewBlobClient(key)

	// Create SAS query parameters
	sasPermissions := sas.BlobPermissions{Write: true, Create: true}

	startTime := time.Now().UTC().Add(-10 * time.Minute)
	expiryTime := time.Now().UTC().Add(expires)

	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     startTime,
		ExpiryTime:    expiryTime,
		Permissions:   sasPermissions.String(),
		ContainerName: a.containerName,
		BlobName:      key,
	}.SignWithSharedKey(cred)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to sign upload URL: %w", err)
	}

	url := blobClient.URL() + "?" + sasQueryParams.Encode()

	headers := make(map[string]string)
	headers["x-ms-blob-type"] = "BlockBlob"

	if opts.ContentType != "" {
		headers["x-ms-blob-content-type"] = opts.ContentType
	}

	return &PresignedUpload{
		URL:     url,
		Method:  "PUT",
		Headers: headers,
		Expires: expiryTime,
	}, nil
}

// URL returns the public URL for an object.
func (a *Azure) URL(path string) string {
	if a.baseURL == "" {
		return ""
	}

	path = NormalizePath(path)
	if a.prefix != "" {
		return a.baseURL + "/" + a.prefix + "/" + path
	}
	return a.baseURL + "/" + path
}

// translateError converts Azure errors to storage errors.
func (a *Azure) translateError(err error) error {
	if err == nil {
		return nil
	}

	if bloberror.HasCode(err, bloberror.BlobNotFound) {
		return ErrNotFound
	}

	if bloberror.HasCode(err, bloberror.ContainerNotFound) {
		return ErrBucketNotFound
	}

	if bloberror.HasCode(err, bloberror.AuthorizationFailure, bloberror.AuthorizationPermissionMismatch) {
		return ErrPermissionDenied
	}

	if bloberror.HasCode(err, bloberror.BlobAlreadyExists) {
		return ErrAlreadyExists
	}

	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		switch respErr.StatusCode {
		case 404:
			return ErrNotFound
		case 403:
			return ErrPermissionDenied
		case 409:
			return ErrAlreadyExists
		}
	}

	return fmt.Errorf("storage: Azure error: %w", err)
}

// toInt32Ptr converts an int32 to a pointer.
func toInt32Ptr(v int32) *int32 {
	return &v
}

// toAzureMetadata converts map[string]string to map[string]*string for Azure SDK.
func toAzureMetadata(m map[string]string) map[string]*string {
	if m == nil {
		return nil
	}
	result := make(map[string]*string, len(m))
	for k, v := range m {
		val := v
		result[k] = &val
	}
	return result
}

// fromAzureMetadata converts map[string]*string from Azure SDK to map[string]string.
func fromAzureMetadata(m map[string]*string) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}
