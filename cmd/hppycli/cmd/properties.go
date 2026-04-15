package cmd

import (
	"fmt"

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

		if limit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer")
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

func init() {
	propertiesListCmd.Flags().Int("limit", 0, "maximum number of properties to return (0 = default cap)")
	propertiesCmd.AddCommand(propertiesListCmd)
}
