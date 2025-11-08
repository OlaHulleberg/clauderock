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
	version string
}

// NewManager creates a new migration manager
func NewManager(version string) *Manager {
	return &Manager{version: version}
}

// ShouldRunMigrations returns true if migrations should run
func (m *Manager) ShouldRunMigrations() bool {
	return m.version != "dev"
}

// MigrateProfile runs all profile-level migrations
func (m *Manager) MigrateProfile(profileName string, cfg *config.Config, saver ProfileSaver) error {
	if !m.ShouldRunMigrations() {
		return nil
	}

	// Run migrations in order
	// v0.6.0 must run first to set ProfileType
	if err := m.migrateToV060(profileName, cfg, saver); err != nil {
		return fmt.Errorf("failed to migrate to v0.6.0: %w", err)
	}

	// Skip Bedrock-specific migrations for API profiles
	if cfg.ProfileType != "api" {
		if err := m.migrateToV040(profileName, cfg, saver); err != nil {
			return fmt.Errorf("failed to migrate to v0.4.0: %w", err)
		}

		if err := m.migrateToV050(profileName, cfg, saver); err != nil {
			return fmt.Errorf("failed to migrate to v0.5.0: %w", err)
		}
	}

	return nil
}

// migrateToV040 migrates model names from friendly format to full profile IDs
func (m *Manager) migrateToV040(profileName string, cfg *config.Config, saver ProfileSaver) error {
	// Check if models are already full profile IDs
	modelIsFullID := aws.IsFullProfileID(cfg.Model)
	fastModelIsFullID := aws.IsFullProfileID(cfg.FastModel)

	// If both are already full IDs, no migration needed
	if modelIsFullID && fastModelIsFullID {
		return nil
	}

	fmt.Println("Upgrading config to cache model profile IDs...")

	// Resolve models to full profile IDs
	if !modelIsFullID {
		fullID, err := aws.ResolveModelToProfileID(cfg.Profile, cfg.Region, cfg.CrossRegion, cfg.Model)
		if err != nil {
			return fmt.Errorf("failed to resolve main model: %w", err)
		}
		cfg.Model = fullID
	}

	if !fastModelIsFullID {
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
func (m *Manager) migrateToV050(profileName string, cfg *config.Config, saver ProfileSaver) error {
	// If HeavyModel is already set, no migration needed
	if cfg.HeavyModel != "" {
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
