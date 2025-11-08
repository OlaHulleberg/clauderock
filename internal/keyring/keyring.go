package keyring

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

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

// Store saves an API key to encrypted file storage with the given ID
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

// Get retrieves an API key from encrypted file storage by ID
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

// Delete removes an API key from encrypted file storage by ID
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

// openKeyring opens the file-based keyring with machine-specific encryption
func openKeyring() (keyring.Keyring, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	fileDir := filepath.Join(home, ".clauderock", "keyring")

	return keyring.Open(keyring.Config{
		ServiceName: serviceName,
		FileDir:     fileDir,
		FilePasswordFunc: func(prompt string) (string, error) {
			// Derive password from machine-specific data
			// This prevents keyring file from being portable across machines
			hostname, _ := os.Hostname()
			username := os.Getenv("USER")
			if username == "" {
				username = os.Getenv("USERNAME") // Windows
			}
			return fmt.Sprintf("clauderock-%s-%s", hostname, username), nil
		},
		// Only use file backend (pure Go, no CGO)
		AllowedBackends: []keyring.BackendType{
			keyring.FileBackend,
		},
	})
}
