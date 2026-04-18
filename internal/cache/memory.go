package cache

import (
	"context"
	"strings"
	"sync"
	"time"
)

type MemoryStore struct {
	mu      sync.RWMutex
	entries map[string]memoryEntry
}

type memoryEntry struct {
	backend   string
	expiresAt time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		entries: make(map[string]memoryEntry),
	}
}

func (s *MemoryStore) PreferredBackend(_ context.Context, key string) (string, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", false, nil
	}

	s.mu.RLock()
	entry, ok := s.entries[key]
	s.mu.RUnlock()
	if !ok {
		return "", false, nil
	}
	if time.Now().After(entry.expiresAt) {
		s.mu.Lock()
		delete(s.entries, key)
		s.mu.Unlock()
		return "", false, nil
	}
	return entry.backend, true, nil
}

func (s *MemoryStore) RememberBackend(_ context.Context, key, backend string, ttl time.Duration) error {
	key = strings.TrimSpace(key)
	backend = strings.TrimSpace(backend)
	if key == "" || backend == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}

	s.mu.Lock()
	s.entries[key] = memoryEntry{
		backend:   backend,
		expiresAt: time.Now().Add(ttl),
	}
	s.mu.Unlock()
	return nil
}
