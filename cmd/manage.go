package cmd

import (
	"github.com/spf13/cobra"
)

var manageCmd = &cobra.Command{
	Use:   "manage",
	Short: "Manage clauderock configuration and settings",
	Long:  `Manage clauderock configuration, profiles, models, stats, and updates.`,
}

func init() {
	rootCmd.AddCommand(manageCmd)

	// Add all management subcommands
	manageCmd.AddCommand(configCmd)
	manageCmd.AddCommand(profilesCmd)
	manageCmd.AddCommand(modelsCmd)
	manageCmd.AddCommand(statsCmd)
	manageCmd.AddCommand(updateCmd)
	manageCmd.AddCommand(versionCmd)
}
