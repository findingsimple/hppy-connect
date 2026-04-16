package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolTimeout is the maximum wall-clock duration for a single tool call,
// preventing slow upstream APIs from holding semaphore slots indefinitely.
const toolTimeout = 5 * time.Minute

// sem limits concurrent pagination loops to 3.
var sem = make(chan struct{}, 3)

// maxLimit caps the maximum number of items a single tool call can return.
const maxLimit = 10000

// Input structs for typed tool handlers.

type ListPropertiesInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Maximum number of properties to return. Omit for default (1000)."`
}

type ListUnitsInput struct {
	PropertyID string `json:"property_id" jsonschema:"required,The property ID to list units for."`
	Limit      int    `json:"limit,omitempty" jsonschema:"Maximum number of units to return."`
}

type ListWorkOrdersInput struct {
	PropertyID    string `json:"property_id,omitempty" jsonschema:"Property ID to scope results."`
	UnitID        string `json:"unit_id,omitempty" jsonschema:"Unit ID to scope results (more specific than property_id)."`
	CreatedAfter  string `json:"created_after,omitempty" jsonschema:"Filter: created after this ISO 8601 date."`
	CreatedBefore string `json:"created_before,omitempty" jsonschema:"Filter: created before this ISO 8601 date."`
	Status        string `json:"status,omitempty" jsonschema:"Filter by status: OPEN, ON_HOLD, or COMPLETED."`
	Limit         int    `json:"limit,omitempty" jsonschema:"Maximum number of work orders to return."`
}

type ListInspectionsInput struct {
	PropertyID    string `json:"property_id,omitempty" jsonschema:"Property ID to scope results."`
	UnitID        string `json:"unit_id,omitempty" jsonschema:"Unit ID to scope results."`
	CreatedAfter  string `json:"created_after,omitempty" jsonschema:"Filter: created after this ISO 8601 date."`
	CreatedBefore string `json:"created_before,omitempty" jsonschema:"Filter: created before this ISO 8601 date."`
	Status        string `json:"status,omitempty" jsonschema:"Filter: COMPLETE, EXPIRED, INCOMPLETE, or SCHEDULED."`
	Limit         int    `json:"limit,omitempty" jsonschema:"Maximum number of inspections to return."`
}

// Domain-specific interfaces — each contains only the methods its tools need.
// Test mocks can embed a no-op base struct and override only the methods under test,
// preventing a new mutation from breaking every existing mock.

type accountReader interface {
	GetAccount(ctx context.Context) (*models.Account, error)
	EnsureAuth(ctx context.Context) error
}

type propertyReader interface {
	ListProperties(ctx context.Context, opts models.ListOptions) ([]models.Property, int, error)
	ListUnits(ctx context.Context, propertyID string, opts models.ListOptions) ([]models.Unit, int, error)
}

type workOrderReader interface {
	ListWorkOrders(ctx context.Context, opts models.ListOptions) ([]models.WorkOrder, int, error)
}

type inspectionReader interface {
	ListInspections(ctx context.Context, opts models.ListOptions) ([]models.Inspection, int, error)
}

// Mutation interfaces — added per domain phase. Initially empty, populated in later phases.

type workOrderMutator interface {
	WorkOrderCreate(ctx context.Context, input models.WorkOrderCreateInput) (*models.WorkOrder, error)
	WorkOrderSetStatusAndSubStatus(ctx context.Context, input models.WorkOrderSetStatusAndSubStatusInput) (*models.WorkOrder, error)
	WorkOrderSetAssignee(ctx context.Context, input models.WorkOrderSetAssigneeInput) (*models.WorkOrder, error)
	WorkOrderSetDescription(ctx context.Context, workOrderID, description string) (*models.WorkOrder, error)
	WorkOrderSetPriority(ctx context.Context, workOrderID, priority string) (*models.WorkOrder, error)
	WorkOrderSetScheduledFor(ctx context.Context, workOrderID, scheduledFor string) (*models.WorkOrder, error)
	WorkOrderSetLocation(ctx context.Context, workOrderID, locationID string) (*models.WorkOrder, error)
	WorkOrderSetType(ctx context.Context, workOrderID, woType string) (*models.WorkOrder, error)
	WorkOrderSetEntryNotes(ctx context.Context, workOrderID, entryNotes string) (*models.WorkOrder, error)
	WorkOrderSetPermissionToEnter(ctx context.Context, workOrderID string, permission bool) (*models.WorkOrder, error)
	WorkOrderSetResidentApprovedEntry(ctx context.Context, workOrderID string, approved bool) (*models.WorkOrder, error)
	WorkOrderSetUnitEntered(ctx context.Context, workOrderID string, unitEntered bool) (*models.WorkOrder, error)
	WorkOrderArchive(ctx context.Context, workOrderID string) (*models.WorkOrder, error)
	WorkOrderAddComment(ctx context.Context, workOrderID, comment string) (*models.WorkOrder, error)
	WorkOrderAddTime(ctx context.Context, workOrderID, duration string) (*models.WorkOrder, error)
	WorkOrderAddAttachment(ctx context.Context, input models.WorkOrderAddAttachmentInput) (*models.WorkOrderAddAttachmentResult, error)
	WorkOrderRemoveAttachment(ctx context.Context, workOrderID, attachmentID string) (*models.WorkOrder, error)
	WorkOrderStartTimer(ctx context.Context, workOrderID, startedAt string) (*models.WorkOrder, error)
	WorkOrderStopTimer(ctx context.Context, workOrderID, stoppedAt string) (*models.WorkOrder, error)
}
type inspectionMutator interface {
	InspectionCreate(ctx context.Context, input models.InspectionCreateInput) (*models.Inspection, error)
	InspectionStart(ctx context.Context, inspectionID string) (*models.Inspection, error)
	InspectionComplete(ctx context.Context, inspectionID string) (*models.Inspection, error)
	InspectionReopen(ctx context.Context, inspectionID string) (*models.Inspection, error)
	InspectionArchive(ctx context.Context, inspectionID string) (*models.Inspection, error)
	InspectionExpire(ctx context.Context, inspectionID string) (*models.Inspection, error)
	InspectionUnexpire(ctx context.Context, inspectionID string) (*models.Inspection, error)
	InspectionSetAssignee(ctx context.Context, input models.InspectionSetAssigneeInput) (*models.Inspection, error)
	InspectionSetDueBy(ctx context.Context, input models.InspectionSetDueByInput) (*models.Inspection, error)
	InspectionSetScheduledFor(ctx context.Context, inspectionID, scheduledFor string) (*models.Inspection, error)
	InspectionSetHeaderField(ctx context.Context, input models.InspectionSetHeaderFieldInput) (*models.Inspection, error)
	InspectionSetFooterField(ctx context.Context, input models.InspectionSetFooterFieldInput) (*models.Inspection, error)
	InspectionSetItemNotes(ctx context.Context, input models.InspectionSetItemNotesInput) (*models.Inspection, error)
	InspectionRateItem(ctx context.Context, input models.InspectionRateItemInput) (*models.Inspection, error)
	InspectionAddSection(ctx context.Context, input models.InspectionAddSectionInput) (*models.Inspection, error)
	InspectionDeleteSection(ctx context.Context, input models.InspectionDeleteSectionInput) (*models.Inspection, error)
	InspectionDuplicateSection(ctx context.Context, input models.InspectionDuplicateSectionInput) (*models.Inspection, error)
	InspectionRenameSection(ctx context.Context, input models.InspectionRenameSectionInput) (*models.Inspection, error)
	InspectionAddItem(ctx context.Context, input models.InspectionAddItemInput) (*models.Inspection, error)
	InspectionDeleteItem(ctx context.Context, input models.InspectionDeleteItemInput) (*models.Inspection, error)
	InspectionAddItemPhoto(ctx context.Context, input models.InspectionAddItemPhotoInput) (*models.InspectionAddItemPhotoResult, error)
	InspectionRemoveItemPhoto(ctx context.Context, input models.InspectionRemoveItemPhotoInput) (*models.Inspection, error)
	InspectionMoveItemPhoto(ctx context.Context, input models.InspectionMoveItemPhotoInput) (*models.Inspection, error)
	InspectionSendToGuest(ctx context.Context, input models.InspectionSendToGuestInput) (*models.InspectionGuestLink, error)
}
type projectMutator interface{}
type userMutator interface{}
type membershipMutator interface{}
type propertyAccessMutator interface{}
type roleMutator interface{}
type webhookMutator interface{}

// apiClient composes all domain interfaces. The concrete *api.Client satisfies this.
// Mocks in tests only need to implement the sub-interface their test uses.
type apiClient interface {
	accountReader
	propertyReader
	workOrderReader
	inspectionReader
	workOrderMutator
	inspectionMutator
	projectMutator
	userMutator
	membershipMutator
	propertyAccessMutator
	roleMutator
	webhookMutator
}

func registerTools(server *mcp.Server, client apiClient, debug bool) {
	// get_account — no pagination, no semaphore needed
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_account",
			Description: "Get the authenticated HappyCo account's information including name and ID",
		},
		wrapTool(debug, "get_account", func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			account, err := client.GetAccount(ctx)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(account)
		}),
	)

	// list_properties
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "list_properties",
			Description: "List all properties for the authenticated HappyCo account, including name, address, and creation date",
		},
		wrapTool(debug, "list_properties", func(ctx context.Context, _ *mcp.CallToolRequest, input ListPropertiesInput) (*mcp.CallToolResult, any, error) {
			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()
			if err := acquireSem(ctx, sem); err != nil {
				return toolError(err), nil, nil
			}
			defer releaseSem(sem)

			opts := models.ListOptions{Limit: clampLimit(input.Limit)}
			properties, total, err := client.ListProperties(ctx, opts)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(map[string]any{
				"total":      total,
				"count":      len(properties),
				"properties": emptyIfNil(properties),
			})
		}),
	)

	// list_units
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "list_units",
			Description: "List all units within a specific HappyCo property",
		},
		wrapTool(debug, "list_units", func(ctx context.Context, _ *mcp.CallToolRequest, input ListUnitsInput) (*mcp.CallToolResult, any, error) {
			if errResult := requireID("property_id", input.PropertyID); errResult != nil {
				return errResult, nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()
			if err := acquireSem(ctx, sem); err != nil {
				return toolError(err), nil, nil
			}
			defer releaseSem(sem)

			opts := models.ListOptions{Limit: clampLimit(input.Limit)}
			units, total, err := client.ListUnits(ctx, input.PropertyID, opts)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(map[string]any{
				"total": total,
				"count": len(units),
				"units": emptyIfNil(units),
			})
		}),
	)

	// list_work_orders
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "list_work_orders",
			Description: "List work orders for the authenticated HappyCo account. Can filter by property, unit, date range, and status",
		},
		wrapTool(debug, "list_work_orders", func(ctx context.Context, _ *mcp.CallToolRequest, input ListWorkOrdersInput) (*mcp.CallToolResult, any, error) {
			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()
			if err := acquireSem(ctx, sem); err != nil {
				return toolError(err), nil, nil
			}
			defer releaseSem(sem)

			opts, err := buildListOpts(input.PropertyID, input.UnitID, input.Status, input.CreatedAfter, input.CreatedBefore, input.Limit, models.ValidWorkOrderStatuses)
			if err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			workOrders, total, err := client.ListWorkOrders(ctx, opts)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(map[string]any{
				"total":       total,
				"count":       len(workOrders),
				"work_orders": emptyIfNil(workOrders),
			})
		}),
	)

	// list_inspections
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "list_inspections",
			Description: "List inspections for the authenticated HappyCo account. Can filter by property, unit, date range, and status",
		},
		wrapTool(debug, "list_inspections", func(ctx context.Context, _ *mcp.CallToolRequest, input ListInspectionsInput) (*mcp.CallToolResult, any, error) {
			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()
			if err := acquireSem(ctx, sem); err != nil {
				return toolError(err), nil, nil
			}
			defer releaseSem(sem)

			opts, err := buildListOpts(input.PropertyID, input.UnitID, input.Status, input.CreatedAfter, input.CreatedBefore, input.Limit, models.ValidInspectionStatuses)
			if err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			inspections, total, err := client.ListInspections(ctx, opts)
			if err != nil {
				return toolError(err), nil, nil
			}
			return toolJSON(map[string]any{
				"total":       total,
				"count":       len(inspections),
				"inspections": emptyIfNil(inspections),
			})
		}),
	)

	// Register mutation tools by domain
	registerWorkOrderMutationTools(server, client, debug)
	registerInspectionMutationTools(server, client, debug)
}

// acquireSem acquires a semaphore slot, respecting context cancellation.
func acquireSem(ctx context.Context, s chan struct{}) error {
	select {
	case s <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("api_error: request cancelled while waiting for capacity")
	}
}

func releaseSem(s chan struct{}) { <-s }

// wrapTool adds debug logging around a tool handler.
func wrapTool[In any](debug bool, name string, handler mcp.ToolHandlerFor[In, any]) mcp.ToolHandlerFor[In, any] {
	if !debug {
		return handler
	}
	return func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, any, error) {
		start := time.Now()
		result, out, err := handler(ctx, req, input)
		isErr := err != nil || (result != nil && result.IsError)
		log.Printf("[debug] tool=%s duration=%s error=%v", name, time.Since(start), isErr)
		return result, out, err
	}
}

// toolError converts an API error into a sanitised MCP error result.
// Logs the full error to stderr; returns only the category prefix to the client.
func toolError(err error) *mcp.CallToolResult {
	msg := err.Error()
	// Log the full error for debugging; only expose the category to the MCP client.
	log.Printf("[error] %s", msg)
	category := sanitiseErrorCategory(msg)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: category}},
		IsError: true,
	}
}

// sanitiseErrorCategory extracts "category: generic message" from an api error string,
// avoiding leaking internal details (URLs, HTTP codes) to the MCP client.
func sanitiseErrorCategory(msg string) string {
	categories := map[string]string{
		"auth_failed":   "auth_failed: Authentication failed — check credentials",
		"not_found":     "not_found: The requested resource was not found",
		"invalid_input": "invalid_input: Invalid input parameters",
		"rate_limited":  "rate_limited: API rate limit exceeded — try again later",
		"api_error":     "api_error: An API error occurred — try again later",
	}
	for prefix, friendly := range categories {
		if strings.HasPrefix(msg, prefix) {
			return friendly
		}
	}
	return "api_error: An unexpected error occurred"
}

// toolInputError returns a validation error result to the MCP client.
func toolInputError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "invalid_input: " + msg}},
		IsError: true,
	}
}

// toolJSON marshals a value into a compact JSON text CallToolResult.
// Compact JSON is used because MCP clients (LLMs) do not benefit from indentation
// and large payloads can be significantly smaller without it.
func toolJSON(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("[error] failed to marshal response: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "api_error: failed to marshal response"}},
			IsError: true,
		}, nil, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

// emptyIfNil returns an empty slice if the input is nil, ensuring JSON
// serialisation produces [] rather than null.
func emptyIfNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// clampLimit ensures limit is within safe bounds.
// 0 = API default (1000); negative or over-max are clamped to maxLimit.
func clampLimit(limit int) int {
	if limit <= 0 {
		return 0 // let API client apply its default
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

// validateID checks that an ID string contains only safe characters.
func validateID(name, value string) error {
	return models.ValidateID(name, value)
}

// requireID validates that an ID is non-empty and contains only safe characters.
// Returns an MCP error result on failure, or nil on success.
func requireID(name, value string) *mcp.CallToolResult {
	if value == "" {
		return toolInputError(name + " is required")
	}
	if err := validateID(name, value); err != nil {
		return toolInputError(err.Error())
	}
	return nil
}

// buildListOpts maps common filter fields to models.ListOptions.
// unit_id takes precedence over property_id (more specific scope).
// validStatuses is the set of allowed status values for the calling tool.
func buildListOpts(propertyID, unitID, status, createdAfter, createdBefore string, limit int, validStatuses map[string]bool) (models.ListOptions, error) {
	if err := validateID("property_id", propertyID); err != nil {
		return models.ListOptions{}, err
	}
	if err := validateID("unit_id", unitID); err != nil {
		return models.ListOptions{}, err
	}

	opts := models.ListOptions{Limit: clampLimit(limit)}

	if unitID != "" {
		opts.LocationID = unitID
	} else if propertyID != "" {
		opts.LocationID = propertyID
	}

	statuses, err := models.ValidateStatus(status, validStatuses)
	if err != nil {
		return models.ListOptions{}, err
	}
	opts.Status = statuses

	if createdAfter != "" {
		t, err := parseFlexibleDate(createdAfter)
		if err != nil {
			return models.ListOptions{}, fmt.Errorf("created_after must be ISO 8601 format (e.g. 2026-01-15T00:00:00Z or 2026-01-15)")
		}
		opts.CreatedAfter = &t
	}

	if createdBefore != "" {
		t, err := parseFlexibleDate(createdBefore)
		if err != nil {
			return models.ListOptions{}, fmt.Errorf("created_before must be ISO 8601 format (e.g. 2026-01-15T00:00:00Z or 2026-01-15)")
		}
		opts.CreatedBefore = &t
	}

	if err := models.ValidateDateRange(opts.CreatedAfter, opts.CreatedBefore); err != nil {
		return models.ListOptions{}, err
	}

	return opts, nil
}

// parseFlexibleDate accepts both RFC3339 and YYYY-MM-DD formats.
// YYYY-MM-DD dates are treated as midnight UTC.
func parseFlexibleDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		if t.Format("2006-01-02") != s {
			return time.Time{}, fmt.Errorf("%q is not a valid calendar date", s)
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("%q is not a valid date", s)
}
