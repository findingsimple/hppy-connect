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
	Type              string `json:"type,omitempty" jsonschema:"Type: SERVICE_REQUEST TURN CAPITAL_IMPROVEMENT INSPECTION_RELATED APPLIANCE_REPLACEMENT"`
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

type WorkOrderIDStringInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	Value       string `json:"value" jsonschema:"required,The value to set"`
}

type WorkOrderIDBoolInput struct {
	WorkOrderID string `json:"work_order_id" jsonschema:"required,The work order ID"`
	Value       bool   `json:"value" jsonschema:"required,The boolean value to set"`
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
			apiInput.Description = input.Description
			apiInput.ScheduledFor = input.ScheduledFor
			apiInput.EntryNotes = input.EntryNotes
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
					return toolInputError("type must be one of SERVICE_REQUEST, TURN, CAPITAL_IMPROVEMENT, INSPECTION_RELATED, APPLIANCE_REPLACEMENT"), nil, nil
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
		wrapTool(debug, "work_order_set_description", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDStringInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Value == "" {
				return toolInputError("value (description) is required"), nil, nil
			}
			if err := models.ValidateFreeText("description", input.Value); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetDescription(ctx, input.WorkOrderID, input.Value)
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
		wrapTool(debug, "work_order_set_priority", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDStringInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Value == "" {
				return toolInputError("value (priority) is required"), nil, nil
			}
			upper := strings.ToUpper(input.Value)
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
		wrapTool(debug, "work_order_set_scheduled_for", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDStringInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Value == "" {
				return toolInputError("value (scheduled_for date) is required"), nil, nil
			}
			if err := models.ValidateTimestamp("scheduled_for", input.Value); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetScheduledFor(ctx, input.WorkOrderID, input.Value)
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
		wrapTool(debug, "work_order_set_location", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDStringInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if errResult := requireID("value (location_id)", input.Value); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetLocation(ctx, input.WorkOrderID, input.Value)
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
		wrapTool(debug, "work_order_set_type", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDStringInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Value == "" {
				return toolInputError("value (type) is required"), nil, nil
			}
			upper := strings.ToUpper(input.Value)
			if !models.ValidWorkOrderTypes[upper] {
				return toolInputError("type must be one of SERVICE_REQUEST, TURN, CAPITAL_IMPROVEMENT, INSPECTION_RELATED, APPLIANCE_REPLACEMENT"), nil, nil
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
		wrapTool(debug, "work_order_set_entry_notes", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDStringInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Value == "" {
				return toolInputError("value (entry_notes) is required"), nil, nil
			}
			if err := models.ValidateFreeText("entry_notes", input.Value); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetEntryNotes(ctx, input.WorkOrderID, input.Value)
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
		wrapTool(debug, "work_order_set_permission_to_enter", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDBoolInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetPermissionToEnter(ctx, input.WorkOrderID, input.Value)
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
		wrapTool(debug, "work_order_set_resident_approved_entry", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDBoolInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetResidentApprovedEntry(ctx, input.WorkOrderID, input.Value)
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
		wrapTool(debug, "work_order_set_unit_entered", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDBoolInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderSetUnitEntered(ctx, input.WorkOrderID, input.Value)
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
		wrapTool(debug, "work_order_add_comment", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDStringInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Value == "" {
				return toolInputError("value (comment) is required"), nil, nil
			}
			if err := models.ValidateFreeText("comment", input.Value); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderAddComment(ctx, input.WorkOrderID, input.Value)
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
		wrapTool(debug, "work_order_add_time", func(ctx context.Context, _ *mcp.CallToolRequest, input WorkOrderIDStringInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("work_order_id", input.WorkOrderID); errResult != nil {
				return errResult, nil, nil
			}
			if input.Value == "" {
				return toolInputError("value (duration in ISO 8601 format, e.g. PT1H30M) is required"), nil, nil
			}
			if err := models.ValidateDuration(input.Value); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			wo, err := client.WorkOrderAddTime(ctx, input.WorkOrderID, input.Value)
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
			Description: "Add an attachment to a work order. Returns a signed URL for uploading the file via PUT request",
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
	Expires      bool   `json:"expires" jsonschema:"required,Whether the inspection expires at the due date"`
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

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()

			insp, err := client.InspectionSetDueBy(ctx, models.InspectionSetDueByInput{
				InspectionID: input.InspectionID,
				DueBy:        input.DueBy,
				Expires:      input.Expires,
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
			Description: "Add a photo to an inspection item. Returns a signed URL for uploading the file via PUT request",
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

			result, err := client.InspectionSendToGuest(ctx, apiInput)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(result)
		}),
	)
}
