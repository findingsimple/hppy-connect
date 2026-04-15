package cmd

import (
	"fmt"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

var inspectionsCmd = &cobra.Command{
	Use:   "inspections",
	Short: "Manage inspections",
}

var inspectionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List inspections",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		opts, err := parseListFlags(cmd, models.ValidInspectionStatuses)
		if err != nil {
			return err
		}

		if outputFormat == "raw" {
			raw, err := apiClient.ListInspectionsRaw(ctx, opts)
			if err != nil {
				return fmt.Errorf("listing inspections: %w", err)
			}
			return printOutput(outputData{RawJSON: raw})
		}

		inspections, total, err := apiClient.ListInspections(ctx, opts)
		if err != nil {
			return fmt.Errorf("listing inspections: %w", err)
		}

		rows := make([][]string, len(inspections))
		for i, insp := range inspections {
			template := ""
			if insp.TemplateV2 != nil {
				template = insp.TemplateV2.Name
			}

			rows[i] = []string{
				insp.ID,
				truncateString(insp.Name, 60),
				insp.Status,
				insp.StartedAt,
				insp.EndedAt,
				formatScore(insp.Score, insp.PotentialScore),
				template,
				formatLocation(insp.Location),
			}
		}

		if err := printOutput(outputData{
			Headers: []string{"ID", "NAME", "STATUS", "STARTED AT", "ENDED AT", "SCORE", "TEMPLATE", "LOCATION"},
			Rows:    rows,
			Items:   inspections,
			Count:   total,
		}); err != nil {
			return err
		}

		if total > len(inspections) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Showing %d of %d inspections\n", len(inspections), total)
		}

		return nil
	},
}

func init() {
	addListFlags(inspectionsListCmd, "filter by status: COMPLETE, EXPIRED, INCOMPLETE, SCHEDULED")
	inspectionsCmd.AddCommand(inspectionsListCmd)
}
