package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var validWorkOrderStatuses = map[string]bool{
	"OPEN": true, "ON_HOLD": true, "COMPLETED": true,
}

var workordersCmd = &cobra.Command{
	Use:   "workorders",
	Short: "Manage work orders",
}

var workordersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List work orders",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		opts, err := parseListFlags(cmd, validWorkOrderStatuses)
		if err != nil {
			return err
		}

		if outputFormat == "raw" {
			raw, err := apiClient.ListWorkOrdersRaw(ctx, opts)
			if err != nil {
				return fmt.Errorf("listing work orders: %w", err)
			}
			return printOutput(outputData{RawJSON: raw})
		}

		workOrders, total, err := apiClient.ListWorkOrders(ctx, opts)
		if err != nil {
			return fmt.Errorf("listing work orders: %w", err)
		}

		rows := make([][]string, len(workOrders))
		for i, wo := range workOrders {
			assignee := "unassigned"
			if wo.AssignedTo != nil && wo.AssignedTo.Name != "" {
				assignee = wo.AssignedTo.Name
			}

			rows[i] = []string{
				wo.ID,
				wo.Status,
				wo.Priority,
				truncateString(wo.Description, 60),
				assignee,
				formatLocation(wo.Location),
			}
		}

		if err := printOutput(outputData{
			Headers: []string{"ID", "STATUS", "PRIORITY", "DESCRIPTION", "ASSIGNED TO", "LOCATION"},
			Rows:    rows,
			Items:   workOrders,
			Count:   total,
		}); err != nil {
			return err
		}

		if total > len(workOrders) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Showing %d of %d work orders\n", len(workOrders), total)
		}

		return nil
	},
}

func init() {
	addListFlags(workordersListCmd, "filter by status: OPEN, ON_HOLD, COMPLETED")
	workordersCmd.AddCommand(workordersListCmd)
}
