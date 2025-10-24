package cmd

import (
	"fmt"

	"github.com/OlaHulleberg/clauderock/internal/interactive"
	"github.com/OlaHulleberg/clauderock/internal/profiles"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage clauderock configuration",
	Long: `Manage clauderock configuration.

When run without subcommands, starts an interactive configuration wizard.
You can also use subcommands to set, get, or list configuration values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand specified, run interactive config
		mgr, err := profiles.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create profile manager: %w", err)
		}
		return interactive.RunInteractiveConfig(Version, mgr)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value in the current profile",
	Long: `Set a configuration value in the current profile. Valid keys:
  profile      - AWS profile name
  region       - AWS region (e.g., us-east-1)
  cross-region - Cross-region setting (us, eu, global)
  model        - Main model name (e.g., anthropic.claude-sonnet-4-5)
  fast-model   - Fast model name (e.g., anthropic.claude-haiku-4-5)`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		mgr, err := profiles.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create profile manager: %w", err)
		}

		cfg, err := mgr.GetCurrentConfig(Version)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.Set(key, value); err != nil {
			return err
		}

		current, err := mgr.GetCurrent()
		if err != nil {
			return fmt.Errorf("failed to get current profile: %w", err)
		}

		if err := mgr.Save(current, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Set %s = %s (in profile '%s')\n", key, value, current)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value from the current profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		mgr, err := profiles.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create profile manager: %w", err)
		}

		cfg, err := mgr.GetCurrentConfig(Version)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		value, err := cfg.Get(key)
		if err != nil {
			return err
		}

		fmt.Println(value)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values from the current profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := profiles.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create profile manager: %w", err)
		}

		current, err := mgr.GetCurrent()
		if err != nil {
			return fmt.Errorf("failed to get current profile: %w", err)
		}

		cfg, err := mgr.GetCurrentConfig(Version)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("Configuration (profile: %s):\n", current)
		fmt.Printf("  profile:      %s\n", cfg.Profile)
		fmt.Printf("  region:       %s\n", cfg.Region)
		fmt.Printf("  cross-region: %s\n", cfg.CrossRegion)
		fmt.Printf("  model:        %s\n", cfg.Model)
		fmt.Printf("  fast-model:   %s\n", cfg.FastModel)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
}
