package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/core/log"
	coreredis "github.com/vincentchyu/sonic-lens/core/redis"
)

const (
	defaultRedisCacheTTL      = 5 * time.Minute
	defaultRedisEmptyCacheTTL = 3 * time.Second
	redisCacheKeyPrefix       = "http:cache:"
)

var errRedisHTTPCacheUnavailable = errors.New("redis http cache unavailable")

// httpCacheStore 定义了 HTTP 缓存的存取边界，便于在测试中替换为内存实现。
type httpCacheStore interface {
	Get(ctx context.Context, key string) (*httpCacheEntry, error)
	Set(ctx context.Context, key string, entry *httpCacheEntry, ttl time.Duration) error
}

type httpCacheEntry struct {
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      []byte            `json:"body"`
	ETag      string            `json:"etag"`
	MaxAge    int               `json:"max_age"`
	CreatedAt time.Time         `json:"created_at"`
}

type redisHTTPCacheStore struct{}

func (s redisHTTPCacheStore) Get(ctx context.Context, key string) (*httpCacheEntry, error) {
	client := coreredis.GetRedisClient()
	if client == nil {
		return nil, errRedisHTTPCacheUnavailable
	}

	val, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	var entry httpCacheEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (s redisHTTPCacheStore) Set(ctx context.Context, key string, entry *httpCacheEntry, ttl time.Duration) error {
	client := coreredis.GetRedisClient()
	if client == nil {
		return errRedisHTTPCacheUnavailable
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return client.Set(ctx, key, payload, ttl).Err()
}

type redisCacheOptions struct {
	ttl      time.Duration
	emptyTTL time.Duration
	keyFunc  func(*gin.Context) string
	store    httpCacheStore
	timeout  time.Duration
}

type redisCacheOption func(*redisCacheOptions)

func withRedisCacheTTL(ttl time.Duration) redisCacheOption {
	return func(o *redisCacheOptions) {
		if ttl > 0 {
			o.ttl = ttl
		}
	}
}

func withRedisCacheKeyFunc(keyFunc func(*gin.Context) string) redisCacheOption {
	return func(o *redisCacheOptions) {
		if keyFunc != nil {
			o.keyFunc = keyFunc
		}
	}
}

func withRedisCacheStore(store httpCacheStore) redisCacheOption {
	return func(o *redisCacheOptions) {
		if store != nil {
			o.store = store
		}
	}
}

func withRedisCacheTimeout(timeout time.Duration) redisCacheOption {
	return func(o *redisCacheOptions) {
		if timeout > 0 {
			o.timeout = timeout
		}
	}
}

func withRedisCacheEmptyTTL(ttl time.Duration) redisCacheOption {
	return func(o *redisCacheOptions) {
		if ttl > 0 {
			o.emptyTTL = ttl
		}
	}
}

// redisCache 为读接口提供 Redis 响应缓存，默认使用 5 分钟 TTL，并在响应头输出 ETag。
func redisCache(ttl ...time.Duration) gin.HandlerFunc {
	cacheTTL := defaultRedisCacheTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		cacheTTL = ttl[0]
	}

	return newRedisCacheMiddleware(
		withRedisCacheTTL(cacheTTL),
		withRedisCacheEmptyTTL(defaultRedisEmptyCacheTTL),
		withRedisCacheStore(redisHTTPCacheStore{}),
		withRedisCacheTimeout(250*time.Millisecond),
	)
}

func newRedisCacheMiddleware(opts ...redisCacheOption) gin.HandlerFunc {
	options := redisCacheOptions{
		ttl:      defaultRedisCacheTTL,
		emptyTTL: defaultRedisEmptyCacheTTL,
		store:    redisHTTPCacheStore{},
		timeout:  250 * time.Millisecond,
		keyFunc:  defaultRedisCacheKey,
	}
	for _, opt := range opts {
		opt(&options)
	}

	return func(c *gin.Context) {
		if c.Request == nil || c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Next()
			return
		}

		if shouldBypassCache(c.Request.Header) {
			c.Next()
			return
		}

		cacheKey := options.keyFunc(c)
		if cacheKey == "" || options.store == nil || options.ttl <= 0 {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		if options.timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, options.timeout)
			defer cancel()
		}

		entry, err := options.store.Get(ctx, cacheKey)
		if err != nil && !errors.Is(err, errRedisHTTPCacheUnavailable) {
			log.Warn(c.Request.Context(), "读取 HTTP Redis 缓存失败，将继续执行原始接口", zap.Error(err), zap.String("cache_key", cacheKey))
		}
		if entry != nil {
			serveCachedResponse(c, entry, cacheKey)
			return
		}

		originalWriter := c.Writer
		recorder := newCacheResponseRecorder(originalWriter)
		c.Writer = recorder
		c.Next()

		if shouldSkipCacheWrite(c) {
			return
		}

		status := recorder.Status()
		body := recorder.body.Bytes()
		cacheTTL, cacheable := cacheTTLForResponse(status, body, options.ttl, options.emptyTTL)
		if !cacheable {
			return
		}

		entry = buildHTTPCacheEntry(status, recorder.Header(), body, cacheTTL)
		if entry == nil {
			return
		}

		c.Writer = originalWriter
		writeCacheHeaders(c.Writer, entry, time.Duration(entry.MaxAge)*time.Second, 0)
		c.Writer.WriteHeader(entry.Status)
		if c.Request.Method != http.MethodHead {
			if _, writeErr := c.Writer.Write(body); writeErr != nil {
				log.Warn(c.Request.Context(), "写回 HTTP 缓存响应失败", zap.Error(writeErr), zap.String("cache_key", cacheKey))
			}
		}

		storeCtx := c.Request.Context()
		if options.timeout > 0 {
			var cancel context.CancelFunc
			storeCtx, cancel = context.WithTimeout(storeCtx, options.timeout)
			defer cancel()
		}
		if err := options.store.Set(storeCtx, cacheKey, entry, cacheTTL); err != nil && !errors.Is(err, errRedisHTTPCacheUnavailable) {
			log.Warn(c.Request.Context(), "写入 HTTP Redis 缓存失败", zap.Error(err), zap.String("cache_key", cacheKey))
		}
	}
}

type cacheResponseRecorder struct {
	gin.ResponseWriter
	body   bytes.Buffer
	status int
}

func newCacheResponseRecorder(writer gin.ResponseWriter) *cacheResponseRecorder {
	return &cacheResponseRecorder{
		ResponseWriter: writer,
		status:         http.StatusOK,
	}
}

func (r *cacheResponseRecorder) WriteHeader(code int) {
	r.status = code
}

func (r *cacheResponseRecorder) WriteHeaderNow() {
	r.status = r.statusOrDefault()
}

func (r *cacheResponseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}

func (r *cacheResponseRecorder) WriteString(s string) (int, error) {
	return r.Write([]byte(s))
}

func (r *cacheResponseRecorder) Status() int {
	return r.statusOrDefault()
}

func (r *cacheResponseRecorder) Size() int {
	return r.body.Len()
}

func (r *cacheResponseRecorder) Written() bool {
	return r.body.Len() > 0 || r.status > 0
}

func (r *cacheResponseRecorder) Flush() {
}

func (r *cacheResponseRecorder) statusOrDefault() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func buildRedisCacheKey(method, path string, query url.Values, accept string) string {
	raw := strings.Builder{}
	raw.WriteString(method)
	raw.WriteString(":")
	raw.WriteString(path)
	if normalizedQuery := normalizeQuery(query); normalizedQuery != "" {
		raw.WriteString("?")
		raw.WriteString(normalizedQuery)
	}
	if accept = strings.TrimSpace(accept); accept != "" {
		raw.WriteString("|accept=")
		raw.WriteString(accept)
	}
	sum := sha256.Sum256([]byte(raw.String()))
	return redisCacheKeyPrefix + fmt.Sprintf("%x", sum[:])
}

func defaultRedisCacheKey(c *gin.Context) string {
	return buildRedisCacheKey(c.Request.Method, c.Request.URL.Path, c.Request.URL.Query(), c.GetHeader("Accept"))
}

func normalizeQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	return values.Encode()
}

func shouldBypassCache(header http.Header) bool {
	cacheControl := strings.ToLower(header.Get("Cache-Control"))
	if strings.Contains(cacheControl, "no-cache") || strings.Contains(cacheControl, "no-store") {
		return true
	}
	pragmas := strings.ToLower(header.Get("Pragma"))
	return strings.Contains(pragmas, "no-cache")
}

func buildHTTPCacheEntry(status int, header http.Header, body []byte, ttl time.Duration) *httpCacheEntry {
	entry := &httpCacheEntry{
		Status:    status,
		Headers:   make(map[string]string),
		Body:      append([]byte(nil), body...),
		ETag:      buildETag(body),
		MaxAge:    int(ttl.Seconds()),
		CreatedAt: time.Now().UTC(),
	}

	if contentType := strings.TrimSpace(header.Get("Content-Type")); contentType != "" {
		entry.Headers["Content-Type"] = contentType
	}
	return entry
}

func cacheTTLForResponse(status int, body []byte, ttl, emptyTTL time.Duration) (time.Duration, bool) {
	switch status {
	case http.StatusOK:
		if len(body) == 0 {
			return emptyTTL, true
		}
		return ttl, true
	case http.StatusNotFound, http.StatusNoContent:
		return emptyTTL, true
	default:
		return 0, false
	}
}

func buildETag(body []byte) string {
	sum := sha256.Sum256(body)
	return `"` + fmt.Sprintf("%x", sum[:]) + `"`
}

func serveCachedResponse(c *gin.Context, entry *httpCacheEntry, cacheKey string) {
	etag := entry.ETag
	if etag == "" {
		etag = buildETag(entry.Body)
	}
	status := entry.Status
	if status <= 0 {
		status = http.StatusOK
	}
	cacheTTL := time.Duration(entry.MaxAge) * time.Second
	if cacheTTL <= 0 {
		cacheTTL = defaultRedisCacheTTL
	}

	if matchETag(c.GetHeader("If-None-Match"), etag) {
		writeCacheHeaders(c.Writer, entry, cacheTTL, time.Since(entry.CreatedAt))
		c.Status(http.StatusNotModified)
		c.Abort()
		return
	}

	writeCacheHeaders(c.Writer, entry, cacheTTL, time.Since(entry.CreatedAt))
	c.Status(status)
	if c.Request.Method != http.MethodHead {
		_, writeErr := c.Writer.Write(entry.Body)
		if writeErr != nil {
			log.Warn(c.Request.Context(), "写出 Redis 缓存响应失败", zap.Error(writeErr), zap.String("cache_key", cacheKey))
		}
	}
	c.Abort()
}

func writeCacheHeaders(writer gin.ResponseWriter, entry *httpCacheEntry, ttl time.Duration, age time.Duration) {
	if writer == nil || entry == nil {
		return
	}

	if contentType, ok := entry.Headers["Content-Type"]; ok && contentType != "" {
		writer.Header().Set("Content-Type", contentType)
	}

	if entry.ETag != "" {
		writer.Header().Set("ETag", entry.ETag)
	}

	cacheControl := fmt.Sprintf("public, max-age=%d", int(ttl.Seconds()))
	writer.Header().Set("Cache-Control", cacheControl)

	if age > 0 {
		writer.Header().Set("Age", strconv.Itoa(int(age.Seconds())))
	}
}

func shouldSkipCacheWrite(c *gin.Context) bool {
	if c == nil || c.Writer == nil {
		return true
	}
	return false
}

func matchETag(incoming, expected string) bool {
	if expected == "" || incoming == "" {
		return false
	}
	if strings.TrimSpace(incoming) == "*" {
		return true
	}

	for _, part := range strings.Split(incoming, ",") {
		if normalizeETag(part) == normalizeETag(expected) {
			return true
		}
	}
	return false
}

func normalizeETag(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "W/")
	return trimmed
}
