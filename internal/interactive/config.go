package interactive

import (
	"fmt"
	"os"
	"strings"

	"github.com/OlaHulleberg/clauderock/internal/api"
	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/awsutil"
	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/keyring"
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

	// Step 0: Profile Type Selection
	profileTypeOptions := []SelectOption{
		{ID: "bedrock", Display: "AWS Bedrock (Cross-region inference)"},
		{ID: "api", Display: "API Key (Direct API access)"},
	}

	selectedProfileType, err := InteractiveSelect(
		"Select Profile Type",
		"Choose authentication method...",
		profileTypeOptions,
		cfg.ProfileType,
	)
	if err != nil {
		return fmt.Errorf("profile type selection failed: %w", err)
	}

	cfg.ProfileType = selectedProfileType

	// Branch based on profile type
	if selectedProfileType == "bedrock" {
		return runBedrockConfig(cfg, manager, currentProfile, currentVersion)
	} else if selectedProfileType == "api" {
		return runAPIConfig(cfg, manager, currentProfile, currentVersion)
	}

	return fmt.Errorf("unsupported profile type: %s", selectedProfileType)
}

// runBedrockConfig handles the Bedrock configuration flow
func runBedrockConfig(cfg *config.Config, manager interface {
	Save(name string, cfg *config.Config) error
}, currentProfile, currentVersion string) error {
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

	// Update version to current CLI version (but not for dev builds)
	if currentVersion != "dev" {
		cfg.Version = currentVersion
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

// runAPIConfig handles the API key configuration flow
func runAPIConfig(cfg *config.Config, manager interface {
	Save(name string, cfg *config.Config) error
}, currentProfile, currentVersion string) error {
	// Step 1: Base URL Input
	fmt.Println("\nEnter the base URL for your API gateway:")
	fmt.Println("Examples: api.example.com, https://api.example.com, http://localhost:8080")
	fmt.Print("> ")

	var baseURL string
	if _, err := fmt.Scanln(&baseURL); err != nil {
		return fmt.Errorf("failed to read base URL: %w", err)
	}

	if baseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	// Normalize the base URL
	cfg.BaseURL = baseURL

	// Step 2: API Key Input
	fmt.Println("\nEnter your API key:")
	fmt.Println("(This will be stored securely in your system keychain)")

	// Check environment variable first
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey != "" {
		fmt.Println("\nFound ANTHROPIC_API_KEY in environment.")
		useEnvKey, err := Confirm(
			"API Key Detected",
			"Found ANTHROPIC_API_KEY in environment. Do you want to use it?",
			nil,
		)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}

		if !useEnvKey {
			apiKey = ""
		}
	}

	// Prompt for API key if not using environment variable
	if apiKey == "" {
		fmt.Print("> ")
		if _, err := fmt.Scanln(&apiKey); err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}

		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}
	}

	// Step 3: Fetch available models
	fmt.Println("\nFetching available models from API...")
	models, err := api.FetchAvailableModels(cfg.BaseURL, apiKey)

	var selectedModel, selectedFastModel, selectedHeavyModel string

	// Fall back to manual input if API call fails
	if err != nil || len(models) == 0 {
		fmt.Println("Using manual input mode")
		fmt.Println()

		// Main model input
		selectedModel, err = PromptTextInput(
			"Enter Main Model ID",
			"",
			"claude-sonnet-4-5",
		)
		if err != nil {
			return fmt.Errorf("main model input failed: %w", err)
		}
		if selectedModel == "" {
			return fmt.Errorf("main model ID cannot be empty")
		}

		// Fast model input
		selectedFastModel, err = PromptTextInput(
			"Enter Fast Model ID",
			"",
			"claude-haiku-4-5",
		)
		if err != nil {
			return fmt.Errorf("fast model input failed: %w", err)
		}
		if selectedFastModel == "" {
			return fmt.Errorf("fast model ID cannot be empty")
		}

		// Heavy model input
		selectedHeavyModel, err = PromptTextInput(
			"Enter Heavy Model ID",
			"",
			"claude-opus-4",
		)
		if err != nil {
			return fmt.Errorf("heavy model input failed: %w", err)
		}
		if selectedHeavyModel == "" {
			return fmt.Errorf("heavy model ID cannot be empty")
		}
	} else {
		// Extract model IDs for selection
		modelIDs := make([]string, len(models))
		for i, m := range models {
			modelIDs[i] = m.ID
		}

		// Step 4: Main model selection
		mainModelOptions := buildAPIModelOptions(models, "main")
		selectedModel, err = InteractiveSelect(
			"Select Main Model",
			"Type to filter models...",
			mainModelOptions,
			"",
		)
		if err != nil {
			return fmt.Errorf("main model selection failed: %w", err)
		}

		// Step 5: Fast model selection
		fastModelOptions := buildAPIModelOptions(models, "fast")
		selectedFastModel, err = InteractiveSelect(
			"Select Fast Model",
			"Type to filter models...",
			fastModelOptions,
			"",
		)
		if err != nil {
			return fmt.Errorf("fast model selection failed: %w", err)
		}

		// Step 6: Heavy model selection
		heavyModelOptions := buildAPIModelOptions(models, "heavy")
		selectedHeavyModel, err = InteractiveSelect(
			"Select Heavy Model",
			"Type to filter models...",
			heavyModelOptions,
			"",
		)
		if err != nil {
			return fmt.Errorf("heavy model selection failed: %w", err)
		}
	}

	// Generate keyring ID and store API key
	keyID, err := keyring.GenerateID()
	if err != nil {
		return fmt.Errorf("failed to generate keyring ID: %w", err)
	}

	if err := keyring.Store(keyID, apiKey); err != nil {
		return fmt.Errorf("failed to store API key in keyring: %w", err)
	}

	// Update configuration
	cfg.APIKeyID = keyID
	cfg.Model = selectedModel
	cfg.FastModel = selectedFastModel
	cfg.HeavyModel = selectedHeavyModel

	// Clear Bedrock-specific fields
	cfg.Profile = ""
	cfg.Region = ""
	cfg.CrossRegion = ""

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		// Clean up keyring entry if validation fails
		keyring.Delete(keyID)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Update version to current CLI version (but not for dev builds)
	if currentVersion != "dev" {
		cfg.Version = currentVersion
	}

	// Save configuration to current profile
	if err := manager.Save(currentProfile, cfg); err != nil {
		// Clean up keyring entry if save fails
		keyring.Delete(keyID)
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\n✓ Configuration saved successfully to profile '%s'!\n", currentProfile)
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Profile Type: %s\n", cfg.ProfileType)
	fmt.Printf("  Base URL:     %s\n", cfg.BaseURL)
	fmt.Printf("  Model:        %s\n", cfg.Model)
	fmt.Printf("  Fast Model:   %s\n", cfg.FastModel)
	fmt.Printf("  Heavy Model:  %s\n", cfg.HeavyModel)

	return nil
}

// buildAPIModelOptions creates SelectOptions for API models
func buildAPIModelOptions(models []api.ModelInfo, context string) []SelectOption {
	var options []SelectOption

	// Add "Recommended" section
	var recommendedModel *api.ModelInfo
	for i, m := range models {
		if api.IsRecommendedModel(m, context) {
			recommendedModel = &models[i]
			break
		}
	}

	if recommendedModel != nil {
		options = append(options, SelectOption{
			ID:       "",
			Display:  recommendedSectionHeader,
			IsHeader: true,
		})
		options = append(options, SelectOption{
			ID:      recommendedModel.ID,
			Display: fmt.Sprintf("  ⭐ %s", recommendedModel.Name),
		})
		options = append(options, SelectOption{
			ID:       "",
			Display:  "",
			IsHeader: true,
		})
	}

	// Add all models
	options = append(options, SelectOption{
		ID:       "",
		Display:  "ALL MODELS",
		IsHeader: true,
	})

	for _, m := range models {
		// Skip if already shown in recommended
		if recommendedModel != nil && m.ID == recommendedModel.ID {
			continue
		}
		options = append(options, SelectOption{
			ID:      m.ID,
			Display: fmt.Sprintf("  %s", m.Name),
		})
	}

	return options
}
