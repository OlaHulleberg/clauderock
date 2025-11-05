package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/OlaHulleberg/clauderock/internal/aws"
	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/usage"
)

// Launch executes Claude Code with the proper environment variables for Bedrock
func Launch(cfg *config.Config, mainModelID, fastModelID string, profileName string, args []string) error {
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

	// Prepare environment variables
	env := os.Environ()
	env = append(env,
		"CLAUDE_CODE_USE_BEDROCK=1",
		fmt.Sprintf("ANTHROPIC_MODEL=%s", mainModelID),
		fmt.Sprintf("ANTHROPIC_DEFAULT_HAIKU_MODEL=%s", fastModelID),
		fmt.Sprintf("AWS_PROFILE=%s", cfg.Profile),
		fmt.Sprintf("AWS_REGION=%s", cfg.Region),
	)

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

	// Validate model profile IDs in background
	validationDone := make(chan error, 1)
	go func() {
		validationDone <- aws.ValidateProfileIDs(cfg.Profile, cfg.Region, mainModelID, fastModelID)
	}()

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
		trackSession(cfg, mainModelID, fastModelID, profileName, cwd, sessionStart, sessionEnd, exitCode)

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
		trackSession(cfg, mainModelID, fastModelID, profileName, cwd, sessionStart, sessionEnd, exitCode)

		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	}
}

func trackSession(cfg *config.Config, mainModelID, fastModelID, profileName, cwd string, sessionStart, sessionEnd time.Time, exitCode int) {
	// Track usage after Claude Code exits
	tracker, err := usage.NewTracker()
	if err == nil {
		// Track session with timing information
		trackErr := tracker.TrackSession(usage.SessionInfo{
			StartTime:          sessionStart,
			EndTime:            sessionEnd,
			ProfileName:        profileName,
			WorkingDirectory:   cwd,
			AWSProfile:         cfg.Profile,
			Region:             cfg.Region,
			CrossRegion:        cfg.CrossRegion,
			Model:              cfg.Model,
			ModelProfileID:     mainModelID,
			FastModel:          cfg.FastModel,
			FastModelProfileID: fastModelID,
			ExitCode:           exitCode,
		})
		tracker.Close()
		if trackErr != nil {
			fmt.Printf("Warning: failed to track session: %v\n", trackErr)
		}
	}
}
