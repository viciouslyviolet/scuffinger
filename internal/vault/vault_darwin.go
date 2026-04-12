//go:build darwin

package vault

import (
	"fmt"
	"os/exec"
	"strings"
)

// darwinStore uses the macOS "security" CLI to interact with the system Keychain.
type darwinStore struct{}

func newPlatformStore() Store { return &darwinStore{} }

func (s *darwinStore) Set(key, value string) error {
	// Delete first to avoid "duplicate item" errors on update.
	_ = s.Delete(key)

	cmd := exec.Command("security", "add-generic-password",
		"-a", key,
		"-s", serviceName,
		"-w", value,
		"-U", // update if exists
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("keychain set %q: %w — %s", key, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *darwinStore) Get(key string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-a", key,
		"-s", serviceName,
		"-w", // print password only
	)
	out, err := cmd.Output()
	if err != nil {
		// "security" exits non-zero when the item is not found.
		return "", ErrNotFound
	}
	return strings.TrimSpace(string(out)), nil
}

func (s *darwinStore) Delete(key string) error {
	cmd := exec.Command("security", "delete-generic-password",
		"-a", key,
		"-s", serviceName,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), "could not be found") {
			return ErrNotFound
		}
		return fmt.Errorf("keychain delete %q: %w — %s", key, err, strings.TrimSpace(string(out)))
	}
	return nil
}
