package artwork

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const maxEntries = 256

type Entry struct {
	Data      []byte
	MimeType  string
	CreatedAt time.Time
}

type Store struct {
	mu      sync.RWMutex
	entries map[string]Entry
	order   []string
}

func NewStore() *Store {
	return &Store{
		entries: make(map[string]Entry),
		order:   make([]string, 0, maxEntries),
	}
}

func (s *Store) GetKeyForSeed(seed string) string {
	// sum := sha1.Sum([]byte(seed + mimeType + string(data)))
	sum := sha1.Sum([]byte(seed))
	key := hex.EncodeToString(sum[:])
	return key
}

func (s *Store) Put(seed string, data []byte, mimeType string) string {
	if len(data) == 0 {
		return ""
	}
	key := s.GetKeyForSeed(seed)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.entries[key]; !exists {
		if len(s.order) >= maxEntries {
			oldest := s.order[0]
			s.order = s.order[1:]
			delete(s.entries, oldest)
		}
		s.order = append(s.order, key)
	}

	s.entries[key] = Entry{
		Data:      data,
		MimeType:  mimeType,
		CreatedAt: time.Now(),
	}

	return key
}

func (s *Store) Get(key string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[key]
	return entry, ok
}

func URLForKey(key string) string {
	if key == "" {
		return ""
	}
	return fmt.Sprintf("/api/artwork/%s", key)
}

var DefaultStore = NewStore()
