package main

import (
	"context"
	"strings"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Work Order MCP Tool Input Structs ---

type CreateWorkOrderInput struct {
	LocationID        string `json:"location_id" jsonschema:"required,The property or unit ID for the work order"`
	Description       string `json:"description,omitempty" jsonschema:"Description of work needed"`
	Priority          string `json:"priority,omitempty" jsonschema:"Priority: NORMAL or URGENT (default NORMAL)"`
	Status            string `json:"status,omitempty" jsonschema:"Status: OPEN ON_HOLD or COMPLETED (default OPEN)"`
	Type              string `json:"type,omitempty" jsonschema:"Type: SERVICE_REQUEST TURN CAPITAL_IMPROVEMENT CURB_APPEAL INCIDENT INVENTORY LIFE_SAFETY PREVENTATIVE_MAINTENANCE REGULATORY SEASONAL_MAINTENANCE"`
	ScheduledFor      string `json:"scheduled_for,omitempty" jsonschema:"Scheduled date in ISO 8601 format"`
	EntryNotes        string `json:"entry_notes,omitempty" jsonschema:"Notes about entering the location"`
	PermissionToEnter *bool  `json:"permission_to_enter,omitempty" jsonschema:"Whether permission to enter has been granted"`
	AssigneeID        string `json:"assignee_id,omitempty" jsonschema:"User or vendor ID to assign"`
	AssigneeType      string `json:"assignee_type,omitempty" jsonschema:"Assignee type: USER or VENDOR (default USER)"`
}

type SetWorkOrderStatusInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	Status      string `json:"status" jsonschema:"required,Status: OPEN ON_HOLD or COMPLETED"`
	SubStatus   string `json:"sub_status,omitempty" jsonschema:"Sub-status: CANCELLED or UNKNOWN (default UNKNOWN)"`
	Comment     string `json:"comment,omitempty" jsonschema:"Optional comment when completing"`
}

type SetWorkOrderAssigneeInput struct {
	WorkOrderID  string `json:"work_order_id" jsonschema:"required,The work order ID"`
	AssigneeID   string `json:"assignee_id" jsonschema:"required,The user or vendor ID to assign"`
	AssigneeType string `json:"assignee_type,omitempty" jsonschema:"Assignee type: USER or VENDOR (default USER)"`
}

type SetWorkOrderDescriptionMCPInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	Description string `json:"description" jsonschema:"required,The description text"`
}

type SetWorkOrderPriorityMCPInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	Priority    string `json:"priority" jsonschema:"required,Priority: NORMAL or URGENT"`
}

type SetWorkOrderScheduledForMCPInput struct {
	WorkOrderID  string `json:"work_order_id" jsonschema:"required,The work order ID"`
	ScheduledFor string `json:"scheduled_for" jsonschema:"required,Scheduled date in ISO 8601 format"`
}

type SetWorkOrderLocationMCPInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	LocationID  string `json:"location_id" jsonschema:"required,The property or unit ID"`
}

type SetWorkOrderTypeMCPInput struct {
	WorkOrderID   string `json:"work_order_id" jsonschema:"required,The work order ID"`
	WorkOrderType string `json:"work_order_type" jsonschema:"required,Type: SERVICE_REQUEST TURN CAPITAL_IMPROVEMENT CURB_APPEAL INCIDENT INVENTORY LIFE_SAFETY PREVENTATIVE_MAINTENANCE REGULATORY SEASONAL_MAINTENANCE"`
}

type SetWorkOrderEntryNotesMCPInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	EntryNotes  string `json:"entry_notes" jsonschema:"required,Notes about entering the location"`
}

type AddWorkOrderCommentMCPInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	Comment     string `json:"comment" jsonschema:"required,The comment text"`
}

type AddWorkOrderTimeMCPInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	Duration    string `json:"duration" jsonschema:"required,Time spent in ISO 8601 duration format (e.g. PT1H30M)"`
}

type SetWorkOrderPermissionToEnterMCPInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	// *bool (not bool) so omission is detectable. A plain bool defaults to false
	// silently — for state-flip flags that's a security-meaningful default.
	PermissionToEnter *bool `json:"permission_to_enter" jsonschema:"required,Whether permission to enter has been granted (true or false)"`
}

type SetWorkOrderResidentApprovedEntryMCPInput struct {
	WorkOrderID           string `json:"work_order_id" jsonschema:"required,The work order ID"`
	ResidentApprovedEntry *bool  `json:"resident_approved_entry" jsonschema:"required,Whether the resident has approved entry (true or false)"`
}

type SetWorkOrderUnitEnteredMCPInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	UnitEntered *bool  `json:"unit_entered" jsonschema:"required,Whether the unit has been entered (true or false)"`
}

type WorkOrderIDOnlyInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
}

type AddWorkOrderAttachmentInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	FileName    string `json:"file_name" jsonschema:"required,The attachment file name"`
	MimeType    string `json:"mime_type" jsonschema:"required,The MIME type of the attachment"`
	Size        *int   `json:"size,omitempty" jsonschema:"File size in bytes if known"`
}

type RemoveWorkOrderAttachmentInput struct {
	WorkOrderID  string `json:"work_order_id" jsonschema:"required,The work order ID"`
	AttachmentID string `json:"attachment_id" jsonschema:"required,The attachment ID to remove"`
}

type WorkOrderTimerInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	Timestamp   string `json:"timestamp" jsonschema:"required,The timestamp in ISO 8601 format"`
}

// registerWorkOrderMutationTools registers all 19 work order mutation tools.
func registerWorkOrderMutationTools(server *mcp.Server, client apiClient, debug bool) {
	destructive := true

	// work_order_create
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_create",
			Description: "Create a new work order for a property or unit",
		},
		wrapTool(debug, "work_order_create", func(ctx context.Context, _ *mcp.CallToolRequest, input CreateWorkOrderInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("location_id", input.LocationID); errResult != nil {
				return errResult, nil, nil
			}

			apiInput := models.WorkOrderCreateInput{LocationID: input.LocationID}
			if input.Description != "" {
				if err := models.ValidateFreeText("description", input.Description); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				apiInput.Description = input.Description
			}
			if input.ScheduledFor != "" {
				if err := models.ValidateTimestamp("scheduled_for", input.ScheduledFor); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				apiInput.ScheduledFor = input.ScheduledFor
			}
			if input.EntryNotes != "" {
				if err := models.ValidateFreeText("entry_notes", input.EntryNotes); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				apiInput.EntryNotes = input.EntryNotes
			}
			apiInput.PermissionToEnter = input.PermissionToEnter

			if input.Priority != "" {
				upper := strings.ToUpper(input.Priority)
				if !models.ValidWorkOrderPriorities[upper] {
					return toolInputError("priority must be NORMAL or URGENT"), nil, nil
				}
				apiInput.Priority = upper
			}
			if input.Status != "" {
				upper := strings.ToUpper(input.Status)
				if !models.ValidWorkOrderStatuses[upper] {
					return toolInputError("status must be OPEN, ON_HOLD, or COMPLETED"), nil, nil
				}
				apiInput.Status = upper
			}
			if input.Type != "" {
				upper := strings.ToUpper(input.Type)
				if !models.ValidWorkOrderTypes[upper] {
					return toolInputError("type must be one of SERVICE_REQUEST, TURN, CAPITAL_IMPROVEMENT, CURB_APPEAL, INCIDENT, INVENTORY, LIFE_SAFETY, PREVENTATIVE_MAINTENANCE, REGULATORY, SEASONAL_MAINTENANCE"), nil, nil
				}
				apiInput.Type = upper
			}
			if input.AssigneeID != "" {
				if err := models.ValidateID("assignee_id", input.AssigneeID); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				aType := "USER"
				if input.AssigneeType != "" {
					aType = strings.ToUpper(input.AssigneeType)
					if !models.ValidWorkOrderAssigneeTypes[aType] {
						return toolInputError("assignee_type must be USER or VENDOR"), nil, nil
					}
				}
				apiInput.Assignee = &models.AssignableInput{AssigneeID: input.AssigneeID, AssigneeType: aType}
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderCreate(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_set_status
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_set_status",
			Description: "Set the status and sub-status of a work order",
		},
		wrapTool(debug, "work_order_set_status", func(ctx context.Context, _ *mcp.CallToolRequest, input SetWorkOrderStatusInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Status == "" {
				return toolInputError("status is required"), nil, nil
			}
			upper := strings.ToUpper(input.Status)
			if !models.ValidWorkOrderStatuses[upper] {
				return toolInputError("status must be OPEN, ON_HOLD, or COMPLETED"), nil, nil
			}
			subUpper := "UNKNOWN"
			if input.SubStatus != "" {
				subUpper = strings.ToUpper(input.SubStatus)
				if !models.ValidWorkOrderSubStatuses[subUpper] {
					return toolInputError("sub_status must be CANCELLED or UNKNOWN"), nil, nil
				}
			}
			if input.Comment != "" {
				if err := models.ValidateFreeText("comment", input.Comment); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
			}

			apiInput := models.WorkOrderSetStatusAndSubStatusInput{
				WorkOrderID: input.WorkOrderID,
				Status:      models.WorkOrderStatusInput{Status: upper, Comment: input.Comment},
				SubStatus:   models.WorkOrderSubStatusInput{SubStatus: subUpper},
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetStatusAndSubStatus(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_set_assignee
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_set_assignee",
			Description: "Set the user or vendor assigned to a work order",
		},
		wrapTool(debug, "work_order_set_assignee", func(ctx context.Context, _ *mcp.CallToolRequest, input SetWorkOrderAssigneeInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("assignee_id", input.AssigneeID); errResult != nil {
				return errResult, nil, nil
			}
			aType := "USER"
			if input.AssigneeType != "" {
				aType = strings.ToUpper(input.AssigneeType)
				if !models.ValidWorkOrderAssigneeTypes[aType] {
					return toolInputError("assignee_type must be USER or VENDOR"), nil, nil
				}
			}

			apiInput := models.WorkOrderSetAssigneeInput{
				WorkOrderID: input.WorkOrderID,
				Assignee:    models.AssignableInput{AssigneeID: input.AssigneeID, AssigneeType: aType},
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetAssignee(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_set_description
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_set_description",
			Description: "Set the description of a work order",
		},
		wrapTool(debug, "work_order_set_description", func(ctx context.Context, _ *mcp.CallToolRequest, input SetWorkOrderDescriptionMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Description == "" {
				return toolInputError("description is required"), nil, nil
			}
			if err := models.ValidateFreeText("description", input.Description); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetDescription(ctx, input.WorkOrderID, input.Description)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_set_priority
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_set_priority",
			Description: "Set the priority of a work order (NORMAL or URGENT)",
		},
		wrapTool(debug, "work_order_set_priority", func(ctx context.Context, _ *mcp.CallToolRequest, input SetWorkOrderPriorityMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Priority == "" {
				return toolInputError("priority is required"), nil, nil
			}
			upper := strings.ToUpper(input.Priority)
			if !models.ValidWorkOrderPriorities[upper] {
				return toolInputError("priority must be NORMAL or URGENT"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetPriority(ctx, input.WorkOrderID, upper)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_set_scheduled_for
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_set_scheduled_for",
			Description: "Set the scheduled date for a work order (ISO 8601 format)",
		},
		wrapTool(debug, "work_order_set_scheduled_for", func(ctx context.Context, _ *mcp.CallToolRequest, input SetWorkOrderScheduledForMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.ScheduledFor == "" {
				return toolInputError("scheduled_for is required"), nil, nil
			}
			if err := models.ValidateTimestamp("scheduled_for", input.ScheduledFor); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetScheduledFor(ctx, input.WorkOrderID, input.ScheduledFor)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_set_location
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_set_location",
			Description: "Set the location (property or unit) of a work order",
		},
		wrapTool(debug, "work_order_set_location", func(ctx context.Context, _ *mcp.CallToolRequest, input SetWorkOrderLocationMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("location_id", input.LocationID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetLocation(ctx, input.WorkOrderID, input.LocationID)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_set_type
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_set_type",
			Description: "Set the type of a work order (SERVICE_REQUEST, TURN, etc.)",
		},
		wrapTool(debug, "work_order_set_type", func(ctx context.Context, _ *mcp.CallToolRequest, input SetWorkOrderTypeMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.WorkOrderType == "" {
				return toolInputError("work_order_type is required"), nil, nil
			}
			upper := strings.ToUpper(input.WorkOrderType)
			if !models.ValidWorkOrderTypes[upper] {
				return toolInputError("type must be one of SERVICE_REQUEST, TURN, CAPITAL_IMPROVEMENT, CURB_APPEAL, INCIDENT, INVENTORY, LIFE_SAFETY, PREVENTATIVE_MAINTENANCE, REGULATORY, SEASONAL_MAINTENANCE"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetType(ctx, input.WorkOrderID, upper)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_set_entry_notes
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_set_entry_notes",
			Description: "Set the entry notes of a work order",
		},
		wrapTool(debug, "work_order_set_entry_notes", func(ctx context.Context, _ *mcp.CallToolRequest, input SetWorkOrderEntryNotesMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.EntryNotes == "" {
				return toolInputError("entry_notes is required"), nil, nil
			}
			if err := models.ValidateFreeText("entry_notes", input.EntryNotes); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetEntryNotes(ctx, input.WorkOrderID, input.EntryNotes)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_set_permission_to_enter
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_set_permission_to_enter",
			Description: "Set the permission to enter flag on a work order",
		},
		wrapTool(debug, "work_order_set_permission_to_enter", func(ctx context.Context, _ *mcp.CallToolRequest, input SetWorkOrderPermissionToEnterMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.PermissionToEnter == nil {
				return toolInputError("permission_to_enter is required (true or false)"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetPermissionToEnter(ctx, input.WorkOrderID, *input.PermissionToEnter)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_set_resident_approved_entry
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_set_resident_approved_entry",
			Description: "Set the resident approved entry flag on a work order",
		},
		wrapTool(debug, "work_order_set_resident_approved_entry", func(ctx context.Context, _ *mcp.CallToolRequest, input SetWorkOrderResidentApprovedEntryMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.ResidentApprovedEntry == nil {
				return toolInputError("resident_approved_entry is required (true or false)"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetResidentApprovedEntry(ctx, input.WorkOrderID, *input.ResidentApprovedEntry)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_set_unit_entered
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_set_unit_entered",
			Description: "Set the unit entered flag on a work order",
		},
		wrapTool(debug, "work_order_set_unit_entered", func(ctx context.Context, _ *mcp.CallToolRequest, input SetWorkOrderUnitEnteredMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.UnitEntered == nil {
				return toolInputError("unit_entered is required (true or false)"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetUnitEntered(ctx, input.WorkOrderID, *input.UnitEntered)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_archive (destructive)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_archive",
			Description: "Archive a work order (irreversible)",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "work_order_archive", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDOnlyInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderArchive(ctx, input.WorkOrderID)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_add_comment
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_add_comment",
			Description: "Add a comment to a work order",
		},
		wrapTool(debug, "work_order_add_comment", func(ctx context.Context, _ *mcp.CallToolRequest, input AddWorkOrderCommentMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Comment == "" {
				return toolInputError("comment is required"), nil, nil
			}
			if err := models.ValidateFreeText("comment", input.Comment); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderAddComment(ctx, input.WorkOrderID, input.Comment)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_add_time
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_add_time",
			Description: "Add time spent on a work order (ISO 8601 duration, e.g. PT1H30M)",
		},
		wrapTool(debug, "work_order_add_time", func(ctx context.Context, _ *mcp.CallToolRequest, input AddWorkOrderTimeMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Duration == "" {
				return toolInputError("duration is required (ISO 8601 format, e.g. PT1H30M)"), nil, nil
			}
			if err := models.ValidateDuration(input.Duration); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderAddTime(ctx, input.WorkOrderID, input.Duration)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_add_attachment
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_add_attachment",
			Description: "Add an attachment to a work order. Returns a signed upload URL — give the URL directly to the user so they can upload the file via HTTP PUT",
		},
		wrapTool(debug, "work_order_add_attachment", func(ctx context.Context, _ *mcp.CallToolRequest, input AddWorkOrderAttachmentInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.FileName == "" {
				return toolInputError("file_name is required"), nil, nil
			}
			if err := models.ValidateFileName(input.FileName); err != nil {
				return toolInputError(err.Error()), nil, nil
			}
			if input.MimeType == "" {
				return toolInputError("mime_type is required"), nil, nil
			}
			if err := models.ValidateMIMEType(input.MimeType); err != nil {
				return toolInputError(err.Error()), nil, nil
			}
			if err := models.ValidatePhotoSize(input.Size); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			apiInput := models.WorkOrderAddAttachmentInput{
				WorkOrderID: input.WorkOrderID,
				FileName:    input.FileName,
				MimeType:    input.MimeType,
				Size:        input.Size,
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			result, err := client.WorkOrderAddAttachment(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(result)
		}),
	)

	// work_order_remove_attachment (destructive)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_remove_attachment",
			Description: "Remove an attachment from a work order",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "work_order_remove_attachment", func(ctx context.Context, _ *mcp.CallToolRequest, input RemoveWorkOrderAttachmentInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("attachment_id", input.AttachmentID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderRemoveAttachment(ctx, input.WorkOrderID, input.AttachmentID)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_start_timer
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_start_timer",
			Description: "Start the timer for a work order",
		},
		wrapTool(debug, "work_order_start_timer", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderTimerInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Timestamp == "" {
				return toolInputError("timestamp is required (ISO 8601 format)"), nil, nil
			}
			if err := models.ValidateTimestamp("timestamp", input.Timestamp); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderStartTimer(ctx, input.WorkOrderID, input.Timestamp)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)

	// work_order_stop_timer
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "work_order_stop_timer",
			Description: "Stop the timer for a work order",
		},
		wrapTool(debug, "work_order_stop_timer", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderTimerInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Timestamp == "" {
				return toolInputError("timestamp is required (ISO 8601 format)"), nil, nil
			}
			if err := models.ValidateTimestamp("timestamp", input.Timestamp); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderStopTimer(ctx, input.WorkOrderID, input.Timestamp)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(wo)
		}),
	)
}

// --- Inspection MCP Tool Input Structs ---

type InspectionCreateMCPInput struct {
	LocationID   string `json:"location_id" jsonschema:"required,The property or unit ID for the inspection"`
	TemplateID   string `json:"template_id" jsonschema:"required,The inspection template ID"`
	ScheduledFor string `json:"scheduled_for" jsonschema:"required,Scheduled date in ISO 8601 format"`
	AssigneeID   string `json:"assignee_id,omitempty" jsonschema:"User ID to assign the inspection to"`
	DueBy        string `json:"due_by,omitempty" jsonschema:"Due date in ISO 8601 format"`
	Expires      *bool  `json:"expires,omitempty" jsonschema:"Whether the inspection expires at the due date"`
}

type InspectionIDOnlyInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
}

type InspectionSetAssigneeMCPInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	UserID       string `json:"user_id" jsonschema:"required,The user ID to assign"`
}

type InspectionSetDueByMCPInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	DueBy        string `json:"due_by" jsonschema:"required,Due date in ISO 8601 format"`
	// Expires is *bool (not bool) so we can detect omission. The default-false
	// of a plain bool would silently set expires=false; the CLI requires the
	// flag explicitly and the MCP must match.
	Expires *bool `json:"expires" jsonschema:"required,Whether the inspection expires at the due date (true or false)"`
}

type InspectionSetScheduledForMCPInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	ScheduledFor string `json:"scheduled_for" jsonschema:"required,Scheduled date in ISO 8601 format"`
}

type InspectionFieldInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	Label        string `json:"label" jsonschema:"required,The field label"`
	Value        string `json:"value,omitempty" jsonschema:"The field value (omit to clear)"`
}

type InspectionItemNotesInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	SectionName  string `json:"section_name" jsonschema:"required,The section name"`
	ItemName     string `json:"item_name" jsonschema:"required,The item name"`
	Notes        string `json:"notes,omitempty" jsonschema:"The notes text (omit to clear)"`
}

type InspectionRateItemMCPInput struct {
	InspectionID string   `json:"inspection_id" jsonschema:"required,The inspection ID"`
	SectionName  string   `json:"section_name" jsonschema:"required,The section name"`
	ItemName     string   `json:"item_name" jsonschema:"required,The item name"`
	RatingKey    string   `json:"rating_key" jsonschema:"required,The rating key from the rating group"`
	RatingScore  *float64 `json:"rating_score,omitempty" jsonschema:"The numeric rating score"`
	RatingValue  string   `json:"rating_value,omitempty" jsonschema:"The rating value string"`
}

type InspectionSectionInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	SectionName  string `json:"section_name" jsonschema:"required,The section name"`
}

type InspectionRenameSectionMCPInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	SectionName  string `json:"section_name" jsonschema:"required,The current section name"`
	NewName      string `json:"new_name" jsonschema:"required,The new section name"`
}

type InspectionAddItemMCPInput struct {
	InspectionID  string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	SectionName   string `json:"section_name" jsonschema:"required,The section name"`
	Name          string `json:"name" jsonschema:"required,The item name"`
	RatingGroupID string `json:"rating_group_id" jsonschema:"required,The rating group ID for the item"`
	Info          string `json:"info,omitempty" jsonschema:"Explanatory text about the item"`
}

type InspectionDeleteItemMCPInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	SectionName  string `json:"section_name" jsonschema:"required,The section name"`
	ItemName     string `json:"item_name" jsonschema:"required,The item name to delete"`
}

type InspectionAddItemPhotoMCPInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	SectionName  string `json:"section_name" jsonschema:"required,The section name"`
	ItemName     string `json:"item_name" jsonschema:"required,The item name"`
	MimeType     string `json:"mime_type" jsonschema:"required,The MIME type of the photo (e.g. image/jpeg)"`
	Size         *int   `json:"size,omitempty" jsonschema:"Photo size in bytes if known"`
}

type InspectionRemoveItemPhotoMCPInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	PhotoID      string `json:"photo_id" jsonschema:"required,The photo ID to remove"`
	SectionName  string `json:"section_name" jsonschema:"required,The section name"`
	ItemName     string `json:"item_name" jsonschema:"required,The item name"`
}

type InspectionMoveItemPhotoMCPInput struct {
	InspectionID    string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	PhotoID         string `json:"photo_id" jsonschema:"required,The photo ID to move"`
	FromSectionName string `json:"from_section_name" jsonschema:"required,Source section name"`
	FromItemName    string `json:"from_item_name" jsonschema:"required,Source item name"`
	ToSectionName   string `json:"to_section_name" jsonschema:"required,Destination section name"`
	ToItemName      string `json:"to_item_name" jsonschema:"required,Destination item name"`
}

type InspectionSendToGuestMCPInput struct {
	InspectionID string `json:"inspection_id" jsonschema:"required,The inspection ID"`
	Email        string `json:"email" jsonschema:"required,Guest email address"`
	Name         string `json:"name,omitempty" jsonschema:"Guest name"`
	Message      string `json:"message,omitempty" jsonschema:"Message to include in the email"`
	DueDate      string `json:"due_date,omitempty" jsonschema:"Due date in ISO 8601 format"`
	Expires      *bool  `json:"expires,omitempty" jsonschema:"Whether the guest link expires at the due date"`
}

// --- Project MCP Tool Input Structs ---

type ProjectCreateMCPInput struct {
	TemplateID           string `json:"template_id" jsonschema:"required,The project template ID"`
	LocationID           string `json:"location_id" jsonschema:"required,The property or unit ID for the project"`
	StartAt              string `json:"start_at" jsonschema:"required,Start date in ISO 8601 format"`
	AssigneeID           string `json:"assignee_id,omitempty" jsonschema:"User ID to assign to the project"`
	Priority             string `json:"priority,omitempty" jsonschema:"Priority: NORMAL or URGENT"`
	DueAt                string `json:"due_at,omitempty" jsonschema:"Due date in ISO 8601 format"`
	AvailabilityTargetAt string `json:"availability_target_at,omitempty" jsonschema:"Availability target date in ISO 8601 format"`
	Notes                string `json:"notes,omitempty" jsonschema:"Initial project notes"`
}

type ProjectSetAssigneeMCPInput struct {
	ProjectID  string  `json:"project_id" jsonschema:"required,The project ID"`
	AssigneeID *string `json:"assignee_id" jsonschema:"User ID to assign (null to unassign)"`
}

type SetProjectNotesMCPInput struct {
	ProjectID string `json:"project_id" jsonschema:"required,The project ID"`
	Notes     string `json:"notes" jsonschema:"required,The project notes text"`
}

type SetProjectDueAtMCPInput struct {
	ProjectID string `json:"project_id" jsonschema:"required,The project ID"`
	DueAt     string `json:"due_at" jsonschema:"required,Due date in ISO 8601 format"`
}

type SetProjectStartAtMCPInput struct {
	ProjectID string `json:"project_id" jsonschema:"required,The project ID"`
	StartAt   string `json:"start_at" jsonschema:"required,Start date in ISO 8601 format"`
}

type SetProjectPriorityMCPInput struct {
	ProjectID string `json:"project_id" jsonschema:"required,The project ID"`
	Priority  string `json:"priority" jsonschema:"required,Priority: NORMAL or URGENT"`
}

type SetProjectOnHoldMCPInput struct {
	ProjectID string `json:"project_id" jsonschema:"required,The project ID"`
	OnHold    *bool  `json:"on_hold" jsonschema:"required,Whether the project is on hold (true or false)"`
}

type ProjectSetAvailabilityTargetAtMCPInput struct {
	ProjectID            string  `json:"project_id" jsonschema:"required,The project ID"`
	AvailabilityTargetAt *string `json:"availability_target_at" jsonschema:"Availability target date in ISO 8601 format (null to clear)"`
}

// --- Account MCP Tool Input Structs ---

type UserCreateMCPInput struct {
	AccountID string `json:"account_id" jsonschema:"required,The account ID to create the user in"`
	Email     string `json:"email" jsonschema:"required,Email address for the new user"`
	Name      string `json:"name" jsonschema:"required,Full name of the user"`
	RoleID    string `json:"role_id,omitempty" jsonschema:"Comma-separated role IDs (defaults to account default role)"`
	ShortName string `json:"short_name,omitempty" jsonschema:"Informal/given name (derived from name if omitted)"`
	Phone     string `json:"phone,omitempty" jsonschema:"Phone number"`
	Message   string `json:"message,omitempty" jsonschema:"Personalised greeting in the invitation email"`
}

type UserIDEmailInput struct {
	UserID string `json:"user_id" jsonschema:"required,The user ID"`
	Email  string `json:"email" jsonschema:"required,New email address"`
}

type UserIDNameInput struct {
	UserID string `json:"user_id" jsonschema:"required,The user ID"`
	Name   string `json:"name" jsonschema:"required,New full name"`
}

type SetUserShortNameMCPInput struct {
	UserID    string  `json:"user_id" jsonschema:"required,The user ID"`
	ShortName *string `json:"short_name" jsonschema:"Informal/given name (null to derive from full name)"`
}

type SetUserPhoneMCPInput struct {
	UserID string  `json:"user_id" jsonschema:"required,The user ID"`
	Phone  *string `json:"phone" jsonschema:"Phone number (null to remove)"`
}

type MembershipCreateMCPInput struct {
	AccountID string `json:"account_id" jsonschema:"required,The account ID"`
	UserID    string `json:"user_id" jsonschema:"required,The user ID"`
	RoleID    string `json:"role_id,omitempty" jsonschema:"Comma-separated role IDs (defaults to account default roles)"`
}

type MembershipAccountUserInput struct {
	AccountID string `json:"account_id" jsonschema:"required,The account ID"`
	UserID    string `json:"user_id" jsonschema:"required,The user ID"`
}

type MembershipSetRolesMCPInput struct {
	AccountID string `json:"account_id" jsonschema:"required,The account ID"`
	UserID    string `json:"user_id" jsonschema:"required,The user ID"`
	RoleID    string `json:"role_id,omitempty" jsonschema:"Comma-separated role IDs (defaults to account default roles)"`
}

type PropertyUserAccessMCPInput struct {
	PropertyID string `json:"property_id" jsonschema:"required,The property ID"`
	UserID     string `json:"user_id" jsonschema:"required,Comma-separated user IDs"`
}

type PropertySetAccountWideAccessMCPInput struct {
	PropertyID        string `json:"property_id" jsonschema:"required,The property ID"`
	AccountWideAccess *bool  `json:"account_wide_access" jsonschema:"required,Whether all users in the account can access this property (true or false)"`
}

type UserPropertyAccessMCPInput struct {
	UserID     string `json:"user_id" jsonschema:"required,The user ID"`
	PropertyID string `json:"property_id" jsonschema:"required,Comma-separated property IDs"`
}

// --- Webhook MCP Tool Input Structs ---

type WebhookCreateMCPInput struct {
	SubscriberID   string `json:"subscriber_id" jsonschema:"required,The subscriber ID (account or plugin ID)"`
	SubscriberType string `json:"subscriber_type" jsonschema:"required,Subscriber type: ACCOUNT or PLUGIN"`
	URL            string `json:"url" jsonschema:"required,Webhook endpoint URL (must be HTTPS)"`
	Subjects       string `json:"subjects,omitempty" jsonschema:"Comma-separated subjects: INSPECTIONS, WORK_ORDERS, VENDORS, PLUGIN_SUBSCRIPTIONS"`
	Status         string `json:"status,omitempty" jsonschema:"Initial status: ENABLED or DISABLED (default DISABLED)"`
}

type WebhookUpdateMCPInput struct {
	ID       string `json:"id" jsonschema:"required,The webhook ID to update"`
	URL      string `json:"url,omitempty" jsonschema:"New webhook endpoint URL (must be HTTPS)"`
	Status   string `json:"status,omitempty" jsonschema:"New status: ENABLED or DISABLED"`
	Subjects string `json:"subjects,omitempty" jsonschema:"Comma-separated subjects: INSPECTIONS, WORK_ORDERS, VENDORS, PLUGIN_SUBSCRIPTIONS"`
}

// --- Role MCP Tool Input Structs ---

type RoleCreateMCPInput struct {
	AccountID         string `json:"account_id" jsonschema:"required,The account ID that will own the role"`
	Name              string `json:"name" jsonschema:"required,Display name for the role (must be unique within account)"`
	Description       string `json:"description,omitempty" jsonschema:"Description of the role"`
	PermissionsGrant  string `json:"permissions_grant" jsonschema:"required,Comma-separated permission actions to grant (e.g. task:task.create)"`
	PermissionsRevoke string `json:"permissions_revoke,omitempty" jsonschema:"Comma-separated permission actions to revoke"`
}

type RoleSetNameMCPInput struct {
	AccountID string `json:"account_id" jsonschema:"required,The account ID that owns the role"`
	RoleID    string `json:"role_id" jsonschema:"required,The role ID to update"`
	Name      string `json:"name" jsonschema:"required,New name for the role (must be unique within account)"`
}

type RoleSetDescriptionMCPInput struct {
	AccountID   string  `json:"account_id" jsonschema:"required,The account ID that owns the role"`
	RoleID      string  `json:"role_id" jsonschema:"required,The role ID to update"`
	Description *string `json:"description" jsonschema:"New description (null to remove)"`
}

type RoleSetPermissionsMCPInput struct {
	AccountID         string `json:"account_id" jsonschema:"required,The account ID that owns the role"`
	RoleID            string `json:"role_id" jsonschema:"required,The role ID to update"`
	PermissionsGrant  string `json:"permissions_grant,omitempty" jsonschema:"Comma-separated permission actions to grant"`
	PermissionsRevoke string `json:"permissions_revoke,omitempty" jsonschema:"Comma-separated permission actions to revoke"`
}

// registerInspectionMutationTools registers all 24 inspection mutation tools.
func registerInspectionMutationTools(server *mcp.Server, client apiClient, debug bool) {
	destructive := true

	// inspection_create
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_create",
			Description: "Create a new inspection for a property or unit",
		},
		wrapTool(debug, "inspection_create", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionCreateMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("location_id", input.LocationID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("template_id", input.TemplateID); errResult != nil {
				return errResult, nil, nil
			}
			if input.ScheduledFor == "" {
				return toolInputError("scheduled_for is required"), nil, nil
			}
			if err := models.ValidateTimestamp("scheduled_for", input.ScheduledFor); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			apiInput := models.InspectionCreateInput{
				LocationID:   input.LocationID,
				TemplateID:   input.TemplateID,
				ScheduledFor: input.ScheduledFor,
			}
			if input.AssigneeID != "" {
				if err := models.ValidateID("assignee_id", input.AssigneeID); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				apiInput.AssignedToID = input.AssigneeID
			}
			if input.DueBy != "" {
				if err := models.ValidateTimestamp("due_by", input.DueBy); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				apiInput.DueBy = input.DueBy
			}
			apiInput.Expires = input.Expires

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionCreate(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_start
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_start",
			Description: "Start an inspection (imprints template structure and changes status to INCOMPLETE)",
		},
		wrapTool(debug, "inspection_start", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionIDOnlyInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionStart(ctx, input.InspectionID)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_complete
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_complete",
			Description: "Mark an inspection as complete",
		},
		wrapTool(debug, "inspection_complete", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionIDOnlyInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionComplete(ctx, input.InspectionID)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_reopen
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_reopen",
			Description: "Reopen a completed inspection (changes status to INCOMPLETE)",
		},
		wrapTool(debug, "inspection_reopen", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionIDOnlyInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionReopen(ctx, input.InspectionID)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_archive (destructive)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_archive",
			Description: "Archive an inspection (removes from active list)",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "inspection_archive", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionIDOnlyInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionArchive(ctx, input.InspectionID)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_expire (destructive)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_expire",
			Description: "Expire an inspection (changes status to EXPIRED)",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "inspection_expire", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionIDOnlyInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionExpire(ctx, input.InspectionID)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_unexpire
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_unexpire",
			Description: "Unexpire an inspection (removes due date and expires flag)",
		},
		wrapTool(debug, "inspection_unexpire", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionIDOnlyInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionUnexpire(ctx, input.InspectionID)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_set_assignee
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_set_assignee",
			Description: "Assign a user to an inspection",
		},
		wrapTool(debug, "inspection_set_assignee", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionSetAssigneeMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("user_id", input.UserID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionSetAssignee(ctx, models.InspectionSetAssigneeInput{
				InspectionID: input.InspectionID,
				UserID:       input.UserID,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_set_due_by
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_set_due_by",
			Description: "Set the due date for an inspection",
		},
		wrapTool(debug, "inspection_set_due_by", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionSetDueByMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.DueBy == "" {
				return toolInputError("due_by is required"), nil, nil
			}
			if err := models.ValidateTimestamp("due_by", input.DueBy); err != nil {
				return toolInputError(err.Error()), nil, nil
			}
			if input.Expires == nil {
				return toolInputError("expires is required (true or false)"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionSetDueBy(ctx, models.InspectionSetDueByInput{
				InspectionID: input.InspectionID,
				DueBy:        input.DueBy,
				Expires:      *input.Expires,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_set_scheduled_for
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_set_scheduled_for",
			Description: "Set the scheduled date for an inspection",
		},
		wrapTool(debug, "inspection_set_scheduled_for", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionSetScheduledForMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.ScheduledFor == "" {
				return toolInputError("scheduled_for is required"), nil, nil
			}
			if err := models.ValidateTimestamp("scheduled_for", input.ScheduledFor); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionSetScheduledFor(ctx, input.InspectionID, input.ScheduledFor)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_set_header_field
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_set_header_field",
			Description: "Update a header field on an inspection",
		},
		wrapTool(debug, "inspection_set_header_field", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionFieldInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Label == "" {
				return toolInputError("label is required"), nil, nil
			}
			if err := models.ValidateFreeText("value", input.Value); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionSetHeaderField(ctx, models.InspectionSetHeaderFieldInput{
				InspectionID: input.InspectionID,
				Label:        input.Label,
				Value:        input.Value,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_set_footer_field
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_set_footer_field",
			Description: "Update a footer field on an inspection",
		},
		wrapTool(debug, "inspection_set_footer_field", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionFieldInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Label == "" {
				return toolInputError("label is required"), nil, nil
			}
			if err := models.ValidateFreeText("value", input.Value); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionSetFooterField(ctx, models.InspectionSetFooterFieldInput{
				InspectionID: input.InspectionID,
				Label:        input.Label,
				Value:        input.Value,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_set_item_notes
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_set_item_notes",
			Description: "Set notes on an inspection item. Use list_inspections to find section and item names first",
		},
		wrapTool(debug, "inspection_set_item_notes", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionItemNotesInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.SectionName == "" {
				return toolInputError("section_name is required"), nil, nil
			}
			if input.ItemName == "" {
				return toolInputError("item_name is required"), nil, nil
			}
			if err := models.ValidateFreeText("notes", input.Notes); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionSetItemNotes(ctx, models.InspectionSetItemNotesInput{
				InspectionID: input.InspectionID,
				SectionName:  input.SectionName,
				ItemName:     input.ItemName,
				Notes:        input.Notes,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_rate_item
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_rate_item",
			Description: "Rate an item in an inspection. Use list_inspections to find section and item names first",
		},
		wrapTool(debug, "inspection_rate_item", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionRateItemMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.SectionName == "" {
				return toolInputError("section_name is required"), nil, nil
			}
			if input.ItemName == "" {
				return toolInputError("item_name is required"), nil, nil
			}
			if input.RatingKey == "" {
				return toolInputError("rating_key is required"), nil, nil
			}
			if err := models.ValidateRatingScore(input.RatingScore); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionRateItem(ctx, models.InspectionRateItemInput{
				InspectionID: input.InspectionID,
				SectionName:  input.SectionName,
				ItemName:     input.ItemName,
				Rating: models.InspectionRatingInput{
					Key:   input.RatingKey,
					Score: input.RatingScore,
					Value: input.RatingValue,
				},
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_add_section
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_add_section",
			Description: "Add a new section to an inspection",
		},
		wrapTool(debug, "inspection_add_section", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionSectionInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.SectionName == "" {
				return toolInputError("section_name is required"), nil, nil
			}
			if err := models.ValidateFreeText("section_name", input.SectionName); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionAddSection(ctx, models.InspectionAddSectionInput{
				InspectionID: input.InspectionID,
				Name:         input.SectionName,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_delete_section (destructive)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_delete_section",
			Description: "Delete a section and all its items from an inspection",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "inspection_delete_section", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionSectionInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.SectionName == "" {
				return toolInputError("section_name is required"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionDeleteSection(ctx, models.InspectionDeleteSectionInput{
				InspectionID: input.InspectionID,
				SectionName:  input.SectionName,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_duplicate_section
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_duplicate_section",
			Description: "Duplicate a section in an inspection (creates a clean copy with same items)",
		},
		wrapTool(debug, "inspection_duplicate_section", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionSectionInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.SectionName == "" {
				return toolInputError("section_name is required"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionDuplicateSection(ctx, models.InspectionDuplicateSectionInput{
				InspectionID: input.InspectionID,
				SectionName:  input.SectionName,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_rename_section
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_rename_section",
			Description: "Rename a section in an inspection",
		},
		wrapTool(debug, "inspection_rename_section", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionRenameSectionMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.SectionName == "" {
				return toolInputError("section_name is required"), nil, nil
			}
			if input.NewName == "" {
				return toolInputError("new_name is required"), nil, nil
			}
			if err := models.ValidateFreeText("new_name", input.NewName); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionRenameSection(ctx, models.InspectionRenameSectionInput{
				InspectionID:   input.InspectionID,
				SectionName:    input.SectionName,
				NewSectionName: input.NewName,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_add_item
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_add_item",
			Description: "Add an item to a section in an inspection",
		},
		wrapTool(debug, "inspection_add_item", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionAddItemMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.SectionName == "" {
				return toolInputError("section_name is required"), nil, nil
			}
			if input.Name == "" {
				return toolInputError("name is required"), nil, nil
			}
			if errResult := requireID("rating_group_id", input.RatingGroupID); errResult != nil {
				return errResult, nil, nil
			}
			if err := models.ValidateFreeText("name", input.Name); err != nil {
				return toolInputError(err.Error()), nil, nil
			}
			if err := models.ValidateFreeText("info", input.Info); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionAddItem(ctx, models.InspectionAddItemInput{
				InspectionID:  input.InspectionID,
				SectionName:   input.SectionName,
				Name:          input.Name,
				RatingGroupID: input.RatingGroupID,
				Info:          input.Info,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_delete_item (destructive)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_delete_item",
			Description: "Delete an item from a section in an inspection",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "inspection_delete_item", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionDeleteItemMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.SectionName == "" {
				return toolInputError("section_name is required"), nil, nil
			}
			if input.ItemName == "" {
				return toolInputError("item_name is required"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionDeleteItem(ctx, models.InspectionDeleteItemInput{
				InspectionID: input.InspectionID,
				SectionName:  input.SectionName,
				ItemName:     input.ItemName,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_add_item_photo
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_add_item_photo",
			Description: "Add a photo to an inspection item. Returns a signed upload URL — give the URL directly to the user so they can upload the file via HTTP PUT",
		},
		wrapTool(debug, "inspection_add_item_photo", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionAddItemPhotoMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.SectionName == "" {
				return toolInputError("section_name is required"), nil, nil
			}
			if input.ItemName == "" {
				return toolInputError("item_name is required"), nil, nil
			}
			if input.MimeType == "" {
				return toolInputError("mime_type is required"), nil, nil
			}
			if err := models.ValidateMIMEType(input.MimeType); err != nil {
				return toolInputError(err.Error()), nil, nil
			}
			if err := models.ValidatePhotoSize(input.Size); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			result, err := client.InspectionAddItemPhoto(ctx, models.InspectionAddItemPhotoInput{
				InspectionID: input.InspectionID,
				SectionName:  input.SectionName,
				ItemName:     input.ItemName,
				MimeType:     input.MimeType,
				Size:         input.Size,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(result)
		}),
	)

	// inspection_remove_item_photo (destructive)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_remove_item_photo",
			Description: "Remove a photo from an inspection item",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "inspection_remove_item_photo", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionRemoveItemPhotoMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("photo_id", input.PhotoID); errResult != nil {
				return errResult, nil, nil
			}
			if input.SectionName == "" {
				return toolInputError("section_name is required"), nil, nil
			}
			if input.ItemName == "" {
				return toolInputError("item_name is required"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionRemoveItemPhoto(ctx, models.InspectionRemoveItemPhotoInput{
				InspectionID: input.InspectionID,
				PhotoID:      input.PhotoID,
				SectionName:  input.SectionName,
				ItemName:     input.ItemName,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_move_item_photo
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_move_item_photo",
			Description: "Move a photo from one item to another in an inspection",
		},
		wrapTool(debug, "inspection_move_item_photo", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionMoveItemPhotoMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("photo_id", input.PhotoID); errResult != nil {
				return errResult, nil, nil
			}
			if input.FromSectionName == "" {
				return toolInputError("from_section_name is required"), nil, nil
			}
			if input.FromItemName == "" {
				return toolInputError("from_item_name is required"), nil, nil
			}
			if input.ToSectionName == "" {
				return toolInputError("to_section_name is required"), nil, nil
			}
			if input.ToItemName == "" {
				return toolInputError("to_item_name is required"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionMoveItemPhoto(ctx, models.InspectionMoveItemPhotoInput{
				InspectionID:    input.InspectionID,
				PhotoID:         input.PhotoID,
				FromSectionName: input.FromSectionName,
				FromItemName:    input.FromItemName,
				ToSectionName:   input.ToSectionName,
				ToItemName:      input.ToItemName,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(insp)
		}),
	)

	// inspection_send_to_guest
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "inspection_send_to_guest",
			Description: "Send an inspection to a guest via email. Returns a guest link",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "inspection_send_to_guest", func(ctx context.Context, _ *mcp.CallToolRequest, input InspectionSendToGuestMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("inspection_id", input.InspectionID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Email == "" {
				return toolInputError("email is required"), nil, nil
			}
			if err := models.ValidateEmail(input.Email); err != nil {
				return toolInputError(err.Error()), nil, nil
			}
			if err := models.ValidateFreeText("message", input.Message); err != nil {
				return toolInputError(err.Error()), nil, nil
			}
			if err := models.ValidateFreeText("name", input.Name); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			apiInput := models.InspectionSendToGuestInput{
				InspectionID: input.InspectionID,
				Email:        input.Email,
				Name:         input.Name,
				Message:      input.Message,
				Expires:      input.Expires,
			}
			if input.DueDate != "" {
				if err := models.ValidateTimestamp("due_date", input.DueDate); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				apiInput.DueDate = input.DueDate
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			// Gate concurrent email-sending mutations to prevent fan-out
			// duplicate emails from a misbehaving LLM batch.
			if err := acquireSem(ctx, emailSem); err != nil {
				return toolError(err), nil, nil
			}
			defer releaseSem(emailSem)

			result, err := client.InspectionSendToGuest(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			redactGuestLink(result)
			return toolJSON(result)
		}),
	)
}

// registerProjectMutationTools registers all 8 project mutation tools.
func registerProjectMutationTools(server *mcp.Server, client apiClient, debug bool) {
	// project_create
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "project_create",
			Description: "Create a new project from a template for a property or unit",
		},
		wrapTool(debug, "project_create", func(ctx context.Context, _ *mcp.CallToolRequest, input ProjectCreateMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("template_id", input.TemplateID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("location_id", input.LocationID); errResult != nil {
				return errResult, nil, nil
			}
			if input.StartAt == "" {
				return toolInputError("start_at is required"), nil, nil
			}
			if err := models.ValidateTimestamp("start_at", input.StartAt); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			apiInput := models.ProjectCreateInput{
				ProjectTemplateID: input.TemplateID,
				LocationID:        input.LocationID,
				StartAt:           input.StartAt,
				Notes:             input.Notes,
			}
			if input.AssigneeID != "" {
				if err := models.ValidateID("assignee_id", input.AssigneeID); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				apiInput.AssigneeID = input.AssigneeID
			}
			if input.Priority != "" {
				upper := strings.ToUpper(input.Priority)
				if !models.ValidProjectPriorities[upper] {
					return toolInputError("priority must be NORMAL or URGENT"), nil, nil
				}
				apiInput.Priority = upper
			}
			if input.DueAt != "" {
				if err := models.ValidateTimestamp("due_at", input.DueAt); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				apiInput.DueAt = input.DueAt
			}
			if input.AvailabilityTargetAt != "" {
				if err := models.ValidateTimestamp("availability_target_at", input.AvailabilityTargetAt); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				apiInput.AvailabilityTargetAt = input.AvailabilityTargetAt
			}
			if err := models.ValidateFreeText("notes", input.Notes); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			proj, err := client.ProjectCreate(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(proj)
		}),
	)

	// project_set_assignee
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "project_set_assignee",
			Description: "Set or clear the user assigned to a project (pass null assignee_id to unassign)",
		},
		wrapTool(debug, "project_set_assignee", func(ctx context.Context, _ *mcp.CallToolRequest, input ProjectSetAssigneeMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("project_id", input.ProjectID); errResult != nil {
				return errResult, nil, nil
			}
			// Coerce non-nil empty string to nil — semantically equivalent to
			// "unassign", and avoids sending an unvalidated empty ID through
			// to the API. Without this, an LLM passing assignee_id="" would
			// bypass ValidateID entirely.
			if input.AssigneeID != nil && *input.AssigneeID == "" {
				input.AssigneeID = nil
			}
			if input.AssigneeID != nil {
				if err := models.ValidateID("assignee_id", *input.AssigneeID); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
			}

			apiInput := models.ProjectSetAssigneeInput{
				ProjectID:  input.ProjectID,
				AssigneeID: input.AssigneeID,
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			proj, err := client.ProjectSetAssignee(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(proj)
		}),
	)

	// project_set_notes
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "project_set_notes",
			Description: "Set the notes on a project",
		},
		wrapTool(debug, "project_set_notes", func(ctx context.Context, _ *mcp.CallToolRequest, input SetProjectNotesMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("project_id", input.ProjectID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Notes == "" {
				return toolInputError("notes is required"), nil, nil
			}
			if err := models.ValidateFreeText("notes", input.Notes); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			proj, err := client.ProjectSetNotes(ctx, input.ProjectID, input.Notes)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(proj)
		}),
	)

	// project_set_due_at
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "project_set_due_at",
			Description: "Set the due date for a project (ISO 8601 format)",
		},
		wrapTool(debug, "project_set_due_at", func(ctx context.Context, _ *mcp.CallToolRequest, input SetProjectDueAtMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("project_id", input.ProjectID); errResult != nil {
				return errResult, nil, nil
			}
			if input.DueAt == "" {
				return toolInputError("due_at is required"), nil, nil
			}
			if err := models.ValidateTimestamp("due_at", input.DueAt); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			proj, err := client.ProjectSetDueAt(ctx, input.ProjectID, input.DueAt)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(proj)
		}),
	)

	// project_set_start_at
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "project_set_start_at",
			Description: "Set the start date for a project (ISO 8601 format)",
		},
		wrapTool(debug, "project_set_start_at", func(ctx context.Context, _ *mcp.CallToolRequest, input SetProjectStartAtMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("project_id", input.ProjectID); errResult != nil {
				return errResult, nil, nil
			}
			if input.StartAt == "" {
				return toolInputError("start_at is required"), nil, nil
			}
			if err := models.ValidateTimestamp("start_at", input.StartAt); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			proj, err := client.ProjectSetStartAt(ctx, input.ProjectID, input.StartAt)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(proj)
		}),
	)

	// project_set_priority
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "project_set_priority",
			Description: "Set the priority of a project (NORMAL or URGENT)",
		},
		wrapTool(debug, "project_set_priority", func(ctx context.Context, _ *mcp.CallToolRequest, input SetProjectPriorityMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("project_id", input.ProjectID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Priority == "" {
				return toolInputError("priority is required"), nil, nil
			}
			upper := strings.ToUpper(input.Priority)
			if !models.ValidProjectPriorities[upper] {
				return toolInputError("priority must be NORMAL or URGENT"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			proj, err := client.ProjectSetPriority(ctx, input.ProjectID, upper)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(proj)
		}),
	)

	// project_set_on_hold
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "project_set_on_hold",
			Description: "Set or clear the on-hold status of a project",
		},
		wrapTool(debug, "project_set_on_hold", func(ctx context.Context, _ *mcp.CallToolRequest, input SetProjectOnHoldMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("project_id", input.ProjectID); errResult != nil {
				return errResult, nil, nil
			}
			if input.OnHold == nil {
				return toolInputError("on_hold is required (true or false)"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			proj, err := client.ProjectSetOnHold(ctx, input.ProjectID, *input.OnHold)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(proj)
		}),
	)

	// project_set_availability_target_at
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "project_set_availability_target_at",
			Description: "Set or clear the availability target date for a project (ISO 8601 format, null to clear)",
		},
		wrapTool(debug, "project_set_availability_target_at", func(ctx context.Context, _ *mcp.CallToolRequest, input ProjectSetAvailabilityTargetAtMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("project_id", input.ProjectID); errResult != nil {
				return errResult, nil, nil
			}
			var datePtr *string
			if input.AvailabilityTargetAt != nil && *input.AvailabilityTargetAt != "" {
				if err := models.ValidateTimestamp("availability_target_at", *input.AvailabilityTargetAt); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				datePtr = input.AvailabilityTargetAt
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			proj, err := client.ProjectSetAvailabilityTargetAt(ctx, input.ProjectID, datePtr)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(proj)
		}),
	)
}

// parseIDList wraps models.ParseIDList, converting the error into the MCP
// tool-result shape.
func parseIDList(fieldName, csv string) ([]string, *mcp.CallToolResult) {
	ids, err := models.ParseIDList(fieldName, csv)
	if err != nil {
		return nil, toolInputError(err.Error())
	}
	return ids, nil
}

// redactWebhookSecret overwrites the SigningSecret field on a webhook before it
// is returned to the MCP client. Tool responses persist in the LLM context window
// and conversation logs, so secrets must never appear there. All MCP paths that
// return *models.Webhook must call this; the CLI path is unaffected.
func redactWebhookSecret(w *models.Webhook) {
	if w == nil {
		return
	}
	w.SigningSecret = "[redacted — use CLI for signing secret]"
}

// redactGuestLink overwrites the bearer-style guest URL on an inspection guest
// link before it is returned to the MCP client. The URL grants unauthenticated
// access to the inspection and would otherwise persist in LLM conversation logs.
// The guest already received the link via the email the API sent — the LLM
// does not need it for any downstream tool call.
func redactGuestLink(l *models.InspectionGuestLink) {
	if l == nil {
		return
	}
	l.Link = "[redacted — guest link sent via email; use CLI to retrieve]"
}

// registerAccountMutationTools registers all 14 account mutation tools (users, memberships, access).
func registerAccountMutationTools(server *mcp.Server, client apiClient, debug bool) {
	destructive := true

	// --- User Tools (5) ---

	// user_create (destructive — sends invitation email)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "user_create",
			Description: "Create a new user account and send an invitation email. Use this for people who don't have a HappyCo account yet. For existing users, use membership_create instead",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "user_create", func(ctx context.Context, _ *mcp.CallToolRequest, input UserCreateMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("account_id", input.AccountID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Email == "" {
				return toolInputError("email is required"), nil, nil
			}
			if err := models.ValidateEmail(input.Email); err != nil {
				return toolInputError(err.Error()), nil, nil
			}
			if input.Name == "" {
				return toolInputError("name is required"), nil, nil
			}
			if err := models.ValidateFreeText("name", input.Name); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			apiInput := models.UserCreateInput{
				AccountID: input.AccountID,
				Email:     input.Email,
				Name:      input.Name,
				ShortName: input.ShortName,
				Phone:     input.Phone,
				Message:   input.Message,
			}
			if input.RoleID != "" {
				roleIDs, errResult := parseIDList("role_id", input.RoleID)
				if errResult != nil {
					return errResult, nil, nil
				}
				apiInput.RoleID = roleIDs
			}
			if input.ShortName != "" {
				if err := models.ValidateFreeText("short_name", input.ShortName); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
			}
			if input.Message != "" {
				if err := models.ValidateFreeText("message", input.Message); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
			}
			if input.Phone != "" {
				if err := models.ValidatePhone(input.Phone); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			// Gate concurrent email-sending mutations (user_create sends
			// invitation emails) to prevent fan-out from a misbehaving LLM.
			if err := acquireSem(ctx, emailSem); err != nil {
				return toolError(err), nil, nil
			}
			defer releaseSem(emailSem)

			user, err := client.UserCreate(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(user)
		}),
	)

	// user_set_email
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "user_set_email",
			Description: "Update a user's email address",
		},
		wrapTool(debug, "user_set_email", func(ctx context.Context, _ *mcp.CallToolRequest, input UserIDEmailInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("user_id", input.UserID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Email == "" {
				return toolInputError("email is required"), nil, nil
			}
			if err := models.ValidateEmail(input.Email); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			user, err := client.UserSetEmail(ctx, input.UserID, input.Email)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(user)
		}),
	)

	// user_set_name
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "user_set_name",
			Description: "Update a user's full name",
		},
		wrapTool(debug, "user_set_name", func(ctx context.Context, _ *mcp.CallToolRequest, input UserIDNameInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("user_id", input.UserID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Name == "" {
				return toolInputError("name is required"), nil, nil
			}
			if err := models.ValidateFreeText("name", input.Name); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			user, err := client.UserSetName(ctx, input.UserID, input.Name)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(user)
		}),
	)

	// user_set_short_name
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "user_set_short_name",
			Description: "Set or clear a user's short name (null to derive from full name)",
		},
		wrapTool(debug, "user_set_short_name", func(ctx context.Context, _ *mcp.CallToolRequest, input SetUserShortNameMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("user_id", input.UserID); errResult != nil {
				return errResult, nil, nil
			}
			if input.ShortName != nil && *input.ShortName != "" {
				if err := models.ValidateFreeText("short_name", *input.ShortName); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			user, err := client.UserSetShortName(ctx, input.UserID, input.ShortName)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(user)
		}),
	)

	// user_set_phone
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "user_set_phone",
			Description: "Set or clear a user's phone number (null to remove)",
		},
		wrapTool(debug, "user_set_phone", func(ctx context.Context, _ *mcp.CallToolRequest, input SetUserPhoneMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("user_id", input.UserID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Phone != nil {
				if err := models.ValidatePhone(*input.Phone); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			user, err := client.UserSetPhone(ctx, input.UserID, input.Phone)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(user)
		}),
	)

	// --- Membership Tools (4) ---

	// membership_create
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "membership_create",
			Description: "Admin: Grant an existing user access to an account. For new users who don't have a HappyCo account, use user_create instead",
		},
		wrapTool(debug, "membership_create", func(ctx context.Context, _ *mcp.CallToolRequest, input MembershipCreateMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("account_id", input.AccountID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("user_id", input.UserID); errResult != nil {
				return errResult, nil, nil
			}

			apiInput := models.AccountMembershipCreateInput{
				AccountID: input.AccountID,
				UserID:    input.UserID,
			}
			if input.RoleID != "" {
				roleIDs, errResult := parseIDList("role_id", input.RoleID)
				if errResult != nil {
					return errResult, nil, nil
				}
				apiInput.RoleID = roleIDs
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			membership, err := client.AccountMembershipCreate(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(membership)
		}),
	)

	// membership_activate
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "membership_activate",
			Description: "Admin: Activate a user's membership in an account",
		},
		wrapTool(debug, "membership_activate", func(ctx context.Context, _ *mcp.CallToolRequest, input MembershipAccountUserInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("account_id", input.AccountID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("user_id", input.UserID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			membership, err := client.AccountMembershipActivate(ctx, models.AccountMembershipActivateInput{
				AccountID: input.AccountID,
				UserID:    input.UserID,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(membership)
		}),
	)

	// membership_deactivate (destructive)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "membership_deactivate",
			Description: "Admin: Deactivate a user's membership in an account",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "membership_deactivate", func(ctx context.Context, _ *mcp.CallToolRequest, input MembershipAccountUserInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("account_id", input.AccountID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("user_id", input.UserID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			membership, err := client.AccountMembershipDeactivate(ctx, models.AccountMembershipDeactivateInput{
				AccountID: input.AccountID,
				UserID:    input.UserID,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(membership)
		}),
	)

	// membership_set_roles (destructive — privilege-changing)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "membership_set_roles",
			Description: "Admin: Set the roles for a user's membership in an account",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "membership_set_roles", func(ctx context.Context, _ *mcp.CallToolRequest, input MembershipSetRolesMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("account_id", input.AccountID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("user_id", input.UserID); errResult != nil {
				return errResult, nil, nil
			}

			apiInput := models.AccountMembershipSetRolesInput{
				AccountID: input.AccountID,
				UserID:    input.UserID,
			}
			if input.RoleID != "" {
				roleIDs, errResult := parseIDList("role_id", input.RoleID)
				if errResult != nil {
					return errResult, nil, nil
				}
				apiInput.RoleID = roleIDs
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			membership, err := client.AccountMembershipSetRoles(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(membership)
		}),
	)

	// --- Property Access Tools (3) ---

	// property_grant_access
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "property_grant_access",
			Description: "Admin: Grant one or more users access to a property",
		},
		wrapTool(debug, "property_grant_access", func(ctx context.Context, _ *mcp.CallToolRequest, input PropertyUserAccessMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("property_id", input.PropertyID); errResult != nil {
				return errResult, nil, nil
			}
			userIDs, errResult := parseIDList("user_id", input.UserID)
			if errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			result, err := client.PropertyGrantUserAccess(ctx, models.PropertyGrantUserAccessInput{
				PropertyID: input.PropertyID,
				UserID:     userIDs,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(result)
		}),
	)

	// property_revoke_access (destructive)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "property_revoke_access",
			Description: "Admin: Revoke one or more users' access to a property",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "property_revoke_access", func(ctx context.Context, _ *mcp.CallToolRequest, input PropertyUserAccessMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("property_id", input.PropertyID); errResult != nil {
				return errResult, nil, nil
			}
			userIDs, errResult := parseIDList("user_id", input.UserID)
			if errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			result, err := client.PropertyRevokeUserAccess(ctx, models.PropertyRevokeUserAccessInput{
				PropertyID: input.PropertyID,
				UserID:     userIDs,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(result)
		}),
	)

	// property_set_account_wide_access (destructive — privilege-widening)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "property_set_account_wide_access",
			Description: "Admin: Set whether a property is accessible to all users in the account",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "property_set_account_wide_access", func(ctx context.Context, _ *mcp.CallToolRequest, input PropertySetAccountWideAccessMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("property_id", input.PropertyID); errResult != nil {
				return errResult, nil, nil
			}
			if input.AccountWideAccess == nil {
				return toolInputError("account_wide_access is required (true or false)"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			result, err := client.PropertySetAccountWideAccess(ctx, models.PropertySetAccountWideAccessInput{
				PropertyID:        input.PropertyID,
				AccountWideAccess: *input.AccountWideAccess,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(result)
		}),
	)

	// --- User Property Access Tools (2) ---

	// user_grant_property_access
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "user_grant_property_access",
			Description: "Admin: Grant a user access to one or more properties",
		},
		wrapTool(debug, "user_grant_property_access", func(ctx context.Context, _ *mcp.CallToolRequest, input UserPropertyAccessMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("user_id", input.UserID); errResult != nil {
				return errResult, nil, nil
			}
			propertyIDs, errResult := parseIDList("property_id", input.PropertyID)
			if errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			user, err := client.UserGrantPropertyAccess(ctx, models.UserGrantPropertyAccessInput{
				UserID:     input.UserID,
				PropertyID: propertyIDs,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(user)
		}),
	)

	// user_revoke_property_access (destructive)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "user_revoke_property_access",
			Description: "Admin: Revoke a user's access to one or more properties",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "user_revoke_property_access", func(ctx context.Context, _ *mcp.CallToolRequest, input UserPropertyAccessMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("user_id", input.UserID); errResult != nil {
				return errResult, nil, nil
			}
			propertyIDs, errResult := parseIDList("property_id", input.PropertyID)
			if errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			user, err := client.UserRevokePropertyAccess(ctx, models.UserRevokePropertyAccessInput{
				UserID:     input.UserID,
				PropertyID: propertyIDs,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(user)
		}),
	)
}

// registerRoleMutationTools registers all 4 role mutation tools.
func registerRoleMutationTools(server *mcp.Server, client apiClient, debug bool) {
	destructive := true

	// role_create
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "role_create",
			Description: "Admin: Create a new permission role in an account. Permission actions use format domain:action (e.g. inspection:inspection.create, task:task.delete)",
		},
		wrapTool(debug, "role_create", func(ctx context.Context, _ *mcp.CallToolRequest, input RoleCreateMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("account_id", input.AccountID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Name == "" {
				return toolInputError("name is required"), nil, nil
			}
			if err := models.ValidateFreeText("name", input.Name); err != nil {
				return toolInputError(err.Error()), nil, nil
			}
			if input.Description != "" {
				if err := models.ValidateFreeText("description", input.Description); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
			}

			grant := models.SplitCSV(input.PermissionsGrant)
			revoke := models.SplitCSV(input.PermissionsRevoke)
			if len(grant) == 0 && len(revoke) == 0 {
				return toolInputError("permissions_grant is required"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			role, err := client.RoleCreate(ctx, models.RoleCreateInput{
				AccountID:   input.AccountID,
				Name:        input.Name,
				Description: input.Description,
				Permissions: models.PermissionsInput{
					Grant:  grant,
					Revoke: revoke,
				},
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(role)
		}),
	)

	// role_set_name
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "role_set_name",
			Description: "Admin: Update a role's display name",
		},
		wrapTool(debug, "role_set_name", func(ctx context.Context, _ *mcp.CallToolRequest, input RoleSetNameMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("account_id", input.AccountID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("role_id", input.RoleID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Name == "" {
				return toolInputError("name is required"), nil, nil
			}
			if err := models.ValidateFreeText("name", input.Name); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			role, err := client.RoleSetName(ctx, models.RoleSetNameInput{
				AccountID: input.AccountID,
				RoleID:    input.RoleID,
				Name:      input.Name,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(role)
		}),
	)

	// role_set_description
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "role_set_description",
			Description: "Admin: Update or clear a role's description",
		},
		wrapTool(debug, "role_set_description", func(ctx context.Context, _ *mcp.CallToolRequest, input RoleSetDescriptionMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("account_id", input.AccountID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("role_id", input.RoleID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Description != nil && *input.Description != "" {
				if err := models.ValidateFreeText("description", *input.Description); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			role, err := client.RoleSetDescription(ctx, models.RoleSetDescriptionInput{
				AccountID:   input.AccountID,
				RoleID:      input.RoleID,
				Description: input.Description,
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(role)
		}),
	)

	// role_set_permissions (destructive — privilege-changing)
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "role_set_permissions",
			Description: "Admin: Update permissions for a role. Permission actions use format domain:action (e.g. inspection:inspection.create). Only specified actions are modified; others are unchanged",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: &destructive,
			},
		},
		wrapTool(debug, "role_set_permissions", func(ctx context.Context, _ *mcp.CallToolRequest, input RoleSetPermissionsMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("account_id", input.AccountID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("role_id", input.RoleID); errResult != nil {
				return errResult, nil, nil
			}

			grant := models.SplitCSV(input.PermissionsGrant)
			revoke := models.SplitCSV(input.PermissionsRevoke)
			if len(grant) == 0 && len(revoke) == 0 {
				return toolInputError("at least one of permissions_grant or permissions_revoke is required"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			role, err := client.RoleSetPermissions(ctx, models.RoleSetPermissionsInput{
				AccountID: input.AccountID,
				RoleID:    input.RoleID,
				Permissions: models.PermissionsInput{
					Grant:  grant,
					Revoke: revoke,
				},
			})
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(role)
		}),
	)
}

// registerWebhookMutationTools registers all 2 webhook mutation tools.
func registerWebhookMutationTools(server *mcp.Server, client apiClient, debug bool) {
	// webhook_create
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "webhook_create",
			Description: "Admin: Create a new webhook subscription for event notifications (Standard Webhooks protocol). Note: the signing secret is redacted in the response — use the CLI to retrieve it",
		},
		wrapTool(debug, "webhook_create", func(ctx context.Context, _ *mcp.CallToolRequest, input WebhookCreateMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("subscriber_id", input.SubscriberID); errResult != nil {
				return errResult, nil, nil
			}
			if input.SubscriberType == "" {
				return toolInputError("subscriber_type is required"), nil, nil
			}
			upperType := strings.ToUpper(input.SubscriberType)
			if !models.ValidWebhookSubscriberTypes[upperType] {
				return toolInputError("subscriber_type must be ACCOUNT or PLUGIN"), nil, nil
			}
			url := strings.TrimSpace(input.URL)
			if url == "" {
				return toolInputError("url is required"), nil, nil
			}
			if err := models.ValidateWebhookURL(url); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			createInput := models.WebhookCreateInput{
				SubscriberID:   input.SubscriberID,
				SubscriberType: upperType,
				URL:            url,
			}

			if input.Subjects != "" {
				subjects, err := models.ValidateWebhookSubjects(input.Subjects)
				if err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				if len(subjects) > 0 {
					createInput.Subjects = subjects
				}
			}

			if input.Status != "" {
				upper := strings.ToUpper(input.Status)
				if !models.ValidWebhookStatuses[upper] {
					return toolInputError("status must be ENABLED or DISABLED"), nil, nil
				}
				createInput.Status = upper
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			webhook, err := client.WebhookCreate(ctx, createInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			redactWebhookSecret(webhook)
			return toolJSON(webhook)
		}),
	)

	// webhook_update
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "webhook_update",
			Description: "Admin: Update an existing webhook's URL, status, or subjects",
		},
		wrapTool(debug, "webhook_update", func(ctx context.Context, _ *mcp.CallToolRequest, input WebhookUpdateMCPInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("id", input.ID); errResult != nil {
				return errResult, nil, nil
			}

			updateInput := models.WebhookUpdateInput{
				ID: input.ID,
			}

			url := strings.TrimSpace(input.URL)
			if url != "" {
				if err := models.ValidateWebhookURL(url); err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				updateInput.URL = url
			}

			if input.Status != "" {
				upper := strings.ToUpper(input.Status)
				if !models.ValidWebhookStatuses[upper] {
					return toolInputError("status must be ENABLED or DISABLED"), nil, nil
				}
				updateInput.Status = upper
			}

			if input.Subjects != "" {
				subjects, err := models.ValidateWebhookSubjects(input.Subjects)
				if err != nil {
					return toolInputError(err.Error()), nil, nil
				}
				if len(subjects) > 0 {
					updateInput.Subjects = subjects
				}
			}

			if url == "" && input.Status == "" && input.Subjects == "" {
				return toolInputError("at least one of url, status, or subjects is required"), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			webhook, err := client.WebhookUpdate(ctx, updateInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			// Defence in depth: webhookFields query selection currently omits
			// signingSecret, but a future schema change could include it.
			redactWebhookSecret(webhook)
			return toolJSON(webhook)
		}),
	)
}
