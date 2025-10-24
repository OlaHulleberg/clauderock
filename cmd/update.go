package cmd

import (
	"github.com/OlaHulleberg/clauderock/internal/updater"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for updates and install if available",
	RunE: func(cmd *cobra.Command, args []string) error {
		return updater.Update(Version)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
