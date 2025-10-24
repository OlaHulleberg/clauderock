package cmd

import (
	"fmt"
	"os"

	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/launcher"
	"github.com/OlaHulleberg/clauderock/internal/updater"
	"github.com/spf13/cobra"
)

var (
	profileFlag string
	Version     = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "clauderock",
	Short: "Launch Claude Code with AWS Bedrock configuration",
	Long:  `clauderock configures and launches Claude Code with AWS Bedrock inference profiles.`,
	RunE:  runRoot,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&profileFlag, "profile", "", "Override AWS profile for this run")
}

func runRoot(cmd *cobra.Command, args []string) error {
	// Check for updates in background
	go updater.CheckForUpdates(Version)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override profile if flag is set
	if profileFlag != "" {
		cfg.Profile = profileFlag
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Find inference profile IDs
	mainModelID, fastModelID, err := aws.FindInferenceProfiles(cfg)
	if err != nil {
		return fmt.Errorf("failed to find inference profiles: %w", err)
	}

	fmt.Printf("Using model: %s\n", mainModelID)
	fmt.Printf("Using fast model: %s\n", fastModelID)

	// Launch Claude Code
	return launcher.Launch(cfg, mainModelID, fastModelID)
}
