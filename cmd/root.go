package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/interactive"
	"github.com/OlaHulleberg/clauderock/internal/keyring"
	"github.com/OlaHulleberg/clauderock/internal/launcher"
	"github.com/OlaHulleberg/clauderock/internal/profiles"
	"github.com/OlaHulleberg/clauderock/internal/updater"
	"github.com/spf13/cobra"
)

var (
	clauderockProfileFlag      string
	clauderockProfileTypeFlag  string
	clauderockModelFlag        string
	clauderockFastModelFlag    string
	clauderockHeavyModelFlag   string
	clauderockAWSProfileFlag   string
	clauderockRegionFlag       string
	clauderockCrossRegionFlag  string
	clauderockBaseURLFlag      string
	clauderockAPIKeyFlag       string
	Version                    = "dev"
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
	rootCmd.Flags().StringVar(&clauderockProfileFlag, "clauderock-profile", "", "Use a specific clauderock profile for this run")
	rootCmd.Flags().StringVar(&clauderockProfileTypeFlag, "clauderock-profile-type", "", "Override profile type for this run (bedrock or api)")
	rootCmd.Flags().StringVar(&clauderockModelFlag, "clauderock-model", "", "Override main model for this run")
	rootCmd.Flags().StringVar(&clauderockFastModelFlag, "clauderock-fast-model", "", "Override fast model for this run")
	rootCmd.Flags().StringVar(&clauderockHeavyModelFlag, "clauderock-heavy-model", "", "Override heavy model for this run")
	rootCmd.Flags().StringVar(&clauderockAWSProfileFlag, "clauderock-aws-profile", "", "Override AWS profile for this run (bedrock only)")
	rootCmd.Flags().StringVar(&clauderockRegionFlag, "clauderock-region", "", "Override AWS region for this run (bedrock only)")
	rootCmd.Flags().StringVar(&clauderockCrossRegionFlag, "clauderock-cross-region", "", "Override cross-region setting for this run (bedrock only)")
	rootCmd.Flags().StringVar(&clauderockBaseURLFlag, "clauderock-base-url", "", "Override base URL for this run (api only)")
	rootCmd.Flags().StringVar(&clauderockAPIKeyFlag, "clauderock-api-key", "", "Override API key for this run (api only, ephemeral)")

	// Allow unknown flags to pass through to Claude CLI
	rootCmd.FParseErrWhitelist.UnknownFlags = true
}

func runRoot(cmd *cobra.Command, args []string) error {
	// Collect passthrough args for Claude CLI
	// This includes all non-clauderock flags and positional arguments
	passthroughArgs := collectPassthroughArgs()

	// Check for updates in background
	go updater.CheckForUpdates(Version)

	// Load configuration from profile
	profileMgr, err := profiles.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create profile manager: %w", err)
	}

	var cfg *config.Config
	if clauderockProfileFlag != "" {
		// Load specific profile
		cfg, err = profileMgr.Load(clauderockProfileFlag)
		if err != nil {
			return fmt.Errorf("failed to load profile '%s': %w", clauderockProfileFlag, err)
		}
	} else {
		// Load current profile
		cfg, err = profileMgr.GetCurrentConfig(Version)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// If config is incomplete, launch interactive configurator
	if cfg.IsIncomplete() {
		fmt.Println("Configuration incomplete. Starting interactive setup...")
		if err := interactive.RunInteractiveConfig(Version, profileMgr); err != nil {
			return fmt.Errorf("configuration setup failed: %w", err)
		}
		// Reload config after interactive setup
		cfg, err = profileMgr.GetCurrentConfig(Version)
		if err != nil {
			return fmt.Errorf("failed to reload config after setup: %w", err)
		}
	}

	// Apply overrides from flags
	hasOverrides := false

	// Profile type override
	if clauderockProfileTypeFlag != "" {
		if clauderockProfileTypeFlag != "bedrock" && clauderockProfileTypeFlag != "api" {
			return fmt.Errorf("--clauderock-profile-type must be either 'bedrock' or 'api'")
		}
		cfg.ProfileType = clauderockProfileTypeFlag
		hasOverrides = true
	}

	// Bedrock-specific overrides
	if clauderockAWSProfileFlag != "" {
		if cfg.ProfileType != "bedrock" {
			return fmt.Errorf("--clauderock-aws-profile can only be used with bedrock profile type")
		}
		cfg.Profile = clauderockAWSProfileFlag
		hasOverrides = true
	}
	if clauderockRegionFlag != "" {
		if cfg.ProfileType != "bedrock" {
			return fmt.Errorf("--clauderock-region can only be used with bedrock profile type")
		}
		cfg.Region = clauderockRegionFlag
		hasOverrides = true
	}
	if clauderockCrossRegionFlag != "" {
		if cfg.ProfileType != "bedrock" {
			return fmt.Errorf("--clauderock-cross-region can only be used with bedrock profile type")
		}
		cfg.CrossRegion = clauderockCrossRegionFlag
		hasOverrides = true
	}

	// API-specific overrides
	if clauderockBaseURLFlag != "" {
		if cfg.ProfileType != "api" {
			return fmt.Errorf("--clauderock-base-url can only be used with api profile type")
		}
		cfg.BaseURL = clauderockBaseURLFlag
		hasOverrides = true
	}
	if clauderockAPIKeyFlag != "" {
		if cfg.ProfileType != "api" {
			return fmt.Errorf("--clauderock-api-key can only be used with api profile type")
		}
		// For API key override, create a temporary keyring entry
		tempKeyID, err := keyring.GenerateID()
		if err != nil {
			return fmt.Errorf("failed to generate temporary key ID: %w", err)
		}
		if err := keyring.Store(tempKeyID, clauderockAPIKeyFlag); err != nil {
			return fmt.Errorf("failed to store temporary API key: %w", err)
		}
		// Note: This temporary key will remain in keyring, but that's acceptable for ephemeral use
		cfg.APIKeyID = tempKeyID
		hasOverrides = true
	}

	// Model overrides (works for both profile types)
	if clauderockModelFlag != "" {
		// For bedrock, validate it's a full profile ID
		if cfg.ProfileType == "bedrock" && !aws.IsFullProfileID(clauderockModelFlag) {
			return fmt.Errorf("--clauderock-model must be a full profile ID for bedrock (e.g., 'global.anthropic.claude-sonnet-4-5-20250929-v1:0')\nRun 'clauderock manage models list' to see available models")
		}
		cfg.Model = clauderockModelFlag
		hasOverrides = true
	}
	if clauderockFastModelFlag != "" {
		if cfg.ProfileType == "bedrock" && !aws.IsFullProfileID(clauderockFastModelFlag) {
			return fmt.Errorf("--clauderock-fast-model must be a full profile ID for bedrock (e.g., 'global.anthropic.claude-haiku-4-5-20250929-v1:0')\nRun 'clauderock manage models list' to see available models")
		}
		cfg.FastModel = clauderockFastModelFlag
		hasOverrides = true
	}
	if clauderockHeavyModelFlag != "" {
		if cfg.ProfileType == "bedrock" && !aws.IsFullProfileID(clauderockHeavyModelFlag) {
			return fmt.Errorf("--clauderock-heavy-model must be a full profile ID for bedrock (e.g., 'global.anthropic.claude-opus-4-1-20250514-v1:0')\nRun 'clauderock manage models list' to see available models")
		}
		cfg.HeavyModel = clauderockHeavyModelFlag
		hasOverrides = true
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Show overrides if any
	if hasOverrides {
		fmt.Println("Using overrides:")
		if clauderockProfileTypeFlag != "" {
			fmt.Printf("  Profile Type: %s\n", cfg.ProfileType)
		}
		if clauderockAWSProfileFlag != "" {
			fmt.Printf("  AWS Profile: %s\n", cfg.Profile)
		}
		if clauderockRegionFlag != "" {
			fmt.Printf("  Region: %s\n", cfg.Region)
		}
		if clauderockCrossRegionFlag != "" {
			fmt.Printf("  Cross Region: %s\n", cfg.CrossRegion)
		}
		if clauderockBaseURLFlag != "" {
			fmt.Printf("  Base URL: %s\n", cfg.BaseURL)
		}
		if clauderockAPIKeyFlag != "" {
			fmt.Printf("  API Key: <provided via flag>\n")
		}
		if clauderockModelFlag != "" {
			fmt.Printf("  Model: %s\n", cfg.Model)
		}
		if clauderockFastModelFlag != "" {
			fmt.Printf("  Fast Model: %s\n", cfg.FastModel)
		}
		if clauderockHeavyModelFlag != "" {
			fmt.Printf("  Heavy Model: %s\n", cfg.HeavyModel)
		}
		fmt.Println()
	}

	// Use stored inference profile IDs directly (no AWS query needed!)
	mainModelID := cfg.Model
	fastModelID := cfg.FastModel
	heavyModelID := cfg.HeavyModel

	// Validate that we have full profile IDs (migration should have handled this)
	if mainModelID == "" || fastModelID == "" || heavyModelID == "" {
		return fmt.Errorf("model configuration is incomplete, please run: clauderock manage config")
	}

	// Get current profile name for tracking
	currentProfile := "default"
	if clauderockProfileFlag != "" {
		currentProfile = clauderockProfileFlag
	} else {
		current, err := profileMgr.GetCurrent()
		if err == nil {
			currentProfile = current
		}
	}

	// Launch Claude Code with passthrough args
	return launcher.Launch(cfg, mainModelID, fastModelID, heavyModelID, currentProfile, passthroughArgs)
}

// collectPassthroughArgs separates clauderock flags from Claude CLI args
func collectPassthroughArgs() []string {
	if len(os.Args) <= 1 {
		return nil
	}

	var passthroughArgs []string
	clauderockFlags := map[string]bool{
		"--clauderock-profile":       true,
		"--clauderock-profile-type":  true,
		"--clauderock-model":         true,
		"--clauderock-fast-model":    true,
		"--clauderock-heavy-model":   true,
		"--clauderock-aws-profile":   true,
		"--clauderock-region":        true,
		"--clauderock-cross-region":  true,
		"--clauderock-base-url":      true,
		"--clauderock-api-key":       true,
	}

	skip := false
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		if skip {
			skip = false
			continue
		}

		// Check if this is a clauderock flag
		if strings.HasPrefix(arg, "--clauderock-") {
			// Check if it's a flag with value (--flag=value or --flag value)
			if strings.Contains(arg, "=") {
				// --flag=value format, skip entirely
				continue
			} else if clauderockFlags[arg] {
				// --flag value format, skip this and next arg
				skip = true
				continue
			}
		}

		// This is a passthrough arg
		passthroughArgs = append(passthroughArgs, arg)
	}

	return passthroughArgs
}
