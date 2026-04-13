// Package vault provides a cross-platform secrets storage abstraction.
//
// Backend selection:
//   - macOS:   uses the "security" CLI to interact with the system Keychain
//   - Windows: uses PowerShell to interact with the Windows Credential Manager
//   - Linux:   stores secrets as plain files under ~/.scuffinger/
//   - Other:   in-memory only (lost on process exit)
package vault

import "errors"

// ErrNotFound is returned when a secret does not exist.
var ErrNotFound = errors.New("secret not found")

const serviceName = "scuffinger" //nolint:unused // used in platform-specific files (vault_darwin.go, vault_windows.go)

// Store is the interface every platform backend implements.
type Store interface {
	// Set stores a secret identified by key.
	Set(key, value string) error
	// Get retrieves a secret by key. Returns ErrNotFound if absent.
	Get(key string) (string, error)
	// Delete removes a secret by key. Returns ErrNotFound if absent.
	Delete(key string) error
}

// New returns the best available Store for the current platform.
// See vault_darwin.go, vault_windows.go, vault_linux.go, vault_other.go.
func New() Store {
	return newPlatformStore()
}
