package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/OlaHulleberg/clauderock/internal/config"
	"github.com/OlaHulleberg/clauderock/internal/usage"
)

// Launch executes Claude Code with the proper environment variables for Bedrock
func Launch(cfg *config.Config, mainModelID, fastModelID string, profileName string) error {
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
		fmt.Sprintf("ANTHROPIC_SMALL_FAST_MODEL=%s", fastModelID),
		fmt.Sprintf("AWS_PROFILE=%s", cfg.Profile),
		fmt.Sprintf("AWS_REGION=%s", cfg.Region),
	)

	// Execute claude and wait for it to complete
	cmd := exec.Command(claudePath)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run and wait for Claude Code to exit
	cmdErr := cmd.Run()
	exitCode := 0
	if cmdErr != nil {
		if exitError, ok := cmdErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return fmt.Errorf("failed to execute claude: %w", cmdErr)
		}
	}

	// Track session end
	sessionEnd := time.Now()

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

	// Return the exit code from Claude Code
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}
