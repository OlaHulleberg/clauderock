package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/OlaHulleberg/clauderock/internal/api"
	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/keyring"
	"github.com/OlaHulleberg/clauderock/internal/usage"
)

// Launch executes Claude Code with the proper environment variables (Bedrock or API)
func Launch(cfg *config.Config, mainModelID, fastModelID, heavyModelID string, profileName string, args []string) error {
	// Get current working directory for session tracking
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	// Track session start
	sessionStart := time.Now()

	// Find claude binary
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude binary not found in PATH: %w", err)
	}

	// Prepare environment variables based on profile type
	env := os.Environ()

	// Setup validation channel
	validationDone := make(chan error, 1)

	if cfg.ProfileType == "bedrock" {
		// Bedrock mode: Use AWS credentials
		env = append(env,
			"CLAUDE_CODE_USE_BEDROCK=1",
			fmt.Sprintf("ANTHROPIC_DEFAULT_SONNET_MODEL=%s", mainModelID),
			fmt.Sprintf("ANTHROPIC_DEFAULT_HAIKU_MODEL=%s", fastModelID),
			fmt.Sprintf("ANTHROPIC_DEFAULT_OPUS_MODEL=%s", heavyModelID),
			fmt.Sprintf("AWS_PROFILE=%s", cfg.Profile),
			fmt.Sprintf("AWS_REGION=%s", cfg.Region),
		)

		// Validate model profile IDs in background
		go func() {
			validationDone <- aws.ValidateProfileIDs(cfg.Profile, cfg.Region, mainModelID, fastModelID, heavyModelID)
		}()

	} else if cfg.ProfileType == "api" {
		// API mode: Use API key from keychain
		apiKey, err := keyring.Get(cfg.APIKeyID)
		if err != nil {
			return fmt.Errorf("failed to retrieve API key from keychain: %w", err)
		}

		// Normalize base URL
		normalizedURL := api.NormalizeBaseURL(cfg.BaseURL)

		env = append(env,
			fmt.Sprintf("ANTHROPIC_API_KEY=%s", apiKey),
			fmt.Sprintf("ANTHROPIC_BASE_URL=%s", normalizedURL),
		)

		// Validate models via API in background
		go func() {
			validationDone <- api.ValidateModels(cfg.BaseURL, apiKey, mainModelID, fastModelID, heavyModelID)
		}()
	} else {
		return fmt.Errorf("unsupported profile type: %s", cfg.ProfileType)
	}

	// Execute claude with passthrough args
	cmd := exec.Command(claudePath, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start Claude Code (non-blocking)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude: %w", err)
	}

	// Wait for either validation to complete or Claude Code to exit
	cmdDone := make(chan error, 1)
	go func() {
		cmdDone <- cmd.Wait()
	}()

	// Check validation result
	select {
	case validationErr := <-validationDone:
		if validationErr != nil {
			// Validation failed - kill Claude Code and return error
			cmd.Process.Kill()
			// Wait for process to be killed
			<-cmdDone
			return fmt.Errorf("invalid model configuration: %w", validationErr)
		}
		// Validation succeeded - wait for Claude Code to complete normally
		cmdErr := <-cmdDone
		exitCode := 0
		if cmdErr != nil {
			if exitError, ok := cmdErr.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			} else {
				return fmt.Errorf("claude exited with error: %w", cmdErr)
			}
		}

		// Track session end and return
		sessionEnd := time.Now()
		trackSession(cfg, mainModelID, fastModelID, heavyModelID, profileName, cwd, sessionStart, sessionEnd, exitCode)

		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil

	case cmdErr := <-cmdDone:
		// Claude Code exited before validation completed
		exitCode := 0
		if cmdErr != nil {
			if exitError, ok := cmdErr.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			} else {
				return fmt.Errorf("claude exited with error: %w", cmdErr)
			}
		}

		// Track session end and return
		sessionEnd := time.Now()
		trackSession(cfg, mainModelID, fastModelID, heavyModelID, profileName, cwd, sessionStart, sessionEnd, exitCode)

		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	}
}

func trackSession(cfg *config.Config, mainModelID, fastModelID, heavyModelID, profileName, cwd string, sessionStart, sessionEnd time.Time, exitCode int) {
	// Track usage after Claude Code exits
	tracker, err := usage.NewTracker()
	if err == nil {
		// Track session with timing information
		trackErr := tracker.TrackSession(usage.SessionInfo{
			StartTime:           sessionStart,
			EndTime:             sessionEnd,
			ProfileName:         profileName,
			WorkingDirectory:    cwd,
			AWSProfile:          cfg.Profile,
			Region:              cfg.Region,
			CrossRegion:         cfg.CrossRegion,
			Model:               cfg.Model,
			ModelProfileID:      mainModelID,
			FastModel:           cfg.FastModel,
			FastModelProfileID:  fastModelID,
			HeavyModel:          cfg.HeavyModel,
			HeavyModelProfileID: heavyModelID,
			ExitCode:            exitCode,
		})
		tracker.Close()
		if trackErr != nil {
			fmt.Printf("Warning: failed to track session: %v\n", trackErr)
		}
	}
}
