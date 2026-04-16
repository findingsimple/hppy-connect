package cmd

import (
	"fmt"
	"os"

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

var inspectionsCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new inspection",
	Example: `  hppycli inspections create --location-id=225393 --template-id=tmpl123 --scheduled-for=2026-05-01T00:00:00Z`,
	RunE: func(cmd *cobra.Command, args []string) error {
		locationID, _ := cmd.Flags().GetString("location-id")
		if locationID == "" {
			return fmt.Errorf("--location-id is required")
		}
		if err := models.ValidateID("location-id", locationID); err != nil {
			return err
		}
		templateID, _ := cmd.Flags().GetString("template-id")
		if templateID == "" {
			return fmt.Errorf("--template-id is required")
		}
		if err := models.ValidateID("template-id", templateID); err != nil {
			return err
		}
		scheduledFor, _ := cmd.Flags().GetString("scheduled-for")
		if scheduledFor == "" {
			return fmt.Errorf("--scheduled-for is required (RFC3339 format)")
		}
		if err := models.ValidateTimestamp("scheduled-for", scheduledFor); err != nil {
			return err
		}

		input := models.InspectionCreateInput{
			LocationID:   locationID,
			TemplateID:   templateID,
			ScheduledFor: scheduledFor,
		}

		if v, _ := cmd.Flags().GetString("assignee-id"); v != "" {
			if err := models.ValidateID("assignee-id", v); err != nil {
				return err
			}
			input.AssignedToID = v
		}
		if v, _ := cmd.Flags().GetString("due-by"); v != "" {
			if err := models.ValidateTimestamp("due-by", v); err != nil {
				return err
			}
			input.DueBy = v
		}
		if cmd.Flags().Changed("expires") {
			v, _ := cmd.Flags().GetBool("expires")
			input.Expires = &v
		}

		insp, err := apiClient.InspectionCreate(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("creating inspection: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsStartCmd = &cobra.Command{
	Use:     "start",
	Short:   "Start an inspection",
	Example: `  hppycli inspections start --id=insp123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}

		insp, err := apiClient.InspectionStart(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("starting inspection: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsCompleteCmd = &cobra.Command{
	Use:     "complete",
	Short:   "Mark an inspection as complete",
	Example: `  hppycli inspections complete --id=insp123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}

		insp, err := apiClient.InspectionComplete(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("completing inspection: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsReopenCmd = &cobra.Command{
	Use:     "reopen",
	Short:   "Reopen a completed inspection",
	Example: `  hppycli inspections reopen --id=insp123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}

		insp, err := apiClient.InspectionReopen(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("reopening inspection: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsArchiveCmd = &cobra.Command{
	Use:     "archive",
	Short:   "Archive an inspection",
	Example: `  hppycli inspections archive --id=insp123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		if err := confirmAction(cmd, "archive inspection "+id, os.Stdin, os.Stderr); err != nil {
			return err
		}

		insp, err := apiClient.InspectionArchive(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("archiving inspection: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsExpireCmd = &cobra.Command{
	Use:     "expire",
	Short:   "Expire an inspection",
	Example: `  hppycli inspections expire --id=insp123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		if err := confirmAction(cmd, "expire inspection "+id, os.Stdin, os.Stderr); err != nil {
			return err
		}

		insp, err := apiClient.InspectionExpire(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("expiring inspection: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsUnexpireCmd = &cobra.Command{
	Use:     "unexpire",
	Short:   "Unexpire an inspection",
	Example: `  hppycli inspections unexpire --id=insp123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}

		insp, err := apiClient.InspectionUnexpire(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("unexpiring inspection: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsSetAssigneeCmd = &cobra.Command{
	Use:     "set-assignee",
	Short:   "Assign a user to an inspection",
	Example: `  hppycli inspections set-assignee --id=insp123 --assignee-id=user456`,
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

		insp, err := apiClient.InspectionSetAssignee(cmd.Context(), models.InspectionSetAssigneeInput{
			InspectionID: id,
			UserID:       assigneeID,
		})
		if err != nil {
			return fmt.Errorf("setting inspection assignee: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsSetDueByCmd = &cobra.Command{
	Use:     "set-due-by",
	Short:   "Set the due date for an inspection",
	Example: `  hppycli inspections set-due-by --id=insp123 --due-by=2026-06-01T00:00:00Z --expires=true`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		dueBy, _ := cmd.Flags().GetString("due-by")
		if dueBy == "" {
			return fmt.Errorf("--due-by is required (RFC3339 format)")
		}
		if err := models.ValidateTimestamp("due-by", dueBy); err != nil {
			return err
		}
		if !cmd.Flags().Changed("expires") {
			return fmt.Errorf("--expires is required (true or false)")
		}
		expires, _ := cmd.Flags().GetBool("expires")

		insp, err := apiClient.InspectionSetDueBy(cmd.Context(), models.InspectionSetDueByInput{
			InspectionID: id,
			DueBy:        dueBy,
			Expires:      expires,
		})
		if err != nil {
			return fmt.Errorf("setting inspection due date: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsSetScheduledForCmd = &cobra.Command{
	Use:     "set-scheduled-for",
	Short:   "Set the scheduled date for an inspection",
	Example: `  hppycli inspections set-scheduled-for --id=insp123 --scheduled-for=2026-05-01T09:00:00Z`,
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
			return fmt.Errorf("--scheduled-for is required (RFC3339 format)")
		}
		if err := models.ValidateTimestamp("scheduled-for", scheduledFor); err != nil {
			return err
		}

		insp, err := apiClient.InspectionSetScheduledFor(cmd.Context(), id, scheduledFor)
		if err != nil {
			return fmt.Errorf("setting inspection scheduled date: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsSetHeaderFieldCmd = &cobra.Command{
	Use:     "set-header-field",
	Short:   "Update a header field on an inspection",
	Example: `  hppycli inspections set-header-field --id=insp123 --label="Inspector" --value="John Smith"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		label, _ := cmd.Flags().GetString("label")
		if label == "" {
			return fmt.Errorf("--label is required")
		}
		value, _ := cmd.Flags().GetString("value")

		insp, err := apiClient.InspectionSetHeaderField(cmd.Context(), models.InspectionSetHeaderFieldInput{
			InspectionID: id,
			Label:        label,
			Value:        value,
		})
		if err != nil {
			return fmt.Errorf("setting header field: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsSetFooterFieldCmd = &cobra.Command{
	Use:     "set-footer-field",
	Short:   "Update a footer field on an inspection",
	Example: `  hppycli inspections set-footer-field --id=insp123 --label="Notes" --value="All clear"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		label, _ := cmd.Flags().GetString("label")
		if label == "" {
			return fmt.Errorf("--label is required")
		}
		value, _ := cmd.Flags().GetString("value")

		insp, err := apiClient.InspectionSetFooterField(cmd.Context(), models.InspectionSetFooterFieldInput{
			InspectionID: id,
			Label:        label,
			Value:        value,
		})
		if err != nil {
			return fmt.Errorf("setting footer field: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsSetItemNotesCmd = &cobra.Command{
	Use:     "set-item-notes",
	Short:   "Set notes on an inspection item",
	Example: `  hppycli inspections set-item-notes --id=insp123 --section="Kitchen" --item="Sink" --notes="Needs repair"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		section, _ := cmd.Flags().GetString("section")
		if section == "" {
			return fmt.Errorf("--section is required")
		}
		item, _ := cmd.Flags().GetString("item")
		if item == "" {
			return fmt.Errorf("--item is required")
		}
		notes, _ := cmd.Flags().GetString("notes")

		insp, err := apiClient.InspectionSetItemNotes(cmd.Context(), models.InspectionSetItemNotesInput{
			InspectionID: id,
			SectionName:  section,
			ItemName:     item,
			Notes:        notes,
		})
		if err != nil {
			return fmt.Errorf("setting item notes: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsRateItemCmd = &cobra.Command{
	Use:     "rate-item",
	Short:   "Rate an item in an inspection",
	Example: `  hppycli inspections rate-item --id=insp123 --section="Kitchen" --item="Sink" --rating-key=condition --rating-score=3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		section, _ := cmd.Flags().GetString("section")
		if section == "" {
			return fmt.Errorf("--section is required")
		}
		item, _ := cmd.Flags().GetString("item")
		if item == "" {
			return fmt.Errorf("--item is required")
		}
		ratingKey, _ := cmd.Flags().GetString("rating-key")
		if ratingKey == "" {
			return fmt.Errorf("--rating-key is required")
		}

		rating := models.InspectionRatingInput{Key: ratingKey}
		if cmd.Flags().Changed("rating-score") {
			score, _ := cmd.Flags().GetFloat64("rating-score")
			rating.Score = &score
		}
		if v, _ := cmd.Flags().GetString("rating-value"); v != "" {
			rating.Value = v
		}

		insp, err := apiClient.InspectionRateItem(cmd.Context(), models.InspectionRateItemInput{
			InspectionID: id,
			SectionName:  section,
			ItemName:     item,
			Rating:       rating,
		})
		if err != nil {
			return fmt.Errorf("rating item: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsAddSectionCmd = &cobra.Command{
	Use:     "add-section",
	Short:   "Add a new section to an inspection",
	Example: `  hppycli inspections add-section --id=insp123 --name="Living Room"`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		insp, err := apiClient.InspectionAddSection(cmd.Context(), models.InspectionAddSectionInput{
			InspectionID: id,
			Name:         name,
		})
		if err != nil {
			return fmt.Errorf("adding section: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsDeleteSectionCmd = &cobra.Command{
	Use:     "delete-section",
	Short:   "Delete a section from an inspection",
	Example: `  hppycli inspections delete-section --id=insp123 --section="Living Room"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		section, _ := cmd.Flags().GetString("section")
		if section == "" {
			return fmt.Errorf("--section is required")
		}
		if err := confirmAction(cmd, "delete section \""+section+"\" from inspection "+id, os.Stdin, os.Stderr); err != nil {
			return err
		}

		insp, err := apiClient.InspectionDeleteSection(cmd.Context(), models.InspectionDeleteSectionInput{
			InspectionID: id,
			SectionName:  section,
		})
		if err != nil {
			return fmt.Errorf("deleting section: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsDuplicateSectionCmd = &cobra.Command{
	Use:     "duplicate-section",
	Short:   "Duplicate a section in an inspection",
	Example: `  hppycli inspections duplicate-section --id=insp123 --section="Bedroom 1"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		section, _ := cmd.Flags().GetString("section")
		if section == "" {
			return fmt.Errorf("--section is required")
		}

		insp, err := apiClient.InspectionDuplicateSection(cmd.Context(), models.InspectionDuplicateSectionInput{
			InspectionID: id,
			SectionName:  section,
		})
		if err != nil {
			return fmt.Errorf("duplicating section: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsRenameSectionCmd = &cobra.Command{
	Use:     "rename-section",
	Short:   "Rename a section in an inspection",
	Example: `  hppycli inspections rename-section --id=insp123 --section="Bedroom 1" --new-name="Master Bedroom"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		section, _ := cmd.Flags().GetString("section")
		if section == "" {
			return fmt.Errorf("--section is required")
		}
		newName, _ := cmd.Flags().GetString("new-name")
		if newName == "" {
			return fmt.Errorf("--new-name is required")
		}

		insp, err := apiClient.InspectionRenameSection(cmd.Context(), models.InspectionRenameSectionInput{
			InspectionID:   id,
			SectionName:    section,
			NewSectionName: newName,
		})
		if err != nil {
			return fmt.Errorf("renaming section: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsAddItemCmd = &cobra.Command{
	Use:     "add-item",
	Short:   "Add an item to a section in an inspection",
	Example: `  hppycli inspections add-item --id=insp123 --section="Kitchen" --name="Dishwasher" --rating-group-id=rg456`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		section, _ := cmd.Flags().GetString("section")
		if section == "" {
			return fmt.Errorf("--section is required")
		}
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		ratingGroupID, _ := cmd.Flags().GetString("rating-group-id")
		if ratingGroupID == "" {
			return fmt.Errorf("--rating-group-id is required")
		}
		if err := models.ValidateID("rating-group-id", ratingGroupID); err != nil {
			return err
		}

		input := models.InspectionAddItemInput{
			InspectionID:  id,
			SectionName:   section,
			Name:          name,
			RatingGroupID: ratingGroupID,
		}
		if v, _ := cmd.Flags().GetString("info"); v != "" {
			input.Info = v
		}

		insp, err := apiClient.InspectionAddItem(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("adding item: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsDeleteItemCmd = &cobra.Command{
	Use:     "delete-item",
	Short:   "Delete an item from a section in an inspection",
	Example: `  hppycli inspections delete-item --id=insp123 --section="Kitchen" --item="Dishwasher"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		section, _ := cmd.Flags().GetString("section")
		if section == "" {
			return fmt.Errorf("--section is required")
		}
		item, _ := cmd.Flags().GetString("item")
		if item == "" {
			return fmt.Errorf("--item is required")
		}
		if err := confirmAction(cmd, "delete item \""+item+"\" from section \""+section+"\" in inspection "+id, os.Stdin, os.Stderr); err != nil {
			return err
		}

		insp, err := apiClient.InspectionDeleteItem(cmd.Context(), models.InspectionDeleteItemInput{
			InspectionID: id,
			SectionName:  section,
			ItemName:     item,
		})
		if err != nil {
			return fmt.Errorf("deleting item: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsAddItemPhotoCmd = &cobra.Command{
	Use:     "add-item-photo",
	Short:   "Add a photo to an inspection item (returns signed upload URL)",
	Example: `  hppycli inspections add-item-photo --id=insp123 --section="Kitchen" --item="Sink" --mime-type=image/jpeg`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		section, _ := cmd.Flags().GetString("section")
		if section == "" {
			return fmt.Errorf("--section is required")
		}
		item, _ := cmd.Flags().GetString("item")
		if item == "" {
			return fmt.Errorf("--item is required")
		}
		mimeType, _ := cmd.Flags().GetString("mime-type")
		if mimeType == "" {
			return fmt.Errorf("--mime-type is required")
		}
		if err := models.ValidateMIMEType(mimeType); err != nil {
			return err
		}

		input := models.InspectionAddItemPhotoInput{
			InspectionID: id,
			SectionName:  section,
			ItemName:     item,
			MimeType:     mimeType,
		}
		if cmd.Flags().Changed("size") {
			size, _ := cmd.Flags().GetInt("size")
			input.Size = &size
		}

		result, err := apiClient.InspectionAddItemPhoto(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("adding item photo: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, result)
	},
}

var inspectionsRemoveItemPhotoCmd = &cobra.Command{
	Use:     "remove-item-photo",
	Short:   "Remove a photo from an inspection item",
	Example: `  hppycli inspections remove-item-photo --id=insp123 --photo-id=ph456 --section="Kitchen" --item="Sink"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		photoID, _ := cmd.Flags().GetString("photo-id")
		if photoID == "" {
			return fmt.Errorf("--photo-id is required")
		}
		if err := models.ValidateID("photo-id", photoID); err != nil {
			return err
		}
		section, _ := cmd.Flags().GetString("section")
		if section == "" {
			return fmt.Errorf("--section is required")
		}
		item, _ := cmd.Flags().GetString("item")
		if item == "" {
			return fmt.Errorf("--item is required")
		}
		if err := confirmAction(cmd, "remove photo "+photoID+" from item \""+item+"\" in inspection "+id, os.Stdin, os.Stderr); err != nil {
			return err
		}

		insp, err := apiClient.InspectionRemoveItemPhoto(cmd.Context(), models.InspectionRemoveItemPhotoInput{
			InspectionID: id,
			PhotoID:      photoID,
			SectionName:  section,
			ItemName:     item,
		})
		if err != nil {
			return fmt.Errorf("removing item photo: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsMoveItemPhotoCmd = &cobra.Command{
	Use:     "move-item-photo",
	Short:   "Move a photo between items in an inspection",
	Example: `  hppycli inspections move-item-photo --id=insp123 --photo-id=ph456 --from-section="Kitchen" --from-item="Sink" --to-section="Bathroom" --to-item="Faucet"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		photoID, _ := cmd.Flags().GetString("photo-id")
		if photoID == "" {
			return fmt.Errorf("--photo-id is required")
		}
		if err := models.ValidateID("photo-id", photoID); err != nil {
			return err
		}
		fromSection, _ := cmd.Flags().GetString("from-section")
		if fromSection == "" {
			return fmt.Errorf("--from-section is required")
		}
		fromItem, _ := cmd.Flags().GetString("from-item")
		if fromItem == "" {
			return fmt.Errorf("--from-item is required")
		}
		toSection, _ := cmd.Flags().GetString("to-section")
		if toSection == "" {
			return fmt.Errorf("--to-section is required")
		}
		toItem, _ := cmd.Flags().GetString("to-item")
		if toItem == "" {
			return fmt.Errorf("--to-item is required")
		}

		insp, err := apiClient.InspectionMoveItemPhoto(cmd.Context(), models.InspectionMoveItemPhotoInput{
			InspectionID:    id,
			PhotoID:         photoID,
			FromSectionName: fromSection,
			FromItemName:    fromItem,
			ToSectionName:   toSection,
			ToItemName:      toItem,
		})
		if err != nil {
			return fmt.Errorf("moving item photo: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, insp)
	},
}

var inspectionsSendToGuestCmd = &cobra.Command{
	Use:     "send-to-guest",
	Short:   "Send an inspection to a guest via email",
	Example: `  hppycli inspections send-to-guest --id=insp123 --email=guest@example.com --name="Jane Doe"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}
		email, _ := cmd.Flags().GetString("email")
		if email == "" {
			return fmt.Errorf("--email is required")
		}
		if err := models.ValidateEmail(email); err != nil {
			return err
		}

		input := models.InspectionSendToGuestInput{
			InspectionID: id,
			Email:        email,
		}
		if v, _ := cmd.Flags().GetString("name"); v != "" {
			if err := models.ValidateFreeText("name", v); err != nil {
				return err
			}
			input.Name = v
		}
		if v, _ := cmd.Flags().GetString("message"); v != "" {
			if err := models.ValidateFreeText("message", v); err != nil {
				return err
			}
			input.Message = v
		}
		if v, _ := cmd.Flags().GetString("due-date"); v != "" {
			if err := models.ValidateTimestamp("due-date", v); err != nil {
				return err
			}
			input.DueDate = v
		}
		if cmd.Flags().Changed("expires") {
			v, _ := cmd.Flags().GetBool("expires")
			input.Expires = &v
		}

		if err := confirmAction(cmd, fmt.Sprintf("send inspection %s to %s", id, email), os.Stdin, os.Stderr); err != nil {
			return err
		}

		result, err := apiClient.InspectionSendToGuest(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("sending to guest: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, result)
	},
}

func init() {
	// List command
	addListFlags(inspectionsListCmd, "filter by status: COMPLETE, EXPIRED, INCOMPLETE, SCHEDULED")
	inspectionsCmd.AddCommand(inspectionsListCmd)

	// Create
	inspectionsCreateCmd.Flags().String("location-id", "", "property or unit ID (required)")
	inspectionsCreateCmd.Flags().String("template-id", "", "inspection template ID (required)")
	inspectionsCreateCmd.Flags().String("scheduled-for", "", "scheduled date in RFC3339 (required)")
	inspectionsCreateCmd.Flags().String("assignee-id", "", "user ID to assign")
	inspectionsCreateCmd.Flags().String("due-by", "", "due date in RFC3339")
	inspectionsCreateCmd.Flags().Bool("expires", false, "whether the inspection expires at the due date")
	inspectionsCmd.AddCommand(inspectionsCreateCmd)

	// Start
	inspectionsStartCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsCmd.AddCommand(inspectionsStartCmd)

	// Complete
	inspectionsCompleteCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsCmd.AddCommand(inspectionsCompleteCmd)

	// Reopen
	inspectionsReopenCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsCmd.AddCommand(inspectionsReopenCmd)

	// Archive (destructive)
	inspectionsArchiveCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsArchiveCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	inspectionsCmd.AddCommand(inspectionsArchiveCmd)

	// Expire (destructive)
	inspectionsExpireCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsExpireCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	inspectionsCmd.AddCommand(inspectionsExpireCmd)

	// Unexpire
	inspectionsUnexpireCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsCmd.AddCommand(inspectionsUnexpireCmd)

	// Set Assignee
	inspectionsSetAssigneeCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsSetAssigneeCmd.Flags().String("assignee-id", "", "user ID to assign (required)")
	inspectionsCmd.AddCommand(inspectionsSetAssigneeCmd)

	// Set Due By
	inspectionsSetDueByCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsSetDueByCmd.Flags().String("due-by", "", "due date in RFC3339 (required)")
	inspectionsSetDueByCmd.Flags().Bool("expires", false, "whether the inspection expires at the due date (required)")
	inspectionsCmd.AddCommand(inspectionsSetDueByCmd)

	// Set Scheduled For
	inspectionsSetScheduledForCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsSetScheduledForCmd.Flags().String("scheduled-for", "", "scheduled date in RFC3339 (required)")
	inspectionsCmd.AddCommand(inspectionsSetScheduledForCmd)

	// Set Header Field
	inspectionsSetHeaderFieldCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsSetHeaderFieldCmd.Flags().String("label", "", "field label (required)")
	inspectionsSetHeaderFieldCmd.Flags().String("value", "", "field value (omit to clear)")
	inspectionsCmd.AddCommand(inspectionsSetHeaderFieldCmd)

	// Set Footer Field
	inspectionsSetFooterFieldCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsSetFooterFieldCmd.Flags().String("label", "", "field label (required)")
	inspectionsSetFooterFieldCmd.Flags().String("value", "", "field value (omit to clear)")
	inspectionsCmd.AddCommand(inspectionsSetFooterFieldCmd)

	// Set Item Notes
	inspectionsSetItemNotesCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsSetItemNotesCmd.Flags().String("section", "", "section name (required)")
	inspectionsSetItemNotesCmd.Flags().String("item", "", "item name (required)")
	inspectionsSetItemNotesCmd.Flags().String("notes", "", "notes text (omit to clear)")
	inspectionsCmd.AddCommand(inspectionsSetItemNotesCmd)

	// Rate Item
	inspectionsRateItemCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsRateItemCmd.Flags().String("section", "", "section name (required)")
	inspectionsRateItemCmd.Flags().String("item", "", "item name (required)")
	inspectionsRateItemCmd.Flags().String("rating-key", "", "rating key (required)")
	inspectionsRateItemCmd.Flags().Float64("rating-score", 0, "numeric rating score")
	inspectionsRateItemCmd.Flags().String("rating-value", "", "rating value string")
	inspectionsCmd.AddCommand(inspectionsRateItemCmd)

	// Add Section
	inspectionsAddSectionCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsAddSectionCmd.Flags().String("name", "", "section name (required)")
	inspectionsCmd.AddCommand(inspectionsAddSectionCmd)

	// Delete Section (destructive)
	inspectionsDeleteSectionCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsDeleteSectionCmd.Flags().String("section", "", "section name (required)")
	inspectionsDeleteSectionCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	inspectionsCmd.AddCommand(inspectionsDeleteSectionCmd)

	// Duplicate Section
	inspectionsDuplicateSectionCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsDuplicateSectionCmd.Flags().String("section", "", "section name (required)")
	inspectionsCmd.AddCommand(inspectionsDuplicateSectionCmd)

	// Rename Section
	inspectionsRenameSectionCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsRenameSectionCmd.Flags().String("section", "", "current section name (required)")
	inspectionsRenameSectionCmd.Flags().String("new-name", "", "new section name (required)")
	inspectionsCmd.AddCommand(inspectionsRenameSectionCmd)

	// Add Item
	inspectionsAddItemCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsAddItemCmd.Flags().String("section", "", "section name (required)")
	inspectionsAddItemCmd.Flags().String("name", "", "item name (required)")
	inspectionsAddItemCmd.Flags().String("rating-group-id", "", "rating group ID (required)")
	inspectionsAddItemCmd.Flags().String("info", "", "explanatory text about the item")
	inspectionsCmd.AddCommand(inspectionsAddItemCmd)

	// Delete Item (destructive)
	inspectionsDeleteItemCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsDeleteItemCmd.Flags().String("section", "", "section name (required)")
	inspectionsDeleteItemCmd.Flags().String("item", "", "item name (required)")
	inspectionsDeleteItemCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	inspectionsCmd.AddCommand(inspectionsDeleteItemCmd)

	// Add Item Photo
	inspectionsAddItemPhotoCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsAddItemPhotoCmd.Flags().String("section", "", "section name (required)")
	inspectionsAddItemPhotoCmd.Flags().String("item", "", "item name (required)")
	inspectionsAddItemPhotoCmd.Flags().String("mime-type", "", "photo MIME type (required)")
	inspectionsAddItemPhotoCmd.Flags().Int("size", 0, "photo size in bytes")
	inspectionsCmd.AddCommand(inspectionsAddItemPhotoCmd)

	// Remove Item Photo (destructive)
	inspectionsRemoveItemPhotoCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsRemoveItemPhotoCmd.Flags().String("photo-id", "", "photo ID (required)")
	inspectionsRemoveItemPhotoCmd.Flags().String("section", "", "section name (required)")
	inspectionsRemoveItemPhotoCmd.Flags().String("item", "", "item name (required)")
	inspectionsRemoveItemPhotoCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	inspectionsCmd.AddCommand(inspectionsRemoveItemPhotoCmd)

	// Move Item Photo
	inspectionsMoveItemPhotoCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsMoveItemPhotoCmd.Flags().String("photo-id", "", "photo ID (required)")
	inspectionsMoveItemPhotoCmd.Flags().String("from-section", "", "source section name (required)")
	inspectionsMoveItemPhotoCmd.Flags().String("from-item", "", "source item name (required)")
	inspectionsMoveItemPhotoCmd.Flags().String("to-section", "", "destination section name (required)")
	inspectionsMoveItemPhotoCmd.Flags().String("to-item", "", "destination item name (required)")
	inspectionsCmd.AddCommand(inspectionsMoveItemPhotoCmd)

	// Send To Guest
	inspectionsSendToGuestCmd.Flags().String("id", "", "inspection ID (required)")
	inspectionsSendToGuestCmd.Flags().String("email", "", "guest email address (required)")
	inspectionsSendToGuestCmd.Flags().String("name", "", "guest name")
	inspectionsSendToGuestCmd.Flags().String("message", "", "message to include in the email")
	inspectionsSendToGuestCmd.Flags().String("due-date", "", "due date in RFC3339")
	inspectionsSendToGuestCmd.Flags().Bool("expires", false, "whether the guest link expires")
	inspectionsSendToGuestCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	inspectionsCmd.AddCommand(inspectionsSendToGuestCmd)
}
