package cmd

import (
	"fmt"
	"os"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

var rolesCmd = &cobra.Command{
	Use:   "roles",
	Short: "Manage permission roles",
}

var rolesCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new permission role in an account",
	Example: `  hppycli roles create --name="Inspector" --grant=inspection:inspection.create,inspection:inspection.view --account-id=acct123`,
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

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if err := models.ValidateFreeText("name", name); err != nil {
			return err
		}

		input := models.RoleCreateInput{
			AccountID: accountID,
			Name:      name,
		}

		if v, _ := cmd.Flags().GetString("description"); v != "" {
			if err := models.ValidateFreeText("description", v); err != nil {
				return err
			}
			input.Description = v
		}

		grantStr, _ := cmd.Flags().GetString("grant")
		revokeStr, _ := cmd.Flags().GetString("revoke")
		grant := models.SplitCSV(grantStr)
		revoke := models.SplitCSV(revokeStr)
		if len(grant) == 0 && len(revoke) == 0 {
			return fmt.Errorf("--grant is required")
		}
		input.Permissions = models.PermissionsInput{
			Grant:  grant,
			Revoke: revoke,
		}

		role, err := apiClient.RoleCreate(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("creating role: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, role)
	},
}

var rolesSetNameCmd = &cobra.Command{
	Use:     "set-name",
	Short:   "Update a role's display name",
	Example: `  hppycli roles set-name --id=role123 --name="Senior Inspector" --account-id=acct123`,
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

		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if err := models.ValidateFreeText("name", name); err != nil {
			return err
		}

		role, err := apiClient.RoleSetName(cmd.Context(), models.RoleSetNameInput{
			AccountID: accountID,
			RoleID:    id,
			Name:      name,
		})
		if err != nil {
			return fmt.Errorf("setting role name: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, role)
	},
}

var rolesSetDescriptionCmd = &cobra.Command{
	Use:     "set-description",
	Short:   "Update or clear a role's description",
	Example: `  hppycli roles set-description --id=role123 --description="Can perform inspections" --account-id=acct123`,
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

		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}

		var description *string
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			if v != "" {
				if err := models.ValidateFreeText("description", v); err != nil {
					return err
				}
			}
			description = &v
		}

		role, err := apiClient.RoleSetDescription(cmd.Context(), models.RoleSetDescriptionInput{
			AccountID:   accountID,
			RoleID:      id,
			Description: description,
		})
		if err != nil {
			return fmt.Errorf("setting role description: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, role)
	},
}

var rolesSetPermissionsCmd = &cobra.Command{
	Use:     "set-permissions",
	Short:   "Update permissions for a role",
	Example: `  hppycli roles set-permissions --id=role123 --grant=inspection:inspection.create --revoke=task:task.delete --account-id=acct123`,
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

		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}

		grantStr, _ := cmd.Flags().GetString("grant")
		revokeStr, _ := cmd.Flags().GetString("revoke")
		grant := models.SplitCSV(grantStr)
		revoke := models.SplitCSV(revokeStr)
		if len(grant) == 0 && len(revoke) == 0 {
			return fmt.Errorf("at least one of --grant or --revoke is required")
		}

		if err := confirmAction(cmd, "set role permissions", os.Stdin); err != nil {
			return err
		}

		role, err := apiClient.RoleSetPermissions(cmd.Context(), models.RoleSetPermissionsInput{
			AccountID: accountID,
			RoleID:    id,
			Permissions: models.PermissionsInput{
				Grant:  grant,
				Revoke: revoke,
			},
		})
		if err != nil {
			return fmt.Errorf("setting role permissions: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, role)
	},
}

func init() {
	// Create
	rolesCreateCmd.Flags().String("account-id", "", "account ID (defaults from config)")
	rolesCreateCmd.Flags().String("name", "", "role name (required)")
	rolesCreateCmd.Flags().String("description", "", "role description")
	rolesCreateCmd.Flags().String("grant", "", "comma-separated permission actions to grant (required)")
	rolesCreateCmd.Flags().String("revoke", "", "comma-separated permission actions to revoke")
	rolesCmd.AddCommand(rolesCreateCmd)

	// Set Name
	rolesSetNameCmd.Flags().String("account-id", "", "account ID (defaults from config)")
	rolesSetNameCmd.Flags().String("id", "", "role ID (required)")
	rolesSetNameCmd.Flags().String("name", "", "new role name (required)")
	rolesCmd.AddCommand(rolesSetNameCmd)

	// Set Description
	rolesSetDescriptionCmd.Flags().String("account-id", "", "account ID (defaults from config)")
	rolesSetDescriptionCmd.Flags().String("id", "", "role ID (required)")
	rolesSetDescriptionCmd.Flags().String("description", "", "new description (omit to remove)")
	rolesCmd.AddCommand(rolesSetDescriptionCmd)

	// Set Permissions
	rolesSetPermissionsCmd.Flags().String("account-id", "", "account ID (defaults from config)")
	rolesSetPermissionsCmd.Flags().String("id", "", "role ID (required)")
	rolesSetPermissionsCmd.Flags().String("grant", "", "comma-separated permission actions to grant")
	rolesSetPermissionsCmd.Flags().String("revoke", "", "comma-separated permission actions to revoke")
	rolesSetPermissionsCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	rolesCmd.AddCommand(rolesSetPermissionsCmd)
}
