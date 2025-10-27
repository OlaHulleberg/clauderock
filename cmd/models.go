package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/profiles"
	"github.com/spf13/cobra"
)

var (
	providerFilter     string
	crossRegionFilter  string
	profileFilterModel string
	regionFilter       string
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Manage and list available models",
	Long:  `Commands for listing and managing available models from AWS Bedrock.`,
}

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available models from AWS Bedrock",
	Long: `List available models from AWS Bedrock.

By default, uses settings from the current profile. You can override
specific settings using flags.

Examples:
  clauderock models list
  clauderock models list --provider anthropic
  clauderock models list --cross-region us
  clauderock models list --profile work-dev
  clauderock models list --region us-west-2 --cross-region global`,
	RunE: runModelsList,
}

func init() {
	// Registered by manage.go
	modelsCmd.AddCommand(modelsListCmd)

	modelsListCmd.Flags().StringVar(&providerFilter, "provider", "", "Filter by provider (e.g., anthropic, meta, amazon)")
	modelsListCmd.Flags().StringVar(&crossRegionFilter, "cross-region", "", "Override cross-region setting (us, eu, global)")
	modelsListCmd.Flags().StringVar(&profileFilterModel, "profile", "", "Use settings from a specific profile")
	modelsListCmd.Flags().StringVar(&regionFilter, "region", "", "Override AWS region")
}

func runModelsList(cmd *cobra.Command, args []string) error {
	// Load profile or use flags
	var awsProfile, region, crossRegion string

	mgr, err := profiles.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create profile manager: %w", err)
	}

	var cfg *config.Config

	if profileFilterModel != "" {
		// Load from specified profile
		cfg, err = mgr.Load(profileFilterModel)
		if err != nil {
			return fmt.Errorf("failed to load profile '%s': %w", profileFilterModel, err)
		}
	} else {
		// Use current profile
		cfg, err = mgr.GetCurrentConfig(Version)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	awsProfile = cfg.Profile
	region = cfg.Region
	crossRegion = cfg.CrossRegion

	// Override with flags if provided
	if regionFilter != "" {
		region = regionFilter
	}
	if crossRegionFilter != "" {
		crossRegion = crossRegionFilter
	}

	// Show what we're querying
	fmt.Printf("Fetching models from AWS Bedrock...\n")
	fmt.Printf("  Region: %s\n", region)
	fmt.Printf("  Cross-Region: %s\n", crossRegion)
	if providerFilter != "" {
		fmt.Printf("  Provider Filter: %s\n", providerFilter)
	}
	fmt.Println()

	// Fetch models
	models, err := aws.GetAvailableModelsDetailed(awsProfile, region, crossRegion)
	if err != nil {
		return fmt.Errorf("failed to fetch models: %w", err)
	}

	// Filter by provider if specified
	if providerFilter != "" {
		filtered := []aws.ModelInfo{}
		for _, m := range models {
			if strings.EqualFold(m.Provider, providerFilter) {
				filtered = append(filtered, m)
			}
		}
		models = filtered
	}

	if len(models) == 0 {
		fmt.Println("No models found matching the criteria.")
		return nil
	}

	// Group and display
	grouped := groupModelsByProvider(models)
	displayModels(grouped, region, crossRegion)

	return nil
}

func groupModelsByProvider(models []aws.ModelInfo) map[string][]aws.ModelInfo {
	grouped := make(map[string][]aws.ModelInfo)
	for _, m := range models {
		provider := strings.Title(m.Provider)
		grouped[provider] = append(grouped[provider], m)
	}
	return grouped
}

func displayModels(grouped map[string][]aws.ModelInfo, region, crossRegion string) {
	fmt.Printf("Available models in %s (%s cross-region):\n\n", region, crossRegion)

	// Sort provider names
	providers := make([]string, 0, len(grouped))
	for provider := range grouped {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	totalModels := 0
	for _, provider := range providers {
		models := grouped[provider]
		totalModels += len(models)

		fmt.Printf("%s:\n", provider)
		for _, m := range models {
			indicator := getModelIndicator(m.Model)
			fmt.Printf("  • %s%s\n", m.Name, indicator)
		}
		fmt.Println()
	}

	fmt.Printf("Found %d models across %d providers.\n", totalModels, len(providers))
}

func getModelIndicator(modelName string) string {
	lower := strings.ToLower(modelName)

	// Check for fast models
	if strings.Contains(lower, "haiku") {
		return " (fast)"
	}

	// Check for recommended models
	if strings.Contains(lower, "sonnet-4-5") || strings.Contains(lower, "sonnet-4.5") {
		return " (recommended)"
	}

	return ""
}
