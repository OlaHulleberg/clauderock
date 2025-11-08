package profiles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/keyring"
	"github.com/OlaHulleberg/clauderock/internal/migrations"
)

type Manager struct {
	profilesDir     string
	currentFilePath string
}

func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(home, ".clauderock")
	profilesDir := filepath.Join(baseDir, "profiles")
	currentFilePath := filepath.Join(baseDir, "current-profile.txt")

	return &Manager{
		profilesDir:     profilesDir,
		currentFilePath: currentFilePath,
	}, nil
}

// List returns all available profile names
func (m *Manager) List() ([]string, error) {
	if err := m.ensureProfilesDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(m.profilesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles directory: %w", err)
	}

	var profiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			profiles = append(profiles, name)
		}
	}

	return profiles, nil
}

// Load loads a specific profile by name
func (m *Manager) Load(name string) (*config.Config, error) {
	if err := m.ensureProfilesDir(); err != nil {
		return nil, err
	}

	path := m.profilePath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile '%s' does not exist", name)
		}
		return nil, fmt.Errorf("failed to read profile: %w", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	return &cfg, nil
}

// Save saves a configuration as a named profile
func (m *Manager) Save(name string, cfg *config.Config) error {
	if err := m.ensureProfilesDir(); err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return m.saveWithoutValidation(name, cfg)
}

// saveWithoutValidation saves a config without validation (used internally)
func (m *Manager) saveWithoutValidation(name string, cfg *config.Config) error {
	if err := m.ensureProfilesDir(); err != nil {
		return err
	}

	path := m.profilePath(name)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	return nil
}

// Delete removes a profile and its associated keyring entry (if API profile)
func (m *Manager) Delete(name string) error {
	if name == "default" {
		return fmt.Errorf("cannot delete default profile")
	}

	current, _ := m.GetCurrent()
	if current == name {
		return fmt.Errorf("cannot delete active profile, switch to another profile first")
	}

	// Load profile to check if it has a keyring entry
	cfg, err := m.Load(name)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load profile for cleanup: %w", err)
	}

	// If it's an API profile, delete the keyring entry
	if cfg != nil && cfg.ProfileType == "api" && cfg.APIKeyID != "" {
		if err := keyring.Delete(cfg.APIKeyID); err != nil {
			// Log warning but don't fail deletion
			fmt.Printf("Warning: failed to delete keyring entry: %v\n", err)
		}
	}

	path := m.profilePath(name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("profile '%s' does not exist", name)
		}
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	return nil
}

// Exists checks if a profile exists
func (m *Manager) Exists(name string) bool {
	path := m.profilePath(name)
	_, err := os.Stat(path)
	return err == nil
}

// GetCurrent returns the name of the current active profile
func (m *Manager) GetCurrent() (string, error) {
	data, err := os.ReadFile(m.currentFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Default to "default" if no current profile is set
			return "default", nil
		}
		return "", fmt.Errorf("failed to read current profile: %w", err)
	}

	name := strings.TrimSpace(string(data))
	if name == "" {
		return "default", nil
	}

	return name, nil
}

// SetCurrent sets the current active profile
func (m *Manager) SetCurrent(name string) error {
	if !m.Exists(name) {
		return fmt.Errorf("profile '%s' does not exist", name)
	}

	if err := m.ensureBaseDir(); err != nil {
		return err
	}

	if err := os.WriteFile(m.currentFilePath, []byte(name), 0644); err != nil {
		return fmt.Errorf("failed to set current profile: %w", err)
	}

	return nil
}

// GetCurrentConfig loads the current active profile's configuration
func (m *Manager) GetCurrentConfig(cliVersion string) (*config.Config, error) {
	// Check for migration from legacy config.json first
	if err := m.MigrateFromLegacyConfig(cliVersion); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	current, err := m.GetCurrent()
	if err != nil {
		return nil, err
	}

	// If current profile doesn't exist, create default with current CLI version
	if !m.Exists(current) {
		cfg := m.createDefaultConfig(cliVersion)
		// Save without validation since it's an incomplete fresh install
		if err := m.saveWithoutValidation(current, cfg); err != nil {
			return nil, fmt.Errorf("failed to create default profile: %w", err)
		}
		if err := m.SetCurrent(current); err != nil {
			return nil, fmt.Errorf("failed to set current profile: %w", err)
		}
		return cfg, nil
	}

	cfg, err := m.Load(current)
	if err != nil {
		return nil, err
	}

	// Run migrations only if config version is older than CLI version
	migMgr := migrations.NewManager(cliVersion)
	needsMigration, err := migMgr.NeedsMigration(cfg.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to check migration status: %w", err)
	}

	if needsMigration {
		oldVersion := cfg.Version
		if err := migMgr.MigrateProfile(current, oldVersion, cfg, m); err != nil {
			return nil, fmt.Errorf("failed to migrate profile from %s to %s: %w\nPlease run: clauderock manage config", oldVersion, cliVersion, err)
		}
		// Update config version to current CLI version (but never "dev")
		if cliVersion != "dev" {
			cfg.Version = cliVersion
			if err := m.Save(current, cfg); err != nil {
				return nil, fmt.Errorf("failed to save migrated config: %w", err)
			}
		}
	}

	return cfg, nil
}

// Rename renames a profile
func (m *Manager) Rename(oldName, newName string) error {
	if oldName == "default" {
		return fmt.Errorf("cannot rename default profile")
	}

	if !m.Exists(oldName) {
		return fmt.Errorf("profile '%s' does not exist", oldName)
	}

	if m.Exists(newName) {
		return fmt.Errorf("profile '%s' already exists", newName)
	}

	oldPath := m.profilePath(oldName)
	newPath := m.profilePath(newName)

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename profile: %w", err)
	}

	// Update current profile if it was the renamed one
	current, _ := m.GetCurrent()
	if current == oldName {
		if err := m.SetCurrent(newName); err != nil {
			return fmt.Errorf("failed to update current profile: %w", err)
		}
	}

	return nil
}

// Copy creates a copy of a profile with a new name, including keychain entry for API profiles
func (m *Manager) Copy(sourceName, destName string) error {
	if !m.Exists(sourceName) {
		return fmt.Errorf("profile '%s' does not exist", sourceName)
	}

	if m.Exists(destName) {
		return fmt.Errorf("profile '%s' already exists", destName)
	}

	cfg, err := m.Load(sourceName)
	if err != nil {
		return err
	}

	// If it's an API profile, duplicate the keyring entry with a new ID
	if cfg.ProfileType == "api" && cfg.APIKeyID != "" {
		// Get the API key from keyring
		apiKey, err := keyring.Get(cfg.APIKeyID)
		if err != nil {
			return fmt.Errorf("failed to retrieve API key from keyring: %w", err)
		}

		// Generate new ID for the copy
		newID, err := keyring.GenerateID()
		if err != nil {
			return fmt.Errorf("failed to generate new keyring ID: %w", err)
		}

		// Store with new ID
		if err := keyring.Store(newID, apiKey); err != nil {
			return fmt.Errorf("failed to store API key in keyring: %w", err)
		}

		// Update config with new ID
		cfg.APIKeyID = newID
	}

	return m.Save(destName, cfg)
}

// MigrateFromLegacyConfig migrates old config.json to profiles/default.json
func (m *Manager) MigrateFromLegacyConfig(version string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	legacyPath := filepath.Join(home, ".clauderock", "config.json")

	// Check if legacy config exists
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return nil // No migration needed
	}

	// Check if profiles directory exists
	if err := m.ensureProfilesDir(); err != nil {
		return err
	}

	// Check if default profile already exists (migration already done)
	if m.Exists("default") {
		return nil
	}

	// Load legacy config
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return fmt.Errorf("failed to read legacy config: %w", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse legacy config: %w", err)
	}

	// Run migration on config if needed (for version upgrades)
	// This is handled internally by config, we just need to save it

	// Save as default profile
	if err := m.Save("default", &cfg); err != nil {
		return fmt.Errorf("failed to save default profile: %w", err)
	}

	// Set as current profile
	if err := m.SetCurrent("default"); err != nil {
		return fmt.Errorf("failed to set current profile: %w", err)
	}

	// Rename legacy config to .bak
	bakPath := legacyPath + ".bak"
	if err := os.Rename(legacyPath, bakPath); err != nil {
		// Don't fail if we can't rename, migration is done
		fmt.Printf("Warning: could not rename legacy config to .bak: %v\n", err)
	}

	fmt.Println("Migrated configuration from config.json to profiles/default.json")

	return nil
}

// Helper functions

func (m *Manager) ensureBaseDir() error {
	baseDir := filepath.Dir(m.profilesDir)
	return os.MkdirAll(baseDir, 0755)
}

func (m *Manager) ensureProfilesDir() error {
	return os.MkdirAll(m.profilesDir, 0755)
}

func (m *Manager) profilePath(name string) string {
	return filepath.Join(m.profilesDir, name+".json")
}

func (m *Manager) createDefaultConfig(cliVersion string) *config.Config {
	// Never store "dev" as version - leave empty for dev builds
	cfgVersion := ""
	if cliVersion != "dev" {
		cfgVersion = cliVersion
	}

	cfg := &config.Config{
		Version:     cfgVersion, // Store CLI version (e.g., "v0.6.1") or empty for dev
		ProfileType: "bedrock",  // Default to bedrock for backward compatibility
		Profile:     "",         // Empty on fresh install - user must configure
		Region:      "us-east-1",
		CrossRegion: "global",
		Model:       "",        // Empty on fresh install - user must configure
		FastModel:   "",        // Empty on fresh install - user must configure
		HeavyModel:  "",        // Empty on fresh install - user must configure
	}

	// Note: Models are empty on fresh install to avoid migration errors
	// User should configure AWS profile and models using: clauderock manage config
	return cfg
}
