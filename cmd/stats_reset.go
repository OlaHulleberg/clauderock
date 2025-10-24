package cmd

import (
	"fmt"

	"github.com/OlaHulleberg/clauderock/internal/interactive"
	"github.com/OlaHulleberg/clauderock/internal/usage"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	resetForce bool
)

var statsResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset usage statistics (DESTRUCTIVE)",
	Long: `Reset and clear all usage statistics from the database.

WARNING: This is a destructive operation that permanently deletes all session data.

This command will delete all session records including:
  - Token usage data
  - TPM/RPM metrics
  - Cache hit rates
  - Session durations

Examples:
  clauderock stats reset          # Reset all data (with confirmation)
  clauderock stats reset --force  # Skip confirmation (dangerous!)`,
	RunE: runStatsReset,
}

func init() {
	statsCmd.AddCommand(statsResetCmd)

	statsResetCmd.Flags().BoolVar(&resetForce, "force", false, "Skip confirmation prompt")
}

func runStatsReset(cmd *cobra.Command, args []string) error {
	// Open database
	db, err := usage.NewDatabase()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Count records
	sessionCount, err := db.CountSessions()
	if err != nil {
		return err
	}

	// Check if there's any data to delete
	if sessionCount == 0 {
		fmt.Println(mutedStyle.Render("No usage data found. Database is already empty."))
		return nil
	}

	// Build details for confirmation
	details := []string{
		fmt.Sprintf("Sessions: %d records", sessionCount),
		"",
		"This includes all token usage, TPM/RPM metrics, and cache data.",
	}

	// Confirm unless --force is used
	if !resetForce {
		confirmed, err := interactive.Confirm(
			"WARNING: This will permanently delete ALL session data",
			"This action cannot be undone. All statistics and history will be lost.",
			details,
		)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !confirmed {
			fmt.Println(mutedStyle.Render("Operation cancelled."))
			return nil
		}
	}

	// Perform deletion
	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))

	if err := db.ClearSessions(); err != nil {
		return err
	}
	fmt.Println(successStyle.Render("✓") + " Deleted " + fmt.Sprintf("%d session records", sessionCount))

	return nil
}
