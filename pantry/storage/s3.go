package storage

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3Config configures the S3 storage backend.
type S3Config struct {
	// Bucket is the S3 bucket name (required).
	Bucket string

	// Region is the AWS region (e.g., "us-east-1").
	// If empty, uses AWS_REGION env var or default region.
	Region string

	// AccessKeyID is the AWS access key.
	// If empty, uses default credential chain.
	AccessKeyID string

	// SecretAccessKey is the AWS secret key.
	// If empty, uses default credential chain.
	SecretAccessKey string

	// SessionToken is an optional session token for temporary credentials.
	SessionToken string

	// Endpoint is a custom S3 endpoint (for S3-compatible services like MinIO).
	Endpoint string

	// UsePathStyle forces path-style URLs (required for some S3-compatible services).
	UsePathStyle bool

	// Prefix is a path prefix for all objects.
	Prefix string

	// BaseURL is the base URL for generating public URLs.
	// If empty, uses the default S3 URL format.
	BaseURL string

	// DefaultACL is the default ACL for new objects.
	// Common values: "private", "public-read", "bucket-owner-full-control"
	DefaultACL string

	// ServerSideEncryption sets default server-side encryption.
	// Common values: "AES256", "aws:kms"
	ServerSideEncryption string

	// StorageClass sets the default storage class.
	// Common values: "STANDARD", "REDUCED_REDUNDANCY", "GLACIER", "INTELLIGENT_TIERING"
	StorageClass string

	// CloudFront configuration (optional).
	// When configured, PresignedURL uses CloudFront URLs instead of S3 presigned URLs.
	//
	// Two modes are supported:
	// 1. Public distribution: Set only CloudFrontURL. Returns unsigned URLs.
	// 2. Restricted access: Set CloudFrontURL + CloudFrontKeyPairID + key. Returns signed URLs.

	// CloudFrontURL is the CloudFront distribution URL (e.g., "https://d1234.cloudfront.net").
	// If set without signing keys, returns unsigned CloudFront URLs (public distribution).
	// If set with signing keys, returns signed CloudFront URLs (restricted access).
	CloudFrontURL string

	// CloudFrontKeyPairID is the CloudFront key pair ID for signing URLs.
	// Optional. When set, enables signed URLs for restricted access distributions.
	CloudFrontKeyPairID string

	// CloudFrontPrivateKey is the PEM-encoded RSA private key for signing.
	// Either this or CloudFrontPrivateKeyPath must be set when CloudFrontURL is set.
	CloudFrontPrivateKey string

	// CloudFrontPrivateKeyPath is the path to the PEM-encoded RSA private key file.
	// Either this or CloudFrontPrivateKey must be set when CloudFrontURL is set.
	CloudFrontPrivateKeyPath string
}

// S3 is a storage backend that uses Amazon S3 or S3-compatible services.
type S3 struct {
	client               *s3.Client
	presignClient        *s3.PresignClient
	bucket               string
	prefix               string
	baseURL              string
	defaultACL           string
	serverSideEncryption string
	storageClass         string

	// CloudFront signing (optional)
	cfURL    string          // CloudFront distribution URL
	cfSigner *sign.URLSigner // CloudFront URL signer (nil if not configured)
}

// NewS3 creates a new S3 storage backend.
func NewS3(ctx context.Context, cfg S3Config) (*S3, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("%w: Bucket is required", ErrInvalidConfig)
	}

	// Build AWS config options
	var opts []func(*config.LoadOptions) error

	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}

	// Use explicit credentials if provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		creds := credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			cfg.SessionToken,
		)
		opts = append(opts, config.WithCredentialsProvider(creds))
	}

	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to load AWS config: %w", err)
	}

	// Build S3 client options
	var s3Opts []func(*s3.Options)

	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	if cfg.UsePathStyle {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)
	presignClient := s3.NewPresignClient(client)

	// Determine base URL
	baseURL := cfg.BaseURL
	if baseURL == "" && cfg.Endpoint == "" {
		// Use default S3 URL format
		region := awsCfg.Region
		if region == "" {
			region = "us-east-1"
		}
		if region == "us-east-1" {
			baseURL = fmt.Sprintf("https://%s.s3.amazonaws.com", cfg.Bucket)
		} else {
			baseURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, region)
		}
	}

	// Initialize CloudFront signer if configured
	// If CloudFrontURL is set but no keys are provided, use unsigned URLs (public distribution)
	// If CloudFrontURL and keys are provided, use signed URLs (restricted access)
	var cfSigner *sign.URLSigner
	cfURL := cfg.CloudFrontURL
	if cfURL != "" && cfg.CloudFrontKeyPairID != "" {
		// Load private key from string or file
		var keyBytes []byte
		if cfg.CloudFrontPrivateKey != "" {
			keyBytes = []byte(cfg.CloudFrontPrivateKey)
		} else if cfg.CloudFrontPrivateKeyPath != "" {
			var err error
			keyBytes, err = os.ReadFile(cfg.CloudFrontPrivateKeyPath)
			if err != nil {
				return nil, fmt.Errorf("storage: failed to read CloudFront private key file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("%w: CloudFrontPrivateKey or CloudFrontPrivateKeyPath is required when CloudFrontKeyPairID is set", ErrInvalidConfig)
		}

		privKey, err := parseRSAPrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("storage: failed to parse CloudFront private key: %w", err)
		}

		cfSigner = sign.NewURLSigner(cfg.CloudFrontKeyPairID, privKey)
	}

	return &S3{
		client:               client,
		presignClient:        presignClient,
		bucket:               cfg.Bucket,
		prefix:               NormalizePath(cfg.Prefix),
		baseURL:              baseURL,
		defaultACL:           cfg.DefaultACL,
		serverSideEncryption: cfg.ServerSideEncryption,
		storageClass:         cfg.StorageClass,
		cfURL:                cfURL,
		cfSigner:             cfSigner,
	}, nil
}

// Backend returns the backend type identifier.
func (s *S3) Backend() string {
	return "s3"
}

// fullKey returns the full S3 key for an object.
func (s *S3) fullKey(path string) (string, error) {
	path = NormalizePath(path)
	if err := ValidatePath(path); err != nil {
		return "", err
	}

	if s.prefix != "" {
		return s.prefix + "/" + path, nil
	}
	return path, nil
}

// Put uploads an object to S3.
func (s *S3) Put(ctx context.Context, path string, r io.Reader, opts *PutOptions) error {
	key, err := s.fullKey(path)
	if err != nil {
		return err
	}

	if opts == nil {
		opts = &PutOptions{}
	}

	// Check if object exists for IfNotExists
	if opts.IfNotExists {
		exists, err := s.Exists(ctx, path)
		if err != nil {
			return err
		}
		if exists {
			return ErrAlreadyExists
		}
	}

	// Detect content type if not provided
	contentType := opts.ContentType
	if contentType == "" {
		contentType = DetectContentType(path, nil)
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
	}

	if opts.ContentDisposition != "" {
		input.ContentDisposition = aws.String(opts.ContentDisposition)
	}

	if opts.CacheControl != "" {
		input.CacheControl = aws.String(opts.CacheControl)
	}

	if opts.ContentEncoding != "" {
		input.ContentEncoding = aws.String(opts.ContentEncoding)
	}

	// Set ACL
	acl := opts.ACL
	if acl == "" {
		acl = s.defaultACL
	}
	if acl != "" {
		input.ACL = types.ObjectCannedACL(acl)
	}

	// Set server-side encryption
	sse := opts.ServerSideEncryption
	if sse == "" {
		sse = s.serverSideEncryption
	}
	if sse != "" {
		input.ServerSideEncryption = types.ServerSideEncryption(sse)
	}

	// Set storage class
	sc := opts.StorageClass
	if sc == "" {
		sc = s.storageClass
	}
	if sc != "" {
		input.StorageClass = types.StorageClass(sc)
	}

	// Set metadata
	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}

	_, err = s.client.PutObject(ctx, input)
	if err != nil {
		return s.translateError(err)
	}

	return nil
}

// PutBytes uploads bytes to S3.
func (s *S3) PutBytes(ctx context.Context, path string, data []byte, opts *PutOptions) error {
	return s.Put(ctx, path, bytes.NewReader(data), opts)
}

// Get retrieves an object from S3.
func (s *S3) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	key, err := s.fullKey(path)
	if err != nil {
		return nil, err
	}

	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, s.translateError(err)
	}

	return output.Body, nil
}

// GetBytes retrieves an object as bytes.
func (s *S3) GetBytes(ctx context.Context, path string) ([]byte, error) {
	rc, err := s.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// GetWithInfo retrieves an object along with its metadata.
func (s *S3) GetWithInfo(ctx context.Context, path string) (io.ReadCloser, *ObjectInfo, error) {
	key, err := s.fullKey(path)
	if err != nil {
		return nil, nil, err
	}

	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, nil, s.translateError(err)
	}

	info := &ObjectInfo{
		Path:         path,
		Size:         aws.ToInt64(output.ContentLength),
		ContentType:  aws.ToString(output.ContentType),
		ETag:         aws.ToString(output.ETag),
		Metadata:     output.Metadata,
		StorageClass: string(output.StorageClass),
	}

	if output.LastModified != nil {
		info.LastModified = *output.LastModified
	}

	return output.Body, info, nil
}

// Head returns metadata about an object without downloading it.
func (s *S3) Head(ctx context.Context, path string) (*ObjectInfo, error) {
	key, err := s.fullKey(path)
	if err != nil {
		return nil, err
	}

	output, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, s.translateError(err)
	}

	info := &ObjectInfo{
		Path:         path,
		Size:         aws.ToInt64(output.ContentLength),
		ContentType:  aws.ToString(output.ContentType),
		ETag:         aws.ToString(output.ETag),
		Metadata:     output.Metadata,
		StorageClass: string(output.StorageClass),
	}

	if output.LastModified != nil {
		info.LastModified = *output.LastModified
	}

	return info, nil
}

// Delete removes an object from S3.
func (s *S3) Delete(ctx context.Context, path string) error {
	key, err := s.fullKey(path)
	if err != nil {
		return err
	}

	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return s.translateError(err)
	}

	return nil
}

// DeleteMany removes multiple objects from S3.
func (s *S3) DeleteMany(ctx context.Context, paths []string) (int, error) {
	if len(paths) == 0 {
		return 0, nil
	}

	// Build object identifiers
	objects := make([]types.ObjectIdentifier, len(paths))
	for i, path := range paths {
		key, err := s.fullKey(path)
		if err != nil {
			return 0, err
		}
		objects[i] = types.ObjectIdentifier{
			Key: aws.String(key),
		}
	}

	output, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	})
	if err != nil {
		return 0, s.translateError(err)
	}

	deleted := len(paths) - len(output.Errors)
	return deleted, nil
}

// Exists checks if an object exists in S3.
func (s *S3) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.Head(ctx, path)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// List returns objects matching the given prefix.
func (s *S3) List(ctx context.Context, prefix string, opts *ListOptions) (*ListResult, error) {
	if opts == nil {
		opts = &ListOptions{}
	}

	maxKeys := opts.MaxKeys
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	// Build full prefix
	fullPrefix := s.prefix
	prefix = NormalizePath(prefix)
	if prefix != "" && prefix != "." {
		if fullPrefix != "" {
			fullPrefix = fullPrefix + "/" + prefix
		} else {
			fullPrefix = prefix
		}
	}

	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucket),
		MaxKeys: aws.Int32(int32(maxKeys)),
	}

	if fullPrefix != "" {
		input.Prefix = aws.String(fullPrefix)
	}

	if opts.Delimiter != "" {
		input.Delimiter = aws.String(opts.Delimiter)
	}

	if opts.ContinuationToken != "" {
		input.ContinuationToken = aws.String(opts.ContinuationToken)
	}

	output, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, s.translateError(err)
	}

	result := &ListResult{
		Objects:        make([]ObjectInfo, 0, len(output.Contents)),
		CommonPrefixes: make([]string, 0, len(output.CommonPrefixes)),
		IsTruncated:    aws.ToBool(output.IsTruncated),
	}

	if output.NextContinuationToken != nil {
		result.NextContinuationToken = *output.NextContinuationToken
	}

	// Convert S3 objects to ObjectInfo
	for _, obj := range output.Contents {
		key := aws.ToString(obj.Key)

		// Remove prefix from key
		path := key
		if s.prefix != "" && len(key) > len(s.prefix) {
			path = key[len(s.prefix)+1:]
		}

		info := ObjectInfo{
			Path:         path,
			Size:         aws.ToInt64(obj.Size),
			ETag:         aws.ToString(obj.ETag),
			StorageClass: string(obj.StorageClass),
		}

		if obj.LastModified != nil {
			info.LastModified = *obj.LastModified
		}

		result.Objects = append(result.Objects, info)
	}

	// Convert common prefixes
	for _, cp := range output.CommonPrefixes {
		prefix := aws.ToString(cp.Prefix)
		// Remove storage prefix
		if s.prefix != "" && len(prefix) > len(s.prefix) {
			prefix = prefix[len(s.prefix)+1:]
		}
		result.CommonPrefixes = append(result.CommonPrefixes, prefix)
	}

	return result, nil
}

// Copy copies an object from src to dst within S3.
func (s *S3) Copy(ctx context.Context, src, dst string) error {
	srcKey, err := s.fullKey(src)
	if err != nil {
		return err
	}

	dstKey, err := s.fullKey(dst)
	if err != nil {
		return err
	}

	copySource := fmt.Sprintf("%s/%s", s.bucket, srcKey)

	_, err = s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return s.translateError(err)
	}

	return nil
}

// Move moves an object from src to dst within S3.
func (s *S3) Move(ctx context.Context, src, dst string) error {
	if err := s.Copy(ctx, src, dst); err != nil {
		return err
	}
	return s.Delete(ctx, src)
}

// PresignedURL generates a presigned URL for downloading an object.
func (s *S3) PresignedURL(ctx context.Context, path string, opts *PresignOptions) (string, error) {
	key, err := s.fullKey(path)
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

	// Use CloudFront URL if configured
	if s.cfURL != "" {
		resourceURL := fmt.Sprintf("%s/%s", s.cfURL, key)
		// If signer is configured, return signed URL (restricted access)
		// Otherwise return unsigned URL (public distribution)
		if s.cfSigner != nil {
			signedURL, err := s.cfSigner.Sign(resourceURL, time.Now().Add(expires))
			if err != nil {
				return "", fmt.Errorf("storage: failed to sign CloudFront URL: %w", err)
			}
			return signedURL, nil
		}
		return resourceURL, nil
	}

	// Fall back to S3 presigned URL
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if opts.ContentType != "" {
		input.ResponseContentType = aws.String(opts.ContentType)
	}

	if opts.ContentDisposition != "" {
		input.ResponseContentDisposition = aws.String(opts.ContentDisposition)
	}

	if opts.ResponseCacheControl != "" {
		input.ResponseCacheControl = aws.String(opts.ResponseCacheControl)
	}

	presigned, err := s.presignClient.PresignGetObject(ctx, input, func(po *s3.PresignOptions) {
		po.Expires = expires
	})
	if err != nil {
		return "", fmt.Errorf("storage: failed to presign URL: %w", err)
	}

	return presigned.URL, nil
}

// PresignedUploadURL generates a presigned URL for uploading an object.
func (s *S3) PresignedUploadURL(ctx context.Context, path string, opts *PresignUploadOptions) (*PresignedUpload, error) {
	key, err := s.fullKey(path)
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

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}

	if opts.ACL != "" {
		input.ACL = types.ObjectCannedACL(opts.ACL)
	} else if s.defaultACL != "" {
		input.ACL = types.ObjectCannedACL(s.defaultACL)
	}

	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}

	presigned, err := s.presignClient.PresignPutObject(ctx, input, func(po *s3.PresignOptions) {
		po.Expires = expires
	})
	if err != nil {
		return nil, fmt.Errorf("storage: failed to presign upload URL: %w", err)
	}

	headers := make(map[string]string)
	for key, values := range presigned.SignedHeader {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return &PresignedUpload{
		URL:     presigned.URL,
		Method:  presigned.Method,
		Headers: headers,
		Expires: time.Now().Add(expires),
	}, nil
}

// URL returns the public URL for an object.
func (s *S3) URL(path string) string {
	if s.baseURL == "" {
		return ""
	}

	path = NormalizePath(path)
	if s.prefix != "" {
		return s.baseURL + "/" + s.prefix + "/" + path
	}
	return s.baseURL + "/" + path
}

// translateError converts AWS errors to storage errors.
func (s *S3) translateError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound", "404":
			return ErrNotFound
		case "NoSuchBucket":
			return ErrBucketNotFound
		case "AccessDenied", "403":
			return ErrPermissionDenied
		case "BucketAlreadyExists", "BucketAlreadyOwnedByYou":
			return ErrAlreadyExists
		}
	}

	return fmt.Errorf("storage: S3 error: %w", err)
}

// parseRSAPrivateKey parses a PEM-encoded RSA private key.
func parseRSAPrivateKey(keyBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try PKCS#1 format first
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}

	// Try PKCS#8 format
	keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	rsaKey, ok := keyInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}

	return rsaKey, nil
}
