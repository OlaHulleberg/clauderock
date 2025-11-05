package interactive

import (
	"fmt"
	"strings"

	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/awsutil"
	"github.com/OlaHulleberg/clauderock/internal/config"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	recommendedSectionHeader = "RECOMMENDED"
)

// formatModelDisplay formats a model name with provider
func formatModelDisplay(model string, showProvider bool) string {
	// Parse provider.model-name format
	parts := strings.SplitN(model, ".", 2)

	if len(parts) == 2 {
		caser := cases.Title(language.English)
		provider := caser.String(parts[0])
		modelName := parts[1]

		if showProvider {
			return fmt.Sprintf("  ⭐ %s - %s", provider, modelName)
		}
		return fmt.Sprintf("  %s", modelName)
	}

	if showProvider {
		return fmt.Sprintf("  ⭐ %s", model)
	}
	return fmt.Sprintf("  %s", model)
}

// buildModelOptions creates SelectOptions with headers for recommended and provider sections
func buildModelOptions(models []string, context string) []SelectOption {
	var options []SelectOption

	// Add "Recommended" section
	var recommendedModel string
	for _, m := range models {
		if aws.IsRecommendedModel(m, context) {
			recommendedModel = m
			break
		}
	}

	if recommendedModel != "" {
		options = append(options, SelectOption{
			ID:       "",
			Display:  recommendedSectionHeader,
			IsHeader: true,
		})
		options = append(options, SelectOption{
			ID:      recommendedModel,
			Display: formatModelDisplay(recommendedModel, true), // Show provider for recommended
		})
		options = append(options, SelectOption{
			ID:       "",
			Display:  "",
			IsHeader: true,
		})
	}

	// Group models by provider
	providerModels := make(map[string][]string)
	var providers []string

	for _, m := range models {
		parts := strings.SplitN(m, ".", 2)
		if len(parts) == 2 {
			provider := parts[0]
			if _, exists := providerModels[provider]; !exists {
				providers = append(providers, provider)
			}
			providerModels[provider] = append(providerModels[provider], m)
		}
	}

	// Add sections for each provider
	for _, provider := range providers {
		// Add provider header
		options = append(options, SelectOption{
			ID:       "",
			Display:  strings.ToUpper(provider),
			IsHeader: true,
		})

		// Add models for this provider
		for _, m := range providerModels[provider] {
			options = append(options, SelectOption{
				ID:      m,
				Display: formatModelDisplay(m, false), // Don't show provider for grouped models
			})
		}

		// Add empty line between providers
		options = append(options, SelectOption{
			ID:       "",
			Display:  "",
			IsHeader: true,
		})
	}

	return options
}

// RunInteractiveConfig runs an interactive configuration wizard
func RunInteractiveConfig(currentVersion string, mgr interface{}) error {
	// Type assert the manager (we'll accept any interface to avoid circular dependencies)
	type ConfigManager interface {
		GetCurrentConfig(version string) (*config.Config, error)
		GetCurrent() (string, error)
		Save(name string, cfg *config.Config) error
	}

	manager, ok := mgr.(ConfigManager)
	if !ok {
		return fmt.Errorf("invalid manager type")
	}

	// Load existing config (or defaults)
	cfg, err := manager.GetCurrentConfig(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	currentProfile, err := manager.GetCurrent()
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
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

	// Step 5: Main model selection
	// Build model options with headers for main context
	mainModelOptions := buildModelOptions(models, "main")

	selectedModel, err = InteractiveSelect(
		"Select Main Model",
		"Type to filter models...",
		mainModelOptions,
		selectedModel,
	)
	if err != nil {
		return fmt.Errorf("main model selection failed: %w", err)
	}

	// Step 6: Fast model selection
	// Build model options with headers for fast context
	fastModelOptions := buildModelOptions(models, "fast")

	selectedFastModel, err = InteractiveSelect(
		"Select Fast Model",
		"Type to filter models...",
		fastModelOptions,
		selectedFastModel,
	)
	if err != nil {
		return fmt.Errorf("fast model selection failed: %w", err)
	}

	// Step 7: Heavy model selection
	// Build model options with headers for heavy context
	heavyModelOptions := buildModelOptions(models, "heavy")

	selectedHeavyModel, err := InteractiveSelect(
		"Select Heavy Model",
		"Type to filter models...",
		heavyModelOptions,
		"",
	)
	if err != nil {
		return fmt.Errorf("heavy model selection failed: %w", err)
	}

	// Update configuration with selections
	cfg.Profile = selectedProfile
	cfg.Region = selectedRegion
	cfg.CrossRegion = selectedCrossRegion

	// Resolve friendly model names to full profile IDs
	fmt.Println("\nResolving model profile IDs...")
	mainModelID, err := aws.ResolveModelToProfileID(selectedProfile, selectedRegion, selectedCrossRegion, selectedModel)
	if err != nil {
		return fmt.Errorf("failed to resolve main model: %w", err)
	}
	cfg.Model = mainModelID

	fastModelID, err := aws.ResolveModelToProfileID(selectedProfile, selectedRegion, selectedCrossRegion, selectedFastModel)
	if err != nil {
		return fmt.Errorf("failed to resolve fast model: %w", err)
	}
	cfg.FastModel = fastModelID

	heavyModelID, err := aws.ResolveModelToProfileID(selectedProfile, selectedRegion, selectedCrossRegion, selectedHeavyModel)
	if err != nil {
		return fmt.Errorf("failed to resolve heavy model: %w", err)
	}
	cfg.HeavyModel = heavyModelID

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save configuration to current profile
	if err := manager.Save(currentProfile, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\n✓ Configuration saved successfully to profile '%s'!\n", currentProfile)
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Profile:      %s\n", cfg.Profile)
	fmt.Printf("  Region:       %s\n", cfg.Region)
	fmt.Printf("  Cross Region: %s\n", cfg.CrossRegion)
	fmt.Printf("  Model:        %s\n", cfg.Model)
	fmt.Printf("  Fast Model:   %s\n", cfg.FastModel)
	fmt.Printf("  Heavy Model:  %s\n", cfg.HeavyModel)

	return nil
}
