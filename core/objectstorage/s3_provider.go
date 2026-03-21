package objectstorage

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/metrics/smithyotelmetrics"
	"github.com/aws/smithy-go/tracing/smithyoteltracing"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/telemetry"
)

// S3Provider 使用 S3 兼容接口（MinIO/R2/AWS）实现对象存储能力。
type S3Provider struct {
	client *s3.Client
	cfg    config.ObjectStorageConfig
}

// NewS3Provider 创建 S3 兼容 Provider。
func NewS3Provider(cfg config.ObjectStorageConfig) (*S3Provider, error) {
	endpoint := normalizeEndpoint(cfg.Endpoint, cfg.UseSSL)
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}

	awsCfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
	}
	if endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(endpoint)
	}

	client := s3.NewFromConfig(
		awsCfg, func(o *s3.Options) {
			o.UsePathStyle = cfg.ForcePathStyle || endpoint != ""
			o.TracerProvider = smithyoteltracing.Adapt(telemetry.GetTracerProvider())
			o.MeterProvider = smithyotelmetrics.Adapt(telemetry.GetMeterProvider())
		},
	)

	provider := &S3Provider{
		client: client,
		cfg:    cfg,
	}
	if err := provider.ensureBucket(context.Background()); err != nil {
		return nil, err
	}
	return provider, nil
}

func (s *S3Provider) ensureBucket(ctx context.Context) error {
	bucket := strings.TrimSpace(s.cfg.Bucket)
	if bucket == "" {
		return fmt.Errorf("object storage bucket is empty")
	}
	_, err := s.client.HeadBucket(
		ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucket),
		},
	)
	if err == nil {
		return nil
	}

	_, createErr := s.client.CreateBucket(
		ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucket),
		},
	)
	if createErr != nil {
		return fmt.Errorf("head/create bucket failed: %w", err)
	}
	return nil
}

func (s *S3Provider) CheckObjectExists(ctx context.Context, objectKey string) (bool, string, error) {
	key := JoinObjectKey(objectKey)
	out, err := s.client.HeadObject(
		ctx, &s3.HeadObjectInput{
			Bucket: aws.String(s.cfg.Bucket),
			Key:    aws.String(key),
		},
	)
	if err != nil {
		var notFound *types.NotFound
		if strings.Contains(strings.ToLower(err.Error()), "not found") || strings.Contains(
			strings.ToLower(err.Error()), "no such key",
		) || strings.Contains(strings.ToLower(err.Error()), "status code: 404") || errors.As(err, &notFound) {
			return false, "", nil
		}
		return false, "", err
	}

	return true, strings.TrimSpace(aws.ToString(out.ContentType)), nil
}

func (s *S3Provider) UploadFileToObject(ctx context.Context, objectKey, filePath, contentType string) error {
	payload, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return s.UploadBytesToObject(ctx, objectKey, payload, contentType)
}

func (s *S3Provider) UploadBytesToObject(ctx context.Context, objectKey string, payload []byte, contentType string) error {
	key := JoinObjectKey(objectKey)
	if len(payload) == 0 {
		return nil
	}
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(payload),
	}
	if strings.TrimSpace(contentType) != "" {
		input.ContentType = aws.String(strings.TrimSpace(contentType))
	}
	_, err := s.client.PutObject(ctx, input)
	return err
}

func (s *S3Provider) DeleteObject(ctx context.Context, objectKey string) error {
	key := JoinObjectKey(objectKey)
	_, err := s.client.DeleteObject(
		ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.cfg.Bucket),
			Key:    aws.String(key),
		},
	)
	return err
}

func (s *S3Provider) DeleteObjects(ctx context.Context, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}
	objects := make([]types.ObjectIdentifier, 0, len(objectKeys))
	for _, key := range objectKeys {
		normalized := JoinObjectKey(key)
		if normalized == "" {
			continue
		}
		objects = append(objects, types.ObjectIdentifier{Key: aws.String(normalized)})
	}
	if len(objects) == 0 {
		return nil
	}

	_, err := s.client.DeleteObjects(
		ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.cfg.Bucket),
			Delete: &types.Delete{
				Objects: objects,
				Quiet:   aws.Bool(true),
			},
		},
	)
	return err
}

func (s *S3Provider) GetObjectCDNURL(objectKey string) string {
	key := JoinObjectKey(objectKey)
	if key == "" {
		return ""
	}
	cdn := strings.TrimSpace(s.cfg.CDNURL)
	if cdn != "" {
		// 仅返回相对路径，不返回域名，交由调用方按当前服务地址拼接。
		if strings.HasPrefix(cdn, "/") {
			return strings.TrimRight(cdn, "/") + "/" + escapeObjectKey(key)
		}
		if parsed, err := url.Parse(cdn); err == nil {
			base := strings.TrimRight(parsed.Path, "/")
			if base == "" {
				base = "/" + strings.Trim(strings.TrimSpace(s.cfg.Bucket), "/")
			}
			return base + "/" + escapeObjectKey(key)
		}
	}
	return "/" + strings.Trim(strings.TrimSpace(s.cfg.Bucket), "/") + "/" + escapeObjectKey(key)
}

func (s *S3Provider) BuildOriginalObjectKey(seed string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(seed)))
	hash := hex.EncodeToString(sum[:])
	return JoinObjectKey(s.cfg.BasePrefix, s.cfg.OriginalPrefix, hash)
}

func normalizeEndpoint(endpoint string, useSSL bool) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return strings.TrimRight(trimmed, "/")
	}
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, strings.TrimRight(trimmed, "/"))
}

func escapeObjectKey(key string) string {
	parts := strings.Split(key, "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	return strings.Join(escaped, "/")
}
