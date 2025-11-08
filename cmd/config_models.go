package cmd

import (
	"fmt"

	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/interactive"
	"github.com/OlaHulleberg/clauderock/internal/profiles"
	"github.com/spf13/cobra"
)

var configModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Interactively reconfigure models for the current profile",
	Long: `Interactively reconfigure the main, fast, and heavy models for the current profile.

This command allows you to update model selections without changing other configuration
settings like AWS profile, region, API keys, or base URL. It works for both Bedrock
and API profile types.

The command will:
  - Detect your current profile type (Bedrock or API)
  - Fetch available models from the appropriate source
  - Present an interactive selector for each model type (main, fast, heavy)
  - Save the updated configuration

Example usage:
  clauderock manage config models`,
	RunE: runConfigModels,
}

func runConfigModels(cmd *cobra.Command, args []string) error {
	// Create profile manager
	mgr, err := profiles.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create profile manager: %w", err)
	}

	// Load current profile configuration
	cfg, err := mgr.GetCurrentConfig(Version)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current profile name
	currentProfile, err := mgr.GetCurrent()
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
	}

	fmt.Printf("Configuring models for profile '%s' (type: %s)\n", currentProfile, cfg.ProfileType)

	// Branch based on profile type
	switch cfg.ProfileType {
	case "bedrock":
		if err := interactive.SelectBedrockModels(cfg); err != nil {
			return err
		}
	case "api":
		if err := interactive.SelectAPIModels(cfg); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported profile type: %s", cfg.ProfileType)
	}

	// Update version to current CLI version (but not for dev builds)
	if Version != "dev" {
		cfg.Version = Version
	}

	// Save updated configuration
	if err := mgr.Save(currentProfile, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Display success message with updated models
	fmt.Printf("\n✓ Configuration saved successfully to profile '%s'!\n", currentProfile)
	fmt.Printf("\nUpdated models:\n")

	// Display friendly model names for better readability
	if cfg.ProfileType == "bedrock" {
		fmt.Printf("  Main Model:  %s\n", aws.ExtractFriendlyModelName(cfg.Model))
		fmt.Printf("  Fast Model:  %s\n", aws.ExtractFriendlyModelName(cfg.FastModel))
		fmt.Printf("  Heavy Model: %s\n", aws.ExtractFriendlyModelName(cfg.HeavyModel))
	} else {
		fmt.Printf("  Main Model:  %s\n", cfg.Model)
		fmt.Printf("  Fast Model:  %s\n", cfg.FastModel)
		fmt.Printf("  Heavy Model: %s\n", cfg.HeavyModel)
	}

	return nil
}
