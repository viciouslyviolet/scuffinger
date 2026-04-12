//go:build !darwin && !windows && !linux

package vault

import "sync"

// memoryStore is used on platforms without a native secrets backend.
// Secrets are stored in memory and lost when the process exits.
type memoryStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func newPlatformStore() Store {
	return &memoryStore{data: make(map[string]string)}
}

func (s *memoryStore) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *memoryStore) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (s *memoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; !ok {
		return ErrNotFound
	}
	delete(s.data, key)
	return nil
}
