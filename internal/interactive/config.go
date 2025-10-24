package interactive

import (
	"fmt"
	"strings"

	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/awsutil"
	"github.com/OlaHulleberg/clauderock/internal/config"
)

// RunInteractiveConfig runs an interactive configuration wizard
func RunInteractiveConfig(currentVersion string) error {
	// Load existing config (or defaults)
	cfg, err := config.Load(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Variables to hold user selections
	var (
		selectedProfile     string
		selectedRegion      string
		selectedCrossRegion string
		selectedModel       string
		selectedFastModel   string
	)

	// Initialize with current values
	selectedProfile = cfg.Profile
	selectedRegion = cfg.Region
	selectedCrossRegion = cfg.CrossRegion
	selectedModel = cfg.Model
	selectedFastModel = cfg.FastModel

	// Step 1: Profile selection
	profiles, err := awsutil.GetProfiles()
	if err != nil {
		return fmt.Errorf("failed to get AWS profiles: %w", err)
	}

	profileOptions := make([]SelectOption, len(profiles))
	for i, p := range profiles {
		profileOptions[i] = SelectOption{ID: p, Display: p}
	}

	selectedProfile, err = InteractiveSelect(
		"Select AWS Profile",
		"Type to filter profiles...",
		profileOptions,
		selectedProfile,
	)
	if err != nil {
		return fmt.Errorf("profile selection failed: %w", err)
	}

	// Step 2: Region selection
	selectedRegion, err = SelectRegionWithSearch(selectedRegion)
	if err != nil {
		return fmt.Errorf("region selection failed: %w", err)
	}

	// Step 3: Cross-region selection
	crossRegionOptions := []SelectOption{
		{ID: "global", Display: "Global"},
		{ID: "us", Display: "US"},
		{ID: "eu", Display: "EU"},
	}

	selectedCrossRegion, err = InteractiveSelect(
		"Select Cross Region",
		"Type to filter...",
		crossRegionOptions,
		selectedCrossRegion,
	)
	if err != nil {
		return fmt.Errorf("cross-region selection failed: %w", err)
	}

	// Step 4: Fetch available models
	fmt.Println("\nFetching available models...")
	models, err := aws.GetAvailableModels(selectedProfile, selectedRegion, selectedCrossRegion)
	if err != nil {
		return fmt.Errorf("failed to fetch models: %w", err)
	}

	if len(models) == 0 {
		return fmt.Errorf("no models available for the selected configuration")
	}

	// Convert models to SelectOptions with friendly display names
	modelOptions := make([]SelectOption, len(models))
	for i, m := range models {
		// Parse provider.model-name format
		parts := strings.SplitN(m, ".", 2)
		var displayName string
		if len(parts) == 2 {
			// Capitalize provider name for display
			provider := strings.Title(parts[0])
			modelName := parts[1]
			displayName = fmt.Sprintf("%s: %s", provider, modelName)
		} else {
			// Fallback if format is unexpected
			displayName = m
		}
		modelOptions[i] = SelectOption{ID: m, Display: displayName}
	}

	// Step 5: Main model selection
	selectedModel, err = InteractiveSelect(
		"Select Main Model",
		"Type to filter models...",
		modelOptions,
		selectedModel,
	)
	if err != nil {
		return fmt.Errorf("main model selection failed: %w", err)
	}

	// Step 6: Fast model selection
	selectedFastModel, err = InteractiveSelect(
		"Select Fast Model",
		"Type to filter models...",
		modelOptions,
		selectedFastModel,
	)
	if err != nil {
		return fmt.Errorf("fast model selection failed: %w", err)
	}

	// Update configuration with selections
	cfg.Profile = selectedProfile
	cfg.Region = selectedRegion
	cfg.CrossRegion = selectedCrossRegion
	cfg.Model = selectedModel
	cfg.FastModel = selectedFastModel

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("\n✓ Configuration saved successfully!")
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Profile:      %s\n", cfg.Profile)
	fmt.Printf("  Region:       %s\n", cfg.Region)
	fmt.Printf("  Cross Region: %s\n", cfg.CrossRegion)
	fmt.Printf("  Model:        %s\n", cfg.Model)
	fmt.Printf("  Fast Model:   %s\n", cfg.FastModel)

	return nil
}
