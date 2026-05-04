package jwt

import (
	"context"
	"sync"
	"time"
)

// InMemoryRevocationStore is a non-persistent store for use in tests and local dev.
type InMemoryRevocationStore struct {
	mu      sync.RWMutex
	revoked map[string]struct{}
}

func NewInMemoryRevocationStore() *InMemoryRevocationStore {
	return &InMemoryRevocationStore{revoked: make(map[string]struct{})}
}

func (s *InMemoryRevocationStore) Revoke(_ context.Context, jti string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked[jti] = struct{}{}
	return nil
}

func (s *InMemoryRevocationStore) IsRevoked(_ context.Context, jti string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.revoked[jti]
	return ok, nil
}
