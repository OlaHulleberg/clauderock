package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Version     string `json:"version"`
	ProfileType string `json:"profile-type"` // "bedrock" or "api"

	// Bedrock-specific fields (only used when ProfileType == "bedrock")
	Profile     string `json:"profile,omitempty"`
	Region      string `json:"region,omitempty"`
	CrossRegion string `json:"cross-region,omitempty"`

	// API-specific fields (only used when ProfileType == "api")
	BaseURL  string `json:"base-url,omitempty"`
	APIKeyID string `json:"api-key-id,omitempty"` // Reference to encrypted keyring entry

	// Model fields (used by both types)
	Model      string `json:"model"`
	FastModel  string `json:"fast-model"`
	HeavyModel string `json:"heavy-model"`
}

var validCrossRegions = map[string]bool{
	"us":     true,
	"eu":     true,
	"global": true,
}

// compareVersions compares two semantic version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
// Handles versions like "0.1.0", "0.2.0", "dev", etc.
func compareVersions(v1, v2 string) int {
	// Handle special cases
	if v1 == v2 {
		return 0
	}
	if v1 == "dev" {
		return 1 // dev is always considered newer
	}
	if v2 == "dev" {
		return -1
	}
	if v1 == "" || v1 == "0" {
		v1 = "0.0.0"
	}
	if v2 == "" || v2 == "0" {
		v2 = "0.0.0"
	}

	// Split versions into parts
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Pad to ensure same length
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}
	for len(parts1) < maxLen {
		parts1 = append(parts1, "0")
	}
	for len(parts2) < maxLen {
		parts2 = append(parts2, "0")
	}

	// Compare each part
	for i := 0; i < maxLen; i++ {
		num1, err1 := strconv.Atoi(parts1[i])
		num2, err2 := strconv.Atoi(parts2[i])

		// If either part is not a number, fall back to string comparison
		if err1 != nil || err2 != nil {
			if parts1[i] < parts2[i] {
				return -1
			}
			if parts1[i] > parts2[i] {
				return 1
			}
			continue
		}

		if num1 < num2 {
			return -1
		}
		if num1 > num2 {
			return 1
		}
	}

	return 0
}

// migrate runs all necessary migrations based on config version
func (c *Config) migrate(currentVersion string) bool {
	// Never run migrations in dev mode
	if currentVersion == "dev" {
		return false
	}

	migrated := false

	// Migration for v0.2.0: Add provider prefix to model names
	if compareVersions(c.Version, "0.2.0") < 0 {
		c.migrateToV020()
		migrated = true
	}

	// Migration for v0.6.0: Add profile type field
	if compareVersions(c.Version, "0.6.0") < 0 {
		c.migrateToV060()
		migrated = true
	}

	// Update version to current binary version
	if migrated {
		c.Version = currentVersion
	}

	return migrated
}

// migrateToV020 migrates model format from "claude-sonnet-4-5" to "anthropic.claude-sonnet-4-5"
func (c *Config) migrateToV020() {
	c.Model = migrateModelFormat(c.Model)
	c.FastModel = migrateModelFormat(c.FastModel)
}

// migrateToV060 sets ProfileType to "bedrock" for existing configs
func (c *Config) migrateToV060() {
	// If ProfileType is already set, don't override
	if c.ProfileType != "" {
		return
	}

	// Default to bedrock for backward compatibility
	c.ProfileType = "bedrock"
}

// migrateModelFormat adds provider prefix to model name if missing
func migrateModelFormat(model string) string {
	// If already has provider prefix, return as-is
	if strings.Contains(model, ".") {
		return model
	}

	// Map model prefixes to providers
	modelPrefixToProvider := map[string]string{
		"claude":  "anthropic",
		"llama":   "meta",
		"titan":   "amazon",
		"j2":      "ai21",
		"command": "cohere",
		"mistral": "mistral",
		"jamba":   "ai21",
	}

	// Find matching provider
	for prefix, provider := range modelPrefixToProvider {
		if strings.HasPrefix(model, prefix) {
			return fmt.Sprintf("%s.%s", provider, model)
		}
	}

	// Default: assume anthropic for unknown models (most common case)
	return fmt.Sprintf("anthropic.%s", model)
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".clauderock", "config.json"), nil
}

func Load(currentVersion string) (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	// Create default config if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// In dev mode, don't set version field
		version := currentVersion
		if currentVersion == "dev" {
			version = ""
		}

		cfg := &Config{
			Version:     version,
			ProfileType: "bedrock", // Default to bedrock for backward compatibility
			Profile:     "default",
			Region:      "us-east-1",
			CrossRegion: "global",
			Model:       "anthropic.claude-sonnet-4-5",
			FastModel:   "anthropic.claude-haiku-4-5",
			HeavyModel:  "anthropic.claude-opus-4-1",
		}
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Run migrations if needed (skips automatically in dev mode)
	if cfg.migrate(currentVersion) {
		// Save migrated config
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("failed to save migrated config: %w", err)
		}
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Config) Validate() error {
	// Validate ProfileType
	if c.ProfileType != "bedrock" && c.ProfileType != "api" {
		return fmt.Errorf("profile-type must be either 'bedrock' or 'api'")
	}

	// Validate based on profile type
	if c.ProfileType == "bedrock" {
		if c.Profile == "" {
			return fmt.Errorf("profile is required for bedrock profile type")
		}
		if c.Region == "" {
			return fmt.Errorf("region is required for bedrock profile type")
		}
		if c.CrossRegion == "" {
			return fmt.Errorf("cross-region is required for bedrock profile type")
		}
		if !validCrossRegions[c.CrossRegion] {
			return fmt.Errorf("invalid cross-region: %s (must be one of: us, eu, global)", c.CrossRegion)
		}
	} else if c.ProfileType == "api" {
		if c.BaseURL == "" {
			return fmt.Errorf("base-url is required for api profile type")
		}
		if c.APIKeyID == "" {
			return fmt.Errorf("api-key-id is required for api profile type")
		}
	}

	// Models are required for both profile types
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.FastModel == "" {
		return fmt.Errorf("fast-model is required")
	}
	if c.HeavyModel == "" {
		return fmt.Errorf("heavy-model is required")
	}

	return nil
}

func (c *Config) Set(key, value string) error {
	switch key {
	case "profile-type":
		if value != "bedrock" && value != "api" {
			return fmt.Errorf("profile-type must be either 'bedrock' or 'api'")
		}
		c.ProfileType = value
	case "profile":
		c.Profile = value
	case "region":
		c.Region = value
	case "cross-region":
		if !validCrossRegions[value] {
			return fmt.Errorf("invalid cross-region: %s (must be one of: us, eu, global)", value)
		}
		c.CrossRegion = value
	case "base-url":
		c.BaseURL = value
	case "api-key-id":
		c.APIKeyID = value
	case "model":
		c.Model = value
	case "fast-model":
		c.FastModel = value
	case "heavy-model":
		c.HeavyModel = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

func (c *Config) Get(key string) (string, error) {
	switch key {
	case "profile-type":
		return c.ProfileType, nil
	case "profile":
		return c.Profile, nil
	case "region":
		return c.Region, nil
	case "cross-region":
		return c.CrossRegion, nil
	case "base-url":
		return c.BaseURL, nil
	case "api-key-id":
		return c.APIKeyID, nil
	case "model":
		return c.Model, nil
	case "fast-model":
		return c.FastModel, nil
	case "heavy-model":
		return c.HeavyModel, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}
