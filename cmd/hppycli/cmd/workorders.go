package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

var workordersCmd = &cobra.Command{
	Use:   "workorders",
	Short: "Manage work orders",
}

var workordersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List work orders",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		opts, err := parseListFlags(cmd, models.ValidWorkOrderStatuses)
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

var workordersCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new work order",
	Example: `  hppycli workorders create --location-id=225393 --description="Fix leaking faucet" --priority=URGENT`,
	RunE: func(cmd *cobra.Command, args []string) error {
		locationID, _ := cmd.Flags().GetString("location-id")
		if locationID == "" {
			return fmt.Errorf("--location-id is required")
		}
		if err := models.ValidateID("location-id", locationID); err != nil {
			return err
		}

		input := models.WorkOrderCreateInput{LocationID: locationID}

		if v, _ := cmd.Flags().GetString("description"); v != "" {
			if err := models.ValidateFreeText("description", v); err != nil {
				return err
			}
			input.Description = v
		}
		if v, _ := cmd.Flags().GetString("priority"); v != "" {
			upper := strings.ToUpper(v)
			if !models.ValidWorkOrderPriorities[upper] {
				return fmt.Errorf("invalid --priority %q: must be NORMAL or URGENT", v)
			}
			input.Priority = upper
		}
		if v, _ := cmd.Flags().GetString("status"); v != "" {
			upper := strings.ToUpper(v)
			if !models.ValidWorkOrderStatuses[upper] {
				return fmt.Errorf("invalid --status %q: must be OPEN, ON_HOLD, or COMPLETED", v)
			}
			input.Status = upper
		}
		if v, _ := cmd.Flags().GetString("type"); v != "" {
			upper := strings.ToUpper(v)
			if !models.ValidWorkOrderTypes[upper] {
				return fmt.Errorf("invalid --type %q: must be one of SERVICE_REQUEST, TURN, CAPITAL_IMPROVEMENT, CURB_APPEAL, INCIDENT, INVENTORY, LIFE_SAFETY, PREVENTATIVE_MAINTENANCE, REGULATORY, SEASONAL_MAINTENANCE", v)
			}
			input.Type = upper
		}
		if v, _ := cmd.Flags().GetString("scheduled-for"); v != "" {
			if err := models.ValidateTimestamp("scheduled-for", v); err != nil {
				return err
			}
			input.ScheduledFor = v
		}
		if v, _ := cmd.Flags().GetString("entry-notes"); v != "" {
			if err := models.ValidateFreeText("entry-notes", v); err != nil {
				return err
			}
			input.EntryNotes = v
		}
		if cmd.Flags().Changed("permission-to-enter") {
			v, _ := cmd.Flags().GetBool("permission-to-enter")
			input.PermissionToEnter = &v
		}
		if assigneeID, _ := cmd.Flags().GetString("assignee-id"); assigneeID != "" {
			if err := models.ValidateID("assignee-id", assigneeID); err != nil {
				return err
			}
			assigneeType, _ := cmd.Flags().GetString("assignee-type")
			if assigneeType == "" {
				assigneeType = "USER"
			}
			upper := strings.ToUpper(assigneeType)
			if !models.ValidWorkOrderAssigneeTypes[upper] {
				return fmt.Errorf("invalid --assignee-type %q: must be USER or VENDOR", assigneeType)
			}
			input.Assignee = &models.AssignableInput{
				AssigneeID:   assigneeID,
				AssigneeType: upper,
			}
		}

		wo, err := apiClient.WorkOrderCreate(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("creating work order: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersSetStatusCmd = &cobra.Command{
	Use:     "set-status",
	Short:   "Set the status and sub-status of a work order",
	Example: `  hppycli workorders set-status --id=abc123 --status=COMPLETED --sub-status=UNKNOWN`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		status, _ := cmd.Flags().GetString("status")
		if status == "" {
			return fmt.Errorf("--status is required")
		}
		upper := strings.ToUpper(status)
		if !models.ValidWorkOrderStatuses[upper] {
			return fmt.Errorf("invalid --status %q: must be OPEN, ON_HOLD, or COMPLETED", status)
		}

		subStatus, _ := cmd.Flags().GetString("sub-status")
		subUpper := "UNKNOWN"
		if subStatus != "" {
			subUpper = strings.ToUpper(subStatus)
			if !models.ValidWorkOrderSubStatuses[subUpper] {
				return fmt.Errorf("invalid --sub-status %q: must be CANCELLED or UNKNOWN", subStatus)
			}
		}

		input := models.WorkOrderSetStatusAndSubStatusInput{
			WorkOrderID: id,
			Status: models.WorkOrderStatusInput{
				Status: upper,
			},
			SubStatus: models.WorkOrderSubStatusInput{
				SubStatus: subUpper,
			},
		}

		if v, _ := cmd.Flags().GetString("comment"); v != "" {
			input.Status.Comment = v
		}

		wo, err := apiClient.WorkOrderSetStatusAndSubStatus(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("setting work order status: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersSetAssigneeCmd = &cobra.Command{
	Use:     "set-assignee",
	Short:   "Set the assignee of a work order",
	Example: `  hppycli workorders set-assignee --id=abc123 --assignee-id=user456 --assignee-type=USER`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		assigneeID, _ := cmd.Flags().GetString("assignee-id")
		if assigneeID == "" {
			return fmt.Errorf("--assignee-id is required")
		}
		if err := models.ValidateID("assignee-id", assigneeID); err != nil {
			return err
		}
		assigneeType, _ := cmd.Flags().GetString("assignee-type")
		if assigneeType == "" {
			assigneeType = "USER"
		}
		upper := strings.ToUpper(assigneeType)
		if !models.ValidWorkOrderAssigneeTypes[upper] {
			return fmt.Errorf("invalid --assignee-type %q: must be USER or VENDOR", assigneeType)
		}

		input := models.WorkOrderSetAssigneeInput{
			WorkOrderID: id,
			Assignee: models.AssignableInput{
				AssigneeID:   assigneeID,
				AssigneeType: upper,
			},
		}

		wo, err := apiClient.WorkOrderSetAssignee(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("setting work order assignee: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersSetDescriptionCmd = &cobra.Command{
	Use:     "set-description",
	Short:   "Set the description of a work order",
	Example: `  hppycli workorders set-description --id=abc123 --description="Updated description"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		desc, _ := cmd.Flags().GetString("description")
		if desc == "" {
			return fmt.Errorf("--description is required")
		}
		if err := models.ValidateFreeText("description", desc); err != nil {
			return err
		}

		wo, err := apiClient.WorkOrderSetDescription(cmd.Context(), id, desc)
		if err != nil {
			return fmt.Errorf("setting work order description: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersSetPriorityCmd = &cobra.Command{
	Use:     "set-priority",
	Short:   "Set the priority of a work order",
	Example: `  hppycli workorders set-priority --id=abc123 --priority=URGENT`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		priority, _ := cmd.Flags().GetString("priority")
		if priority == "" {
			return fmt.Errorf("--priority is required")
		}
		upper := strings.ToUpper(priority)
		if !models.ValidWorkOrderPriorities[upper] {
			return fmt.Errorf("invalid --priority %q: must be NORMAL or URGENT", priority)
		}

		wo, err := apiClient.WorkOrderSetPriority(cmd.Context(), id, upper)
		if err != nil {
			return fmt.Errorf("setting work order priority: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersSetScheduledForCmd = &cobra.Command{
	Use:     "set-scheduled-for",
	Short:   "Set the scheduled date for a work order",
	Example: `  hppycli workorders set-scheduled-for --id=abc123 --scheduled-for=2026-05-01T09:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		scheduledFor, _ := cmd.Flags().GetString("scheduled-for")
		if scheduledFor == "" {
			return fmt.Errorf("--scheduled-for is required")
		}
		if err := models.ValidateTimestamp("scheduled-for", scheduledFor); err != nil {
			return err
		}

		wo, err := apiClient.WorkOrderSetScheduledFor(cmd.Context(), id, scheduledFor)
		if err != nil {
			return fmt.Errorf("setting work order scheduled date: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersSetLocationCmd = &cobra.Command{
	Use:     "set-location",
	Short:   "Set the location of a work order",
	Example: `  hppycli workorders set-location --id=abc123 --location-id=prop456`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		locationID, _ := cmd.Flags().GetString("location-id")
		if locationID == "" {
			return fmt.Errorf("--location-id is required")
		}
		if err := models.ValidateID("location-id", locationID); err != nil {
			return err
		}

		wo, err := apiClient.WorkOrderSetLocation(cmd.Context(), id, locationID)
		if err != nil {
			return fmt.Errorf("setting work order location: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersSetTypeCmd = &cobra.Command{
	Use:     "set-type",
	Short:   "Set the type of a work order",
	Example: `  hppycli workorders set-type --id=abc123 --type=SERVICE_REQUEST`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		woType, _ := cmd.Flags().GetString("type")
		if woType == "" {
			return fmt.Errorf("--type is required")
		}
		upper := strings.ToUpper(woType)
		if !models.ValidWorkOrderTypes[upper] {
			return fmt.Errorf("invalid --type %q: must be one of SERVICE_REQUEST, TURN, CAPITAL_IMPROVEMENT, CURB_APPEAL, INCIDENT, INVENTORY, LIFE_SAFETY, PREVENTATIVE_MAINTENANCE, REGULATORY, SEASONAL_MAINTENANCE", woType)
		}

		wo, err := apiClient.WorkOrderSetType(cmd.Context(), id, upper)
		if err != nil {
			return fmt.Errorf("setting work order type: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersSetEntryNotesCmd = &cobra.Command{
	Use:     "set-entry-notes",
	Short:   "Set the entry notes of a work order",
	Example: `  hppycli workorders set-entry-notes --id=abc123 --entry-notes="Ring doorbell twice"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		notes, _ := cmd.Flags().GetString("entry-notes")
		if notes == "" {
			return fmt.Errorf("--entry-notes is required")
		}
		if err := models.ValidateFreeText("entry-notes", notes); err != nil {
			return err
		}

		wo, err := apiClient.WorkOrderSetEntryNotes(cmd.Context(), id, notes)
		if err != nil {
			return fmt.Errorf("setting work order entry notes: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersSetPermissionToEnterCmd = &cobra.Command{
	Use:     "set-permission-to-enter",
	Short:   "Set the permission to enter flag on a work order",
	Example: `  hppycli workorders set-permission-to-enter --id=abc123 --permission-to-enter=true`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		if !cmd.Flags().Changed("permission-to-enter") {
			return fmt.Errorf("--permission-to-enter is required")
		}
		permission, _ := cmd.Flags().GetBool("permission-to-enter")

		wo, err := apiClient.WorkOrderSetPermissionToEnter(cmd.Context(), id, permission)
		if err != nil {
			return fmt.Errorf("setting permission to enter: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersSetResidentApprovedEntryCmd = &cobra.Command{
	Use:     "set-resident-approved-entry",
	Short:   "Set the resident approved entry flag on a work order",
	Example: `  hppycli workorders set-resident-approved-entry --id=abc123 --resident-approved-entry=true`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		if !cmd.Flags().Changed("resident-approved-entry") {
			return fmt.Errorf("--resident-approved-entry is required")
		}
		approved, _ := cmd.Flags().GetBool("resident-approved-entry")

		wo, err := apiClient.WorkOrderSetResidentApprovedEntry(cmd.Context(), id, approved)
		if err != nil {
			return fmt.Errorf("setting resident approved entry: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersSetUnitEnteredCmd = &cobra.Command{
	Use:     "set-unit-entered",
	Short:   "Set the unit entered flag on a work order",
	Example: `  hppycli workorders set-unit-entered --id=abc123 --unit-entered=true`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		if !cmd.Flags().Changed("unit-entered") {
			return fmt.Errorf("--unit-entered is required")
		}
		entered, _ := cmd.Flags().GetBool("unit-entered")

		wo, err := apiClient.WorkOrderSetUnitEntered(cmd.Context(), id, entered)
		if err != nil {
			return fmt.Errorf("setting unit entered: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersArchiveCmd = &cobra.Command{
	Use:     "archive",
	Short:   "Archive a work order",
	Example: `  hppycli workorders archive --id=abc123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		if err := confirmAction(cmd, "archive work order "+id, os.Stdin, os.Stderr); err != nil {
			return err
		}

		wo, err := apiClient.WorkOrderArchive(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("archiving work order: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersAddCommentCmd = &cobra.Command{
	Use:     "add-comment",
	Short:   "Add a comment to a work order",
	Example: `  hppycli workorders add-comment --id=abc123 --comment="Parts ordered"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		comment, _ := cmd.Flags().GetString("comment")
		if comment == "" {
			return fmt.Errorf("--comment is required")
		}
		if err := models.ValidateFreeText("comment", comment); err != nil {
			return err
		}

		wo, err := apiClient.WorkOrderAddComment(cmd.Context(), id, comment)
		if err != nil {
			return fmt.Errorf("adding comment: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersAddTimeCmd = &cobra.Command{
	Use:     "add-time",
	Short:   "Add time spent on a work order",
	Example: `  hppycli workorders add-time --id=abc123 --duration=PT1H30M`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		duration, _ := cmd.Flags().GetString("duration")
		if duration == "" {
			return fmt.Errorf("--duration is required (ISO 8601 format, e.g. PT1H30M)")
		}
		if err := models.ValidateDuration(duration); err != nil {
			return err
		}

		wo, err := apiClient.WorkOrderAddTime(cmd.Context(), id, duration)
		if err != nil {
			return fmt.Errorf("adding time: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersAddAttachmentCmd = &cobra.Command{
	Use:     "add-attachment",
	Short:   "Add an attachment to a work order (returns signed upload URL)",
	Example: `  hppycli workorders add-attachment --id=abc123 --file-name=photo.jpg --mime-type=image/jpeg`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		fileName, _ := cmd.Flags().GetString("file-name")
		if fileName == "" {
			return fmt.Errorf("--file-name is required")
		}
		if err := models.ValidateFileName(fileName); err != nil {
			return err
		}
		mimeType, _ := cmd.Flags().GetString("mime-type")
		if mimeType == "" {
			return fmt.Errorf("--mime-type is required")
		}
		if err := models.ValidateMIMEType(mimeType); err != nil {
			return err
		}

		input := models.WorkOrderAddAttachmentInput{
			WorkOrderID: id,
			FileName:    fileName,
			MimeType:    mimeType,
		}
		if cmd.Flags().Changed("size") {
			size, _ := cmd.Flags().GetInt("size")
			input.Size = &size
		}

		result, err := apiClient.WorkOrderAddAttachment(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("adding attachment: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, result)
	},
}

var workordersRemoveAttachmentCmd = &cobra.Command{
	Use:     "remove-attachment",
	Short:   "Remove an attachment from a work order",
	Example: `  hppycli workorders remove-attachment --id=abc123 --attachment-id=att456`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		attachmentID, _ := cmd.Flags().GetString("attachment-id")
		if attachmentID == "" {
			return fmt.Errorf("--attachment-id is required")
		}
		if err := models.ValidateID("attachment-id", attachmentID); err != nil {
			return err
		}
		if err := confirmAction(cmd, "remove attachment "+attachmentID+" from work order "+id, os.Stdin, os.Stderr); err != nil {
			return err
		}

		wo, err := apiClient.WorkOrderRemoveAttachment(cmd.Context(), id, attachmentID)
		if err != nil {
			return fmt.Errorf("removing attachment: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersStartTimerCmd = &cobra.Command{
	Use:     "start-timer",
	Short:   "Start the timer for a work order",
	Example: `  hppycli workorders start-timer --id=abc123 --started-at=2026-05-01T09:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		startedAt, _ := cmd.Flags().GetString("started-at")
		if startedAt == "" {
			return fmt.Errorf("--started-at is required (RFC3339 format)")
		}
		if err := models.ValidateTimestamp("started-at", startedAt); err != nil {
			return err
		}

		wo, err := apiClient.WorkOrderStartTimer(cmd.Context(), id, startedAt)
		if err != nil {
			return fmt.Errorf("starting timer: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

var workordersStopTimerCmd = &cobra.Command{
	Use:     "stop-timer",
	Short:   "Stop the timer for a work order",
	Example: `  hppycli workorders stop-timer --id=abc123 --stopped-at=2026-05-01T10:30:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		stoppedAt, _ := cmd.Flags().GetString("stopped-at")
		if stoppedAt == "" {
			return fmt.Errorf("--stopped-at is required (RFC3339 format)")
		}
		if err := models.ValidateTimestamp("stopped-at", stoppedAt); err != nil {
			return err
		}

		wo, err := apiClient.WorkOrderStopTimer(cmd.Context(), id, stoppedAt)
		if err != nil {
			return fmt.Errorf("stopping timer: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, wo)
	},
}

func init() {
	// List command
	addListFlags(workordersListCmd, "filter by status: OPEN, ON_HOLD, COMPLETED")
	workordersCmd.AddCommand(workordersListCmd)

	// Create
	workordersCreateCmd.Flags().String("location-id", "", "property or unit ID (required)")
	workordersCreateCmd.Flags().String("description", "", "work order description")
	workordersCreateCmd.Flags().String("priority", "", "priority: NORMAL or URGENT")
	workordersCreateCmd.Flags().String("status", "", "status: OPEN, ON_HOLD, COMPLETED")
	workordersCreateCmd.Flags().String("type", "", "type: SERVICE_REQUEST, TURN, CAPITAL_IMPROVEMENT, CURB_APPEAL, INCIDENT, INVENTORY, LIFE_SAFETY, PREVENTATIVE_MAINTENANCE, REGULATORY, SEASONAL_MAINTENANCE")
	workordersCreateCmd.Flags().String("scheduled-for", "", "scheduled date (RFC3339)")
	workordersCreateCmd.Flags().String("entry-notes", "", "entry notes")
	workordersCreateCmd.Flags().Bool("permission-to-enter", false, "permission to enter")
	workordersCreateCmd.Flags().String("assignee-id", "", "assignee user or vendor ID")
	workordersCreateCmd.Flags().String("assignee-type", "USER", "assignee type: USER or VENDOR")
	workordersCmd.AddCommand(workordersCreateCmd)

	// Set Status
	workordersSetStatusCmd.Flags().String("id", "", "work order ID (required)")
	workordersSetStatusCmd.Flags().String("status", "", "status: OPEN, ON_HOLD, COMPLETED (required)")
	workordersSetStatusCmd.Flags().String("sub-status", "", "sub-status: CANCELLED or UNKNOWN (default UNKNOWN)")
	workordersSetStatusCmd.Flags().String("comment", "", "optional comment when completing")
	workordersCmd.AddCommand(workordersSetStatusCmd)

	// Set Assignee
	workordersSetAssigneeCmd.Flags().String("id", "", "work order ID (required)")
	workordersSetAssigneeCmd.Flags().String("assignee-id", "", "assignee ID (required)")
	workordersSetAssigneeCmd.Flags().String("assignee-type", "USER", "assignee type: USER or VENDOR")
	workordersCmd.AddCommand(workordersSetAssigneeCmd)

	// Set Description
	workordersSetDescriptionCmd.Flags().String("id", "", "work order ID (required)")
	workordersSetDescriptionCmd.Flags().String("description", "", "new description (required)")
	workordersCmd.AddCommand(workordersSetDescriptionCmd)

	// Set Priority
	workordersSetPriorityCmd.Flags().String("id", "", "work order ID (required)")
	workordersSetPriorityCmd.Flags().String("priority", "", "priority: NORMAL or URGENT (required)")
	workordersCmd.AddCommand(workordersSetPriorityCmd)

	// Set Scheduled For
	workordersSetScheduledForCmd.Flags().String("id", "", "work order ID (required)")
	workordersSetScheduledForCmd.Flags().String("scheduled-for", "", "scheduled date in RFC3339 (required)")
	workordersCmd.AddCommand(workordersSetScheduledForCmd)

	// Set Location
	workordersSetLocationCmd.Flags().String("id", "", "work order ID (required)")
	workordersSetLocationCmd.Flags().String("location-id", "", "new location/property/unit ID (required)")
	workordersCmd.AddCommand(workordersSetLocationCmd)

	// Set Type
	workordersSetTypeCmd.Flags().String("id", "", "work order ID (required)")
	workordersSetTypeCmd.Flags().String("type", "", "work order type (required)")
	workordersCmd.AddCommand(workordersSetTypeCmd)

	// Set Entry Notes
	workordersSetEntryNotesCmd.Flags().String("id", "", "work order ID (required)")
	workordersSetEntryNotesCmd.Flags().String("entry-notes", "", "entry notes (required)")
	workordersCmd.AddCommand(workordersSetEntryNotesCmd)

	// Set Permission to Enter
	workordersSetPermissionToEnterCmd.Flags().String("id", "", "work order ID (required)")
	workordersSetPermissionToEnterCmd.Flags().Bool("permission-to-enter", false, "use --permission-to-enter=true or =false (required)")
	workordersCmd.AddCommand(workordersSetPermissionToEnterCmd)

	// Set Resident Approved Entry
	workordersSetResidentApprovedEntryCmd.Flags().String("id", "", "work order ID (required)")
	workordersSetResidentApprovedEntryCmd.Flags().Bool("resident-approved-entry", false, "use --resident-approved-entry=true or =false (required)")
	workordersCmd.AddCommand(workordersSetResidentApprovedEntryCmd)

	// Set Unit Entered
	workordersSetUnitEnteredCmd.Flags().String("id", "", "work order ID (required)")
	workordersSetUnitEnteredCmd.Flags().Bool("unit-entered", false, "use --unit-entered=true or =false (required)")
	workordersCmd.AddCommand(workordersSetUnitEnteredCmd)

	// Archive (destructive)
	workordersArchiveCmd.Flags().String("id", "", "work order ID (required)")
	workordersArchiveCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	workordersCmd.AddCommand(workordersArchiveCmd)

	// Add Comment
	workordersAddCommentCmd.Flags().String("id", "", "work order ID (required)")
	workordersAddCommentCmd.Flags().String("comment", "", "comment text (required)")
	workordersCmd.AddCommand(workordersAddCommentCmd)

	// Add Time
	workordersAddTimeCmd.Flags().String("id", "", "work order ID (required)")
	workordersAddTimeCmd.Flags().String("duration", "", "ISO 8601 duration, e.g. PT1H30M (required)")
	workordersCmd.AddCommand(workordersAddTimeCmd)

	// Add Attachment
	workordersAddAttachmentCmd.Flags().String("id", "", "work order ID (required)")
	workordersAddAttachmentCmd.Flags().String("file-name", "", "attachment file name (required)")
	workordersAddAttachmentCmd.Flags().String("mime-type", "", "attachment MIME type (required)")
	workordersAddAttachmentCmd.Flags().Int("size", 0, "attachment size in bytes")
	workordersCmd.AddCommand(workordersAddAttachmentCmd)

	// Remove Attachment (destructive)
	workordersRemoveAttachmentCmd.Flags().String("id", "", "work order ID (required)")
	workordersRemoveAttachmentCmd.Flags().String("attachment-id", "", "attachment ID (required)")
	workordersRemoveAttachmentCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	workordersCmd.AddCommand(workordersRemoveAttachmentCmd)

	// Start Timer
	workordersStartTimerCmd.Flags().String("id", "", "work order ID (required)")
	workordersStartTimerCmd.Flags().String("started-at", "", "start time in RFC3339 (required)")
	workordersCmd.AddCommand(workordersStartTimerCmd)

	// Stop Timer
	workordersStopTimerCmd.Flags().String("id", "", "work order ID (required)")
	workordersStopTimerCmd.Flags().String("stopped-at", "", "stop time in RFC3339 (required)")
	workordersCmd.AddCommand(workordersStopTimerCmd)
}
