package interactive

import (
	"fmt"

	"github.com/OlaHulleberg/clauderock/internal/api"
	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/keyring"
)

// SelectBedrockModels interactively selects models for a Bedrock profile
// Updates cfg.Model, cfg.FastModel, and cfg.HeavyModel with full profile IDs
func SelectBedrockModels(cfg *config.Config) error {
	// Fetch available models using current AWS configuration
	fmt.Println("\nFetching available models...")
	models, err := aws.GetAvailableModels(cfg.Profile, cfg.Region, cfg.CrossRegion)
	if err != nil {
		return fmt.Errorf("failed to fetch models: %w", err)
	}

	if len(models) == 0 {
		return fmt.Errorf("no models available for the selected configuration")
	}

	// Extract current friendly names for defaults
	currentMain := aws.ExtractFriendlyModelName(cfg.Model)
	currentFast := aws.ExtractFriendlyModelName(cfg.FastModel)
	currentHeavy := aws.ExtractFriendlyModelName(cfg.HeavyModel)

	// Main model selection
	mainModelOptions := buildModelOptions(models, "main")
	selectedMain, err := InteractiveSelect(
		"Select Main Model",
		"Type to filter models...",
		mainModelOptions,
		currentMain,
	)
	if err != nil {
		return fmt.Errorf("main model selection failed: %w", err)
	}

	// Fast model selection
	fastModelOptions := buildModelOptions(models, "fast")
	selectedFast, err := InteractiveSelect(
		"Select Fast Model",
		"Type to filter models...",
		fastModelOptions,
		currentFast,
	)
	if err != nil {
		return fmt.Errorf("fast model selection failed: %w", err)
	}

	// Heavy model selection
	heavyModelOptions := buildModelOptions(models, "heavy")
	selectedHeavy, err := InteractiveSelect(
		"Select Heavy Model",
		"Type to filter models...",
		heavyModelOptions,
		currentHeavy,
	)
	if err != nil {
		return fmt.Errorf("heavy model selection failed: %w", err)
	}

	// Resolve friendly model names to full profile IDs
	fmt.Println("\nResolving model profile IDs...")
	mainModelID, err := aws.ResolveModelToProfileID(cfg.Profile, cfg.Region, cfg.CrossRegion, selectedMain)
	if err != nil {
		return fmt.Errorf("failed to resolve main model: %w", err)
	}

	fastModelID, err := aws.ResolveModelToProfileID(cfg.Profile, cfg.Region, cfg.CrossRegion, selectedFast)
	if err != nil {
		return fmt.Errorf("failed to resolve fast model: %w", err)
	}

	heavyModelID, err := aws.ResolveModelToProfileID(cfg.Profile, cfg.Region, cfg.CrossRegion, selectedHeavy)
	if err != nil {
		return fmt.Errorf("failed to resolve heavy model: %w", err)
	}

	// Update config with resolved model IDs
	cfg.Model = mainModelID
	cfg.FastModel = fastModelID
	cfg.HeavyModel = heavyModelID

	return nil
}

// SelectAPIModels interactively selects models for an API profile
// Updates cfg.Model, cfg.FastModel, and cfg.HeavyModel with model IDs
func SelectAPIModels(cfg *config.Config) error {
	// Retrieve API key from keyring
	apiKey, err := keyring.Get(cfg.APIKeyID)
	if err != nil {
		return fmt.Errorf("failed to retrieve API key from keyring: %w", err)
	}

	// Fetch available models from API
	fmt.Println("\nFetching available models from API...")
	models, err := api.FetchAvailableModels(cfg.BaseURL, apiKey)
	if err != nil {
		return fmt.Errorf("failed to fetch models: %w", err)
	}

	if len(models) == 0 {
		return fmt.Errorf("no models available from API")
	}

	// Main model selection
	mainModelOptions := buildAPIModelOptions(models, "main")
	selectedMain, err := InteractiveSelect(
		"Select Main Model",
		"Type to filter models...",
		mainModelOptions,
		cfg.Model,
	)
	if err != nil {
		return fmt.Errorf("main model selection failed: %w", err)
	}

	// Fast model selection
	fastModelOptions := buildAPIModelOptions(models, "fast")
	selectedFast, err := InteractiveSelect(
		"Select Fast Model",
		"Type to filter models...",
		fastModelOptions,
		cfg.FastModel,
	)
	if err != nil {
		return fmt.Errorf("fast model selection failed: %w", err)
	}

	// Heavy model selection
	heavyModelOptions := buildAPIModelOptions(models, "heavy")
	selectedHeavy, err := InteractiveSelect(
		"Select Heavy Model",
		"Type to filter models...",
		heavyModelOptions,
		cfg.HeavyModel,
	)
	if err != nil {
		return fmt.Errorf("heavy model selection failed: %w", err)
	}

	// Update config with selected model IDs (no resolution needed for API)
	cfg.Model = selectedMain
	cfg.FastModel = selectedFast
	cfg.HeavyModel = selectedHeavy

	return nil
}
