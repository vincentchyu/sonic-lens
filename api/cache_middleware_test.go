package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeHTTPCacheStore struct {
	mu       sync.Mutex
	getEntry *httpCacheEntry
	getErr   error
	setCalls []fakeHTTPCacheSetCall
}

type fakeHTTPCacheSetCall struct {
	key string
	ttl time.Duration
}

func (s *fakeHTTPCacheStore) Get(ctx context.Context, key string) (*httpCacheEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getEntry == nil {
		return nil, nil
	}

	cloned := *s.getEntry
	cloned.Body = append([]byte(nil), s.getEntry.Body...)
	if s.getEntry.Headers != nil {
		cloned.Headers = make(map[string]string, len(s.getEntry.Headers))
		for k, v := range s.getEntry.Headers {
			cloned.Headers[k] = v
		}
	}
	return &cloned, nil
}

func (s *fakeHTTPCacheStore) Set(ctx context.Context, key string, entry *httpCacheEntry, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.setCalls = append(s.setCalls, fakeHTTPCacheSetCall{key: key, ttl: ttl})
	cloned := *entry
	cloned.Body = append([]byte(nil), entry.Body...)
	if entry.Headers != nil {
		cloned.Headers = make(map[string]string, len(entry.Headers))
		for k, v := range entry.Headers {
			cloned.Headers[k] = v
		}
	}
	s.getEntry = &cloned
	return nil
}

func TestRedisCacheMiddlewareCachesResponseWithDefaultTTL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeHTTPCacheStore{}
	engine := gin.New()
	engine.GET("/cacheable", newRedisCacheMiddleware(withRedisCacheStore(store)), func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cacheable?foo=bar", nil)
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if got := recorder.Header().Get("ETag"); got == "" {
		t.Fatalf("expected etag header")
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Fatalf("unexpected cache-control: %q", got)
	}
	if len(store.setCalls) != 1 {
		t.Fatalf("expected one cache write, got %d", len(store.setCalls))
	}
	if store.setCalls[0].ttl != defaultRedisCacheTTL {
		t.Fatalf("unexpected ttl: %s", store.setCalls[0].ttl)
	}
	if store.getEntry == nil || store.getEntry.MaxAge != int(defaultRedisCacheTTL.Seconds()) {
		t.Fatalf("unexpected cached max-age: %+v", store.getEntry)
	}
}

func TestRedisCacheMiddlewareCachesEmptyResponseWithShortTTL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeHTTPCacheStore{}
	engine := gin.New()
	engine.GET("/empty", newRedisCacheMiddleware(withRedisCacheStore(store)), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/empty", nil)
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=3" {
		t.Fatalf("unexpected cache-control: %q", got)
	}
	if len(store.setCalls) != 1 {
		t.Fatalf("expected one cache write, got %d", len(store.setCalls))
	}
	if store.setCalls[0].ttl != defaultRedisEmptyCacheTTL {
		t.Fatalf("unexpected ttl: %s", store.setCalls[0].ttl)
	}
	if store.getEntry == nil || store.getEntry.MaxAge != int(defaultRedisEmptyCacheTTL.Seconds()) {
		t.Fatalf("unexpected cached max-age: %+v", store.getEntry)
	}
	if len(store.getEntry.Body) != 0 {
		t.Fatalf("expected empty cached body, got %q", string(store.getEntry.Body))
	}
}

func TestRedisCacheMiddlewareServesCachedResponseAndHonorsETag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"hello":"world"}`)
	entry := &httpCacheEntry{
		Status:    http.StatusOK,
		Headers:   map[string]string{"Content-Type": "application/json; charset=utf-8"},
		Body:      body,
		ETag:      buildETag(body),
		MaxAge:    12,
		CreatedAt: time.Now().Add(-2 * time.Second),
	}
	store := &fakeHTTPCacheStore{getEntry: entry}
	calls := 0

	engine := gin.New()
	engine.GET("/cacheable", newRedisCacheMiddleware(withRedisCacheStore(store)), func(c *gin.Context) {
		calls++
		c.JSON(http.StatusOK, gin.H{"should": "not-run"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cacheable", nil)
	engine.ServeHTTP(recorder, req)

	if calls != 0 {
		t.Fatalf("expected cached response to skip handler, got %d calls", calls)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if got := recorder.Body.String(); got != string(body) {
		t.Fatalf("unexpected body: %q", got)
	}
	if got := recorder.Header().Get("ETag"); got != entry.ETag {
		t.Fatalf("unexpected etag: %q", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=12" {
		t.Fatalf("unexpected cache-control: %q", got)
	}

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/cacheable", nil)
	req.Header.Set("If-None-Match", entry.ETag)
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotModified {
		t.Fatalf("unexpected status for conditional request: %d", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty body for 304, got %q", recorder.Body.String())
	}
	if got := recorder.Header().Get("ETag"); got != entry.ETag {
		t.Fatalf("unexpected etag on 304: %q", got)
	}
}
