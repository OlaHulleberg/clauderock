package aws

import (
	"context"
	"fmt"
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
	// Expected format: {cross-region}.anthropic.{model}*
	// Example: global.anthropic.claude-sonnet-4-5-20250929-v1:0
	prefix := fmt.Sprintf("%s.anthropic.%s", crossRegion, model)

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
