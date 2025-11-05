package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/launcher"
	"github.com/OlaHulleberg/clauderock/internal/profiles"
	"github.com/OlaHulleberg/clauderock/internal/updater"
	"github.com/spf13/cobra"
)

var (
	clauderockProfileFlag     string
	clauderockModelFlag       string
	clauderockFastModelFlag   string
	clauderockAWSProfileFlag  string
	clauderockRegionFlag      string
	clauderockCrossRegionFlag string
	Version                   = "dev"
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
	rootCmd.Flags().StringVar(&clauderockModelFlag, "clauderock-model", "", "Override main model for this run")
	rootCmd.Flags().StringVar(&clauderockFastModelFlag, "clauderock-fast-model", "", "Override fast model for this run")
	rootCmd.Flags().StringVar(&clauderockAWSProfileFlag, "clauderock-aws-profile", "", "Override AWS profile for this run")
	rootCmd.Flags().StringVar(&clauderockRegionFlag, "clauderock-region", "", "Override AWS region for this run")
	rootCmd.Flags().StringVar(&clauderockCrossRegionFlag, "clauderock-cross-region", "", "Override cross-region setting for this run")

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

	// Apply overrides from flags
	hasOverrides := false
	if clauderockAWSProfileFlag != "" {
		cfg.Profile = clauderockAWSProfileFlag
		hasOverrides = true
	}
	if clauderockRegionFlag != "" {
		cfg.Region = clauderockRegionFlag
		hasOverrides = true
	}
	if clauderockCrossRegionFlag != "" {
		cfg.CrossRegion = clauderockCrossRegionFlag
		hasOverrides = true
	}
	if clauderockModelFlag != "" {
		if !aws.IsFullProfileID(clauderockModelFlag) {
			return fmt.Errorf("--clauderock-model must be a full profile ID (e.g., 'global.anthropic.claude-sonnet-4-5-20250929-v1:0')\nRun 'clauderock manage models list' to see available models")
		}
		cfg.Model = clauderockModelFlag
		hasOverrides = true
	}
	if clauderockFastModelFlag != "" {
		if !aws.IsFullProfileID(clauderockFastModelFlag) {
			return fmt.Errorf("--clauderock-fast-model must be a full profile ID (e.g., 'global.anthropic.claude-haiku-4-5-20250929-v1:0')\nRun 'clauderock manage models list' to see available models")
		}
		cfg.FastModel = clauderockFastModelFlag
		hasOverrides = true
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Show overrides if any
	if hasOverrides {
		fmt.Println("Using overrides:")
		if clauderockAWSProfileFlag != "" {
			fmt.Printf("  AWS Profile: %s\n", cfg.Profile)
		}
		if clauderockRegionFlag != "" {
			fmt.Printf("  Region: %s\n", cfg.Region)
		}
		if clauderockCrossRegionFlag != "" {
			fmt.Printf("  Cross Region: %s\n", cfg.CrossRegion)
		}
		if clauderockModelFlag != "" {
			fmt.Printf("  Model: %s\n", cfg.Model)
		}
		if clauderockFastModelFlag != "" {
			fmt.Printf("  Fast Model: %s\n", cfg.FastModel)
		}
		fmt.Println()
	}

	// Use stored inference profile IDs directly (no AWS query needed!)
	mainModelID := cfg.Model
	fastModelID := cfg.FastModel

	// Validate that we have full profile IDs (migration should have handled this)
	if mainModelID == "" || fastModelID == "" {
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
	return launcher.Launch(cfg, mainModelID, fastModelID, currentProfile, passthroughArgs)
}

// collectPassthroughArgs separates clauderock flags from Claude CLI args
func collectPassthroughArgs() []string {
	if len(os.Args) <= 1 {
		return nil
	}

	var passthroughArgs []string
	clauderockFlags := map[string]bool{
		"--clauderock-profile":      true,
		"--clauderock-model":        true,
		"--clauderock-fast-model":   true,
		"--clauderock-aws-profile":  true,
		"--clauderock-region":       true,
		"--clauderock-cross-region": true,
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
