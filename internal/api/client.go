package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ModelInfo represents a model from the API
type ModelInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Recommended []string `json:"recommended,omitempty"`
}

// ModelsResponse represents the response from /v1/models endpoint
type ModelsResponse struct {
	Data []ModelInfo `json:"data"`
}

// NormalizeBaseURL ensures the base URL has a protocol (defaults to https://)
// If user explicitly provided http:// or https://, keeps it as-is
func NormalizeBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)

	// If already has protocol, return as-is
	if strings.HasPrefix(baseURL, "http://") || strings.HasPrefix(baseURL, "https://") {
		return strings.TrimSuffix(baseURL, "/")
	}

	// Default to https://
	return "https://" + strings.TrimSuffix(baseURL, "/")
}

// FetchAvailableModels fetches available models from the API's /v1/models endpoint
func FetchAvailableModels(baseURL, apiKey string) ([]ModelInfo, error) {
	normalizedURL := NormalizeBaseURL(baseURL)
	endpoint := normalizedURL + "/v1/models"

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Authorization header with Bearer token (OpenRouter style)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no models available from API")
	}

	return result.Data, nil
}

// ValidateModels validates that the given model IDs exist in the API
func ValidateModels(baseURL, apiKey string, modelIDs ...string) error {
	models, err := FetchAvailableModels(baseURL, apiKey)
	if err != nil {
		return fmt.Errorf("failed to fetch models for validation: %w", err)
	}

	// Build a set of available model IDs
	availableModels := make(map[string]bool)
	for _, model := range models {
		availableModels[model.ID] = true
	}

	// Validate each provided model ID
	var missing []string
	for _, id := range modelIDs {
		if !availableModels[id] {
			missing = append(missing, id)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("models not available: %v", missing)
	}

	return nil
}

// IsRecommendedModel returns true if the model is recommended for the given context
// Checks the model's Recommended field for matching context
func IsRecommendedModel(model ModelInfo, context string) bool {
	// Map clauderock context to API context names
	contextMap := map[string]string{
		"main":  "code",
		"fast":  "code-fast",
		"heavy": "code-heavy",
	}

	apiContext := contextMap[context]
	for _, rec := range model.Recommended {
		if rec == apiContext {
			return true
		}
	}
	return false
}

// ExtractFriendlyName extracts a display name from the model ID
// e.g., "anthropic/claude-sonnet-4-5" -> "Claude Sonnet 4.5"
func ExtractFriendlyName(modelID string) string {
	// Remove provider prefix if present
	parts := strings.Split(modelID, "/")
	name := modelID
	if len(parts) > 1 {
		name = parts[len(parts)-1]
	}

	// Replace hyphens with spaces and title case
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.Title(name)

	return name
}
