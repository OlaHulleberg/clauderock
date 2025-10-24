package cmd

import (
	"fmt"

	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/interactive"
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
		return interactive.RunInteractiveConfig(Version)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Valid keys:
  profile      - AWS profile name
  region       - AWS region (e.g., us-east-1)
  cross-region - Cross-region setting (us, eu, global)
  model        - Main model name (e.g., claude-sonnet-4-5)
  fast-model   - Fast model name (e.g., claude-haiku-4-5)`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		cfg, err := config.Load(Version)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.Set(key, value); err != nil {
			return err
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Set %s = %s\n", key, value)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		cfg, err := config.Load(Version)
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
	Short: "List all configuration values",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(Version)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Println("Configuration:")
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
