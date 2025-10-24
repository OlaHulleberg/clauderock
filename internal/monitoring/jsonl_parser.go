package monitoring

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ClaudeMessage represents a message from Claude Code's JSONL file
type ClaudeMessage struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Message   struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// APICall represents a single API call extracted from JSONL
type APICall struct {
	Timestamp           time.Time
	Model               string
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
}

// SessionMetrics contains aggregated metrics for a session
type SessionMetrics struct {
	SessionUUID         string
	TotalRequests       int
	TotalInputTokens    int64
	TotalOutputTokens   int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	AvgTPM              float64
	PeakTPM             float64
	P95TPM              float64
	AvgRPM              float64
	PeakRPM             float64
	P95RPM              float64
	CacheHitRate        float64
	APICalls            []APICall
}

// FindSessionJSONL finds the JSONL file for a session based on working directory and start time
func FindSessionJSONL(workingDir string, sessionStart time.Time) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Encode working directory to Claude Code's format
	// Replace "/" with "-" (keep the leading dash - it represents root "/")
	encodedDir := strings.ReplaceAll(workingDir, "/", "-")

	projectDir := filepath.Join(home, ".claude", "projects", encodedDir)

	// Check if directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return "", fmt.Errorf("project directory not found: %s", projectDir)
	}

	// Find all JSONL files in the directory
	files, err := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
	if err != nil {
		return "", fmt.Errorf("failed to glob JSONL files: %w", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no JSONL files found in %s", projectDir)
	}

	// If only one file, return it
	if len(files) == 1 {
		return files[0], nil
	}

	// Find the file with modification time closest to session start
	type fileWithTime struct {
		path    string
		modTime time.Time
	}

	var filesWithTime []fileWithTime
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		// Only consider files created at or after session start
		// This prevents accidentally picking up old sessions from regular claude runs
		if info.ModTime().Before(sessionStart) {
			continue
		}
		filesWithTime = append(filesWithTime, fileWithTime{
			path:    file,
			modTime: info.ModTime(),
		})
	}

	if len(filesWithTime) == 0 {
		return "", fmt.Errorf("no readable JSONL files found")
	}

	// Sort by modification time and find closest to session start
	sort.Slice(filesWithTime, func(i, j int) bool {
		diffI := filesWithTime[i].modTime.Sub(sessionStart)
		diffJ := filesWithTime[j].modTime.Sub(sessionStart)
		if diffI < 0 {
			diffI = -diffI
		}
		if diffJ < 0 {
			diffJ = -diffJ
		}
		return diffI < diffJ
	})

	return filesWithTime[0].path, nil
}

// ParseSessionJSONL parses a JSONL file and extracts session metrics
func ParseSessionJSONL(jsonlPath string) (*SessionMetrics, error) {
	file, err := os.Open(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSONL file: %w", err)
	}
	defer file.Close()

	metrics := &SessionMetrics{
		APICalls: []APICall{},
	}

	// Extract session UUID from filename
	base := filepath.Base(jsonlPath)
	metrics.SessionUUID = strings.TrimSuffix(base, ".jsonl")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var msg ClaudeMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			// Skip malformed lines
			continue
		}

		// Only process assistant messages (these have usage data)
		if msg.Type != "assistant" {
			continue
		}

		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339, msg.Timestamp)
		if err != nil {
			continue
		}

		// Extract API call data
		apiCall := APICall{
			Timestamp:           timestamp,
			Model:               msg.Message.Model,
			InputTokens:         msg.Message.Usage.InputTokens,
			OutputTokens:        msg.Message.Usage.OutputTokens,
			CacheReadTokens:     msg.Message.Usage.CacheReadInputTokens,
			CacheCreationTokens: msg.Message.Usage.CacheCreationInputTokens,
		}

		metrics.APICalls = append(metrics.APICalls, apiCall)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading JSONL file: %w", err)
	}

	if len(metrics.APICalls) == 0 {
		return metrics, nil // Empty session, no API calls
	}

	// Calculate aggregated metrics
	calculateMetrics(metrics)

	return metrics, nil
}

func calculateMetrics(metrics *SessionMetrics) {
	if len(metrics.APICalls) == 0 {
		return
	}

	// Calculate totals
	metrics.TotalRequests = len(metrics.APICalls)
	for _, call := range metrics.APICalls {
		metrics.TotalInputTokens += call.InputTokens
		metrics.TotalOutputTokens += call.OutputTokens
		metrics.CacheReadTokens += call.CacheReadTokens
		metrics.CacheCreationTokens += call.CacheCreationTokens
	}

	// Calculate session duration from first to last API call
	firstCall := metrics.APICalls[0].Timestamp
	lastCall := metrics.APICalls[len(metrics.APICalls)-1].Timestamp
	duration := lastCall.Sub(firstCall)
	durationMinutes := duration.Minutes()

	// Handle very short sessions
	if durationMinutes < 0.01 {
		durationMinutes = 0.01
	}

	// Calculate TPM using AWS Bedrock's formula: Input + Output + CacheCreation
	// (CacheRead tokens are NOT counted - they save you tokens!)
	totalTokens := metrics.TotalInputTokens + metrics.TotalOutputTokens + metrics.CacheCreationTokens

	// Calculate average TPM and RPM
	metrics.AvgTPM = float64(totalTokens) / durationMinutes
	metrics.AvgRPM = float64(metrics.TotalRequests) / durationMinutes

	// Calculate peak and P95 TPM/RPM
	metrics.PeakTPM, metrics.P95TPM = calculatePeakAndP95Tokens(metrics.APICalls)
	metrics.PeakRPM, metrics.P95RPM = calculatePeakAndP95Requests(metrics.APICalls)

	// Calculate cache hit rate
	totalInputTokensIncludingCache := metrics.TotalInputTokens + metrics.CacheReadTokens
	if totalInputTokensIncludingCache > 0 {
		metrics.CacheHitRate = float64(metrics.CacheReadTokens) / float64(totalInputTokensIncludingCache) * 100.0
	}
}

// calculatePeakAndP95Tokens calculates peak and P95 TPM using 1-minute rolling windows
func calculatePeakAndP95Tokens(calls []APICall) (float64, float64) {
	if len(calls) == 0 {
		return 0, 0
	}

	// Group API calls into 1-minute buckets
	buckets := make(map[int64]int64)
	for _, call := range calls {
		bucket := call.Timestamp.Unix() / 60 // 1-minute buckets
		// AWS formula: Input + Output + CacheCreation (CacheRead tokens don't count)
		tokens := call.InputTokens + call.OutputTokens + call.CacheCreationTokens
		buckets[bucket] += tokens
	}

	// Extract bucket values
	var values []float64
	for _, tokens := range buckets {
		values = append(values, float64(tokens))
	}

	if len(values) == 0 {
		return 0, 0
	}

	// Sort for P95 calculation
	sort.Float64s(values)

	// Peak is the max value
	peak := values[len(values)-1]

	// P95 is the 95th percentile
	p95Index := int(float64(len(values)) * 0.95)
	if p95Index >= len(values) {
		p95Index = len(values) - 1
	}
	p95 := values[p95Index]

	return peak, p95
}

// calculatePeakAndP95Requests calculates peak and P95 RPM using 1-minute rolling windows
func calculatePeakAndP95Requests(calls []APICall) (float64, float64) {
	if len(calls) == 0 {
		return 0, 0
	}

	// Group API calls into 1-minute buckets
	buckets := make(map[int64]int)
	for _, call := range calls {
		bucket := call.Timestamp.Unix() / 60 // 1-minute buckets
		buckets[bucket]++
	}

	// Extract bucket values
	var values []float64
	for _, count := range buckets {
		values = append(values, float64(count))
	}

	if len(values) == 0 {
		return 0, 0
	}

	// Sort for P95 calculation
	sort.Float64s(values)

	// Peak is the max value
	peak := values[len(values)-1]

	// P95 is the 95th percentile
	p95Index := int(float64(len(values)) * 0.95)
	if p95Index >= len(values) {
		p95Index = len(values) - 1
	}
	p95 := values[p95Index]

	return peak, p95
}
