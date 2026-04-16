package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

var membershipsCmd = &cobra.Command{
	Use:   "memberships",
	Short: "Manage account memberships",
}

var membershipsCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a user's membership in an account",
	Example: `  hppycli memberships create --user-id=user456 --account-id=acct123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, _ := cmd.Flags().GetString("account-id")
		if accountID == "" {
			accountID = configAccountID
		}
		if accountID == "" {
			return fmt.Errorf("--account-id is required (or set account_id in config)")
		}
		if err := models.ValidateID("account-id", accountID); err != nil {
			return err
		}
		userID, _ := cmd.Flags().GetString("user-id")
		if userID == "" {
			return fmt.Errorf("--user-id is required")
		}
		if err := models.ValidateID("user-id", userID); err != nil {
			return err
		}

		input := models.AccountMembershipCreateInput{
			AccountID: accountID,
			UserID:    userID,
		}
		if v, _ := cmd.Flags().GetString("role-id"); v != "" {
			for _, p := range strings.Split(v, ",") {
				id := strings.TrimSpace(p)
				if id == "" {
					continue
				}
				if err := models.ValidateID("role-id", id); err != nil {
					return err
				}
				input.RoleID = append(input.RoleID, id)
			}
		}

		membership, err := apiClient.AccountMembershipCreate(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("creating membership: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, membership)
	},
}

var membershipsActivateCmd = &cobra.Command{
	Use:     "activate",
	Short:   "Activate a user's membership in an account",
	Example: `  hppycli memberships activate --user-id=user456 --account-id=acct123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, _ := cmd.Flags().GetString("account-id")
		if accountID == "" {
			accountID = configAccountID
		}
		if accountID == "" {
			return fmt.Errorf("--account-id is required (or set account_id in config)")
		}
		if err := models.ValidateID("account-id", accountID); err != nil {
			return err
		}
		userID, _ := cmd.Flags().GetString("user-id")
		if userID == "" {
			return fmt.Errorf("--user-id is required")
		}
		if err := models.ValidateID("user-id", userID); err != nil {
			return err
		}

		membership, err := apiClient.AccountMembershipActivate(cmd.Context(), models.AccountMembershipActivateInput{
			AccountID: accountID,
			UserID:    userID,
		})
		if err != nil {
			return fmt.Errorf("activating membership: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, membership)
	},
}

var membershipsDeactivateCmd = &cobra.Command{
	Use:     "deactivate",
	Short:   "Deactivate a user's membership in an account",
	Example: `  hppycli memberships deactivate --user-id=user456 --account-id=acct123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, _ := cmd.Flags().GetString("account-id")
		if accountID == "" {
			accountID = configAccountID
		}
		if accountID == "" {
			return fmt.Errorf("--account-id is required (or set account_id in config)")
		}
		if err := models.ValidateID("account-id", accountID); err != nil {
			return err
		}
		userID, _ := cmd.Flags().GetString("user-id")
		if userID == "" {
			return fmt.Errorf("--user-id is required")
		}
		if err := models.ValidateID("user-id", userID); err != nil {
			return err
		}

		if err := confirmAction(cmd, "deactivate membership", os.Stdin); err != nil {
			return err
		}

		membership, err := apiClient.AccountMembershipDeactivate(cmd.Context(), models.AccountMembershipDeactivateInput{
			AccountID: accountID,
			UserID:    userID,
		})
		if err != nil {
			return fmt.Errorf("deactivating membership: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, membership)
	},
}

var membershipsSetRolesCmd = &cobra.Command{
	Use:     "set-roles",
	Short:   "Set the roles for a user's membership",
	Example: `  hppycli memberships set-roles --user-id=user456 --role-id=role1,role2 --account-id=acct123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, _ := cmd.Flags().GetString("account-id")
		if accountID == "" {
			accountID = configAccountID
		}
		if accountID == "" {
			return fmt.Errorf("--account-id is required (or set account_id in config)")
		}
		if err := models.ValidateID("account-id", accountID); err != nil {
			return err
		}
		userID, _ := cmd.Flags().GetString("user-id")
		if userID == "" {
			return fmt.Errorf("--user-id is required")
		}
		if err := models.ValidateID("user-id", userID); err != nil {
			return err
		}

		if err := confirmAction(cmd, "set membership roles", os.Stdin); err != nil {
			return err
		}

		input := models.AccountMembershipSetRolesInput{
			AccountID: accountID,
			UserID:    userID,
		}
		if v, _ := cmd.Flags().GetString("role-id"); v != "" {
			for _, p := range strings.Split(v, ",") {
				id := strings.TrimSpace(p)
				if id == "" {
					continue
				}
				if err := models.ValidateID("role-id", id); err != nil {
					return err
				}
				input.RoleID = append(input.RoleID, id)
			}
		}

		membership, err := apiClient.AccountMembershipSetRoles(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("setting membership roles: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, membership)
	},
}

func init() {
	// Create
	membershipsCreateCmd.Flags().String("account-id", "", "account ID (defaults from config)")
	membershipsCreateCmd.Flags().String("user-id", "", "user ID (required)")
	membershipsCreateCmd.Flags().String("role-id", "", "comma-separated role IDs")
	membershipsCmd.AddCommand(membershipsCreateCmd)

	// Activate
	membershipsActivateCmd.Flags().String("account-id", "", "account ID (defaults from config)")
	membershipsActivateCmd.Flags().String("user-id", "", "user ID (required)")
	membershipsCmd.AddCommand(membershipsActivateCmd)

	// Deactivate
	membershipsDeactivateCmd.Flags().String("account-id", "", "account ID (defaults from config)")
	membershipsDeactivateCmd.Flags().String("user-id", "", "user ID (required)")
	membershipsDeactivateCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	membershipsCmd.AddCommand(membershipsDeactivateCmd)

	// Set Roles
	membershipsSetRolesCmd.Flags().String("account-id", "", "account ID (defaults from config)")
	membershipsSetRolesCmd.Flags().String("user-id", "", "user ID (required)")
	membershipsSetRolesCmd.Flags().String("role-id", "", "comma-separated role IDs")
	membershipsSetRolesCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	membershipsCmd.AddCommand(membershipsSetRolesCmd)
}
