package cmd

import (
	"fmt"
	"os"

	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/launcher"
	"github.com/OlaHulleberg/clauderock/internal/profiles"
	"github.com/OlaHulleberg/clauderock/internal/updater"
	"github.com/spf13/cobra"
)

var (
	profileFlag     string
	modelFlag       string
	fastModelFlag   string
	awsProfileFlag  string
	regionFlag      string
	crossRegionFlag string
	Version         = "dev"
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
	rootCmd.Flags().StringVar(&profileFlag, "profile", "", "Use a specific profile for this run")
	rootCmd.Flags().StringVar(&modelFlag, "model", "", "Override main model for this run")
	rootCmd.Flags().StringVar(&fastModelFlag, "fast-model", "", "Override fast model for this run")
	rootCmd.Flags().StringVar(&awsProfileFlag, "aws-profile", "", "Override AWS profile for this run")
	rootCmd.Flags().StringVar(&regionFlag, "region", "", "Override AWS region for this run")
	rootCmd.Flags().StringVar(&crossRegionFlag, "cross-region", "", "Override cross-region setting for this run")
}

func runRoot(cmd *cobra.Command, args []string) error {
	// Check for updates in background
	go updater.CheckForUpdates(Version)

	// Load configuration from profile
	profileMgr, err := profiles.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create profile manager: %w", err)
	}

	var cfg *config.Config
	if profileFlag != "" {
		// Load specific profile
		cfg, err = profileMgr.Load(profileFlag)
		if err != nil {
			return fmt.Errorf("failed to load profile '%s': %w", profileFlag, err)
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
	if awsProfileFlag != "" {
		cfg.Profile = awsProfileFlag
		hasOverrides = true
	}
	if regionFlag != "" {
		cfg.Region = regionFlag
		hasOverrides = true
	}
	if crossRegionFlag != "" {
		cfg.CrossRegion = crossRegionFlag
		hasOverrides = true
	}
	if modelFlag != "" {
		cfg.Model = modelFlag
		hasOverrides = true
	}
	if fastModelFlag != "" {
		cfg.FastModel = fastModelFlag
		hasOverrides = true
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Show overrides if any
	if hasOverrides {
		fmt.Println("Using overrides:")
		if awsProfileFlag != "" {
			fmt.Printf("  AWS Profile: %s\n", cfg.Profile)
		}
		if regionFlag != "" {
			fmt.Printf("  Region: %s\n", cfg.Region)
		}
		if crossRegionFlag != "" {
			fmt.Printf("  Cross Region: %s\n", cfg.CrossRegion)
		}
		if modelFlag != "" {
			fmt.Printf("  Model: %s\n", cfg.Model)
		}
		if fastModelFlag != "" {
			fmt.Printf("  Fast Model: %s\n", cfg.FastModel)
		}
		fmt.Println()
	}

	// Find inference profile IDs
	mainModelID, fastModelID, err := aws.FindInferenceProfiles(cfg)
	if err != nil {
		return fmt.Errorf("failed to find inference profiles: %w", err)
	}

	fmt.Printf("Using model: %s\n", mainModelID)
	fmt.Printf("Using fast model: %s\n", fastModelID)

	// Get current profile name for tracking
	currentProfile := "default"
	if profileFlag != "" {
		currentProfile = profileFlag
	} else {
		current, err := profileMgr.GetCurrent()
		if err == nil {
			currentProfile = current
		}
	}

	// Launch Claude Code
	return launcher.Launch(cfg, mainModelID, fastModelID, currentProfile)
}
