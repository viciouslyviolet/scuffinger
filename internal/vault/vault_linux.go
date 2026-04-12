//go:build linux

package vault

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// linuxStore persists secrets as plain files under ~/.scuffinger/.
type linuxStore struct {
	dir string
}

func newPlatformStore() Store {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".scuffinger")
	return &linuxStore{dir: dir}
}

func (s *linuxStore) path(key string) string {
	return filepath.Join(s.dir, key)
}

func (s *linuxStore) Set(key, value string) error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("create vault dir: %w", err)
	}
	return os.WriteFile(s.path(key), []byte(value), 0600)
}

func (s *linuxStore) Get(key string) (string, error) {
	data, err := os.ReadFile(s.path(key))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNotFound
		}
		return "", err
	}
	return string(data), nil
}

func (s *linuxStore) Delete(key string) error {
	err := os.Remove(s.path(key))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
