package cmd

import (
	"fmt"

	"github.com/OlaHulleberg/clauderock/internal/profiles"
	"github.com/spf13/cobra"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List all available profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := profiles.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create profile manager: %w", err)
		}

		profileList, err := mgr.List()
		if err != nil {
			return fmt.Errorf("failed to list profiles: %w", err)
		}

		current, err := mgr.GetCurrent()
		if err != nil {
			return fmt.Errorf("failed to get current profile: %w", err)
		}

		if len(profileList) == 0 {
			fmt.Println("No profiles found")
			return nil
		}

		fmt.Println("Available profiles:")
		for _, name := range profileList {
			if name == current {
				fmt.Printf("  * %s (active)\n", name)
			} else {
				fmt.Printf("    %s\n", name)
			}
		}

		return nil
	},
}

var profileSaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save current configuration as a named profile",
	Long: `Save current configuration as a named profile.

Example:
  clauderock config save --name work-dev`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName, _ := cmd.Flags().GetString("name")
		if profileName == "" {
			return fmt.Errorf("profile name is required (use --name)")
		}

		mgr, err := profiles.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create profile manager: %w", err)
		}

		// Load current config
		cfg, err := mgr.GetCurrentConfig(Version)
		if err != nil {
			return fmt.Errorf("failed to load current config: %w", err)
		}

		// Check if profile already exists
		if mgr.Exists(profileName) {
			return fmt.Errorf("profile '%s' already exists, use 'config delete' first or choose a different name", profileName)
		}

		// Save as new profile
		if err := mgr.Save(profileName, cfg); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}

		fmt.Printf("Saved current configuration as profile '%s'\n", profileName)
		return nil
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a profile",
	Long: `Delete a named profile.

Example:
  clauderock config delete --name old-project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName, _ := cmd.Flags().GetString("name")
		if profileName == "" {
			return fmt.Errorf("profile name is required (use --name)")
		}

		mgr, err := profiles.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create profile manager: %w", err)
		}

		if err := mgr.Delete(profileName); err != nil {
			return err
		}

		fmt.Printf("Deleted profile '%s'\n", profileName)
		return nil
	},
}

var profileRenameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Rename a profile",
	Long: `Rename a profile.

Example:
  clauderock config rename --from old-name --to new-name`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fromName, _ := cmd.Flags().GetString("from")
		toName, _ := cmd.Flags().GetString("to")

		if fromName == "" || toName == "" {
			return fmt.Errorf("both --from and --to are required")
		}

		mgr, err := profiles.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create profile manager: %w", err)
		}

		if err := mgr.Rename(fromName, toName); err != nil {
			return err
		}

		fmt.Printf("Renamed profile '%s' to '%s'\n", fromName, toName)
		return nil
	},
}

var profileCopyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy a profile",
	Long: `Copy a profile to a new name.

Example:
  clauderock config copy --from template --to new-project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fromName, _ := cmd.Flags().GetString("from")
		toName, _ := cmd.Flags().GetString("to")

		if fromName == "" || toName == "" {
			return fmt.Errorf("both --from and --to are required")
		}

		mgr, err := profiles.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create profile manager: %w", err)
		}

		if err := mgr.Copy(fromName, toName); err != nil {
			return err
		}

		fmt.Printf("Copied profile '%s' to '%s'\n", fromName, toName)
		return nil
	},
}

var profileSwitchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch to a different profile",
	Long: `Switch the active profile.

Example:
  clauderock config switch --name work-dev`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName, _ := cmd.Flags().GetString("name")
		if profileName == "" {
			return fmt.Errorf("profile name is required (use --name)")
		}

		mgr, err := profiles.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create profile manager: %w", err)
		}

		if err := mgr.SetCurrent(profileName); err != nil {
			return err
		}

		fmt.Printf("Switched to profile '%s'\n", profileName)
		return nil
	},
}

func init() {
	// Add profiles command to config
	configCmd.AddCommand(profilesCmd)

	// Add profile management commands
	profileSaveCmd.Flags().String("name", "", "Name for the profile")
	configCmd.AddCommand(profileSaveCmd)

	profileDeleteCmd.Flags().String("name", "", "Name of the profile to delete")
	configCmd.AddCommand(profileDeleteCmd)

	profileRenameCmd.Flags().String("from", "", "Current name of the profile")
	profileRenameCmd.Flags().String("to", "", "New name for the profile")
	configCmd.AddCommand(profileRenameCmd)

	profileCopyCmd.Flags().String("from", "", "Name of the profile to copy")
	profileCopyCmd.Flags().String("to", "", "Name for the new profile")
	configCmd.AddCommand(profileCopyCmd)

	profileSwitchCmd.Flags().String("name", "", "Name of the profile to switch to")
	configCmd.AddCommand(profileSwitchCmd)
}
