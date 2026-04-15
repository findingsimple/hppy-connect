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
