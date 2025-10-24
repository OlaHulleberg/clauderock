package aws

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
)

// FindInferenceProfiles finds the main and fast model inference profile IDs
func FindInferenceProfiles(cfg *config.Config) (string, string, error) {
	ctx := context.Background()

	// Load AWS config with specified profile and region
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithSharedConfigProfile(cfg.Profile),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Bedrock client
	client := bedrock.NewFromConfig(awsCfg)

	// List cross-region inference profiles (SYSTEM_DEFINED type only)
	result, err := client.ListInferenceProfiles(ctx, &bedrock.ListInferenceProfilesInput{
		TypeEquals: types.InferenceProfileTypeSystemDefined,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to list inference profiles: %w", err)
	}

	// Find matching profiles
	mainModelID, err := findMatchingProfile(result.InferenceProfileSummaries, cfg.CrossRegion, cfg.Model)
	if err != nil {
		return "", "", fmt.Errorf("main model: %w\nAvailable profiles:\n%s",
			err, formatAvailableProfiles(result.InferenceProfileSummaries))
	}

	fastModelID, err := findMatchingProfile(result.InferenceProfileSummaries, cfg.CrossRegion, cfg.FastModel)
	if err != nil {
		return "", "", fmt.Errorf("fast model: %w\nAvailable profiles:\n%s",
			err, formatAvailableProfiles(result.InferenceProfileSummaries))
	}

	return mainModelID, fastModelID, nil
}

func findMatchingProfile(profiles []types.InferenceProfileSummary, crossRegion, model string) (string, error) {
	// Model format: {provider}.{model-name}
	// Example input: "anthropic.claude-sonnet-4-5"
	// Expected profile format: {cross-region}.{provider}.{model-name}-{version}
	// Example: global.anthropic.claude-sonnet-4-5-20250929-v1:0

	// Build prefix from cross-region and model (which includes provider)
	prefix := fmt.Sprintf("%s.%s", crossRegion, model)

	for _, profile := range profiles {
		if profile.InferenceProfileId != nil {
			profileID := aws.ToString(profile.InferenceProfileId)
			if strings.HasPrefix(profileID, prefix) {
				return profileID, nil
			}
		}
	}

	return "", fmt.Errorf("could not find inference profile for model '%s' with cross-region '%s'", model, crossRegion)
}

func formatAvailableProfiles(profiles []types.InferenceProfileSummary) string {
	var builder strings.Builder
	for _, profile := range profiles {
		if profile.InferenceProfileId != nil {
			builder.WriteString(fmt.Sprintf("  - %s\n", aws.ToString(profile.InferenceProfileId)))
		}
	}
	return builder.String()
}

// GetAvailableModels fetches available models from Bedrock for a given profile, region, and cross-region
// Returns a deduplicated list of model names in format "provider.model-name" (e.g., "anthropic.claude-sonnet-4-5", "meta.llama3-70b")
func GetAvailableModels(profile, region, crossRegion string) ([]string, error) {
	ctx := context.Background()

	// Load AWS config with specified profile and region
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithSharedConfigProfile(profile),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Bedrock client
	client := bedrock.NewFromConfig(awsCfg)

	// List cross-region inference profiles (SYSTEM_DEFINED type only)
	result, err := client.ListInferenceProfiles(ctx, &bedrock.ListInferenceProfilesInput{
		TypeEquals: types.InferenceProfileTypeSystemDefined,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list inference profiles: %w", err)
	}

	// Extract unique model names for the specified cross-region
	modelMap := make(map[string]bool)
	prefix := fmt.Sprintf("%s.", crossRegion)

	for _, profile := range result.InferenceProfileSummaries {
		if profile.InferenceProfileId != nil {
			profileID := aws.ToString(profile.InferenceProfileId)
			if strings.HasPrefix(profileID, prefix) {
				// Extract provider and model name from profile ID
				// Format: {cross-region}.{provider}.{model-name}-{version}
				// Example: global.anthropic.claude-sonnet-4-5-20250929-v1:0
				// We want to extract: anthropic.claude-sonnet-4-5

				// Remove cross-region prefix
				remaining := strings.TrimPrefix(profileID, prefix)

				// Split on first dot to separate provider from rest
				firstDotIndex := strings.Index(remaining, ".")
				if firstDotIndex == -1 {
					continue
				}

				provider := remaining[:firstDotIndex]
				modelWithVersion := remaining[firstDotIndex+1:]

				// Extract model name without version
				// Split by hyphen and take parts that form the model name
				// We need to handle versions like "claude-sonnet-4-5-20250929-v1:0"
				// and extract just "claude-sonnet-4-5"
				parts := strings.Split(modelWithVersion, "-")

				// Build model name by taking parts until we hit a date-like pattern (8 digits)
				var modelParts []string
				for _, part := range parts {
					// Stop if we hit a date pattern (8 digits) or version pattern
					if len(part) == 8 || strings.HasPrefix(part, "v") || strings.Contains(part, ":") {
						break
					}
					modelParts = append(modelParts, part)
				}

				if len(modelParts) > 0 {
					modelName := strings.Join(modelParts, "-")
					// Store in format: provider.model-name
					fullModelName := fmt.Sprintf("%s.%s", provider, modelName)
					modelMap[fullModelName] = true
				}
			}
		}
	}

	// Convert map to slice
	models := make([]string, 0, len(modelMap))
	for model := range modelMap {
		models = append(models, model)
	}

	// Sort models alphabetically (groups by provider, then by model name)
	sort.Strings(models)

	if len(models) == 0 {
		return nil, fmt.Errorf("no models found for cross-region '%s'", crossRegion)
	}

	return models, nil
}
