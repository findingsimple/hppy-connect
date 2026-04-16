package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

var propertiesCmd = &cobra.Command{
	Use:   "properties",
	Short: "Manage properties",
}

var propertiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List properties",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		limit, _ := cmd.Flags().GetInt("limit")

		if err := validateLimit(limit); err != nil {
			return err
		}

		opts := models.ListOptions{Limit: limit}

		if outputFormat == "raw" {
			raw, err := apiClient.ListPropertiesRaw(ctx, opts)
			if err != nil {
				return fmt.Errorf("listing properties: %w", err)
			}
			return printOutput(outputData{RawJSON: raw})
		}

		properties, total, err := apiClient.ListProperties(ctx, opts)
		if err != nil {
			return fmt.Errorf("listing properties: %w", err)
		}

		rows := make([][]string, len(properties))
		for i, p := range properties {
			addr := formatAddress(p.Address.Line1, p.Address.Line2, p.Address.City, p.Address.State, p.Address.PostalCode)
			rows[i] = []string{p.ID, p.Name, addr, p.CreatedAt}
		}

		if err := printOutput(outputData{
			Headers: []string{"ID", "NAME", "ADDRESS", "CREATED AT"},
			Rows:    rows,
			Items:   properties,
			Count:   total,
		}); err != nil {
			return err
		}

		if total > len(properties) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Showing %d of %d properties\n", len(properties), total)
		}

		return nil
	},
}

var propertiesGrantAccessCmd = &cobra.Command{
	Use:     "grant-access",
	Short:   "Grant one or more users access to a property",
	Example: `  hppycli properties grant-access --id=prop123 --user-id=user1,user2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		userIDsRaw, _ := cmd.Flags().GetString("user-id")
		if userIDsRaw == "" {
			return fmt.Errorf("--user-id is required")
		}
		var userIDs []string
		for _, p := range strings.Split(userIDsRaw, ",") {
			uid := strings.TrimSpace(p)
			if uid == "" {
				continue
			}
			if err := models.ValidateID("user-id", uid); err != nil {
				return err
			}
			userIDs = append(userIDs, uid)
		}

		result, err := apiClient.PropertyGrantUserAccess(cmd.Context(), models.PropertyGrantUserAccessInput{
			PropertyID: id,
			UserID:     userIDs,
		})
		if err != nil {
			return fmt.Errorf("granting user access: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, result)
	},
}

var propertiesRevokeAccessCmd = &cobra.Command{
	Use:     "revoke-access",
	Short:   "Revoke one or more users' access to a property",
	Example: `  hppycli properties revoke-access --id=prop123 --user-id=user1,user2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		userIDsRaw, _ := cmd.Flags().GetString("user-id")
		if userIDsRaw == "" {
			return fmt.Errorf("--user-id is required")
		}
		var userIDs []string
		for _, p := range strings.Split(userIDsRaw, ",") {
			uid := strings.TrimSpace(p)
			if uid == "" {
				continue
			}
			if err := models.ValidateID("user-id", uid); err != nil {
				return err
			}
			userIDs = append(userIDs, uid)
		}

		if err := confirmAction(cmd, "revoke user access", os.Stdin); err != nil {
			return err
		}

		result, err := apiClient.PropertyRevokeUserAccess(cmd.Context(), models.PropertyRevokeUserAccessInput{
			PropertyID: id,
			UserID:     userIDs,
		})
		if err != nil {
			return fmt.Errorf("revoking user access: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, result)
	},
}

var propertiesSetAccountWideAccessCmd = &cobra.Command{
	Use:     "set-account-wide-access",
	Short:   "Set whether a property is accessible to all users in the account",
	Example: `  hppycli properties set-account-wide-access --id=prop123 --account-wide-access=true`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		if !cmd.Flags().Changed("account-wide-access") {
			return fmt.Errorf("--account-wide-access is required")
		}
		accountWideAccess, _ := cmd.Flags().GetBool("account-wide-access")

		if err := confirmAction(cmd, "set account-wide access", os.Stdin); err != nil {
			return err
		}

		result, err := apiClient.PropertySetAccountWideAccess(cmd.Context(), models.PropertySetAccountWideAccessInput{
			PropertyID:        id,
			AccountWideAccess: accountWideAccess,
		})
		if err != nil {
			return fmt.Errorf("setting account-wide access: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, result)
	},
}

func init() {
	propertiesListCmd.Flags().Int("limit", 0, "maximum number of properties to return (0 = default cap)")
	propertiesCmd.AddCommand(propertiesListCmd)

	// Grant Access
	propertiesGrantAccessCmd.Flags().String("id", "", "property ID (required)")
	propertiesGrantAccessCmd.Flags().String("user-id", "", "comma-separated user IDs (required)")
	propertiesCmd.AddCommand(propertiesGrantAccessCmd)

	// Revoke Access
	propertiesRevokeAccessCmd.Flags().String("id", "", "property ID (required)")
	propertiesRevokeAccessCmd.Flags().String("user-id", "", "comma-separated user IDs (required)")
	propertiesRevokeAccessCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	propertiesCmd.AddCommand(propertiesRevokeAccessCmd)

	// Set Account Wide Access
	propertiesSetAccountWideAccessCmd.Flags().String("id", "", "property ID (required)")
	propertiesSetAccountWideAccessCmd.Flags().Bool("account-wide-access", false, "whether all users can access this property (required)")
	propertiesSetAccountWideAccessCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	propertiesCmd.AddCommand(propertiesSetAccountWideAccessCmd)
}
