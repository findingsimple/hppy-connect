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

var membershipsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List account memberships",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		search, _ := cmd.Flags().GetString("search")
		includeInactive, _ := cmd.Flags().GetBool("include-inactive")
		limit, _ := cmd.Flags().GetInt("limit")

		if err := validateLimit(limit); err != nil {
			return err
		}
		if err := models.ValidateFreeText("search", search); err != nil {
			return err
		}

		opts := models.ListOptions{
			Limit:           limit,
			Search:          search,
			IncludeInactive: includeInactive,
		}

		if outputFormat == "raw" {
			raw, err := apiClient.ListMembersRaw(ctx, opts)
			if err != nil {
				return fmt.Errorf("listing members: %w", err)
			}
			return printOutput(outputData{RawJSON: raw})
		}

		members, total, err := apiClient.ListMembers(ctx, opts)
		if err != nil {
			return fmt.Errorf("listing members: %w", err)
		}

		rows := make([][]string, len(members))
		for i, m := range members {
			userName := ""
			userEmail := ""
			userID := ""
			if m.User != nil {
				userName = m.User.Name
				userEmail = m.User.Email
				userID = m.User.ID
			}
			active := "yes"
			if !m.IsActive {
				active = "no"
			}
			roleNames := ""
			if m.Roles != nil {
				names := make([]string, len(m.Roles.Nodes))
				for j, r := range m.Roles.Nodes {
					names[j] = r.Name
				}
				roleNames = strings.Join(names, ", ")
			}
			rows[i] = []string{userID, userName, userEmail, active, roleNames}
		}

		if err := printOutput(outputData{
			Headers: []string{"USER ID", "NAME", "EMAIL", "ACTIVE", "ROLES"},
			Rows:    rows,
			Items:   members,
			Count:   total,
		}); err != nil {
			return err
		}

		if total > len(members) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Showing %d of %d members\n", len(members), total)
		}

		return nil
	},
}

var membershipsCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a user's membership in an account",
	Example: `  hppycli memberships create --user-id=user456 --account-id=acct123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, err := resolveAccountID(cmd)
		if err != nil {
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
			ids, err := parseIDList("role-id", v)
			if err != nil {
				return err
			}
			input.RoleID = ids
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
		accountID, err := resolveAccountID(cmd)
		if err != nil {
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
		accountID, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}
		userID, _ := cmd.Flags().GetString("user-id")
		if userID == "" {
			return fmt.Errorf("--user-id is required")
		}
		if err := models.ValidateID("user-id", userID); err != nil {
			return err
		}

		if err := confirmAction(cmd, "deactivate membership", os.Stdin, os.Stderr); err != nil {
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
		accountID, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}
		userID, _ := cmd.Flags().GetString("user-id")
		if userID == "" {
			return fmt.Errorf("--user-id is required")
		}
		if err := models.ValidateID("user-id", userID); err != nil {
			return err
		}

		if err := confirmAction(cmd, "set membership roles", os.Stdin, os.Stderr); err != nil {
			return err
		}

		input := models.AccountMembershipSetRolesInput{
			AccountID: accountID,
			UserID:    userID,
		}
		if v, _ := cmd.Flags().GetString("role-id"); v != "" {
			ids, err := parseIDList("role-id", v)
			if err != nil {
				return err
			}
			input.RoleID = ids
		}

		membership, err := apiClient.AccountMembershipSetRoles(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("setting membership roles: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, membership)
	},
}

func init() {
	// List
	membershipsListCmd.Flags().String("search", "", "search by user name or email")
	membershipsListCmd.Flags().Bool("include-inactive", false, "include deactivated memberships")
	membershipsListCmd.Flags().Int("limit", 0, "maximum number of members to return (0 = default cap)")
	membershipsCmd.AddCommand(membershipsListCmd)

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
