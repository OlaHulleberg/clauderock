package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/OlaHulleberg/clauderock/internal/config"
)

// Launch executes Claude Code with the proper environment variables for Bedrock
func Launch(cfg *config.Config, mainModelID, fastModelID string) error {
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

	// Execute claude, replacing the current process
	// This ensures claude runs in the foreground and inherits all I/O
	if err := syscall.Exec(claudePath, []string{"claude"}, env); err != nil {
		return fmt.Errorf("failed to execute claude: %w", err)
	}

	// This line should never be reached if syscall.Exec succeeds
	return nil
}
