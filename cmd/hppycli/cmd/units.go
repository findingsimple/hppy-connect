package cmd

import (
	"fmt"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

var unitsCmd = &cobra.Command{
	Use:   "units",
	Short: "Manage units",
}

var unitsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List units for a property",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		propertyID, _ := cmd.Flags().GetString("property-id")
		limit, _ := cmd.Flags().GetInt("limit")

		if limit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer")
		}

		opts := models.ListOptions{Limit: limit}

		if outputFormat == "raw" {
			raw, err := apiClient.ListUnitsRaw(ctx, propertyID)
			if err != nil {
				return fmt.Errorf("listing units: %w", err)
			}
			return printOutput(outputData{RawJSON: raw})
		}

		units, total, err := apiClient.ListUnits(ctx, propertyID, opts)
		if err != nil {
			return fmt.Errorf("listing units: %w", err)
		}

		rows := make([][]string, len(units))
		for i, u := range units {
			rows[i] = []string{u.ID, u.Name}
		}

		if err := printOutput(outputData{
			Headers: []string{"ID", "NAME"},
			Rows:    rows,
			Items:   units,
			Count:   total,
		}); err != nil {
			return err
		}

		if total > len(units) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Showing %d of %d units\n", len(units), total)
		}

		return nil
	},
}

func init() {
	unitsListCmd.Flags().String("property-id", "", "property ID (required)")
	unitsListCmd.MarkFlagRequired("property-id")
	unitsListCmd.Flags().Int("limit", 0, "maximum number of units to return (0 = default cap)")
	unitsCmd.AddCommand(unitsListCmd)
}
