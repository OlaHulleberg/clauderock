package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Profile     string `json:"profile"`
	Region      string `json:"region"`
	CrossRegion string `json:"cross-region"`
	Model       string `json:"model"`
	FastModel   string `json:"fast-model"`
}

var validCrossRegions = map[string]bool{
	"us":     true,
	"eu":     true,
	"global": true,
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".clauderock", "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	// Create default config if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := &Config{
			Profile:     "default",
			Region:      "us-east-1",
			CrossRegion: "global",
			Model:       "claude-sonnet-4-5",
			FastModel:   "claude-haiku-4-5",
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
	if c.Profile == "" {
		return fmt.Errorf("profile is required")
	}
	if c.Region == "" {
		return fmt.Errorf("region is required")
	}
	if c.CrossRegion == "" {
		return fmt.Errorf("cross-region is required")
	}
	if !validCrossRegions[c.CrossRegion] {
		return fmt.Errorf("invalid cross-region: %s (must be one of: us, eu, global)", c.CrossRegion)
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.FastModel == "" {
		return fmt.Errorf("fast-model is required")
	}
	return nil
}

func (c *Config) Set(key, value string) error {
	switch key {
	case "profile":
		c.Profile = value
	case "region":
		c.Region = value
	case "cross-region":
		if !validCrossRegions[value] {
			return fmt.Errorf("invalid cross-region: %s (must be one of: us, eu, global)", value)
		}
		c.CrossRegion = value
	case "model":
		c.Model = value
	case "fast-model":
		c.FastModel = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

func (c *Config) Get(key string) (string, error) {
	switch key {
	case "profile":
		return c.Profile, nil
	case "region":
		return c.Region, nil
	case "cross-region":
		return c.CrossRegion, nil
	case "model":
		return c.Model, nil
	case "fast-model":
		return c.FastModel, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}
