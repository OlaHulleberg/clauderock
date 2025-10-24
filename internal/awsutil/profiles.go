package awsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

// GetProfiles returns a list of available AWS profiles from ~/.aws/config and ~/.aws/credentials
func GetProfiles() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	profileMap := make(map[string]bool)

	// Parse credentials file
	credentialsPath := filepath.Join(home, ".aws", "credentials")
	if _, err := os.Stat(credentialsPath); err == nil {
		cfg, err := ini.Load(credentialsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse credentials file: %w", err)
		}

		for _, section := range cfg.Sections() {
			name := section.Name()
			if name != "DEFAULT" && name != "" {
				profileMap[name] = true
			}
		}
	}

	// Parse config file
	configPath := filepath.Join(home, ".aws", "config")
	if _, err := os.Stat(configPath); err == nil {
		cfg, err := ini.Load(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}

		for _, section := range cfg.Sections() {
			name := section.Name()
			// Skip DEFAULT, empty sections, and sso-session sections
			if name == "DEFAULT" || name == "" || strings.HasPrefix(name, "sso-session ") {
				continue
			}
			// Only process sections that start with "profile "
			if strings.HasPrefix(name, "profile ") {
				profileName := strings.TrimPrefix(name, "profile ")
				profileMap[profileName] = true
			}
		}
	}

	// If no profiles found, check if files exist
	if len(profileMap) == 0 {
		if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				return nil, fmt.Errorf("no AWS config files found. Please run 'aws configure' to set up your AWS credentials")
			}
		}
	}

	// Convert map to slice
	profiles := make([]string, 0, len(profileMap))
	for profile := range profileMap {
		profiles = append(profiles, profile)
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no AWS profiles found. Please run 'aws configure' to set up your AWS credentials")
	}

	return profiles, nil
}
