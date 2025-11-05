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

// ModelInfo contains detailed model information
type ModelInfo struct {
	Name     string // e.g., "anthropic.claude-sonnet-4-5"
	Provider string // e.g., "anthropic"
	Model    string // e.g., "claude-sonnet-4-5"
}

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

// extractModelNameFromVersion removes version suffixes from a model string
// Input: "claude-sonnet-4-5-20250929-v1:0"
// Output: "claude-sonnet-4-5"
func extractModelNameFromVersion(modelWithVersion string) string {
	parts := strings.Split(modelWithVersion, "-")
	var modelParts []string
	for _, part := range parts {
		// Stop if we hit a date pattern (8 digits) or version pattern
		if len(part) == 8 || strings.HasPrefix(part, "v") || strings.Contains(part, ":") {
			break
		}
		modelParts = append(modelParts, part)
	}
	return strings.Join(modelParts, "-")
}

// parseProfileID extracts provider and model name from a profile ID
// Input: "global.anthropic.claude-sonnet-4-5-20250929-v1:0", "global"
// Output: "anthropic", "claude-sonnet-4-5", true
func parseProfileID(profileID, crossRegionPrefix string) (provider, modelName string, ok bool) {
	if !strings.HasPrefix(profileID, crossRegionPrefix+".") {
		return "", "", false
	}

	// Remove cross-region prefix
	remaining := strings.TrimPrefix(profileID, crossRegionPrefix+".")

	// Split on first dot to separate provider from rest
	firstDotIndex := strings.Index(remaining, ".")
	if firstDotIndex == -1 {
		return "", "", false
	}

	provider = remaining[:firstDotIndex]
	modelWithVersion := remaining[firstDotIndex+1:]

	// Extract model name without version using helper
	modelName = extractModelNameFromVersion(modelWithVersion)
	if modelName == "" {
		return "", "", false
	}

	return provider, modelName, true
}

// parseModelName splits a model name in format "provider.model-name" into parts
// Returns provider, modelName, and ok flag
// Input: "anthropic.claude-sonnet-4-5" → "anthropic", "claude-sonnet-4-5", true
// Input: "invalid" → "", "", false
func parseModelName(fullModelName string) (provider, modelName string, ok bool) {
	parts := strings.SplitN(fullModelName, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// IsFullProfileID checks if a string is a full profile ID
// Input: "global.anthropic.claude-sonnet-4-5-20250929-v1:0" → true
// Input: "anthropic.claude-sonnet-4-5" → false
func IsFullProfileID(id string) bool {
	parts := strings.SplitN(id, ".", 2)
	if len(parts) < 2 {
		return false
	}
	crossRegions := map[string]bool{"us": true, "eu": true, "global": true}
	return crossRegions[parts[0]]
}

// ExtractFriendlyModelName extracts friendly model name from full profile ID
// Input: "global.anthropic.claude-sonnet-4-5-20250929-v1:0"
// Output: "anthropic.claude-sonnet-4-5"
func ExtractFriendlyModelName(profileID string) string {
	// If it's not a full profile ID, return as-is
	if !IsFullProfileID(profileID) {
		return profileID
	}

	// Remove cross-region prefix (us., eu., global.)
	parts := strings.SplitN(profileID, ".", 2)
	if len(parts) != 2 {
		return profileID
	}

	remaining := parts[1]

	// Split provider from rest
	firstDotIndex := strings.Index(remaining, ".")
	if firstDotIndex == -1 {
		return remaining
	}

	provider := remaining[:firstDotIndex]
	modelWithVersion := remaining[firstDotIndex+1:]

	// Extract model name without version using helper
	modelName := extractModelNameFromVersion(modelWithVersion)
	if modelName != "" {
		return fmt.Sprintf("%s.%s", provider, modelName)
	}

	return profileID
}

// ResolveModelToProfileID resolves a friendly model name to a full profile ID
// Input: "anthropic.claude-sonnet-4-5" with profile, region, crossRegion
// Output: "global.anthropic.claude-sonnet-4-5-20250929-v1:0"
func ResolveModelToProfileID(awsProfile, region, crossRegion, model string) (string, error) {
	// If model already looks like a full profile ID, return it
	if IsFullProfileID(model) {
		return model, nil
	}

	ctx := context.Background()

	// Load AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithSharedConfigProfile(awsProfile),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Bedrock client
	client := bedrock.NewFromConfig(awsCfg)

	// List cross-region inference profiles
	result, err := client.ListInferenceProfiles(ctx, &bedrock.ListInferenceProfilesInput{
		TypeEquals: types.InferenceProfileTypeSystemDefined,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list inference profiles: %w", err)
	}

	// Find matching profile
	profileID, err := findMatchingProfile(result.InferenceProfileSummaries, crossRegion, model)
	if err != nil {
		return "", fmt.Errorf("%w\nAvailable profiles:\n%s",
			err, formatAvailableProfiles(result.InferenceProfileSummaries))
	}

	return profileID, nil
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

	for _, profile := range result.InferenceProfileSummaries {
		if profile.InferenceProfileId != nil {
			profileID := aws.ToString(profile.InferenceProfileId)

			// Use helper to parse profile ID
			provider, modelName, ok := parseProfileID(profileID, crossRegion)
			if ok {
				fullModelName := fmt.Sprintf("%s.%s", provider, modelName)
				modelMap[fullModelName] = true
			}
		}
	}

	// Convert map to slice
	models := make([]string, 0, len(modelMap))
	for model := range modelMap {
		models = append(models, model)
	}

	// Sort models by provider first, then model name
	sort.Slice(models, func(i, j int) bool {
		providerI, modelI, okI := parseModelName(models[i])
		providerJ, modelJ, okJ := parseModelName(models[j])

		if !okI || !okJ {
			return models[i] < models[j]
		}

		// Compare provider first
		if providerI != providerJ {
			return providerI < providerJ
		}

		// If same provider, compare model name
		return modelI < modelJ
	})

	if len(models) == 0 {
		return nil, fmt.Errorf("no models found for cross-region '%s'", crossRegion)
	}

	return models, nil
}

// IsRecommendedModel returns true if the model is recommended for the given context
func IsRecommendedModel(model, context string) bool {
	switch context {
	case "main":
		return model == "anthropic.claude-sonnet-4-5"
	case "fast":
		return model == "anthropic.claude-haiku-4-5"
	default:
		return false
	}
}

// SortModelsWithRecommendation sorts models with the recommended model for the context at the top
func SortModelsWithRecommendation(models []string, context string) []string {
	sorted := make([]string, len(models))
	copy(sorted, models)

	sort.Slice(sorted, func(i, j int) bool {
		iRecommended := IsRecommendedModel(sorted[i], context)
		jRecommended := IsRecommendedModel(sorted[j], context)

		// Recommended model comes first
		if iRecommended && !jRecommended {
			return true
		}
		if !iRecommended && jRecommended {
			return false
		}

		// Otherwise, sort by provider then model name
		providerI, modelI, okI := parseModelName(sorted[i])
		providerJ, modelJ, okJ := parseModelName(sorted[j])

		if !okI || !okJ {
			return sorted[i] < sorted[j]
		}

		// Compare provider first
		if providerI != providerJ {
			return providerI < providerJ
		}

		// If same provider, compare model name
		return modelI < modelJ
	})

	return sorted
}

// ValidateProfileIDs validates that the given profile IDs exist in AWS Bedrock
func ValidateProfileIDs(awsProfile, region string, profileIDs ...string) error {
	ctx := context.Background()

	// Load AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithSharedConfigProfile(awsProfile),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Bedrock client
	client := bedrock.NewFromConfig(awsCfg)

	// List all inference profiles
	result, err := client.ListInferenceProfiles(ctx, &bedrock.ListInferenceProfilesInput{
		TypeEquals: types.InferenceProfileTypeSystemDefined,
	})
	if err != nil {
		return fmt.Errorf("failed to list inference profiles: %w", err)
	}

	// Build a set of valid profile IDs
	validProfiles := make(map[string]bool)
	for _, profile := range result.InferenceProfileSummaries {
		if profile.InferenceProfileId != nil {
			validProfiles[aws.ToString(profile.InferenceProfileId)] = true
		}
	}

	// Validate each requested profile ID
	for _, profileID := range profileIDs {
		if !validProfiles[profileID] {
			return fmt.Errorf("profile ID '%s' does not exist in AWS Bedrock\nRun 'clauderock manage models list' to see available models", profileID)
		}
	}

	return nil
}

// GetAvailableModelsDetailed fetches available models from Bedrock with detailed information
func GetAvailableModelsDetailed(profile, region, crossRegion string) ([]ModelInfo, error) {
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
	modelMap := make(map[string]ModelInfo)

	for _, profile := range result.InferenceProfileSummaries {
		if profile.InferenceProfileId != nil {
			profileID := aws.ToString(profile.InferenceProfileId)

			// Use helper to parse profile ID
			provider, modelName, ok := parseProfileID(profileID, crossRegion)
			if ok {
				fullModelName := fmt.Sprintf("%s.%s", provider, modelName)
				modelMap[fullModelName] = ModelInfo{
					Name:     fullModelName,
					Provider: provider,
					Model:    modelName,
				}
			}
		}
	}

	// Convert map to slice
	models := make([]ModelInfo, 0, len(modelMap))
	for _, model := range modelMap {
		models = append(models, model)
	}

	// Sort models alphabetically (groups by provider, then by model name)
	sort.Slice(models, func(i, j int) bool {
		return models[i].Name < models[j].Name
	})

	if len(models) == 0 {
		return nil, fmt.Errorf("no models found for cross-region '%s'", crossRegion)
	}

	return models, nil
}
