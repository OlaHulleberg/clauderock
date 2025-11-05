package pricing

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ModelPrice struct {
	Provider   string
	Model      string
	InputCost  float64 // Cost per 1M input tokens
	OutputCost float64 // Cost per 1M output tokens
}

// PricingTable contains AWS Bedrock pricing as of October 2025
// Prices are per 1M tokens
var PricingTable = map[string]ModelPrice{
	"anthropic.claude-opus-4": {
		Provider:   "anthropic",
		Model:      "claude-opus-4",
		InputCost:  15.00,
		OutputCost: 75.00,
	},
	"anthropic.claude-sonnet-4-5": {
		Provider:   "anthropic",
		Model:      "claude-sonnet-4-5",
		InputCost:  3.00,
		OutputCost: 15.00,
	},
	"anthropic.claude-haiku-4-5": {
		Provider:   "anthropic",
		Model:      "claude-haiku-4-5",
		InputCost:  0.80,
		OutputCost: 4.00,
	},
	"anthropic.claude-sonnet-3-5": {
		Provider:   "anthropic",
		Model:      "claude-sonnet-3-5",
		InputCost:  3.00,
		OutputCost: 15.00,
	},
	"anthropic.claude-haiku-3-5": {
		Provider:   "anthropic",
		Model:      "claude-haiku-3-5",
		InputCost:  0.80,
		OutputCost: 4.00,
	},
	"meta.llama-3-2-90b": {
		Provider:   "meta",
		Model:      "llama-3-2-90b",
		InputCost:  2.65,
		OutputCost: 3.50,
	},
	"meta.llama-3-2-11b": {
		Provider:   "meta",
		Model:      "llama-3-2-11b",
		InputCost:  0.35,
		OutputCost: 0.40,
	},
	"amazon.titan-text-premier": {
		Provider:   "amazon",
		Model:      "titan-text-premier",
		InputCost:  0.50,
		OutputCost: 1.50,
	},
}

// GetModelPrice looks up pricing for a model
func GetModelPrice(model string) (ModelPrice, bool) {
	price, ok := PricingTable[model]
	return price, ok
}

// EstimateCostPerLaunch estimates average cost per launch
// This is a rough estimate based on typical usage patterns
func EstimateCostPerLaunch(model string) float64 {
	price, ok := GetModelPrice(model)
	if !ok {
		return 0.0
	}

	// Estimate average tokens per Claude Code session
	// These are rough estimates based on typical usage:
	// - Short sessions: ~5k input, ~2k output
	// - Medium sessions: ~20k input, ~8k output
	// - Long sessions: ~50k input, ~20k output
	// Average: ~25k input, ~10k output

	avgInputTokens := 25000.0
	avgOutputTokens := 10000.0

	// Calculate cost
	inputCost := (avgInputTokens / 1_000_000.0) * price.InputCost
	outputCost := (avgOutputTokens / 1_000_000.0) * price.OutputCost

	return inputCost + outputCost
}

// CalculateCost calculates exact cost given token counts
func CalculateCost(model string, inputTokens, outputTokens int64) float64 {
	price, ok := GetModelPrice(model)
	if !ok {
		return 0.0
	}

	inputCost := (float64(inputTokens) / 1_000_000.0) * price.InputCost
	outputCost := (float64(outputTokens) / 1_000_000.0) * price.OutputCost

	return inputCost + outputCost
}

// GetProviderName extracts provider name from model string
func GetProviderName(model string) string {
	parts := strings.SplitN(model, ".", 2)
	if len(parts) == 2 {
		caser := cases.Title(language.English)
		return caser.String(parts[0])
	}
	return "Unknown"
}
