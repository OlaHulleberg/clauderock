package migrations

import (
	"fmt"

	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/config"
)

// ProfileSaver defines the interface for saving profiles
type ProfileSaver interface {
	Save(name string, cfg *config.Config) error
}

// Manager handles all configuration and profile migrations
type Manager struct {
	cliVersion string // Current CLI version (e.g., "v0.6.1")
}

// NewManager creates a new migration manager
func NewManager(cliVersion string) *Manager {
	return &Manager{cliVersion: cliVersion}
}

// NeedsMigration checks if config needs migration based on version comparison
func (m *Manager) NeedsMigration(configVersion string) (bool, error) {
	// Dev builds always skip migration
	if m.cliVersion == "dev" {
		return false, nil
	}

	// Empty config version with current CLI = fresh install, no migration needed
	// Empty config version with old CLI = very old config, needs migration (but shouldn't happen)
	if configVersion == "" {
		return false, nil // Fresh install, no migration
	}

	// Compare versions
	cmp := config.CompareVersions(configVersion, m.cliVersion)
	return cmp < 0, nil // Needs migration if config version < CLI version
}

// MigrateProfile runs all necessary migrations from oldVersion to current CLI version
func (m *Manager) MigrateProfile(profileName, oldVersion string, cfg *config.Config, saver ProfileSaver) error {
	// Dev builds skip migration
	if m.cliVersion == "dev" {
		return nil
	}

	// Determine which migrations need to run based on version comparison
	// v0.6.0 must run first to set ProfileType
	if m.shouldRunMigration(oldVersion, "v0.6.0") {
		if err := m.migrateToV060(profileName, cfg, saver); err != nil {
			return fmt.Errorf("failed to migrate to v0.6.0: %w", err)
		}
	}

	// Skip Bedrock-specific migrations for API profiles
	if cfg.ProfileType != "api" {
		if m.shouldRunMigration(oldVersion, "v0.4.0") {
			if err := m.migrateToV040(profileName, cfg, saver); err != nil {
				return fmt.Errorf("failed to migrate to v0.4.0: %w", err)
			}
		}

		if m.shouldRunMigration(oldVersion, "v0.5.0") {
			if err := m.migrateToV050(profileName, cfg, saver); err != nil {
				return fmt.Errorf("failed to migrate to v0.5.0: %w", err)
			}
		}
	}

	return nil
}

// shouldRunMigration determines if a migration should run based on version comparison
// Returns true if oldVersion < targetVersion (migration is needed)
func (m *Manager) shouldRunMigration(oldVersion, targetVersion string) bool {
	// Empty old version means fresh install or very old config - run migration
	if oldVersion == "" {
		return true
	}

	// Check if old version is less than target version
	return config.CompareVersions(oldVersion, targetVersion) < 0
}

// migrateToV040 migrates model names from friendly format to full profile IDs
// Assumes migration manager has already determined this should run
func (m *Manager) migrateToV040(profileName string, cfg *config.Config, saver ProfileSaver) error {
	// Skip migration if models are empty (fresh install or not yet configured)
	if cfg.Model == "" && cfg.FastModel == "" {
		return nil
	}

	// Check if models are already full profile IDs
	modelIsFullID := cfg.Model == "" || aws.IsFullProfileID(cfg.Model)
	fastModelIsFullID := cfg.FastModel == "" || aws.IsFullProfileID(cfg.FastModel)

	// If both are already full IDs or empty, no migration needed
	if modelIsFullID && fastModelIsFullID {
		return nil
	}

	fmt.Println("Upgrading config to cache model profile IDs...")

	// Resolve models to full profile IDs (skip empty ones)
	if cfg.Model != "" && !modelIsFullID {
		fullID, err := aws.ResolveModelToProfileID(cfg.Profile, cfg.Region, cfg.CrossRegion, cfg.Model)
		if err != nil {
			return fmt.Errorf("failed to resolve main model: %w", err)
		}
		cfg.Model = fullID
	}

	if cfg.FastModel != "" && !fastModelIsFullID {
		fullID, err := aws.ResolveModelToProfileID(cfg.Profile, cfg.Region, cfg.CrossRegion, cfg.FastModel)
		if err != nil {
			return fmt.Errorf("failed to resolve fast model: %w", err)
		}
		cfg.FastModel = fullID
	}

	// Save updated config
	if err := saver.Save(profileName, cfg); err != nil {
		return fmt.Errorf("failed to save migrated config: %w", err)
	}

	fmt.Printf("✓ Cached model profile IDs for faster startup\n")
	return nil
}

// migrateToV050 adds heavy model field if missing
// Assumes migration manager has already determined this should run
func (m *Manager) migrateToV050(profileName string, cfg *config.Config, saver ProfileSaver) error {
	// If HeavyModel is already set, no migration needed
	if cfg.HeavyModel != "" {
		return nil
	}

	// Skip migration if main model is empty (fresh install or not yet configured)
	if cfg.Model == "" {
		return nil
	}

	fmt.Println("Upgrading config to add heavy model support...")

	// Set heavy model to the same as default model (user can change later)
	cfg.HeavyModel = cfg.Model

	// Save updated config
	if err := saver.Save(profileName, cfg); err != nil {
		return fmt.Errorf("failed to save migrated config: %w", err)
	}

	fmt.Printf("✓ Added heavy model support (set to default model)\n")
	return nil
}

// migrateToV060 adds ProfileType field if missing
// Assumes migration manager has already determined this should run
func (m *Manager) migrateToV060(profileName string, cfg *config.Config, saver ProfileSaver) error {
	// If ProfileType is already set, no migration needed
	if cfg.ProfileType != "" {
		return nil
	}

	fmt.Println("Upgrading config to add profile type...")

	// Default to bedrock for backward compatibility
	cfg.ProfileType = "bedrock"

	// Save updated config
	if err := saver.Save(profileName, cfg); err != nil {
		return fmt.Errorf("failed to save migrated config: %w", err)
	}

	fmt.Printf("✓ Added profile type support (set to bedrock)\n")
	return nil
}
