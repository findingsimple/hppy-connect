package cmd

import (
	"fmt"
	"os"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage users",
}

var usersCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new user in an account",
	Example: `  hppycli users create --email=user@example.com --name="Jane Doe" --account-id=acct123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		email, _ := cmd.Flags().GetString("email")
		if email == "" {
			return fmt.Errorf("--email is required")
		}
		if err := models.ValidateEmail(email); err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if err := models.ValidateFreeText("name", name); err != nil {
			return err
		}

		input := models.UserCreateInput{
			AccountID: accountID,
			Email:     email,
			Name:      name,
		}

		if v, _ := cmd.Flags().GetString("role-id"); v != "" {
			ids, err := parseIDList("role-id", v)
			if err != nil {
				return err
			}
			input.RoleID = ids
		}
		if v, _ := cmd.Flags().GetString("short-name"); v != "" {
			if err := models.ValidateFreeText("short-name", v); err != nil {
				return err
			}
			input.ShortName = v
		}
		if v, _ := cmd.Flags().GetString("phone"); v != "" {
			if err := models.ValidatePhone(v); err != nil {
				return err
			}
			input.Phone = v
		}
		if v, _ := cmd.Flags().GetString("message"); v != "" {
			if err := models.ValidateFreeText("message", v); err != nil {
				return err
			}
			input.Message = v
		}

		// Creating a user sends an invitation email — destructive side-effect
		// outside the local environment. Match the MCP tool's DestructiveHint:true.
		if err := confirmAction(cmd, fmt.Sprintf("create user %q and send invitation email to %s", name, email), os.Stdin, os.Stderr); err != nil {
			return err
		}

		user, err := apiClient.UserCreate(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("creating user: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, user)
	},
}

var usersSetEmailCmd = &cobra.Command{
	Use:     "set-email",
	Short:   "Update a user's email address",
	Example: `  hppycli users set-email --id=user123 --email=new@example.com`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}
		email, _ := cmd.Flags().GetString("email")
		if email == "" {
			return fmt.Errorf("--email is required")
		}
		if err := models.ValidateEmail(email); err != nil {
			return err
		}

		user, err := apiClient.UserSetEmail(cmd.Context(), id, email)
		if err != nil {
			return fmt.Errorf("setting user email: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, user)
	},
}

var usersSetNameCmd = &cobra.Command{
	Use:     "set-name",
	Short:   "Update a user's full name",
	Example: `  hppycli users set-name --id=user123 --name="Jane Smith"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if err := models.ValidateFreeText("name", name); err != nil {
			return err
		}

		user, err := apiClient.UserSetName(cmd.Context(), id, name)
		if err != nil {
			return fmt.Errorf("setting user name: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, user)
	},
}

var usersSetShortNameCmd = &cobra.Command{
	Use:     "set-short-name",
	Short:   "Set or clear a user's short name",
	Example: `  hppycli users set-short-name --id=user123 --short-name=Jane`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}

		var shortName *string
		if cmd.Flags().Changed("short-name") {
			v, _ := cmd.Flags().GetString("short-name")
			if v != "" {
				if err := models.ValidateFreeText("short-name", v); err != nil {
					return err
				}
			}
			shortName = &v
		}

		user, err := apiClient.UserSetShortName(cmd.Context(), id, shortName)
		if err != nil {
			return fmt.Errorf("setting user short name: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, user)
	},
}

var usersSetPhoneCmd = &cobra.Command{
	Use:     "set-phone",
	Short:   "Set or clear a user's phone number",
	Example: `  hppycli users set-phone --id=user123 --phone="+1-555-0100"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}

		var phone *string
		if cmd.Flags().Changed("phone") {
			v, _ := cmd.Flags().GetString("phone")
			if err := models.ValidatePhone(v); err != nil {
				return err
			}
			phone = &v
		}

		user, err := apiClient.UserSetPhone(cmd.Context(), id, phone)
		if err != nil {
			return fmt.Errorf("setting user phone: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, user)
	},
}

var usersGrantPropertyAccessCmd = &cobra.Command{
	Use:     "grant-property-access",
	Short:   "Grant a user access to one or more properties",
	Example: `  hppycli users grant-property-access --id=user123 --property-id=prop1,prop2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}
		propIDsRaw, _ := cmd.Flags().GetString("property-id")
		if propIDsRaw == "" {
			return fmt.Errorf("--property-id is required")
		}
		propIDs, err := parseIDList("property-id", propIDsRaw)
		if err != nil {
			return err
		}

		result, err := apiClient.UserGrantPropertyAccess(cmd.Context(), models.UserGrantPropertyAccessInput{
			UserID:     id,
			PropertyID: propIDs,
		})
		if err != nil {
			return fmt.Errorf("granting property access: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, result)
	},
}

var usersRevokePropertyAccessCmd = &cobra.Command{
	Use:     "revoke-property-access",
	Short:   "Revoke a user's access to one or more properties",
	Example: `  hppycli users revoke-property-access --id=user123 --property-id=prop1,prop2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}
		propIDsRaw, _ := cmd.Flags().GetString("property-id")
		if propIDsRaw == "" {
			return fmt.Errorf("--property-id is required")
		}
		propIDs, err := parseIDList("property-id", propIDsRaw)
		if err != nil {
			return err
		}

		if err := confirmAction(cmd, "revoke property access", os.Stdin, os.Stderr); err != nil {
			return err
		}

		result, err := apiClient.UserRevokePropertyAccess(cmd.Context(), models.UserRevokePropertyAccessInput{
			UserID:     id,
			PropertyID: propIDs,
		})
		if err != nil {
			return fmt.Errorf("revoking property access: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, result)
	},
}

func init() {
	// Create
	usersCreateCmd.Flags().String("account-id", "", "account ID (defaults from config)")
	usersCreateCmd.Flags().String("email", "", "email address (required)")
	usersCreateCmd.Flags().String("name", "", "full name (required)")
	usersCreateCmd.Flags().String("role-id", "", "comma-separated role IDs")
	usersCreateCmd.Flags().String("short-name", "", "informal/given name")
	usersCreateCmd.Flags().String("phone", "", "phone number")
	usersCreateCmd.Flags().String("message", "", "personalised invitation email greeting")
	usersCreateCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	usersCmd.AddCommand(usersCreateCmd)

	// Set Email
	usersSetEmailCmd.Flags().String("id", "", "user ID (required)")
	usersSetEmailCmd.Flags().String("email", "", "new email address (required)")
	usersCmd.AddCommand(usersSetEmailCmd)

	// Set Name
	usersSetNameCmd.Flags().String("id", "", "user ID (required)")
	usersSetNameCmd.Flags().String("name", "", "new full name (required)")
	usersCmd.AddCommand(usersSetNameCmd)

	// Set Short Name
	usersSetShortNameCmd.Flags().String("id", "", "user ID (required)")
	usersSetShortNameCmd.Flags().String("short-name", "", "short name (omit value to derive from name)")
	usersCmd.AddCommand(usersSetShortNameCmd)

	// Set Phone
	usersSetPhoneCmd.Flags().String("id", "", "user ID (required)")
	usersSetPhoneCmd.Flags().String("phone", "", "phone number (omit to remove)")
	usersCmd.AddCommand(usersSetPhoneCmd)

	// Grant Property Access
	usersGrantPropertyAccessCmd.Flags().String("id", "", "user ID (required)")
	usersGrantPropertyAccessCmd.Flags().String("property-id", "", "comma-separated property IDs (required)")
	usersCmd.AddCommand(usersGrantPropertyAccessCmd)

	// Revoke Property Access
	usersRevokePropertyAccessCmd.Flags().String("id", "", "user ID (required)")
	usersRevokePropertyAccessCmd.Flags().String("property-id", "", "comma-separated property IDs (required)")
	usersRevokePropertyAccessCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	usersCmd.AddCommand(usersRevokePropertyAccessCmd)
}
