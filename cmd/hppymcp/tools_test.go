package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock API client
// ---------------------------------------------------------------------------

type mockClient struct {
	account     *models.Account
	properties  []models.Property
	propTotal   int
	units       []models.Unit
	unitTotal   int
	members     []models.AccountMembership
	memberTotal int
	workOrders  []models.WorkOrder
	woTotal     int
	inspections []models.Inspection
	inspTotal   int
	err         error // returned by all methods when set

	// Mutation response fields
	mutatedWorkOrder *models.WorkOrder
	attachmentResult *models.WorkOrderAddAttachmentResult

	// Inspection mutation response fields
	mutatedInspection *models.Inspection
	photoResult       *models.InspectionAddItemPhotoResult
	guestLinkResult   *models.InspectionGuestLink

	// Capture call args for verification.
	lastListOpts    models.ListOptions
	lastPropertyID  string
	lastMutationID  string
	lastCreateInput models.WorkOrderCreateInput
	lastStatusInput models.WorkOrderSetStatusAndSubStatusInput
	lastAssignInput models.WorkOrderSetAssigneeInput
	lastAttachInput models.WorkOrderAddAttachmentInput
	lastStringValue string
	lastBoolValue   bool

	// Project mutation response fields
	mutatedProject *models.Project

	// Project capture fields
	lastProjCreateInput models.ProjectCreateInput
	lastProjAssignInput models.ProjectSetAssigneeInput

	// Account mutation response fields
	mutatedUser       *models.User
	mutatedMembership *models.AccountMembership
	mutatedAccess     *models.PropertyAccess

	// Account capture fields
	lastUserCreateInput     models.UserCreateInput
	lastMembershipInput     models.AccountMembershipCreateInput
	lastMembershipRoleInput models.AccountMembershipSetRolesInput
	lastPropertyAccessInput models.PropertyGrantUserAccessInput
	lastUserAccessInput     models.UserGrantPropertyAccessInput

	// Role mutation response fields
	mutatedRole *models.Role

	// Role capture fields
	lastRoleCreateInput  models.RoleCreateInput
	lastRoleSetNameInput models.RoleSetNameInput
	lastRoleSetDescInput models.RoleSetDescriptionInput
	lastRolePermInput    models.RoleSetPermissionsInput

	// Webhook mutation response fields
	mutatedWebhook *models.Webhook

	// Webhook capture fields
	lastWebhookCreateInput models.WebhookCreateInput
	lastWebhookUpdateInput models.WebhookUpdateInput

	// Inspection capture fields
	lastInspCreateInput        models.InspectionCreateInput
	lastInspAssignInput        models.InspectionSetAssigneeInput
	lastInspDueByInput         models.InspectionSetDueByInput
	lastInspHeaderInput        models.InspectionSetHeaderFieldInput
	lastInspFooterInput        models.InspectionSetFooterFieldInput
	lastInspItemNotesInput     models.InspectionSetItemNotesInput
	lastInspRateItemInput      models.InspectionRateItemInput
	lastInspAddSectionInput    models.InspectionAddSectionInput
	lastInspDeleteSectionInput models.InspectionDeleteSectionInput
	lastInspDupSectionInput    models.InspectionDuplicateSectionInput
	lastInspRenameSectionInput models.InspectionRenameSectionInput
	lastInspAddItemInput       models.InspectionAddItemInput
	lastInspDeleteItemInput    models.InspectionDeleteItemInput
	lastInspAddPhotoInput      models.InspectionAddItemPhotoInput
	lastInspRemovePhotoInput   models.InspectionRemoveItemPhotoInput
	lastInspMovePhotoInput     models.InspectionMoveItemPhotoInput
	lastInspSendToGuestInput   models.InspectionSendToGuestInput
}

func (m *mockClient) GetAccount(_ context.Context) (*models.Account, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.account, nil
}

func (m *mockClient) ListProperties(_ context.Context, opts models.ListOptions) ([]models.Property, int, error) {
	m.lastListOpts = opts
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.properties, m.propTotal, nil
}

func (m *mockClient) ListUnits(_ context.Context, propertyID string, opts models.ListOptions) ([]models.Unit, int, error) {
	m.lastPropertyID = propertyID
	m.lastListOpts = opts
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.units, m.unitTotal, nil
}

func (m *mockClient) ListMembers(_ context.Context, opts models.ListOptions) ([]models.AccountMembership, int, error) {
	m.lastListOpts = opts
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.members, m.memberTotal, nil
}

func (m *mockClient) ListWorkOrders(_ context.Context, opts models.ListOptions) ([]models.WorkOrder, int, error) {
	m.lastListOpts = opts
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.workOrders, m.woTotal, nil
}

func (m *mockClient) ListInspections(_ context.Context, opts models.ListOptions) ([]models.Inspection, int, error) {
	m.lastListOpts = opts
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.inspections, m.inspTotal, nil
}

func (m *mockClient) EnsureAuth(_ context.Context) error {
	return m.err
}

// --- Work Order Mutation Mocks ---

func (m *mockClient) woResult() *models.WorkOrder {
	if m.mutatedWorkOrder != nil {
		return m.mutatedWorkOrder
	}
	return &models.WorkOrder{ID: "wo-mock", Status: "OPEN"}
}

func (m *mockClient) WorkOrderCreate(_ context.Context, input models.WorkOrderCreateInput) (*models.WorkOrder, error) {
	m.lastCreateInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderSetStatusAndSubStatus(_ context.Context, input models.WorkOrderSetStatusAndSubStatusInput) (*models.WorkOrder, error) {
	m.lastMutationID = input.WorkOrderID
	m.lastStatusInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderSetAssignee(_ context.Context, input models.WorkOrderSetAssigneeInput) (*models.WorkOrder, error) {
	m.lastMutationID = input.WorkOrderID
	m.lastAssignInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderSetDescription(_ context.Context, id, value string) (*models.WorkOrder, error) {
	m.lastStringValue = value
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderSetPriority(_ context.Context, id, value string) (*models.WorkOrder, error) {
	m.lastStringValue = value
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderSetScheduledFor(_ context.Context, id, _ string) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderSetLocation(_ context.Context, id, _ string) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderSetType(_ context.Context, id, _ string) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderSetEntryNotes(_ context.Context, id, _ string) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderSetPermissionToEnter(_ context.Context, id string, _ bool) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderSetResidentApprovedEntry(_ context.Context, id string, _ bool) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderSetUnitEntered(_ context.Context, id string, _ bool) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderArchive(_ context.Context, id string) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderAddComment(_ context.Context, id, _ string) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderAddTime(_ context.Context, id, _ string) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderAddAttachment(_ context.Context, input models.WorkOrderAddAttachmentInput) (*models.WorkOrderAddAttachmentResult, error) {
	m.lastAttachInput = input
	if m.err != nil {
		return nil, m.err
	}
	if m.attachmentResult != nil {
		return m.attachmentResult, nil
	}
	return &models.WorkOrderAddAttachmentResult{
		WorkOrder:  *m.woResult(),
		Attachment: models.WorkOrderAttachment{ID: "att-1", Name: "photo.jpg"},
		SignedURL:  "https://storage.example.com/upload/att-1",
	}, nil
}

func (m *mockClient) WorkOrderRemoveAttachment(_ context.Context, id, _ string) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderStartTimer(_ context.Context, id, _ string) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

func (m *mockClient) WorkOrderStopTimer(_ context.Context, id, _ string) (*models.WorkOrder, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.woResult(), nil
}

// --- Inspection Mutation Mocks ---

func (m *mockClient) inspResult() *models.Inspection {
	if m.mutatedInspection != nil {
		return m.mutatedInspection
	}
	return &models.Inspection{ID: "insp-mock", Status: "SCHEDULED"}
}

func (m *mockClient) InspectionCreate(_ context.Context, input models.InspectionCreateInput) (*models.Inspection, error) {
	m.lastInspCreateInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionStart(_ context.Context, id string) (*models.Inspection, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionComplete(_ context.Context, id string) (*models.Inspection, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionReopen(_ context.Context, id string) (*models.Inspection, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionArchive(_ context.Context, id string) (*models.Inspection, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionExpire(_ context.Context, id string) (*models.Inspection, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionUnexpire(_ context.Context, id string) (*models.Inspection, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionSetAssignee(_ context.Context, input models.InspectionSetAssigneeInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspAssignInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionSetDueBy(_ context.Context, input models.InspectionSetDueByInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspDueByInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionSetScheduledFor(_ context.Context, id, _ string) (*models.Inspection, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionSetHeaderField(_ context.Context, input models.InspectionSetHeaderFieldInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspHeaderInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionSetFooterField(_ context.Context, input models.InspectionSetFooterFieldInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspFooterInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionSetItemNotes(_ context.Context, input models.InspectionSetItemNotesInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspItemNotesInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionRateItem(_ context.Context, input models.InspectionRateItemInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspRateItemInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionAddSection(_ context.Context, input models.InspectionAddSectionInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspAddSectionInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionDeleteSection(_ context.Context, input models.InspectionDeleteSectionInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspDeleteSectionInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionDuplicateSection(_ context.Context, input models.InspectionDuplicateSectionInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspDupSectionInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionRenameSection(_ context.Context, input models.InspectionRenameSectionInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspRenameSectionInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionAddItem(_ context.Context, input models.InspectionAddItemInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspAddItemInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionDeleteItem(_ context.Context, input models.InspectionDeleteItemInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspDeleteItemInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionAddItemPhoto(_ context.Context, input models.InspectionAddItemPhotoInput) (*models.InspectionAddItemPhotoResult, error) {
	m.lastInspAddPhotoInput = input
	if m.err != nil {
		return nil, m.err
	}
	if m.photoResult != nil {
		return m.photoResult, nil
	}
	return &models.InspectionAddItemPhotoResult{
		Inspection:      *m.inspResult(),
		InspectionPhoto: models.InspectionPhoto{ID: "photo-1", MimeType: "image/jpeg"},
		SignedURL:       "https://storage.example.com/upload/photo-1",
	}, nil
}

func (m *mockClient) InspectionRemoveItemPhoto(_ context.Context, input models.InspectionRemoveItemPhotoInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspRemovePhotoInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionMoveItemPhoto(_ context.Context, input models.InspectionMoveItemPhotoInput) (*models.Inspection, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspMovePhotoInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.inspResult(), nil
}

func (m *mockClient) InspectionSendToGuest(_ context.Context, input models.InspectionSendToGuestInput) (*models.InspectionGuestLink, error) {
	m.lastMutationID = input.InspectionID
	m.lastInspSendToGuestInput = input
	if m.err != nil {
		return nil, m.err
	}
	if m.guestLinkResult != nil {
		return m.guestLinkResult, nil
	}
	return &models.InspectionGuestLink{
		InspectionID: input.InspectionID,
		Link:         "https://app.happyco.com/inspect/guest/abc123",
	}, nil
}

// --- Project mutation mock methods ---

func (m *mockClient) projResult() *models.Project {
	if m.mutatedProject != nil {
		return m.mutatedProject
	}
	return &models.Project{ID: "proj-1", Status: "PLANNED", Priority: "NORMAL"}
}

func (m *mockClient) ProjectCreate(_ context.Context, input models.ProjectCreateInput) (*models.Project, error) {
	m.lastProjCreateInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.projResult(), nil
}

func (m *mockClient) ProjectSetAssignee(_ context.Context, input models.ProjectSetAssigneeInput) (*models.Project, error) {
	m.lastMutationID = input.ProjectID
	m.lastProjAssignInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.projResult(), nil
}

func (m *mockClient) ProjectSetNotes(_ context.Context, id, notes string) (*models.Project, error) {
	m.lastMutationID = id
	m.lastStringValue = notes
	if m.err != nil {
		return nil, m.err
	}
	return m.projResult(), nil
}

func (m *mockClient) ProjectSetDueAt(_ context.Context, id, _ string) (*models.Project, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.projResult(), nil
}

func (m *mockClient) ProjectSetStartAt(_ context.Context, id, _ string) (*models.Project, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.projResult(), nil
}

func (m *mockClient) ProjectSetPriority(_ context.Context, id, _ string) (*models.Project, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.projResult(), nil
}

func (m *mockClient) ProjectSetOnHold(_ context.Context, id string, _ bool) (*models.Project, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.projResult(), nil
}

func (m *mockClient) ProjectSetAvailabilityTargetAt(_ context.Context, id string, _ *string) (*models.Project, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.projResult(), nil
}

// --- User mutation mock methods ---

func (m *mockClient) userResult() *models.User {
	if m.mutatedUser != nil {
		return m.mutatedUser
	}
	return &models.User{ID: "user-1", Name: "Test User", Email: "test@example.com"}
}

func (m *mockClient) UserCreate(_ context.Context, input models.UserCreateInput) (*models.User, error) {
	m.lastUserCreateInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.userResult(), nil
}

func (m *mockClient) UserSetEmail(_ context.Context, id, email string) (*models.User, error) {
	m.lastMutationID = id
	m.lastStringValue = email
	if m.err != nil {
		return nil, m.err
	}
	return m.userResult(), nil
}

func (m *mockClient) UserSetName(_ context.Context, id, name string) (*models.User, error) {
	m.lastMutationID = id
	m.lastStringValue = name
	if m.err != nil {
		return nil, m.err
	}
	return m.userResult(), nil
}

func (m *mockClient) UserSetShortName(_ context.Context, id string, _ *string) (*models.User, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.userResult(), nil
}

func (m *mockClient) UserSetPhone(_ context.Context, id string, _ *string) (*models.User, error) {
	m.lastMutationID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.userResult(), nil
}

func (m *mockClient) UserGrantPropertyAccess(_ context.Context, input models.UserGrantPropertyAccessInput) (*models.User, error) {
	m.lastUserAccessInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.userResult(), nil
}

func (m *mockClient) UserRevokePropertyAccess(_ context.Context, input models.UserRevokePropertyAccessInput) (*models.User, error) {
	m.lastMutationID = input.UserID
	if m.err != nil {
		return nil, m.err
	}
	return m.userResult(), nil
}

// --- Membership mutation mock methods ---

func (m *mockClient) membershipResult() *models.AccountMembership {
	if m.mutatedMembership != nil {
		return m.mutatedMembership
	}
	return &models.AccountMembership{
		IsActive: true,
		Account:  &models.Account{ID: "acct-1"},
		User:     &models.User{ID: "user-1", Name: "Test User", Email: "test@example.com"},
	}
}

func (m *mockClient) AccountMembershipCreate(_ context.Context, input models.AccountMembershipCreateInput) (*models.AccountMembership, error) {
	m.lastMembershipInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.membershipResult(), nil
}

func (m *mockClient) AccountMembershipActivate(_ context.Context, input models.AccountMembershipActivateInput) (*models.AccountMembership, error) {
	m.lastMutationID = input.UserID
	if m.err != nil {
		return nil, m.err
	}
	return m.membershipResult(), nil
}

func (m *mockClient) AccountMembershipDeactivate(_ context.Context, input models.AccountMembershipDeactivateInput) (*models.AccountMembership, error) {
	m.lastMutationID = input.UserID
	if m.err != nil {
		return nil, m.err
	}
	return m.membershipResult(), nil
}

func (m *mockClient) AccountMembershipSetRoles(_ context.Context, input models.AccountMembershipSetRolesInput) (*models.AccountMembership, error) {
	m.lastMembershipRoleInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.membershipResult(), nil
}

// --- Property access mutation mock methods ---

func (m *mockClient) accessResult() *models.PropertyAccess {
	if m.mutatedAccess != nil {
		return m.mutatedAccess
	}
	return &models.PropertyAccess{PropertyID: "prop-1", AccountWideAccess: false}
}

func (m *mockClient) PropertyGrantUserAccess(_ context.Context, input models.PropertyGrantUserAccessInput) (*models.PropertyAccess, error) {
	m.lastPropertyAccessInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.accessResult(), nil
}

func (m *mockClient) PropertyRevokeUserAccess(_ context.Context, input models.PropertyRevokeUserAccessInput) (*models.PropertyAccess, error) {
	m.lastMutationID = input.PropertyID
	if m.err != nil {
		return nil, m.err
	}
	return m.accessResult(), nil
}

func (m *mockClient) PropertySetAccountWideAccess(_ context.Context, input models.PropertySetAccountWideAccessInput) (*models.PropertyAccess, error) {
	m.lastMutationID = input.PropertyID
	if m.err != nil {
		return nil, m.err
	}
	return m.accessResult(), nil
}

// --- Role mutation mock methods ---

func (m *mockClient) roleResult() *models.Role {
	if m.mutatedRole != nil {
		return m.mutatedRole
	}
	return &models.Role{
		ID:   "role-1",
		Name: "Test Role",
		Permissions: []models.Permission{
			{Action: "test:action", Description: "Test action"},
		},
	}
}

func (m *mockClient) RoleCreate(_ context.Context, input models.RoleCreateInput) (*models.Role, error) {
	m.lastRoleCreateInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.roleResult(), nil
}

func (m *mockClient) RoleSetName(_ context.Context, input models.RoleSetNameInput) (*models.Role, error) {
	m.lastRoleSetNameInput = input
	m.lastMutationID = input.RoleID
	if m.err != nil {
		return nil, m.err
	}
	return m.roleResult(), nil
}

func (m *mockClient) RoleSetDescription(_ context.Context, input models.RoleSetDescriptionInput) (*models.Role, error) {
	m.lastRoleSetDescInput = input
	m.lastMutationID = input.RoleID
	if m.err != nil {
		return nil, m.err
	}
	return m.roleResult(), nil
}

func (m *mockClient) RoleSetPermissions(_ context.Context, input models.RoleSetPermissionsInput) (*models.Role, error) {
	m.lastRolePermInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.roleResult(), nil
}

func (m *mockClient) webhookResult() *models.Webhook {
	if m.mutatedWebhook != nil {
		return m.mutatedWebhook
	}
	return &models.Webhook{
		ID:       "wh-1",
		URL:      "https://example.com/webhook",
		Status:   "DISABLED",
		Subjects: []string{"INSPECTIONS"},
		Subscriber: &models.WebhookSubscriber{
			Type: "ACCOUNT",
			ID:   "acct-1",
		},
		SigningSecret: "whsec_test123",
	}
}

func (m *mockClient) WebhookCreate(_ context.Context, input models.WebhookCreateInput) (*models.Webhook, error) {
	m.lastWebhookCreateInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.webhookResult(), nil
}

func (m *mockClient) WebhookUpdate(_ context.Context, input models.WebhookUpdateInput) (*models.Webhook, error) {
	m.lastWebhookUpdateInput = input
	if m.err != nil {
		return nil, m.err
	}
	return m.webhookResult(), nil
}

// Verify mockClient satisfies the interface at compile time.
var _ apiClient = (*mockClient)(nil)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newTestServer creates an MCP server with tools, resources, and prompts registered
// against the given mock, connects it via in-memory transport, and returns the
// client session. The caller must defer cs.Close().
func newTestServer(t *testing.T, mock *mockClient) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	server := mcp.NewServer(
		&mcp.Implementation{Name: "hppymcp-test", Version: "test"},
		&mcp.ServerOptions{Instructions: "test"},
	)
	registerTools(server, mock, false)
	registerResources(server, mock)
	registerPrompts(server)

	ct, st := mcp.NewInMemoryTransports()
	_, err := server.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(
		&mcp.Implementation{Name: "test-client", Version: "test"},
		nil,
	)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { cs.Close() })
	return cs
}

func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	require.NoError(t, err)
	return result
}

func toolText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
	tc, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "expected TextContent, got %T", result.Content[0])
	return tc.Text
}

// ---------------------------------------------------------------------------
// Tool Tests
// ---------------------------------------------------------------------------

// TestRegisteredToolCount pins the total number of MCP tools registered.
// The README claims a specific count (currently 77 = 6 read + 71 mutation).
// If you add or remove a tool, update this count AND the README claims at:
//   - README.md headline ("77 tools across 8 domains")
//   - README.md "MCP Tools (N total: ...)" section header
//   - README.md "Mutation Tools (N total)" subsection header
func TestRegisteredToolCount(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	tools, err := cs.ListTools(context.Background(), nil)
	require.NoError(t, err)

	// Total registered tools. If this number changes, update README.
	const expectedTotal = 77
	assert.Equal(t, expectedTotal, len(tools.Tools),
		"registered tool count drifted from README (%d). Update README.md headline AND section headers.", expectedTotal)
}

func TestToolGetAccount(t *testing.T) {
	mock := &mockClient{
		account: &models.Account{ID: "54522", Name: "Test Account"},
	}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "get_account", nil)
	assert.False(t, result.IsError)

	var account models.Account
	require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &account))
	assert.Equal(t, "54522", account.ID)
	assert.Equal(t, "Test Account", account.Name)
}

func TestToolListProperties(t *testing.T) {
	t.Run("happy path with payload verification", func(t *testing.T) {
		mock := &mockClient{
			properties: []models.Property{
				{ID: "p1", Name: "Sunrise Apartments", Address: models.Address{City: "Austin"}},
				{ID: "p2", Name: "Oakwood Estates", Address: models.Address{City: "Dallas"}},
			},
			propTotal: 2,
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_properties", nil)
		assert.False(t, result.IsError)

		var parsed struct {
			Total      int               `json:"total"`
			Count      int               `json:"count"`
			Properties []models.Property `json:"properties"`
		}
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &parsed))
		assert.Equal(t, 2, parsed.Total)
		assert.Equal(t, 2, parsed.Count)
		require.Len(t, parsed.Properties, 2)
		assert.Equal(t, "p1", parsed.Properties[0].ID)
		assert.Equal(t, "Sunrise Apartments", parsed.Properties[0].Name)
		assert.Equal(t, "Austin", parsed.Properties[0].Address.City)
		assert.Equal(t, "p2", parsed.Properties[1].ID)
	})

	t.Run("with limit", func(t *testing.T) {
		mock := &mockClient{
			properties: []models.Property{{ID: "p1", Name: "One"}},
			propTotal:  1,
		}
		cs := newTestServer(t, mock)

		callTool(t, cs, "list_properties", map[string]any{"limit": 5})
		assert.Equal(t, 5, mock.lastListOpts.Limit)
	})

	t.Run("nil slice serialises as empty array", func(t *testing.T) {
		mock := &mockClient{properties: nil, propTotal: 0}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_properties", nil)
		assert.False(t, result.IsError)
		text := toolText(t, result)
		assert.Contains(t, text, `"properties":[]`)
		assert.NotContains(t, text, "null")
	})
}

func TestToolListUnits(t *testing.T) {
	t.Run("happy path with payload verification", func(t *testing.T) {
		mock := &mockClient{
			units:     []models.Unit{{ID: "u1", Name: "101"}, {ID: "u2", Name: "102"}},
			unitTotal: 2,
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_units", map[string]any{"property_id": "p1"})
		assert.False(t, result.IsError)
		assert.Equal(t, "p1", mock.lastPropertyID)

		var parsed struct {
			Total int           `json:"total"`
			Count int           `json:"count"`
			Units []models.Unit `json:"units"`
		}
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &parsed))
		assert.Equal(t, 2, parsed.Total)
		require.Len(t, parsed.Units, 2)
		assert.Equal(t, "u1", parsed.Units[0].ID)
		assert.Equal(t, "101", parsed.Units[0].Name)
	})

	t.Run("missing property_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_units", nil)
		assert.True(t, result.IsError)
		text := toolText(t, result)
		// SDK schema validation catches the required field before our handler runs.
		assert.Contains(t, text, "property_id")
	})

	t.Run("empty property_id returns invalid_input", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_units", map[string]any{"property_id": ""})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolListWorkOrders(t *testing.T) {
	t.Run("happy path with payload verification", func(t *testing.T) {
		mock := &mockClient{
			workOrders: []models.WorkOrder{
				{ID: "wo1", Status: "OPEN", Summary: "Leaky faucet", Priority: "URGENT"},
			},
			woTotal: 1,
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_work_orders", nil)
		assert.False(t, result.IsError)

		var parsed struct {
			Total      int                `json:"total"`
			Count      int                `json:"count"`
			WorkOrders []models.WorkOrder `json:"work_orders"`
		}
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &parsed))
		assert.Equal(t, 1, parsed.Total)
		require.Len(t, parsed.WorkOrders, 1)
		assert.Equal(t, "wo1", parsed.WorkOrders[0].ID)
		assert.Equal(t, "OPEN", parsed.WorkOrders[0].Status)
		assert.Equal(t, "Leaky faucet", parsed.WorkOrders[0].Summary)
	})

	t.Run("with all filters", func(t *testing.T) {
		mock := &mockClient{woTotal: 0}
		cs := newTestServer(t, mock)

		callTool(t, cs, "list_work_orders", map[string]any{
			"property_id":    "prop-1",
			"status":         "OPEN",
			"created_after":  "2026-01-01T00:00:00Z",
			"created_before": "2026-04-01T00:00:00Z",
			"limit":          50,
		})
		assert.Equal(t, "prop-1", mock.lastListOpts.LocationID)
		assert.Equal(t, []string{"OPEN"}, mock.lastListOpts.Status)
		assert.NotNil(t, mock.lastListOpts.CreatedAfter)
		assert.NotNil(t, mock.lastListOpts.CreatedBefore)
		assert.Equal(t, 50, mock.lastListOpts.Limit)
	})

	t.Run("unit_id takes precedence over property_id", func(t *testing.T) {
		mock := &mockClient{woTotal: 0}
		cs := newTestServer(t, mock)

		callTool(t, cs, "list_work_orders", map[string]any{
			"property_id": "prop-1",
			"unit_id":     "unit-99",
		})
		assert.Equal(t, "unit-99", mock.lastListOpts.LocationID)
	})

	t.Run("invalid status rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_work_orders", map[string]any{"status": "INVALID"})
		assert.True(t, result.IsError)
		text := toolText(t, result)
		assert.Contains(t, text, "invalid_input")
		assert.Contains(t, text, "INVALID")
	})

	t.Run("lowercase status normalised to uppercase", func(t *testing.T) {
		mock := &mockClient{woTotal: 0}
		cs := newTestServer(t, mock)

		callTool(t, cs, "list_work_orders", map[string]any{"status": "open"})
		assert.Equal(t, []string{"OPEN"}, mock.lastListOpts.Status)
	})
}

func TestToolListInspections(t *testing.T) {
	t.Run("happy path with payload verification", func(t *testing.T) {
		score := 85.0
		mock := &mockClient{
			inspections: []models.Inspection{
				{ID: "insp1", Name: "Move-in", Status: "COMPLETE", Score: &score},
			},
			inspTotal: 1,
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_inspections", map[string]any{"property_id": "prop-1"})
		assert.False(t, result.IsError)
		assert.Equal(t, "prop-1", mock.lastListOpts.LocationID)

		var parsed struct {
			Total       int                 `json:"total"`
			Count       int                 `json:"count"`
			Inspections []models.Inspection `json:"inspections"`
		}
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &parsed))
		assert.Equal(t, 1, parsed.Total)
		require.Len(t, parsed.Inspections, 1)
		assert.Equal(t, "insp1", parsed.Inspections[0].ID)
		assert.Equal(t, "Move-in", parsed.Inspections[0].Name)
		assert.Equal(t, 85.0, *parsed.Inspections[0].Score)
	})

	t.Run("with status filter", func(t *testing.T) {
		mock := &mockClient{inspTotal: 0}
		cs := newTestServer(t, mock)

		callTool(t, cs, "list_inspections", map[string]any{"status": "SCHEDULED"})
		assert.Equal(t, []string{"SCHEDULED"}, mock.lastListOpts.Status)
	})

	t.Run("with date filters", func(t *testing.T) {
		mock := &mockClient{inspTotal: 0}
		cs := newTestServer(t, mock)

		callTool(t, cs, "list_inspections", map[string]any{
			"created_after":  "2026-03-01T00:00:00Z",
			"created_before": "2026-04-01T00:00:00Z",
		})
		require.NotNil(t, mock.lastListOpts.CreatedAfter)
		require.NotNil(t, mock.lastListOpts.CreatedBefore)
		assert.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), *mock.lastListOpts.CreatedAfter)
		assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), *mock.lastListOpts.CreatedBefore)
	})

	t.Run("invalid inspection status rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_inspections", map[string]any{"status": "OPEN"})
		assert.True(t, result.IsError)
		text := toolText(t, result)
		assert.Contains(t, text, "invalid_input")
		// OPEN is valid for work orders but not inspections
		assert.Contains(t, text, "OPEN")
	})
}

func TestToolListMembers(t *testing.T) {
	t.Run("happy path with payload verification", func(t *testing.T) {
		mock := &mockClient{
			members: []models.AccountMembership{
				{
					IsActive:  true,
					User:      &models.User{ID: "u1", Name: "Alice Smith", Email: "alice@example.com"},
					CreatedAt: "2026-01-01T00:00:00Z",
					Roles:     &models.MembershipRoles{Nodes: []models.Role{{ID: "r1", Name: "Admin"}}},
				},
				{
					IsActive: false,
					User:     &models.User{ID: "u2", Name: "Bob Jones", Email: "bob@example.com"},
				},
			},
			memberTotal: 2,
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_members", nil)
		assert.False(t, result.IsError)

		var parsed struct {
			Total   int                        `json:"total"`
			Count   int                        `json:"count"`
			Members []models.AccountMembership `json:"members"`
		}
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &parsed))
		assert.Equal(t, 2, parsed.Total)
		assert.Equal(t, 2, parsed.Count)
		require.Len(t, parsed.Members, 2)
		assert.Equal(t, "u1", parsed.Members[0].User.ID)
		assert.Equal(t, "Alice Smith", parsed.Members[0].User.Name)
		assert.True(t, parsed.Members[0].IsActive)
		require.NotNil(t, parsed.Members[0].Roles)
		require.Len(t, parsed.Members[0].Roles.Nodes, 1)
		assert.Equal(t, "r1", parsed.Members[0].Roles.Nodes[0].ID)
		assert.Equal(t, "Admin", parsed.Members[0].Roles.Nodes[0].Name)
		assert.Equal(t, "u2", parsed.Members[1].User.ID)
		assert.False(t, parsed.Members[1].IsActive)
	})

	t.Run("default include_inactive is false", func(t *testing.T) {
		mock := &mockClient{memberTotal: 0}
		cs := newTestServer(t, mock)

		callTool(t, cs, "list_members", nil)
		assert.False(t, mock.lastListOpts.IncludeInactive)
	})

	t.Run("with search and include_inactive", func(t *testing.T) {
		mock := &mockClient{memberTotal: 0}
		cs := newTestServer(t, mock)

		callTool(t, cs, "list_members", map[string]any{
			"search":           "alice",
			"include_inactive": true,
			"limit":            25,
		})
		assert.Equal(t, "alice", mock.lastListOpts.Search)
		assert.True(t, mock.lastListOpts.IncludeInactive)
		assert.Equal(t, 25, mock.lastListOpts.Limit)
	})

	t.Run("nil user in membership serialises safely", func(t *testing.T) {
		mock := &mockClient{
			members:     []models.AccountMembership{{IsActive: true, User: nil}},
			memberTotal: 1,
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_members", nil)
		assert.False(t, result.IsError)
		text := toolText(t, result)
		assert.Contains(t, text, `"user":null`)
	})

	t.Run("nil slice serialises as empty array", func(t *testing.T) {
		mock := &mockClient{members: nil, memberTotal: 0}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_members", nil)
		assert.False(t, result.IsError)
		text := toolText(t, result)
		assert.Contains(t, text, `"members":[]`)
		assert.NotContains(t, text, "null")
	})

	t.Run("API error propagated", func(t *testing.T) {
		mock := &mockClient{err: fmt.Errorf("api_error: HTTP 500")}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_members", nil)
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "api_error")
	})

	t.Run("emails redacted by default to avoid PII in LLM context", func(t *testing.T) {
		mock := &mockClient{
			members: []models.AccountMembership{
				{IsActive: true, User: &models.User{ID: "u1", Name: "Alice", Email: "alice@example.com"}},
				{IsActive: true, User: &models.User{ID: "u2", Name: "Bob", Email: "bob@example.com"}},
			},
			memberTotal: 2,
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_members", nil)
		require.False(t, result.IsError)

		body := toolText(t, result)
		assert.NotContains(t, body, "alice@example.com", "email leaked into LLM-visible response")
		assert.NotContains(t, body, "bob@example.com", "email leaked into LLM-visible response")
	})

	t.Run("emails included when include_emails=true", func(t *testing.T) {
		mock := &mockClient{
			members: []models.AccountMembership{
				{IsActive: true, User: &models.User{ID: "u1", Name: "Alice", Email: "alice@example.com"}},
			},
			memberTotal: 1,
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "list_members", map[string]any{"include_emails": true})
		require.False(t, result.IsError)

		assert.Contains(t, toolText(t, result), "alice@example.com")
	})
}

// ---------------------------------------------------------------------------
// Work Order Mutation Tool Tests
// ---------------------------------------------------------------------------

func TestToolWorkOrderCreate(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{
			mutatedWorkOrder: &models.WorkOrder{ID: "wo-new", Status: "OPEN", Priority: "URGENT", Description: "Fix leak"},
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_create", map[string]any{
			"location_id": "loc-123",
			"description": "Fix leak",
			"priority":    "URGENT",
		})
		assert.False(t, result.IsError)

		var wo models.WorkOrder
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &wo))
		assert.Equal(t, "wo-new", wo.ID)
		assert.Equal(t, "URGENT", wo.Priority)
	})

	t.Run("missing location_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_create", map[string]any{
			"description": "Fix leak",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "location_id")
	})

	t.Run("invalid location_id rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_create", map[string]any{
			"location_id": "../../etc/passwd",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("invalid priority rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_create", map[string]any{
			"location_id": "loc-123",
			"priority":    "INVALID",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "priority must be NORMAL or URGENT")
	})

	t.Run("lowercase priority normalised", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_create", map[string]any{
			"location_id": "loc-123",
			"priority":    "urgent",
		})
		assert.False(t, result.IsError)
	})
}

func TestToolWorkOrderSetStatus(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_status", map[string]any{
			"work_order_id": "wo-123",
			"status":        "COMPLETED",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_status", map[string]any{
			"status": "OPEN",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})

	t.Run("invalid status rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_status", map[string]any{
			"work_order_id": "wo-123",
			"status":        "INVALID",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "status must be")
	})

	t.Run("explicit sub_status reaches mock", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_status", map[string]any{
			"work_order_id": "wo-123",
			"status":        "COMPLETED",
			"sub_status":    "CANCELLED",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "CANCELLED", mock.lastStatusInput.SubStatus.SubStatus)
	})

	t.Run("default sub_status is UNKNOWN", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_status", map[string]any{
			"work_order_id": "wo-123",
			"status":        "OPEN",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "UNKNOWN", mock.lastStatusInput.SubStatus.SubStatus)
	})

	t.Run("invalid sub_status rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_status", map[string]any{
			"work_order_id": "wo-123",
			"status":        "OPEN",
			"sub_status":    "INVALID",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "sub_status must be")
	})
}

func TestToolWorkOrderSetAssignee(t *testing.T) {
	t.Run("happy path with VENDOR", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_assignee", map[string]any{
			"work_order_id": "wo-123",
			"assignee_id":   "vendor-456",
			"assignee_type": "VENDOR",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastAssignInput.WorkOrderID)
		assert.Equal(t, "vendor-456", mock.lastAssignInput.Assignee.AssigneeID)
		assert.Equal(t, "VENDOR", mock.lastAssignInput.Assignee.AssigneeType)
	})

	t.Run("default assignee_type is USER", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_assignee", map[string]any{
			"work_order_id": "wo-123",
			"assignee_id":   "user-789",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "USER", mock.lastAssignInput.Assignee.AssigneeType)
	})

	t.Run("invalid assignee_type rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_assignee", map[string]any{
			"work_order_id": "wo-123",
			"assignee_id":   "user-789",
			"assignee_type": "ROBOT",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "assignee_type must be")
	})

	t.Run("missing assignee_id rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_assignee", map[string]any{
			"work_order_id": "wo-123",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "assignee_id")
	})
}

func TestToolWorkOrderArchive(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_archive", map[string]any{
			"work_order_id": "wo-123",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("invalid ID rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_archive", map[string]any{
			"work_order_id": "../bad",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolWorkOrderAddAttachment(t *testing.T) {
	t.Run("happy path returns signed URL", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_add_attachment", map[string]any{
			"work_order_id": "wo-123",
			"file_name":     "photo.jpg",
			"mime_type":     "image/jpeg",
		})
		assert.False(t, result.IsError)
		text := toolText(t, result)
		assert.Contains(t, text, "signedURL")
		assert.Contains(t, text, "photo.jpg")
	})

	t.Run("missing required fields", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_add_attachment", map[string]any{
			"work_order_id": "wo-123",
		})
		assert.True(t, result.IsError)
	})
}

func TestToolWorkOrderSetPriority(t *testing.T) {
	t.Run("valid priority", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_priority", map[string]any{
			"work_order_id": "wo-123",
			"priority":      "URGENT",
		})
		assert.False(t, result.IsError)
	})

	t.Run("invalid priority", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_priority", map[string]any{
			"work_order_id": "wo-123",
			"priority":      "HIGH",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "NORMAL or URGENT")
	})
}

func TestToolWorkOrderSetDescription(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_description", map[string]any{
			"work_order_id": "wo-123",
			"description":   "Fix the leaky faucet in unit 4B",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
		assert.Equal(t, "Fix the leaky faucet in unit 4B", mock.lastStringValue)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_description", map[string]any{
			"description": "some description",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})

	t.Run("missing value rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_description", map[string]any{
			"work_order_id": "wo-123",
			"description":   "",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("oversized value rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_description", map[string]any{
			"work_order_id": "wo-123",
			"description":   string(make([]byte, models.MaxFreeTextLength+1)),
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolWorkOrderSetScheduledFor(t *testing.T) {
	t.Run("happy path with valid RFC3339 timestamp", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_scheduled_for", map[string]any{
			"work_order_id": "wo-123",
			"scheduled_for": "2026-05-01T09:00:00Z",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_scheduled_for", map[string]any{
			"scheduled_for": "2026-05-01T09:00:00Z",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})

	t.Run("missing value rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_scheduled_for", map[string]any{
			"work_order_id": "wo-123",
			"scheduled_for": "",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("invalid timestamp rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_scheduled_for", map[string]any{
			"work_order_id": "wo-123",
			"scheduled_for": "not-a-timestamp",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolWorkOrderSetLocation(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_location", map[string]any{
			"work_order_id": "wo-123",
			"location_id":   "loc-456",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_location", map[string]any{
			"location_id": "loc-456",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})

	t.Run("missing value rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_location", map[string]any{
			"work_order_id": "wo-123",
			"location_id":   "",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("invalid location ID rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_location", map[string]any{
			"work_order_id": "wo-123",
			"location_id":   "../../etc/passwd",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolWorkOrderSetType(t *testing.T) {
	t.Run("happy path with valid type", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_type", map[string]any{
			"work_order_id":   "wo-123",
			"work_order_type": "SERVICE_REQUEST",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_type", map[string]any{
			"work_order_type": "TURN",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})

	t.Run("missing value rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_type", map[string]any{
			"work_order_id":   "wo-123",
			"work_order_type": "",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("invalid type rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_type", map[string]any{
			"work_order_id":   "wo-123",
			"work_order_type": "URGENT",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "SERVICE_REQUEST")
	})

	t.Run("lowercase type normalised", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_type", map[string]any{
			"work_order_id":   "wo-123",
			"work_order_type": "turn",
		})
		assert.False(t, result.IsError)
	})
}

func TestToolWorkOrderSetEntryNotes(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_entry_notes", map[string]any{
			"work_order_id": "wo-123",
			"entry_notes":   "Please knock before entering",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_entry_notes", map[string]any{
			"entry_notes": "Please knock",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})

	t.Run("missing value rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_entry_notes", map[string]any{
			"work_order_id": "wo-123",
			"entry_notes":   "",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("oversized value rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_entry_notes", map[string]any{
			"work_order_id": "wo-123",
			"entry_notes":   string(make([]byte, models.MaxFreeTextLength+1)),
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolWorkOrderSetPermissionToEnter(t *testing.T) {
	t.Run("happy path true", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_permission_to_enter", map[string]any{
			"work_order_id":       "wo-123",
			"permission_to_enter": true,
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("happy path false", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_permission_to_enter", map[string]any{
			"work_order_id":       "wo-123",
			"permission_to_enter": false,
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_permission_to_enter", map[string]any{
			"permission_to_enter": true,
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})
}

func TestToolWorkOrderSetResidentApprovedEntry(t *testing.T) {
	t.Run("happy path true", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_resident_approved_entry", map[string]any{
			"work_order_id":           "wo-123",
			"resident_approved_entry": true,
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("happy path false", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_resident_approved_entry", map[string]any{
			"work_order_id":           "wo-123",
			"resident_approved_entry": false,
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_resident_approved_entry", map[string]any{
			"resident_approved_entry": false,
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})
}

func TestToolWorkOrderSetUnitEntered(t *testing.T) {
	t.Run("happy path true", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_unit_entered", map[string]any{
			"work_order_id": "wo-123",
			"unit_entered":  true,
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("happy path false", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_unit_entered", map[string]any{
			"work_order_id": "wo-123",
			"unit_entered":  false,
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_set_unit_entered", map[string]any{
			"unit_entered": true,
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})
}

func TestToolWorkOrderAddComment(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_add_comment", map[string]any{
			"work_order_id": "wo-123",
			"comment":       "Technician will arrive at 10am",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_add_comment", map[string]any{
			"comment": "some comment",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})

	t.Run("missing value rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_add_comment", map[string]any{
			"work_order_id": "wo-123",
			"comment":       "",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("oversized comment rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_add_comment", map[string]any{
			"work_order_id": "wo-123",
			"comment":       string(make([]byte, models.MaxFreeTextLength+1)),
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolWorkOrderAddTime(t *testing.T) {
	t.Run("happy path with valid duration", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_add_time", map[string]any{
			"work_order_id": "wo-123",
			"duration":      "PT1H30M",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_add_time", map[string]any{
			"duration": "PT1H",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})

	t.Run("missing value rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_add_time", map[string]any{
			"work_order_id": "wo-123",
			"duration":      "",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("invalid duration rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_add_time", map[string]any{
			"work_order_id": "wo-123",
			"duration":      "1h30m",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("bare PT rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_add_time", map[string]any{
			"work_order_id": "wo-123",
			"duration":      "PT",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolWorkOrderRemoveAttachment(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_remove_attachment", map[string]any{
			"work_order_id": "wo-123",
			"attachment_id": "att-456",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_remove_attachment", map[string]any{
			"attachment_id": "att-456",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})

	t.Run("missing attachment_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_remove_attachment", map[string]any{
			"work_order_id": "wo-123",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "attachment_id")
	})

	t.Run("invalid attachment_id rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_remove_attachment", map[string]any{
			"work_order_id": "wo-123",
			"attachment_id": "../../bad",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolWorkOrderStartTimer(t *testing.T) {
	t.Run("happy path with valid timestamp", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_start_timer", map[string]any{
			"work_order_id": "wo-123",
			"timestamp":     "2026-04-16T10:00:00Z",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_start_timer", map[string]any{
			"timestamp": "2026-04-16T10:00:00Z",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})

	t.Run("missing timestamp rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_start_timer", map[string]any{
			"work_order_id": "wo-123",
			"timestamp":     "",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("invalid timestamp rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_start_timer", map[string]any{
			"work_order_id": "wo-123",
			"timestamp":     "2026-04-16",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolWorkOrderStopTimer(t *testing.T) {
	t.Run("happy path with valid timestamp", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_stop_timer", map[string]any{
			"work_order_id": "wo-123",
			"timestamp":     "2026-04-16T11:30:00Z",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "wo-123", mock.lastMutationID)
	})

	t.Run("missing work_order_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_stop_timer", map[string]any{
			"timestamp": "2026-04-16T11:30:00Z",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "work_order_id")
	})

	t.Run("missing timestamp rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_stop_timer", map[string]any{
			"work_order_id": "wo-123",
			"timestamp":     "",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("invalid timestamp rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "work_order_stop_timer", map[string]any{
			"work_order_id": "wo-123",
			"timestamp":     "not-a-date",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolWorkOrderMutationAPIError(t *testing.T) {
	mock := &mockClient{err: fmt.Errorf("api_error: HTTP 500")}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "work_order_archive", map[string]any{
		"work_order_id": "wo-123",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, toolText(t, result), "api_error")
}

// ---------------------------------------------------------------------------
// Inspection Mutation Tool Tests
// ---------------------------------------------------------------------------

func TestToolInspectionCreate(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{
			mutatedInspection: &models.Inspection{ID: "insp-new", Status: "SCHEDULED", Name: "Move-in"},
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_create", map[string]any{
			"location_id":   "loc-123",
			"template_id":   "tmpl-456",
			"scheduled_for": "2026-05-01T09:00:00Z",
		})
		assert.False(t, result.IsError)

		var insp models.Inspection
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &insp))
		assert.Equal(t, "insp-new", insp.ID)
		assert.Equal(t, "SCHEDULED", insp.Status)
		assert.Equal(t, "loc-123", mock.lastInspCreateInput.LocationID)
		assert.Equal(t, "tmpl-456", mock.lastInspCreateInput.TemplateID)
	})

	t.Run("missing location_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_create", map[string]any{
			"template_id":   "tmpl-456",
			"scheduled_for": "2026-05-01T09:00:00Z",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "location_id")
	})

	t.Run("missing template_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_create", map[string]any{
			"location_id":   "loc-123",
			"scheduled_for": "2026-05-01T09:00:00Z",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "template_id")
	})

	t.Run("invalid location_id rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_create", map[string]any{
			"location_id":   "../../bad",
			"template_id":   "tmpl-456",
			"scheduled_for": "2026-05-01T09:00:00Z",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("invalid scheduled_for rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_create", map[string]any{
			"location_id":   "loc-123",
			"template_id":   "tmpl-456",
			"scheduled_for": "not-a-date",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolInspectionArchive(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_archive", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastMutationID)
	})

	t.Run("missing inspection_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_archive", map[string]any{})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "inspection_id")
	})

	t.Run("invalid ID rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_archive", map[string]any{
			"inspection_id": "../bad",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolInspectionStart(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{
			mutatedInspection: &models.Inspection{ID: "insp-123", Status: "IN_PROGRESS"},
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_start", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastMutationID)

		var insp models.Inspection
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &insp))
		assert.Equal(t, "insp-123", insp.ID)
		assert.Equal(t, "IN_PROGRESS", insp.Status)
	})

	t.Run("invalid ID rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_start", map[string]any{
			"inspection_id": "../bad",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolInspectionComplete(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{
			mutatedInspection: &models.Inspection{ID: "insp-123", Status: "COMPLETE"},
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_complete", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastMutationID)

		var insp models.Inspection
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &insp))
		assert.Equal(t, "insp-123", insp.ID)
		assert.Equal(t, "COMPLETE", insp.Status)
	})
}

func TestToolInspectionReopen(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{
			mutatedInspection: &models.Inspection{ID: "insp-123", Status: "SCHEDULED"},
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_reopen", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastMutationID)

		var insp models.Inspection
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &insp))
		assert.Equal(t, "SCHEDULED", insp.Status)
	})

	t.Run("missing inspection_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_reopen", map[string]any{})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "inspection_id")
	})
}

func TestToolInspectionExpire(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_expire", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastMutationID)
	})

	t.Run("invalid ID rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_expire", map[string]any{
			"inspection_id": "../bad",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolInspectionUnexpire(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_unexpire", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastMutationID)
	})
}

func TestToolInspectionSetScheduledFor(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_scheduled_for", map[string]any{
			"inspection_id": "insp-123",
			"scheduled_for": "2026-05-01T09:00:00Z",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastMutationID)
	})

	t.Run("missing scheduled_for rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_scheduled_for", map[string]any{
			"inspection_id": "insp-123",
			"scheduled_for": "",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("invalid timestamp rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_scheduled_for", map[string]any{
			"inspection_id": "insp-123",
			"scheduled_for": "not-a-date",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolInspectionSetAssignee(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_assignee", map[string]any{
			"inspection_id": "insp-123",
			"user_id":       "user-456",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspAssignInput.InspectionID)
		assert.Equal(t, "user-456", mock.lastInspAssignInput.UserID)
	})

	t.Run("missing user_id rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_assignee", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "user_id")
	})
}

func TestToolInspectionSetDueBy(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_due_by", map[string]any{
			"inspection_id": "insp-123",
			"due_by":        "2026-06-01T00:00:00Z",
			"expires":       true,
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspDueByInput.InspectionID)
		assert.Equal(t, "2026-06-01T00:00:00Z", mock.lastInspDueByInput.DueBy)
		assert.True(t, mock.lastInspDueByInput.Expires)
	})

	t.Run("missing due_by rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_due_by", map[string]any{
			"inspection_id": "insp-123",
			"expires":       true,
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "due_by")
	})

	t.Run("missing expires rejected", func(t *testing.T) {
		// Parity with CLI: --expires is explicitly required (CLI fails when not
		// passed). MCP must not silently default to false.
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_due_by", map[string]any{
			"inspection_id": "insp-123",
			"due_by":        "2026-06-01T00:00:00Z",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "expires")
	})

	t.Run("expires false explicitly", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_due_by", map[string]any{
			"inspection_id": "insp-123",
			"due_by":        "2026-06-01T00:00:00Z",
			"expires":       false,
		})
		assert.False(t, result.IsError, "unexpected error: %s", toolText(t, result))
		assert.False(t, mock.lastInspDueByInput.Expires)
	})
}

func TestToolInspectionAddSection(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_add_section", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspAddSectionInput.InspectionID)
		assert.Equal(t, "Kitchen", mock.lastInspAddSectionInput.Name)
	})

	t.Run("missing section_name rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_add_section", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "section_name")
	})
}

func TestToolInspectionDeleteSection(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_delete_section", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspDeleteSectionInput.InspectionID)
		assert.Equal(t, "Kitchen", mock.lastInspDeleteSectionInput.SectionName)
	})

	t.Run("missing section_name rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_delete_section", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "section_name")
	})
}

func TestToolInspectionSetFooterField(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_footer_field", map[string]any{
			"inspection_id": "insp-123",
			"label":         "Inspector Signature",
			"value":         "J. Smith",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspFooterInput.InspectionID)
		assert.Equal(t, "Inspector Signature", mock.lastInspFooterInput.Label)
		assert.Equal(t, "J. Smith", mock.lastInspFooterInput.Value)
	})

	t.Run("missing label rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_footer_field", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "label")
	})
}

func TestToolInspectionSetItemNotes(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_item_notes", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
			"item_name":     "Stove",
			"notes":         "Burner not working",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspItemNotesInput.InspectionID)
		assert.Equal(t, "Kitchen", mock.lastInspItemNotesInput.SectionName)
		assert.Equal(t, "Stove", mock.lastInspItemNotesInput.ItemName)
		assert.Equal(t, "Burner not working", mock.lastInspItemNotesInput.Notes)
	})

	t.Run("missing section_name rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_item_notes", map[string]any{
			"inspection_id": "insp-123",
			"item_name":     "Stove",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "section_name")
	})

	t.Run("missing item_name rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_item_notes", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "item_name")
	})

	t.Run("oversized notes rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_item_notes", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
			"item_name":     "Stove",
			"notes":         string(make([]byte, models.MaxFreeTextLength+1)),
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolInspectionDuplicateSection(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_duplicate_section", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspDupSectionInput.InspectionID)
		assert.Equal(t, "Kitchen", mock.lastInspDupSectionInput.SectionName)
	})

	t.Run("missing section_name rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_duplicate_section", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "section_name")
	})
}

func TestToolInspectionRenameSection(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_rename_section", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
			"new_name":      "Kitchen (Updated)",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspRenameSectionInput.InspectionID)
		assert.Equal(t, "Kitchen", mock.lastInspRenameSectionInput.SectionName)
		assert.Equal(t, "Kitchen (Updated)", mock.lastInspRenameSectionInput.NewSectionName)
	})

	t.Run("missing new_name rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_rename_section", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "new_name")
	})
}

func TestToolInspectionDeleteItem(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_delete_item", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
			"item_name":     "Stove",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspDeleteItemInput.InspectionID)
		assert.Equal(t, "Kitchen", mock.lastInspDeleteItemInput.SectionName)
		assert.Equal(t, "Stove", mock.lastInspDeleteItemInput.ItemName)
	})

	t.Run("missing item_name rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_delete_item", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "item_name")
	})
}

func TestToolInspectionAddItem(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_add_item", map[string]any{
			"inspection_id":   "insp-123",
			"section_name":    "Kitchen",
			"name":            "Stove",
			"rating_group_id": "rg-1",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspAddItemInput.InspectionID)
		assert.Equal(t, "Kitchen", mock.lastInspAddItemInput.SectionName)
		assert.Equal(t, "Stove", mock.lastInspAddItemInput.Name)
		assert.Equal(t, "rg-1", mock.lastInspAddItemInput.RatingGroupID)
	})

	t.Run("missing required fields rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_add_item", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.True(t, result.IsError)
	})
}

func TestToolInspectionAddItemPhoto(t *testing.T) {
	t.Run("happy path returns signed URL and verifies capture", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_add_item_photo", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
			"item_name":     "Stove",
			"mime_type":     "image/jpeg",
		})
		assert.False(t, result.IsError)
		text := toolText(t, result)
		assert.Contains(t, text, "signedURL")
		assert.Contains(t, text, "photo-1")

		assert.Equal(t, "insp-123", mock.lastInspAddPhotoInput.InspectionID)
		assert.Equal(t, "Kitchen", mock.lastInspAddPhotoInput.SectionName)
		assert.Equal(t, "Stove", mock.lastInspAddPhotoInput.ItemName)
		assert.Equal(t, "image/jpeg", mock.lastInspAddPhotoInput.MimeType)
	})

	t.Run("invalid mime_type rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_add_item_photo", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
			"item_name":     "Stove",
			"mime_type":     "not a mime",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("negative size rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_add_item_photo", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
			"item_name":     "Stove",
			"mime_type":     "image/jpeg",
			"size":          -1,
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("missing required fields rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_add_item_photo", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.True(t, result.IsError)
	})
}

func TestToolInspectionRemoveItemPhoto(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_remove_item_photo", map[string]any{
			"inspection_id": "insp-123",
			"photo_id":      "photo-456",
			"section_name":  "Kitchen",
			"item_name":     "Stove",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspRemovePhotoInput.InspectionID)
		assert.Equal(t, "photo-456", mock.lastInspRemovePhotoInput.PhotoID)
	})
}

func TestToolInspectionMoveItemPhoto(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_move_item_photo", map[string]any{
			"inspection_id":     "insp-123",
			"photo_id":          "photo-456",
			"from_section_name": "Kitchen",
			"from_item_name":    "Stove",
			"to_section_name":   "Bathroom",
			"to_item_name":      "Sink",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspMovePhotoInput.InspectionID)
		assert.Equal(t, "photo-456", mock.lastInspMovePhotoInput.PhotoID)
		assert.Equal(t, "Kitchen", mock.lastInspMovePhotoInput.FromSectionName)
		assert.Equal(t, "Stove", mock.lastInspMovePhotoInput.FromItemName)
		assert.Equal(t, "Bathroom", mock.lastInspMovePhotoInput.ToSectionName)
		assert.Equal(t, "Sink", mock.lastInspMovePhotoInput.ToItemName)
	})

	t.Run("missing required fields rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_move_item_photo", map[string]any{
			"inspection_id": "insp-123",
			"photo_id":      "photo-456",
		})
		assert.True(t, result.IsError)
	})
}

func TestToolInspectionSendToGuest(t *testing.T) {
	t.Run("happy path returns guest link", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_send_to_guest", map[string]any{
			"inspection_id": "insp-123",
			"email":         "guest@example.com",
		})
		assert.False(t, result.IsError)
		text := toolText(t, result)
		assert.Contains(t, text, "inspectionId")
		assert.Contains(t, text, "link")
		assert.Equal(t, "guest@example.com", mock.lastInspSendToGuestInput.Email)
	})

	t.Run("missing email rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_send_to_guest", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "email")
	})

	t.Run("invalid email rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_send_to_guest", map[string]any{
			"inspection_id": "insp-123",
			"email":         "notanemail",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("with optional fields", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_send_to_guest", map[string]any{
			"inspection_id": "insp-123",
			"email":         "guest@example.com",
			"name":          "John Doe",
			"message":       "Please complete this inspection",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "John Doe", mock.lastInspSendToGuestInput.Name)
		assert.Equal(t, "Please complete this inspection", mock.lastInspSendToGuestInput.Message)
	})

	// Guest link is a bearer-style capability token; must be redacted from
	// MCP responses to avoid persisting in LLM conversation logs.
	t.Run("guest link is redacted from MCP response", func(t *testing.T) {
		const leakedURL = "https://app.happyco.com/inspect/guest/abc123"
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_send_to_guest", map[string]any{
			"inspection_id": "insp-123",
			"email":         "guest@example.com",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		body := toolText(t, result)
		assert.NotContains(t, body, leakedURL, "guest URL leaked into MCP response — bearer-style capability must be redacted")
		assert.Contains(t, body, "redacted", "response should mark guest link as redacted")

		var link models.InspectionGuestLink
		require.NoError(t, json.Unmarshal([]byte(body), &link))
		assert.Equal(t, "insp-123", link.InspectionID)
		assert.NotEqual(t, leakedURL, link.Link)
	})

	t.Run("redactGuestLink tolerates nil", func(t *testing.T) {
		require.NotPanics(t, func() { redactGuestLink(nil) })
	})
}

func TestToolInspectionRateItem(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_rate_item", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
			"item_name":     "Stove",
			"rating_key":    "condition",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspRateItemInput.InspectionID)
		assert.Equal(t, "Kitchen", mock.lastInspRateItemInput.SectionName)
		assert.Equal(t, "Stove", mock.lastInspRateItemInput.ItemName)
		assert.Equal(t, "condition", mock.lastInspRateItemInput.Rating.Key)
	})

	t.Run("with score", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_rate_item", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
			"item_name":     "Stove",
			"rating_key":    "condition",
			"rating_score":  4.5,
		})
		assert.False(t, result.IsError)
		require.NotNil(t, mock.lastInspRateItemInput.Rating.Score)
		assert.Equal(t, 4.5, *mock.lastInspRateItemInput.Rating.Score)
	})

	t.Run("negative score rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_rate_item", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
			"item_name":     "Stove",
			"rating_key":    "condition",
			"rating_score":  -1.0,
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("missing rating_key rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_rate_item", map[string]any{
			"inspection_id": "insp-123",
			"section_name":  "Kitchen",
			"item_name":     "Stove",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "rating_key")
	})
}

func TestToolInspectionSetHeaderField(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_header_field", map[string]any{
			"inspection_id": "insp-123",
			"label":         "Tenant Name",
			"value":         "John Doe",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "insp-123", mock.lastInspHeaderInput.InspectionID)
		assert.Equal(t, "Tenant Name", mock.lastInspHeaderInput.Label)
		assert.Equal(t, "John Doe", mock.lastInspHeaderInput.Value)
	})

	t.Run("missing label rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "inspection_set_header_field", map[string]any{
			"inspection_id": "insp-123",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "label")
	})
}

func TestToolInspectionMutationAPIError(t *testing.T) {
	mock := &mockClient{err: fmt.Errorf("api_error: HTTP 500")}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "inspection_archive", map[string]any{
		"inspection_id": "insp-123",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, toolText(t, result), "api_error")
}

// ---------------------------------------------------------------------------
// Error Tests
// ---------------------------------------------------------------------------

func TestToolErrorCategories(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		err      error
		wantText string
	}{
		{
			name:     "auth failure via get_account",
			tool:     "get_account",
			err:      fmt.Errorf("auth_failed: invalid credentials"),
			wantText: "auth_failed",
		},
		{
			name:     "not found via get_account",
			tool:     "get_account",
			err:      fmt.Errorf("not_found: account does not exist"),
			wantText: "not_found",
		},
		{
			name:     "api error via get_account",
			tool:     "get_account",
			err:      fmt.Errorf("api_error: HTTP 500"),
			wantText: "api_error",
		},
		{
			name:     "api error via list_work_orders (through semaphore path)",
			tool:     "list_work_orders",
			err:      fmt.Errorf("api_error: connection refused"),
			wantText: "api_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClient{err: tt.err}
			cs := newTestServer(t, mock)

			result := callTool(t, cs, tt.tool, tt.args)
			assert.True(t, result.IsError)
			assert.Contains(t, toolText(t, result), tt.wantText)
		})
	}
}

// ---------------------------------------------------------------------------
// Resource Tests
// ---------------------------------------------------------------------------

func TestResourceAccount(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{
			account: &models.Account{ID: "54522", Name: "Test Account"},
		}
		cs := newTestServer(t, mock)

		result, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{
			URI: "happyco://account",
		})
		require.NoError(t, err)
		require.Len(t, result.Contents, 1)

		var account models.Account
		require.NoError(t, json.Unmarshal([]byte(result.Contents[0].Text), &account))
		assert.Equal(t, "54522", account.ID)
		assert.Equal(t, "Test Account", account.Name)
	})

	t.Run("API error returns user-friendly message", func(t *testing.T) {
		mock := &mockClient{err: fmt.Errorf("api_error: HTTP 500 internal")}
		cs := newTestServer(t, mock)

		_, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{
			URI: "happyco://account",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve account information")
		assert.NotContains(t, err.Error(), "500", "should not leak HTTP status")
	})
}

func TestResourcePropertyDetails(t *testing.T) {
	t.Run("happy path with payload verification", func(t *testing.T) {
		mock := &mockClient{
			properties: []models.Property{
				{ID: "p1", Name: "Sunrise Apartments", CreatedAt: "2025-06-01T00:00:00Z", Address: models.Address{City: "Austin", State: "TX"}},
			},
			propTotal: 1,
			units:     []models.Unit{{ID: "u1"}, {ID: "u2"}, {ID: "u3"}},
			unitTotal: 3,
		}
		cs := newTestServer(t, mock)

		result, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{
			URI: "happyco://properties/p1",
		})
		require.NoError(t, err)
		require.Len(t, result.Contents, 1)

		var parsed map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(result.Contents[0].Text), &parsed))

		var unitCount int
		require.NoError(t, json.Unmarshal(parsed["unit_count"], &unitCount))
		assert.Equal(t, 3, unitCount)

		var name string
		require.NoError(t, json.Unmarshal(parsed["name"], &name))
		assert.Equal(t, "Sunrise Apartments", name)

		var id string
		require.NoError(t, json.Unmarshal(parsed["id"], &id))
		assert.Equal(t, "p1", id)
	})

	t.Run("property not found", func(t *testing.T) {
		mock := &mockClient{properties: nil, propTotal: 0}
		cs := newTestServer(t, mock)

		_, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{
			URI: "happyco://properties/nonexistent",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "property not found")
	})

	t.Run("API error on property fetch", func(t *testing.T) {
		mock := &mockClient{err: fmt.Errorf("api_error: timeout")}
		cs := newTestServer(t, mock)

		_, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{
			URI: "happyco://properties/p1",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve property")
		assert.NotContains(t, err.Error(), "timeout", "should not leak API error details")
	})
}

// ---------------------------------------------------------------------------
// Prompt Tests
// ---------------------------------------------------------------------------

func TestPromptPropertySummary(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	t.Run("happy path", func(t *testing.T) {
		result, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
			Name:      "property_summary",
			Arguments: map[string]string{"property_id": "p1"},
		})
		require.NoError(t, err)
		require.Len(t, result.Messages, 1)
		assert.Equal(t, mcp.Role("user"), result.Messages[0].Role)

		text := result.Messages[0].Content.(*mcp.TextContent).Text
		// Verify property_id is interpolated into the correct tool call instructions
		assert.Contains(t, text, `property_id "p1"`)
		assert.Contains(t, text, "list_units")
		assert.Contains(t, text, "list_work_orders")
		assert.Contains(t, text, `status "OPEN"`)
		// Verify description includes the property ID
		assert.Contains(t, result.Description, "p1")
	})

	t.Run("missing property_id returns error", func(t *testing.T) {
		_, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
			Name:      "property_summary",
			Arguments: map[string]string{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "property_id is required")
	})

	t.Run("invalid property_id returns error", func(t *testing.T) {
		_, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
			Name:      "property_summary",
			Arguments: map[string]string{"property_id": "../../etc"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid characters")
	})
}

func TestPromptMaintenanceReport(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	t.Run("default days_back", func(t *testing.T) {
		result, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
			Name:      "maintenance_report",
			Arguments: map[string]string{"property_id": "p1"},
		})
		require.NoError(t, err)
		text := result.Messages[0].Content.(*mcp.TextContent).Text
		assert.Contains(t, text, "last 30 days")
		assert.Contains(t, text, `property_id "p1"`)
		assert.Contains(t, text, "list_inspections")
		assert.Contains(t, result.Description, "p1")
		assert.Contains(t, result.Description, "30 days")
	})

	t.Run("custom days_back", func(t *testing.T) {
		result, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
			Name:      "maintenance_report",
			Arguments: map[string]string{"property_id": "p1", "days_back": "7"},
		})
		require.NoError(t, err)
		text := result.Messages[0].Content.(*mcp.TextContent).Text
		assert.Contains(t, text, "last 7 days")
		assert.Contains(t, result.Description, "7 days")
	})

	t.Run("missing property_id returns error", func(t *testing.T) {
		_, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
			Name:      "maintenance_report",
			Arguments: map[string]string{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "property_id is required")
	})

	t.Run("invalid days_back returns error", func(t *testing.T) {
		_, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
			Name:      "maintenance_report",
			Arguments: map[string]string{"property_id": "p1", "days_back": "-5"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "days_back must be a positive integer")
	})
}

// ---------------------------------------------------------------------------
// Unit tests for helper functions
// ---------------------------------------------------------------------------

func TestBuildListOpts(t *testing.T) {
	refTime := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	refStr := refTime.Format(time.RFC3339)

	// Use work order statuses as the default for most tests.
	woStatuses := models.ValidWorkOrderStatuses

	tests := []struct {
		name          string
		propertyID    string
		unitID        string
		status        string
		createdAfter  string
		createdBefore string
		limit         int
		statuses      map[string]bool
		wantLocation  string
		wantStatus    []string
		wantAfter     *time.Time
		wantBefore    *time.Time
		wantLimit     int
		wantErr       string
	}{
		{
			name:         "empty input uses defaults",
			statuses:     woStatuses,
			wantLocation: "",
			wantStatus:   nil,
			wantLimit:    0,
		},
		{
			name:         "property_id sets location",
			propertyID:   "prop-123",
			statuses:     woStatuses,
			wantLocation: "prop-123",
		},
		{
			name:         "unit_id takes precedence over property_id",
			propertyID:   "prop-123",
			unitID:       "unit-456",
			statuses:     woStatuses,
			wantLocation: "unit-456",
		},
		{
			name:         "unit_id alone sets location",
			unitID:       "unit-789",
			statuses:     woStatuses,
			wantLocation: "unit-789",
		},
		{
			name:       "status is wrapped in slice",
			status:     "OPEN",
			statuses:   woStatuses,
			wantStatus: []string{"OPEN"},
		},
		{
			name:       "lowercase status normalised",
			status:     "open",
			statuses:   woStatuses,
			wantStatus: []string{"OPEN"},
		},
		{
			name:     "invalid status rejected",
			status:   "INVALID",
			statuses: woStatuses,
			wantErr:  `invalid status "INVALID"`,
		},
		{
			name:     "inspection status rejected for work orders",
			status:   "COMPLETE",
			statuses: woStatuses,
			wantErr:  `invalid status "COMPLETE"`,
		},
		{
			name:       "inspection status accepted for inspections",
			status:     "COMPLETE",
			statuses:   models.ValidInspectionStatuses,
			wantStatus: []string{"COMPLETE"},
		},
		{
			name:         "valid created_after is parsed",
			createdAfter: refStr,
			statuses:     woStatuses,
			wantAfter:    &refTime,
		},
		{
			name:          "valid created_before is parsed",
			createdBefore: refStr,
			statuses:      woStatuses,
			wantBefore:    &refTime,
		},
		{
			name:         "invalid created_after returns error",
			createdAfter: "not-a-date",
			statuses:     woStatuses,
			wantErr:      "created_after must be ISO 8601 format",
		},
		{
			name:          "invalid created_before returns error",
			createdBefore: "2026-13-99",
			statuses:      woStatuses,
			wantErr:       "created_before must be ISO 8601 format",
		},
		{
			name:         "date error does not leak Go internals",
			createdAfter: "bad",
			statuses:     woStatuses,
			wantErr:      "(e.g. 2026-01-15T00:00:00Z or 2026-01-15)",
		},
		{
			name:          "inverted date range rejected",
			createdAfter:  "2026-12-01T00:00:00Z",
			createdBefore: "2026-01-01T00:00:00Z",
			statuses:      woStatuses,
			wantErr:       "created_after must be before created_before",
		},
		{
			name:      "limit is clamped to max",
			limit:     99999,
			statuses:  woStatuses,
			wantLimit: maxLimit,
		},
		{
			name:      "negative limit treated as default",
			limit:     -1,
			statuses:  woStatuses,
			wantLimit: 0,
		},
		{
			name:      "normal limit preserved",
			limit:     50,
			statuses:  woStatuses,
			wantLimit: 50,
		},
		{
			name:       "invalid property_id rejected",
			propertyID: "../../etc/passwd",
			statuses:   woStatuses,
			wantErr:    "property_id contains invalid characters",
		},
		{
			name:     "invalid unit_id rejected",
			unitID:   "unit id with spaces",
			statuses: woStatuses,
			wantErr:  "unit_id contains invalid characters",
		},
		{
			name:         "YYYY-MM-DD date accepted for created_after",
			createdAfter: "2026-01-15",
			statuses:     woStatuses,
			wantAfter:    func() *time.Time { t := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC); return &t }(),
		},
		{
			name:          "YYYY-MM-DD date accepted for created_before",
			createdBefore: "2026-04-01",
			statuses:      woStatuses,
			wantBefore:    func() *time.Time { t := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC); return &t }(),
		},
		{
			name:         "invalid calendar date rejected",
			createdAfter: "2026-02-30",
			statuses:     woStatuses,
			wantErr:      "created_after must be ISO 8601 format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := buildListOpts(tt.propertyID, tt.unitID, tt.status, tt.createdAfter, tt.createdBefore, tt.limit, tt.statuses)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLocation, opts.LocationID)
			assert.Equal(t, tt.wantLimit, opts.Limit)

			if tt.wantStatus != nil {
				assert.Equal(t, tt.wantStatus, opts.Status)
			} else {
				assert.Nil(t, opts.Status)
			}

			if tt.wantAfter != nil {
				require.NotNil(t, opts.CreatedAfter)
				assert.True(t, tt.wantAfter.Equal(*opts.CreatedAfter))
			} else {
				assert.Nil(t, opts.CreatedAfter)
			}

			if tt.wantBefore != nil {
				require.NotNil(t, opts.CreatedBefore)
				assert.True(t, tt.wantBefore.Equal(*opts.CreatedBefore))
			} else {
				assert.Nil(t, opts.CreatedBefore)
			}
		})
	}
}

func TestExtractPropertyID(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{"valid URI", "happyco://properties/12345", "12345"},
		{"valid UUID", "happyco://properties/abc-def-123", "abc-def-123"},
		{"empty string", "", ""},
		{"prefix only", "happyco://properties/", ""},
		{"wrong scheme returns empty", "http://properties/12345", ""},
		{"no property segment", "happyco://account", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPropertyID(tt.uri)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClampLimit(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"zero returns zero (API default)", 0, 0},
		{"negative returns zero", -5, 0},
		{"normal preserved", 100, 100},
		{"at max preserved", maxLimit, maxLimit},
		{"over max clamped", maxLimit + 1, maxLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, clampLimit(tt.input))
		})
	}
}

func TestValidateID(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		wantErr   bool
		errMsg    string
	}{
		{"empty is valid", "id", "", false, ""},
		{"numeric", "id", "12345", false, ""},
		{"alphanumeric", "id", "abc123", false, ""},
		{"UUID-style", "id", "abc-def-123", false, ""},
		{"underscore allowed", "id", "prop_123", false, ""},
		{"path traversal", "property_id", "../../etc", true, "property_id contains invalid characters"},
		{"spaces", "unit_id", "has spaces", true, "unit_id contains invalid characters"},
		{"newlines", "id", "line\nbreak", true, "id contains invalid characters"},
		{"slashes", "id", "a/b/c", true, "id contains invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateID(tt.fieldName, tt.value)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitiseErrorCategory(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"auth error", "auth_failed: HTTP 401", "auth_failed: Authentication failed — check credentials"},
		{"not found", "not_found: entity xyz not found", "not_found: The requested resource was not found"},
		{"invalid input", "invalid_input: missing field", "invalid_input: Invalid input parameters"},
		{"rate limited", "rate_limited: too many requests", "rate_limited: API rate limit exceeded — try again later"},
		{"api error", "api_error: HTTP 500", "api_error: An API error occurred — try again later"},
		{"unknown error", "something unexpected", "api_error: An unexpected error occurred"},
		{"empty string", "", "api_error: An unexpected error occurred"},
		{"graphql error leaks nothing", "api_error: parsing response: invalid JSON at position 42", "api_error: An API error occurred — try again later"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitiseErrorCategory(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToolJSON(t *testing.T) {
	t.Run("valid value", func(t *testing.T) {
		result, _, err := toolJSON(map[string]string{"key": "value"})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)

		var parsed map[string]string
		text := result.Content[0].(*mcp.TextContent).Text
		require.NoError(t, json.Unmarshal([]byte(text), &parsed))
		assert.Equal(t, "value", parsed["key"])
	})

	t.Run("unmarshalable value returns error", func(t *testing.T) {
		result, _, err := toolJSON(make(chan int))
		require.NoError(t, err) // no Go-level error; error is in the result
		require.NotNil(t, result)
		assert.True(t, result.IsError)
		text := result.Content[0].(*mcp.TextContent).Text
		assert.NotContains(t, text, "chan int", "error should not leak Go type info")
	})
}

func TestEmptyIfNil(t *testing.T) {
	t.Run("nil returns empty slice", func(t *testing.T) {
		var s []string
		result := emptyIfNil(s)
		require.NotNil(t, result)
		assert.Len(t, result, 0)
		// Verify JSON serialisation
		data, _ := json.Marshal(result)
		assert.Equal(t, "[]", string(data))
	})

	t.Run("non-nil returned as-is", func(t *testing.T) {
		s := []string{"a", "b"}
		result := emptyIfNil(s)
		assert.Equal(t, s, result)
	})
}

func TestRequirePropertyID(t *testing.T) {
	t.Run("valid ID", func(t *testing.T) {
		id, err := requirePropertyID(map[string]string{"property_id": "p1"})
		require.NoError(t, err)
		assert.Equal(t, "p1", id)
	})

	t.Run("missing returns error", func(t *testing.T) {
		_, err := requirePropertyID(map[string]string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "property_id is required")
	})

	t.Run("invalid characters returns error", func(t *testing.T) {
		_, err := requirePropertyID(map[string]string{"property_id": "../bad"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid characters")
	})
}

func TestAcquireSem(t *testing.T) {
	// Use a local semaphore — acquireSem/releaseSem accept the channel as a
	// parameter, so no global mutation needed.
	testSem := make(chan struct{}, 3)

	t.Run("successful acquire and release", func(t *testing.T) {
		err := acquireSem(context.Background(), testSem)
		require.NoError(t, err)
		assert.Equal(t, 1, len(testSem), "semaphore should have 1 slot occupied")
		releaseSem(testSem)
		assert.Equal(t, 0, len(testSem), "semaphore should be empty after release")
	})

	t.Run("cancelled context returns error", func(t *testing.T) {
		// Fill all slots
		for i := 0; i < cap(testSem); i++ {
			testSem <- struct{}{}
		}
		defer func() {
			for i := 0; i < cap(testSem); i++ {
				<-testSem
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := acquireSem(ctx, testSem)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})
}

func TestToolInputError(t *testing.T) {
	result := toolInputError("field is required")
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Equal(t, "invalid_input: field is required", text)
}

// ---------------------------------------------------------------------------
// Debug wrapper test
// ---------------------------------------------------------------------------

// newTestServerWithDebug creates an MCP server with debug=true to exercise
// the wrapTool debug logging path.
func newTestServerWithDebug(t *testing.T, mock *mockClient) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	server := mcp.NewServer(
		&mcp.Implementation{Name: "hppymcp-test", Version: "test"},
		&mcp.ServerOptions{Instructions: "test"},
	)
	registerTools(server, mock, true) // debug enabled
	registerResources(server, mock)
	registerPrompts(server)

	ct, st := mcp.NewInMemoryTransports()
	_, err := server.Connect(ctx, st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(
		&mcp.Implementation{Name: "test-client", Version: "test"},
		nil,
	)
	cs, err := client.Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { cs.Close() })
	return cs
}

func TestWrapToolDebugModeReturnsCorrectResults(t *testing.T) {
	mock := &mockClient{
		account: &models.Account{ID: "54522", Name: "Test Account"},
	}
	cs := newTestServerWithDebug(t, mock)

	// Verify the debug wrapper does not interfere with normal results
	result := callTool(t, cs, "get_account", nil)
	assert.False(t, result.IsError)

	var account models.Account
	require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &account))
	assert.Equal(t, "54522", account.ID)
	assert.Equal(t, "Test Account", account.Name)
}

func TestWrapToolDebugModePassesErrorsThrough(t *testing.T) {
	mock := &mockClient{err: fmt.Errorf("api_error: something broke")}
	cs := newTestServerWithDebug(t, mock)

	result := callTool(t, cs, "get_account", nil)
	assert.True(t, result.IsError)
	assert.Contains(t, toolText(t, result), "api_error")
}

// ---------------------------------------------------------------------------
// Project Mutation Tool Tests
// ---------------------------------------------------------------------------

func TestToolProjectCreate(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{
			mutatedProject: &models.Project{ID: "proj-new", Status: "PLANNED", Priority: "URGENT"},
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_create", map[string]any{
			"template_id": "tmpl-1",
			"location_id": "loc-123",
			"start_at":    "2026-05-01T00:00:00Z",
			"priority":    "URGENT",
			"notes":       "Test project",
		})
		assert.False(t, result.IsError)

		var proj models.Project
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &proj))
		assert.Equal(t, "proj-new", proj.ID)
		assert.Equal(t, "URGENT", proj.Priority)

		// Verify input was passed correctly
		assert.Equal(t, "tmpl-1", mock.lastProjCreateInput.ProjectTemplateID)
		assert.Equal(t, "loc-123", mock.lastProjCreateInput.LocationID)
		assert.Equal(t, "Test project", mock.lastProjCreateInput.Notes)
	})

	t.Run("missing template_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_create", map[string]any{
			"location_id": "loc-123",
			"start_at":    "2026-05-01T00:00:00Z",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "template_id")
	})

	t.Run("missing location_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_create", map[string]any{
			"template_id": "tmpl-1",
			"start_at":    "2026-05-01T00:00:00Z",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "location_id")
	})

	t.Run("missing start_at returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_create", map[string]any{
			"template_id": "tmpl-1",
			"location_id": "loc-123",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "start_at")
	})

	t.Run("invalid priority rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_create", map[string]any{
			"template_id": "tmpl-1",
			"location_id": "loc-123",
			"start_at":    "2026-05-01T00:00:00Z",
			"priority":    "CRITICAL",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "priority must be NORMAL or URGENT")
	})

	t.Run("lowercase priority normalised", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_create", map[string]any{
			"template_id": "tmpl-1",
			"location_id": "loc-123",
			"start_at":    "2026-05-01T00:00:00Z",
			"priority":    "urgent",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "URGENT", mock.lastProjCreateInput.Priority)
	})

	t.Run("invalid ID rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_create", map[string]any{
			"template_id": "../../etc/passwd",
			"location_id": "loc-123",
			"start_at":    "2026-05-01T00:00:00Z",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolProjectSetAssignee(t *testing.T) {
	t.Run("happy path set assignee", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		assigneeID := "user-456"
		result := callTool(t, cs, "project_set_assignee", map[string]any{
			"project_id":  "proj-1",
			"assignee_id": assigneeID,
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "proj-1", mock.lastProjAssignInput.ProjectID)
		require.NotNil(t, mock.lastProjAssignInput.AssigneeID)
		assert.Equal(t, "user-456", *mock.lastProjAssignInput.AssigneeID)
	})

	t.Run("missing project_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_set_assignee", map[string]any{})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "project_id")
	})
}

func TestToolProjectSetNotes(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_set_notes", map[string]any{
			"project_id": "proj-1",
			"notes":      "Updated notes",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "proj-1", mock.lastMutationID)
		assert.Equal(t, "Updated notes", mock.lastStringValue)
	})

	t.Run("missing notes returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_set_notes", map[string]any{
			"project_id": "proj-1",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "notes")
	})
}

func TestToolProjectSetPriority(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_set_priority", map[string]any{
			"project_id": "proj-1",
			"priority":   "URGENT",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "proj-1", mock.lastMutationID)
	})

	t.Run("invalid priority rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_set_priority", map[string]any{
			"project_id": "proj-1",
			"priority":   "HIGH",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "priority must be NORMAL or URGENT")
	})
}

func TestToolProjectSetOnHold(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "project_set_on_hold", map[string]any{
		"project_id": "proj-1",
		"on_hold":    true,
	})
	assert.False(t, result.IsError)
	assert.Equal(t, "proj-1", mock.lastMutationID)
}

func TestToolProjectSetDueAt(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_set_due_at", map[string]any{
			"project_id": "proj-1",
			"due_at":     "2026-06-01T00:00:00Z",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "proj-1", mock.lastMutationID)
	})

	t.Run("invalid timestamp rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_set_due_at", map[string]any{
			"project_id": "proj-1",
			"due_at":     "not-a-date",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "RFC3339")
	})
}

func TestToolProjectSetStartAt(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "project_set_start_at", map[string]any{
		"project_id": "proj-1",
		"start_at":   "2026-05-01T00:00:00Z",
	})
	assert.False(t, result.IsError)
	assert.Equal(t, "proj-1", mock.lastMutationID)
}

func TestToolProjectSetAvailabilityTargetAt(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_set_availability_target_at", map[string]any{
			"project_id":             "proj-1",
			"availability_target_at": "2026-05-15T00:00:00Z",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "proj-1", mock.lastMutationID)
	})

	t.Run("missing project_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "project_set_availability_target_at", map[string]any{
			"availability_target_at": "2026-05-15T00:00:00Z",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "project_id")
	})
}

func TestToolProjectMutationAPIError(t *testing.T) {
	mock := &mockClient{err: fmt.Errorf("api_error: server error")}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "project_create", map[string]any{
		"template_id": "tmpl-1",
		"location_id": "loc-123",
		"start_at":    "2026-05-01T00:00:00Z",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, toolText(t, result), "api_error")
}

// ---------------------------------------------------------------------------
// Account Mutation Tool Tests
// ---------------------------------------------------------------------------

func TestToolUserCreate(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{
			mutatedUser: &models.User{ID: "user-new", Name: "Jane Doe", Email: "jane@example.com"},
		}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "user_create", map[string]any{
			"account_id": "acct-1",
			"email":      "jane@example.com",
			"name":       "Jane Doe",
			"role_id":    "role1,role2",
		})
		assert.False(t, result.IsError)

		var user models.User
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &user))
		assert.Equal(t, "user-new", user.ID)
		assert.Equal(t, "Jane Doe", user.Name)

		// Verify input passed correctly
		assert.Equal(t, "acct-1", mock.lastUserCreateInput.AccountID)
		assert.Equal(t, "jane@example.com", mock.lastUserCreateInput.Email)
		assert.Equal(t, []string{"role1", "role2"}, mock.lastUserCreateInput.RoleID)
	})

	t.Run("missing account_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "user_create", map[string]any{
			"email": "jane@example.com",
			"name":  "Jane Doe",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "account_id")
	})

	t.Run("missing email returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "user_create", map[string]any{
			"account_id": "acct-1",
			"name":       "Jane Doe",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "email")
	})

	t.Run("invalid email rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "user_create", map[string]any{
			"account_id": "acct-1",
			"email":      "not-an-email",
			"name":       "Jane Doe",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})

	t.Run("missing name returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "user_create", map[string]any{
			"account_id": "acct-1",
			"email":      "jane@example.com",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "name")
	})

	t.Run("invalid ID rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "user_create", map[string]any{
			"account_id": "../../etc/passwd",
			"email":      "jane@example.com",
			"name":       "Jane Doe",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolUserSetEmail(t *testing.T) {
	t.Run("happy path forwards email", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "user_set_email", map[string]any{
			"user_id": "user-1",
			"email":   "new@example.com",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "user-1", mock.lastMutationID)
		assert.Equal(t, "new@example.com", mock.lastStringValue)
	})

	t.Run("missing user_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "user_set_email", map[string]any{
			"email": "new@example.com",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "user_id")
	})

	t.Run("invalid email rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "user_set_email", map[string]any{
			"user_id": "user-1",
			"email":   "bad",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolUserSetName(t *testing.T) {
	t.Run("happy path forwards name", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "user_set_name", map[string]any{
			"user_id": "user-1",
			"name":    "New Name",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "user-1", mock.lastMutationID)
		assert.Equal(t, "New Name", mock.lastStringValue)
	})

	t.Run("missing name returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "user_set_name", map[string]any{
			"user_id": "user-1",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "name")
	})
}

func TestToolUserSetShortName(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "user_set_short_name", map[string]any{
		"user_id":    "user-1",
		"short_name": "Jay",
	})
	assert.False(t, result.IsError)
	assert.Equal(t, "user-1", mock.lastMutationID)
}

func TestToolUserSetPhone(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "user_set_phone", map[string]any{
		"user_id": "user-1",
		"phone":   "+1-555-0100",
	})
	assert.False(t, result.IsError)
	assert.Equal(t, "user-1", mock.lastMutationID)
}

func TestToolMembershipCreate(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "membership_create", map[string]any{
			"account_id": "acct-1",
			"user_id":    "user-1",
			"role_id":    "role-a",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "acct-1", mock.lastMembershipInput.AccountID)
		assert.Equal(t, "user-1", mock.lastMembershipInput.UserID)
		assert.Equal(t, []string{"role-a"}, mock.lastMembershipInput.RoleID)
	})

	t.Run("missing account_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "membership_create", map[string]any{
			"user_id": "user-1",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "account_id")
	})

	t.Run("missing user_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "membership_create", map[string]any{
			"account_id": "acct-1",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "user_id")
	})
}

func TestToolMembershipActivate(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "membership_activate", map[string]any{
		"account_id": "acct-1",
		"user_id":    "user-1",
	})
	assert.False(t, result.IsError)
	assert.Equal(t, "user-1", mock.lastMutationID)
}

func TestToolMembershipDeactivate(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "membership_deactivate", map[string]any{
		"account_id": "acct-1",
		"user_id":    "user-1",
	})
	assert.False(t, result.IsError)
	assert.Equal(t, "user-1", mock.lastMutationID)
}

func TestToolMembershipSetRoles(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "membership_set_roles", map[string]any{
			"account_id": "acct-1",
			"user_id":    "user-1",
			"role_id":    "role-a,role-b",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "acct-1", mock.lastMembershipRoleInput.AccountID)
		assert.Equal(t, []string{"role-a", "role-b"}, mock.lastMembershipRoleInput.RoleID)
	})

	t.Run("missing account_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "membership_set_roles", map[string]any{
			"user_id": "user-1",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "account_id")
	})

	t.Run("missing user_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "membership_set_roles", map[string]any{
			"account_id": "acct-1",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "user_id")
	})
}

func TestToolPropertyGrantAccess(t *testing.T) {
	t.Run("happy path with multiple users", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "property_grant_access", map[string]any{
			"property_id": "prop-1",
			"user_id":     "user-1,user-2",
		})
		assert.False(t, result.IsError)
		assert.Equal(t, "prop-1", mock.lastPropertyAccessInput.PropertyID)
		assert.Equal(t, []string{"user-1", "user-2"}, mock.lastPropertyAccessInput.UserID)
	})

	t.Run("missing property_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "property_grant_access", map[string]any{
			"user_id": "user-1",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "property_id")
	})

	t.Run("missing user_id returns error", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "property_grant_access", map[string]any{
			"property_id": "prop-1",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "user_id")
	})

	t.Run("invalid user_id rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "property_grant_access", map[string]any{
			"property_id": "prop-1",
			"user_id":     "../../../etc/passwd",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolPropertyRevokeAccess(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "property_revoke_access", map[string]any{
		"property_id": "prop-1",
		"user_id":     "user-1",
	})
	assert.False(t, result.IsError)
	assert.Equal(t, "prop-1", mock.lastMutationID)
}

func TestToolPropertySetAccountWideAccess(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "property_set_account_wide_access", map[string]any{
		"property_id":         "prop-1",
		"account_wide_access": true,
	})
	assert.False(t, result.IsError)
	assert.Equal(t, "prop-1", mock.lastMutationID)
}

func TestToolUserGrantPropertyAccess(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "user_grant_property_access", map[string]any{
		"user_id":     "user-1",
		"property_id": "prop-a,prop-b",
	})
	assert.False(t, result.IsError)
	assert.Equal(t, "user-1", mock.lastUserAccessInput.UserID)
	assert.Equal(t, []string{"prop-a", "prop-b"}, mock.lastUserAccessInput.PropertyID)
}

func TestToolUserRevokePropertyAccess(t *testing.T) {
	mock := &mockClient{}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "user_revoke_property_access", map[string]any{
		"user_id":     "user-1",
		"property_id": "prop-a",
	})
	assert.False(t, result.IsError)
	assert.Equal(t, "user-1", mock.lastMutationID)
}

func TestToolAccountMutationAPIError(t *testing.T) {
	mock := &mockClient{err: fmt.Errorf("api_error: server error")}
	cs := newTestServer(t, mock)

	result := callTool(t, cs, "user_create", map[string]any{
		"account_id": "acct-1",
		"email":      "test@example.com",
		"name":       "Test User",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, toolText(t, result), "api_error")
}

// ---------------------------------------------------------------------------
// Role mutation tool tests
// ---------------------------------------------------------------------------

func TestToolRoleCreate(t *testing.T) {
	t.Run("success with grant and revoke", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_create", map[string]any{
			"account_id":         "acct-1",
			"name":               "Inspector",
			"description":        "Can perform inspections",
			"permissions_grant":  "inspection:inspection.create,inspection:inspection.view",
			"permissions_revoke": "inspection:inspection.delete",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		assert.Equal(t, "acct-1", mock.lastRoleCreateInput.AccountID)
		assert.Equal(t, "Inspector", mock.lastRoleCreateInput.Name)
		assert.Equal(t, "Can perform inspections", mock.lastRoleCreateInput.Description)
		assert.Equal(t, []string{"inspection:inspection.create", "inspection:inspection.view"}, mock.lastRoleCreateInput.Permissions.Grant)
		assert.Equal(t, []string{"inspection:inspection.delete"}, mock.lastRoleCreateInput.Permissions.Revoke)

		var role models.Role
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &role))
		assert.Equal(t, "role-1", role.ID)
	})

	t.Run("missing account_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_create", map[string]any{
			"name":              "Inspector",
			"permissions_grant": "test:action",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "account_id")
	})

	t.Run("missing name", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_create", map[string]any{
			"account_id":        "acct-1",
			"permissions_grant": "test:action",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "name")
	})

	t.Run("missing permissions", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_create", map[string]any{
			"account_id": "acct-1",
			"name":       "Inspector",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "permissions_grant")
	})

	t.Run("invalid account_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_create", map[string]any{
			"account_id":        "bad id!",
			"name":              "Inspector",
			"permissions_grant": "test:action",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid_input")
	})
}

func TestToolRoleSetName(t *testing.T) {
	t.Run("success forwards all fields", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_name", map[string]any{
			"account_id": "acct-1",
			"role_id":    "role-1",
			"name":       "Senior Inspector",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		assert.Equal(t, "acct-1", mock.lastRoleSetNameInput.AccountID)
		assert.Equal(t, "role-1", mock.lastRoleSetNameInput.RoleID)
		assert.Equal(t, "Senior Inspector", mock.lastRoleSetNameInput.Name)

		var role models.Role
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &role))
		assert.Equal(t, "role-1", role.ID)
	})

	t.Run("missing account_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_name", map[string]any{
			"role_id": "role-1",
			"name":    "New Name",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "account_id")
	})

	t.Run("missing role_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_name", map[string]any{
			"account_id": "acct-1",
			"name":       "New Name",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "role_id")
	})

	t.Run("missing name", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_name", map[string]any{
			"account_id": "acct-1",
			"role_id":    "role-1",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "name")
	})
}

func TestToolRoleSetDescription(t *testing.T) {
	t.Run("success forwards description", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_description", map[string]any{
			"account_id":  "acct-1",
			"role_id":     "role-1",
			"description": "Updated description",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		assert.Equal(t, "acct-1", mock.lastRoleSetDescInput.AccountID)
		assert.Equal(t, "role-1", mock.lastRoleSetDescInput.RoleID)
		require.NotNil(t, mock.lastRoleSetDescInput.Description)
		assert.Equal(t, "Updated description", *mock.lastRoleSetDescInput.Description)
	})

	t.Run("null description clears it", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		// Passing null explicitly should forward nil pointer
		result := callTool(t, cs, "role_set_description", map[string]any{
			"account_id":  "acct-1",
			"role_id":     "role-1",
			"description": nil,
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		assert.Equal(t, "role-1", mock.lastRoleSetDescInput.RoleID)
		assert.Nil(t, mock.lastRoleSetDescInput.Description)
	})

	t.Run("missing account_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_description", map[string]any{
			"role_id":     "role-1",
			"description": "test",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "account_id")
	})

	t.Run("missing role_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_description", map[string]any{
			"account_id":  "acct-1",
			"description": "test",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "role_id")
	})
}

func TestToolRoleSetPermissions(t *testing.T) {
	t.Run("success with grant and revoke", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_permissions", map[string]any{
			"account_id":         "acct-1",
			"role_id":            "role-1",
			"permissions_grant":  "task:task.create",
			"permissions_revoke": "task:task.delete",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		assert.Equal(t, "acct-1", mock.lastRolePermInput.AccountID)
		assert.Equal(t, "role-1", mock.lastRolePermInput.RoleID)
		assert.Equal(t, []string{"task:task.create"}, mock.lastRolePermInput.Permissions.Grant)
		assert.Equal(t, []string{"task:task.delete"}, mock.lastRolePermInput.Permissions.Revoke)
	})

	t.Run("grant only", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_permissions", map[string]any{
			"account_id":        "acct-1",
			"role_id":           "role-1",
			"permissions_grant": "task:task.create,task:task.view",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		assert.Equal(t, []string{"task:task.create", "task:task.view"}, mock.lastRolePermInput.Permissions.Grant)
		assert.Nil(t, mock.lastRolePermInput.Permissions.Revoke)
	})

	t.Run("revoke only", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_permissions", map[string]any{
			"account_id":         "acct-1",
			"role_id":            "role-1",
			"permissions_revoke": "task:task.delete",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		assert.Nil(t, mock.lastRolePermInput.Permissions.Grant)
		assert.Equal(t, []string{"task:task.delete"}, mock.lastRolePermInput.Permissions.Revoke)
	})

	t.Run("missing both grant and revoke", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_permissions", map[string]any{
			"account_id": "acct-1",
			"role_id":    "role-1",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "permissions_grant")
	})

	t.Run("missing account_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_permissions", map[string]any{
			"role_id":           "role-1",
			"permissions_grant": "task:task.create",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "account_id")
	})

	t.Run("missing role_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_permissions", map[string]any{
			"account_id":        "acct-1",
			"permissions_grant": "task:task.create",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "role_id")
	})
}

func TestToolRoleMutationAPIError(t *testing.T) {
	t.Run("role_create", func(t *testing.T) {
		mock := &mockClient{err: fmt.Errorf("api_error: server error")}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_create", map[string]any{
			"account_id":        "acct-1",
			"name":              "Test Role",
			"permissions_grant": "test:action",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "api_error")
	})

	t.Run("role_set_permissions", func(t *testing.T) {
		mock := &mockClient{err: fmt.Errorf("api_error: server error")}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "role_set_permissions", map[string]any{
			"account_id":        "acct-1",
			"role_id":           "role-1",
			"permissions_grant": "test:action",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "api_error")
	})
}

// ---------------------------------------------------------------------------
// Webhook mutation tool tests
// ---------------------------------------------------------------------------

func TestToolWebhookCreate(t *testing.T) {
	t.Run("success with all fields", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "ACCOUNT",
			"url":             "https://example.com/webhook",
			"subjects":        "INSPECTIONS,WORK_ORDERS",
			"status":          "ENABLED",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		assert.Equal(t, "acct-1", mock.lastWebhookCreateInput.SubscriberID)
		assert.Equal(t, "ACCOUNT", mock.lastWebhookCreateInput.SubscriberType)
		assert.Equal(t, "https://example.com/webhook", mock.lastWebhookCreateInput.URL)
		assert.Equal(t, []string{"INSPECTIONS", "WORK_ORDERS"}, mock.lastWebhookCreateInput.Subjects)
		assert.Equal(t, "ENABLED", mock.lastWebhookCreateInput.Status)

		var webhook models.Webhook
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &webhook))
		assert.Equal(t, "wh-1", webhook.ID)
	})

	t.Run("minimal required fields", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "ACCOUNT",
			"url":             "https://example.com/hook",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))
		assert.Equal(t, "acct-1", mock.lastWebhookCreateInput.SubscriberID)
		assert.Empty(t, mock.lastWebhookCreateInput.Subjects)
		assert.Empty(t, mock.lastWebhookCreateInput.Status)
	})

	t.Run("missing subscriber_id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_type": "ACCOUNT",
			"url":             "https://example.com/hook",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "subscriber_id")
	})

	t.Run("missing subscriber_type", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id": "acct-1",
			"url":           "https://example.com/hook",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "subscriber_type")
	})

	t.Run("missing url", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "ACCOUNT",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "url")
	})

	t.Run("non-HTTPS url rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "ACCOUNT",
			"url":             "http://example.com/hook",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "HTTPS")
	})

	t.Run("invalid subscriber_type rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "INVALID",
			"url":             "https://example.com/hook",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "subscriber_type")
	})

	t.Run("invalid subject rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "ACCOUNT",
			"url":             "https://example.com/hook",
			"subjects":        "INVALID_SUBJECT",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid webhook subject")
	})

	t.Run("invalid status rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "ACCOUNT",
			"url":             "https://example.com/hook",
			"status":          "INVALID",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "status")
	})

	t.Run("lowercase inputs normalised to uppercase", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "account",
			"url":             "https://example.com/hook",
			"subjects":        "inspections,work_orders",
			"status":          "enabled",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		assert.Equal(t, "ACCOUNT", mock.lastWebhookCreateInput.SubscriberType)
		assert.Equal(t, []string{"INSPECTIONS", "WORK_ORDERS"}, mock.lastWebhookCreateInput.Subjects)
		assert.Equal(t, "ENABLED", mock.lastWebhookCreateInput.Status)
	})

	t.Run("whitespace-only url treated as empty", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "ACCOUNT",
			"url":             "   ",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "url")
	})

	t.Run("url with credentials rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "ACCOUNT",
			"url":             "https://user:pass@example.com/hook",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "credentials")
	})
}

func TestToolWebhookUpdate(t *testing.T) {
	t.Run("success updates url and status", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_update", map[string]any{
			"id":     "wh-1",
			"url":    "https://new-endpoint.com/hook",
			"status": "ENABLED",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		assert.Equal(t, "wh-1", mock.lastWebhookUpdateInput.ID)
		assert.Equal(t, "https://new-endpoint.com/hook", mock.lastWebhookUpdateInput.URL)
		assert.Equal(t, "ENABLED", mock.lastWebhookUpdateInput.Status)

		var webhook models.Webhook
		require.NoError(t, json.Unmarshal([]byte(toolText(t, result)), &webhook))
		assert.Equal(t, "wh-1", webhook.ID)
	})

	t.Run("update subjects only", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_update", map[string]any{
			"id":       "wh-1",
			"subjects": "WORK_ORDERS,VENDORS",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))
		assert.Equal(t, []string{"WORK_ORDERS", "VENDORS"}, mock.lastWebhookUpdateInput.Subjects)
	})

	t.Run("missing id", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_update", map[string]any{
			"url": "https://example.com/hook",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "id")
	})

	t.Run("no update fields", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_update", map[string]any{
			"id": "wh-1",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "at least one")
	})

	t.Run("non-HTTPS url rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_update", map[string]any{
			"id":  "wh-1",
			"url": "http://example.com/hook",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "HTTPS")
	})

	t.Run("invalid subject rejected", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_update", map[string]any{
			"id":       "wh-1",
			"subjects": "BAD_SUBJECT",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "invalid webhook subject")
	})

	t.Run("update url only", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_update", map[string]any{
			"id":  "wh-1",
			"url": "https://new-endpoint.com/hook",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))
		assert.Equal(t, "https://new-endpoint.com/hook", mock.lastWebhookUpdateInput.URL)
		assert.Empty(t, mock.lastWebhookUpdateInput.Status)
		assert.Nil(t, mock.lastWebhookUpdateInput.Subjects)
	})

	t.Run("update status only", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_update", map[string]any{
			"id":     "wh-1",
			"status": "disabled",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))
		assert.Equal(t, "DISABLED", mock.lastWebhookUpdateInput.Status)
		assert.Empty(t, mock.lastWebhookUpdateInput.URL)
	})

	t.Run("whitespace-only url not counted as update", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_update", map[string]any{
			"id":  "wh-1",
			"url": "   ",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "at least one")
	})
}

// TestWebhookSigningSecretRedaction asserts that no MCP tool returning a
// *models.Webhook ever exposes the SigningSecret to the LLM. The mock returns
// "whsec_test123"; if any tool path ever leaks it, the test fails.
func TestWebhookSigningSecretRedaction(t *testing.T) {
	const leakedSecret = "whsec_test123"

	t.Run("webhook_create response is redacted", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "ACCOUNT",
			"url":             "https://example.com/webhook",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		body := toolText(t, result)
		assert.NotContains(t, body, leakedSecret, "webhook_create leaked signing secret to MCP response")
		assert.Contains(t, body, "redacted", "webhook_create response should mark signing secret as redacted")

		var webhook models.Webhook
		require.NoError(t, json.Unmarshal([]byte(body), &webhook))
		assert.NotEqual(t, leakedSecret, webhook.SigningSecret)
	})

	t.Run("webhook_update response is redacted", func(t *testing.T) {
		mock := &mockClient{}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_update", map[string]any{
			"id":     "wh-1",
			"status": "ENABLED",
		})
		require.False(t, result.IsError, "unexpected error: %s", toolText(t, result))

		body := toolText(t, result)
		assert.NotContains(t, body, leakedSecret, "webhook_update leaked signing secret to MCP response")
		assert.Contains(t, body, "redacted", "webhook_update response should mark signing secret as redacted")

		var webhook models.Webhook
		require.NoError(t, json.Unmarshal([]byte(body), &webhook))
		assert.NotEqual(t, leakedSecret, webhook.SigningSecret)
	})

	t.Run("redactWebhookSecret tolerates nil", func(t *testing.T) {
		require.NotPanics(t, func() { redactWebhookSecret(nil) })
	})
}

func TestToolWebhookMutationAPIError(t *testing.T) {
	t.Run("webhook_create", func(t *testing.T) {
		mock := &mockClient{err: fmt.Errorf("api_error: server error")}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_create", map[string]any{
			"subscriber_id":   "acct-1",
			"subscriber_type": "ACCOUNT",
			"url":             "https://example.com/hook",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "api_error")
	})

	t.Run("webhook_update", func(t *testing.T) {
		mock := &mockClient{err: fmt.Errorf("api_error: server error")}
		cs := newTestServer(t, mock)

		result := callTool(t, cs, "webhook_update", map[string]any{
			"id":     "wh-1",
			"status": "ENABLED",
		})
		assert.True(t, result.IsError)
		assert.Contains(t, toolText(t, result), "api_error")
	})
}
