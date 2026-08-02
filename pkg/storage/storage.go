// Package storage provides object storage abstraction over MinIO and Azure Blob.
package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"

	"github.com/goshield/pkg/config"
)

// Client is the storage client interface.
type Client interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	HealthCheck(ctx context.Context) error
}

// minioClient implements Client using MinIO SDK (S3-compatible).
type minioClient struct {
	client *minio.Client
	bucket string
	logger *zap.Logger
}

// NewMinIOClient creates a MinIO-backed storage client.
func NewMinIOClient(cfg config.StorageConfig, logger *zap.Logger) (Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	// Ensure bucket exists.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := mc.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := mc.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
		logger.Info("created storage bucket", zap.String("bucket", cfg.Bucket))
	}

	logger.Info("MinIO storage connected",
		zap.String("endpoint", cfg.Endpoint),
		zap.String("bucket", cfg.Bucket),
	)

	return &minioClient{client: mc, bucket: cfg.Bucket, logger: logger}, nil
}

// NewClient creates a storage client based on config provider.
func NewClient(cfg config.StorageConfig, logger *zap.Logger) (Client, error) {
	switch cfg.Provider {
	case "minio", "s3":
		return NewMinIOClient(cfg, logger)
	default:
		// Default to MinIO for local development.
		return NewMinIOClient(cfg, logger)
	}
}

func (c *minioClient) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}
	if contentType == "" {
		opts.ContentType = detectContentType(key)
	}

	info, err := c.client.PutObject(ctx, c.bucket, key, reader, size, opts)
	if err != nil {
		return "", fmt.Errorf("upload object %s: %w", key, err)
	}

	url := fmt.Sprintf("%s/%s/%s", c.client.EndpointURL(), c.bucket, key)
	c.logger.Debug("object uploaded",
		zap.String("key", key),
		zap.Int64("size", info.Size),
	)
	return url, nil
}

func (c *minioClient) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("download object %s: %w", key, err)
	}
	return obj, nil
}

func (c *minioClient) Delete(ctx context.Context, key string) error {
	return c.client.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}

func (c *minioClient) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := c.client.PresignedGetObject(ctx, c.bucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("presign %s: %w", key, err)
	}
	return u.String(), nil
}

func (c *minioClient) HealthCheck(ctx context.Context) error {
	_, err := c.client.BucketExists(ctx, c.bucket)
	return err
}

func detectContentType(key string) string {
	ext := filepath.Ext(key)
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}
