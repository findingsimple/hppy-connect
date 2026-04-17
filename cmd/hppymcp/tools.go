package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
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

// emailLikePattern matches anything that looks like an email address embedded
// in a free-text field. Conservative: requires `@` followed by a domain-shaped
// trailer. Matches inside surrounding text so "Jane Doe (jane@acme.com)" is
// scrubbed.
var emailLikePattern = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)

// scrubEmailLike replaces email-shaped substrings inside a free-text field with
// a redaction sentinel. Used to defend against users putting their email in
// fields like ShortName or Account.Name. Returns the input unchanged if nothing
// matches.
func scrubEmailLike(s string) string {
	if !strings.Contains(s, "@") {
		return s
	}
	return emailLikePattern.ReplaceAllString(s, "[email redacted]")
}

// redactMembershipEmails strips/scrubs email-bearing fields on each membership
// before returning to the LLM. Bulk membership listings can contain hundreds
// of email addresses; persisting them to LLM context/transcripts is PII
// over-exposure.
//
// Belt-and-braces:
//   - User.Email is cleared outright (its semantic value is the email).
//   - User.Name, User.ShortName, and Account.Name are scrubbed for email-shaped
//     substrings. Users sometimes put emails in these (e.g. ShortName "jdoe@acme")
//     and bypassing redaction by inspecting alternate fields is the obvious
//     escape hatch a security review would test.
//
// Opt back in via list_members.include_emails=true (gated by the
// HPPYMCP_ALLOW_EMAIL_DISCLOSURE env var on the server).
func redactMembershipEmails(members []models.AccountMembership) {
	for i := range members {
		if u := members[i].User; u != nil {
			u.Email = ""
			u.Name = scrubEmailLike(u.Name)
			u.ShortName = scrubEmailLike(u.ShortName)
		}
		if a := members[i].Account; a != nil {
			a.Name = scrubEmailLike(a.Name)
		}
	}
}

// emailDisclosureEnabled reports whether the operator has opted in to allow
// list_members responses to include user emails. Env var is checked at request
// time so tests can flip it; defaults to disabled.
//
// Why an env var: setting include_emails=true via the tool parameter alone is
// prompt-injectable — attacker-controlled text (a property name, a webhook URL
// description) can persuade an LLM to flip the flag. The env var moves the
// decision out-of-band so a model can't override it. See README for setup.
func emailDisclosureEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("HPPYMCP_ALLOW_EMAIL_DISCLOSURE")))
	return v == "1" || v == "true" || v == "yes"
}

// emailSem limits concurrent calls to mutations that send external emails
// (inspection_send_to_guest, user_create). A misbehaving LLM iterating over
// a batch could otherwise fan out to many parallel emails. Capacity 2 keeps
// throughput high enough for normal use while preventing the fan-out case.
//
// Note: this gate is in-process only. The upstream HappyCo API has its own
// rate limiting, but rapid concurrent calls within the gate window can still
// cause user-visible duplicate emails. See CLAUDE.md "Known Limitations" item E.
var emailSem = make(chan struct{}, 2)

// maxLimit caps the maximum number of items a single tool call can return.
//
// Why this is tighter than the CLI: the MCP server returns results into an
// LLM's context window. Capping at 10,000 (vs the API client's hard ceiling
// of 50,000) trades a smaller maximum for predictable token usage. CLI users
// asking for tens of thousands of items know what they want and can see the
// raw output; an LLM cannot.
//
// To raise this for a specific use case, change the value here — do not
// bypass clampLimit.
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

type ListMembersInput struct {
	Search          string `json:"search,omitempty" jsonschema:"Search by user name or email. Email match works regardless of include_emails."`
	IncludeInactive bool   `json:"include_inactive,omitempty" jsonschema:"Include deactivated memberships. Default false (active only)."`
	IncludeEmails   bool   `json:"include_emails,omitempty" jsonschema:"Include user email addresses in the response. Default false. Operator must also set HPPYMCP_ALLOW_EMAIL_DISCLOSURE=1 in the server environment — emails are PII, persist in conversation logs, and the env var prevents prompt-injected toggles."`
	Limit           int    `json:"limit,omitempty" jsonschema:"Maximum number of members to return."`
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
}

type propertyReader interface {
	ListProperties(ctx context.Context, opts models.ListOptions) ([]models.Property, int, error)
	ListUnits(ctx context.Context, propertyID string, opts models.ListOptions) ([]models.Unit, int, error)
}

type workOrderReader interface {
	ListWorkOrders(ctx context.Context, opts models.ListOptions) ([]models.WorkOrder, int, error)
}

type membershipReader interface {
	ListMembers(ctx context.Context, opts models.ListOptions) ([]models.AccountMembership, int, error)
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
type projectMutator interface {
	ProjectCreate(ctx context.Context, input models.ProjectCreateInput) (*models.Project, error)
	ProjectSetAssignee(ctx context.Context, input models.ProjectSetAssigneeInput) (*models.Project, error)
	ProjectSetNotes(ctx context.Context, projectID, notes string) (*models.Project, error)
	ProjectSetDueAt(ctx context.Context, projectID, dueAt string) (*models.Project, error)
	ProjectSetStartAt(ctx context.Context, projectID, startAt string) (*models.Project, error)
	ProjectSetPriority(ctx context.Context, projectID, priority string) (*models.Project, error)
	ProjectSetOnHold(ctx context.Context, projectID string, onHold bool) (*models.Project, error)
	ProjectSetAvailabilityTargetAt(ctx context.Context, projectID string, availabilityTargetAt *string) (*models.Project, error)
}
type userMutator interface {
	UserCreate(ctx context.Context, input models.UserCreateInput) (*models.User, error)
	UserSetEmail(ctx context.Context, userID, email string) (*models.User, error)
	UserSetName(ctx context.Context, userID, name string) (*models.User, error)
	UserSetShortName(ctx context.Context, userID string, shortName *string) (*models.User, error)
	UserSetPhone(ctx context.Context, userID string, phone *string) (*models.User, error)
	UserGrantPropertyAccess(ctx context.Context, input models.UserGrantPropertyAccessInput) (*models.User, error)
	UserRevokePropertyAccess(ctx context.Context, input models.UserRevokePropertyAccessInput) (*models.User, error)
}
type membershipMutator interface {
	AccountMembershipCreate(ctx context.Context, input models.AccountMembershipCreateInput) (*models.AccountMembership, error)
	AccountMembershipActivate(ctx context.Context, input models.AccountMembershipActivateInput) (*models.AccountMembership, error)
	AccountMembershipDeactivate(ctx context.Context, input models.AccountMembershipDeactivateInput) (*models.AccountMembership, error)
	AccountMembershipSetRoles(ctx context.Context, input models.AccountMembershipSetRolesInput) (*models.AccountMembership, error)
}
type propertyAccessMutator interface {
	PropertyGrantUserAccess(ctx context.Context, input models.PropertyGrantUserAccessInput) (*models.PropertyAccess, error)
	PropertyRevokeUserAccess(ctx context.Context, input models.PropertyRevokeUserAccessInput) (*models.PropertyAccess, error)
	PropertySetAccountWideAccess(ctx context.Context, input models.PropertySetAccountWideAccessInput) (*models.PropertyAccess, error)
}
type roleMutator interface {
	RoleCreate(ctx context.Context, input models.RoleCreateInput) (*models.Role, error)
	RoleSetName(ctx context.Context, input models.RoleSetNameInput) (*models.Role, error)
	RoleSetDescription(ctx context.Context, input models.RoleSetDescriptionInput) (*models.Role, error)
	RoleSetPermissions(ctx context.Context, input models.RoleSetPermissionsInput) (*models.Role, error)
}
type webhookMutator interface {
	WebhookCreate(ctx context.Context, input models.WebhookCreateInput) (*models.Webhook, error)
	WebhookUpdate(ctx context.Context, input models.WebhookUpdateInput) (*models.Webhook, error)
}

// apiClient composes all domain interfaces. The concrete *api.Client satisfies this.
// Mocks in tests only need to implement the sub-interface their test uses.
type apiClient interface {
	accountReader
	propertyReader
	workOrderReader
	inspectionReader
	membershipReader
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

	// list_members
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "list_members",
			Description: "List account members (users with memberships). Can search by name or email and optionally include inactive members",
		},
		wrapTool(debug, "list_members", func(ctx context.Context, _ *mcp.CallToolRequest, input ListMembersInput) (*mcp.CallToolResult, any, error) {
			if err := models.ValidateFreeText("search", input.Search); err != nil {
				return toolInputError(err.Error()), nil, nil
			}

			ctx, cancel := context.WithTimeout(ctx, toolTimeout)
			defer cancel()
			if err := acquireSem(ctx, sem); err != nil {
				return toolError(err), nil, nil
			}
			defer releaseSem(sem)

			opts := models.ListOptions{
				Limit:           clampLimit(input.Limit),
				Search:          input.Search,
				IncludeInactive: input.IncludeInactive,
			}
			// include_emails alone is prompt-injectable (see emailDisclosureEnabled
			// docstring). Reject the request unless the operator has explicitly
			// enabled disclosure via env var. The redaction sweep below still
			// runs as belt-and-braces even when disclosure IS enabled.
			if input.IncludeEmails && !emailDisclosureEnabled() {
				return toolInputError("email disclosure is disabled on this server; set HPPYMCP_ALLOW_EMAIL_DISCLOSURE=1 in the server environment to enable, then retry"), nil, nil
			}

			members, total, err := client.ListMembers(ctx, opts)
			if err != nil {
				return toolError(err), nil, nil
			}
			if !input.IncludeEmails {
				redactMembershipEmails(members)
			}
			return toolJSON(map[string]any{
				"total":   total,
				"count":   len(members),
				"members": emptyIfNil(members),
			})
		}),
	)

	// Register mutation tools by domain
	registerWorkOrderMutationTools(server, client, debug)
	registerInspectionMutationTools(server, client, debug)
	registerProjectMutationTools(server, client, debug)
	registerAccountMutationTools(server, client, debug)
	registerRoleMutationTools(server, client, debug)
	registerWebhookMutationTools(server, client, debug)
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
		"auth_failed":        "auth_failed: Authentication failed — check credentials",
		"not_found":          "not_found: The requested resource was not found",
		"invalid_input":      "invalid_input: Invalid input parameters",
		"rate_limited":       "rate_limited: API rate limit exceeded — try again later",
		"api_error":          "api_error: An API error occurred — try again later",
		"already_applied":    "already_applied: The destructive operation likely succeeded on a prior attempt — verify resource state before retrying",
		"pagination_aborted": "pagination_aborted: Server returned more pages than the safety ceiling allows; retrying will not help — narrow the query or contact support",
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
