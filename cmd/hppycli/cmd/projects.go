package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage projects",
}

var projectsCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new project from a template",
	Example: `  hppycli projects create --template-id=tmpl123 --location-id=225393 --start-at=2026-05-01T00:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		templateID, _ := cmd.Flags().GetString("template-id")
		if templateID == "" {
			return fmt.Errorf("--template-id is required")
		}
		if err := models.ValidateID("template-id", templateID); err != nil {
			return err
		}
		locationID, _ := cmd.Flags().GetString("location-id")
		if locationID == "" {
			return fmt.Errorf("--location-id is required")
		}
		if err := models.ValidateID("location-id", locationID); err != nil {
			return err
		}
		startAt, _ := cmd.Flags().GetString("start-at")
		if startAt == "" {
			return fmt.Errorf("--start-at is required (RFC3339 format)")
		}
		if err := models.ValidateTimestamp("start-at", startAt); err != nil {
			return err
		}

		input := models.ProjectCreateInput{
			ProjectTemplateID: templateID,
			LocationID:        locationID,
			StartAt:           startAt,
		}

		if v, _ := cmd.Flags().GetString("assignee-id"); v != "" {
			if err := models.ValidateID("assignee-id", v); err != nil {
				return err
			}
			input.AssigneeID = v
		}
		if v, _ := cmd.Flags().GetString("priority"); v != "" {
			upper := strings.ToUpper(v)
			if !models.ValidProjectPriorities[upper] {
				return fmt.Errorf("invalid --priority %q: must be NORMAL or URGENT", v)
			}
			input.Priority = upper
		}
		if v, _ := cmd.Flags().GetString("due-at"); v != "" {
			if err := models.ValidateTimestamp("due-at", v); err != nil {
				return err
			}
			input.DueAt = v
		}
		if v, _ := cmd.Flags().GetString("availability-target-at"); v != "" {
			if err := models.ValidateTimestamp("availability-target-at", v); err != nil {
				return err
			}
			input.AvailabilityTargetAt = v
		}
		if v, _ := cmd.Flags().GetString("notes"); v != "" {
			if err := models.ValidateFreeText("notes", v); err != nil {
				return err
			}
			input.Notes = v
		}

		proj, err := apiClient.ProjectCreate(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("creating project: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, proj)
	},
}

var projectsSetAssigneeCmd = &cobra.Command{
	Use:     "set-assignee",
	Short:   "Set or clear the project assignee",
	Example: `  hppycli projects set-assignee --id=proj123 --assignee-id=user456`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}

		input := models.ProjectSetAssigneeInput{ProjectID: id}
		if cmd.Flags().Changed("assignee-id") {
			v, _ := cmd.Flags().GetString("assignee-id")
			if v != "" {
				if err := models.ValidateID("assignee-id", v); err != nil {
					return err
				}
			}
			input.AssigneeID = &v
		}

		proj, err := apiClient.ProjectSetAssignee(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("setting project assignee: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, proj)
	},
}

var projectsSetNotesCmd = &cobra.Command{
	Use:     "set-notes",
	Short:   "Set the project notes",
	Example: `  hppycli projects set-notes --id=proj123 --notes="Updated project notes"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}
		notes, _ := cmd.Flags().GetString("notes")
		if notes == "" {
			return fmt.Errorf("--notes is required")
		}
		if err := models.ValidateFreeText("notes", notes); err != nil {
			return err
		}

		proj, err := apiClient.ProjectSetNotes(cmd.Context(), id, notes)
		if err != nil {
			return fmt.Errorf("setting project notes: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, proj)
	},
}

var projectsSetDueAtCmd = &cobra.Command{
	Use:     "set-due-at",
	Short:   "Set the project due date",
	Example: `  hppycli projects set-due-at --id=proj123 --due-at=2026-06-01T00:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}
		dueAt, _ := cmd.Flags().GetString("due-at")
		if dueAt == "" {
			return fmt.Errorf("--due-at is required (RFC3339 format)")
		}
		if err := models.ValidateTimestamp("due-at", dueAt); err != nil {
			return err
		}

		proj, err := apiClient.ProjectSetDueAt(cmd.Context(), id, dueAt)
		if err != nil {
			return fmt.Errorf("setting project due date: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, proj)
	},
}

var projectsSetStartAtCmd = &cobra.Command{
	Use:     "set-start-at",
	Short:   "Set the project start date",
	Example: `  hppycli projects set-start-at --id=proj123 --start-at=2026-05-01T00:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}
		startAt, _ := cmd.Flags().GetString("start-at")
		if startAt == "" {
			return fmt.Errorf("--start-at is required (RFC3339 format)")
		}
		if err := models.ValidateTimestamp("start-at", startAt); err != nil {
			return err
		}

		proj, err := apiClient.ProjectSetStartAt(cmd.Context(), id, startAt)
		if err != nil {
			return fmt.Errorf("setting project start date: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, proj)
	},
}

var projectsSetPriorityCmd = &cobra.Command{
	Use:     "set-priority",
	Short:   "Set the project priority",
	Example: `  hppycli projects set-priority --id=proj123 --priority=URGENT`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}
		priority, _ := cmd.Flags().GetString("priority")
		if priority == "" {
			return fmt.Errorf("--priority is required")
		}
		upper := strings.ToUpper(priority)
		if !models.ValidProjectPriorities[upper] {
			return fmt.Errorf("invalid --priority %q: must be NORMAL or URGENT", priority)
		}

		proj, err := apiClient.ProjectSetPriority(cmd.Context(), id, upper)
		if err != nil {
			return fmt.Errorf("setting project priority: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, proj)
	},
}

var projectsSetOnHoldCmd = &cobra.Command{
	Use:     "set-on-hold",
	Short:   "Set the project on-hold status",
	Example: `  hppycli projects set-on-hold --id=proj123 --on-hold=true`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}
		if !cmd.Flags().Changed("on-hold") {
			return fmt.Errorf("--on-hold is required")
		}
		onHold, _ := cmd.Flags().GetBool("on-hold")

		proj, err := apiClient.ProjectSetOnHold(cmd.Context(), id, onHold)
		if err != nil {
			return fmt.Errorf("setting project on-hold status: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, proj)
	},
}

var projectsSetAvailabilityTargetAtCmd = &cobra.Command{
	Use:     "set-availability-target-at",
	Short:   "Set the project availability target date",
	Example: `  hppycli projects set-availability-target-at --id=proj123 --availability-target-at=2026-05-15T00:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireFlagID(cmd, "id")
		if err != nil {
			return err
		}
		v, _ := cmd.Flags().GetString("availability-target-at")
		if v == "" {
			return fmt.Errorf("--availability-target-at is required (RFC3339 format)")
		}
		if err := models.ValidateTimestamp("availability-target-at", v); err != nil {
			return err
		}

		proj, err := apiClient.ProjectSetAvailabilityTargetAt(cmd.Context(), id, &v)
		if err != nil {
			return fmt.Errorf("setting project availability target date: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, proj)
	},
}

func init() {
	// Create
	projectsCreateCmd.Flags().String("template-id", "", "project template ID (required)")
	projectsCreateCmd.Flags().String("location-id", "", "property or unit ID (required)")
	projectsCreateCmd.Flags().String("start-at", "", "start date in RFC3339 (required)")
	projectsCreateCmd.Flags().String("assignee-id", "", "user ID to assign")
	projectsCreateCmd.Flags().String("priority", "", "priority: NORMAL or URGENT")
	projectsCreateCmd.Flags().String("due-at", "", "due date in RFC3339")
	projectsCreateCmd.Flags().String("availability-target-at", "", "availability target date in RFC3339")
	projectsCreateCmd.Flags().String("notes", "", "initial project notes")
	projectsCmd.AddCommand(projectsCreateCmd)

	// Set Assignee
	projectsSetAssigneeCmd.Flags().String("id", "", "project ID (required)")
	projectsSetAssigneeCmd.Flags().String("assignee-id", "", "user ID to assign (omit value to unassign)")
	projectsCmd.AddCommand(projectsSetAssigneeCmd)

	// Set Notes
	projectsSetNotesCmd.Flags().String("id", "", "project ID (required)")
	projectsSetNotesCmd.Flags().String("notes", "", "project notes (required)")
	projectsCmd.AddCommand(projectsSetNotesCmd)

	// Set Due At
	projectsSetDueAtCmd.Flags().String("id", "", "project ID (required)")
	projectsSetDueAtCmd.Flags().String("due-at", "", "due date in RFC3339 (required)")
	projectsCmd.AddCommand(projectsSetDueAtCmd)

	// Set Start At
	projectsSetStartAtCmd.Flags().String("id", "", "project ID (required)")
	projectsSetStartAtCmd.Flags().String("start-at", "", "start date in RFC3339 (required)")
	projectsCmd.AddCommand(projectsSetStartAtCmd)

	// Set Priority
	projectsSetPriorityCmd.Flags().String("id", "", "project ID (required)")
	projectsSetPriorityCmd.Flags().String("priority", "", "priority: NORMAL or URGENT (required)")
	projectsCmd.AddCommand(projectsSetPriorityCmd)

	// Set On Hold
	projectsSetOnHoldCmd.Flags().String("id", "", "project ID (required)")
	projectsSetOnHoldCmd.Flags().Bool("on-hold", false, "on-hold status (required)")
	projectsCmd.AddCommand(projectsSetOnHoldCmd)

	// Set Availability Target At
	projectsSetAvailabilityTargetAtCmd.Flags().String("id", "", "project ID (required)")
	projectsSetAvailabilityTargetAtCmd.Flags().String("availability-target-at", "", "availability target date in RFC3339 (required)")
	projectsCmd.AddCommand(projectsSetAvailabilityTargetAtCmd)
}
