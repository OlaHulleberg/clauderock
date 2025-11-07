package keyring

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/99designs/keyring"
)

const (
	serviceName = "clauderock"
)

// GenerateID creates a unique identifier for a keychain entry
func GenerateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// Store saves an API key to the OS keychain with the given ID
func Store(id, apiKey string) error {
	ring, err := openKeyring()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	item := keyring.Item{
		Key:  id,
		Data: []byte(apiKey),
	}

	if err := ring.Set(item); err != nil {
		return fmt.Errorf("failed to store API key: %w", err)
	}

	return nil
}

// Get retrieves an API key from the OS keychain by ID
func Get(id string) (string, error) {
	ring, err := openKeyring()
	if err != nil {
		return "", fmt.Errorf("failed to open keyring: %w", err)
	}

	item, err := ring.Get(id)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve API key: %w", err)
	}

	return string(item.Data), nil
}

// Delete removes an API key from the OS keychain by ID
func Delete(id string) error {
	ring, err := openKeyring()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	if err := ring.Remove(id); err != nil {
		// Don't return error if key doesn't exist
		if err == keyring.ErrKeyNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}

// openKeyring opens the OS keyring with appropriate backends
func openKeyring() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName: serviceName,
		// Allow multiple backends for cross-platform support
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,     // macOS
			keyring.SecretServiceBackend, // Linux
			keyring.WinCredBackend,       // Windows
			keyring.FileBackend,          // Fallback (encrypted file)
		},
	})
}
