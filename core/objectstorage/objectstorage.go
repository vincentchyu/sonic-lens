package objectstorage

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/vincentchyu/sonic-lens/config"
)

// Provider 定义对象存储抽象能力，便于未来切换 MinIO/R2/S3。
type Provider interface {
	CheckObjectExists(ctx context.Context, objectKey string) (exists bool, contentType string, err error)
	UploadFileToObject(ctx context.Context, objectKey, filePath, contentType string) error
	UploadBytesToObject(ctx context.Context, objectKey string, payload []byte, contentType string) error
	DeleteObject(ctx context.Context, objectKey string) error
	DeleteObjects(ctx context.Context, objectKeys []string) error
	GetObjectCDNURL(objectKey string) string
	BuildOriginalObjectKey(seed string) string
}

var (
	provider Provider
	mu       sync.RWMutex
)

// Init 初始化对象存储 Provider。未启用时返回 nil 并保持 provider 为空。
func Init(cfg config.ObjectStorageConfig) error {
	if !cfg.Enabled {
		mu.Lock()
		provider = nil
		mu.Unlock()
		return nil
	}

	selected := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if selected == "" {
		selected = "s3"
	}

	var impl Provider
	switch selected {
	case "s3", "minio", "r2":
		s3Provider, err := NewS3Provider(cfg)
		if err != nil {
			return err
		}
		impl = s3Provider
	default:
		return fmt.Errorf("unsupported object storage provider: %s", selected)
	}

	mu.Lock()
	provider = impl
	mu.Unlock()
	return nil
}

// Get 获取全局对象存储 Provider，未初始化或禁用时返回 nil。
func Get() Provider {
	mu.RLock()
	defer mu.RUnlock()
	return provider
}

// JoinObjectKey 拼接对象键并标准化路径。
func JoinObjectKey(parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		normalized = append(normalized, strings.Trim(part, "/"))
	}
	return strings.TrimLeft(path.Clean("/"+strings.Join(normalized, "/")), "/")
}
