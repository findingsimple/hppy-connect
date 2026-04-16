package models

// --- Work Order Inputs ---

// WorkOrderCreateInput is the input for creating a new work order.
type WorkOrderCreateInput struct {
	LocationID          string           `json:"locationId"`
	Description         string           `json:"description,omitempty"`
	Priority            string           `json:"priority,omitempty"`
	Status              string           `json:"status,omitempty"`
	SubStatus           string           `json:"subStatus,omitempty"`
	Type                string           `json:"type,omitempty"`
	ScheduledFor        string           `json:"scheduledFor,omitempty"`
	EntryNotes          string           `json:"entryNotes,omitempty"`
	PermissionToEnter   *bool            `json:"permissionToEnter,omitempty"`
	Assignee            *AssignableInput `json:"assignee,omitempty"`
	WorkCategoryID      string           `json:"workCategoryId,omitempty"`
	ProjectStageID      string           `json:"projectStageId,omitempty"`
	ReportingResidentID string           `json:"reportingResidentId,omitempty"`
	ClientCreatedAt     string           `json:"clientCreatedAt,omitempty"`
}

// AssignableInput is the shared input for assigning a user or vendor.
type AssignableInput struct {
	AssigneeID   string `json:"assigneeId"`
	AssigneeType string `json:"assigneeType"`
}

// WorkOrderSetStatusAndSubStatusInput sets both status and sub-status.
type WorkOrderSetStatusAndSubStatusInput struct {
	WorkOrderID string                  `json:"workOrderId"`
	Status      WorkOrderStatusInput    `json:"Status"`
	SubStatus   WorkOrderSubStatusInput `json:"SubStatus"`
}

// WorkOrderStatusInput is the nested status input for setStatusAndSubStatus.
type WorkOrderStatusInput struct {
	Status            string `json:"status"`
	Comment           string `json:"comment,omitempty"`
	PhotoID           string `json:"photoId,omitempty"`
	UpdateCompletedAt *bool  `json:"updateCompletedAt,omitempty"`
}

// WorkOrderSubStatusInput is the nested sub-status input for setStatusAndSubStatus.
type WorkOrderSubStatusInput struct {
	SubStatus string `json:"subStatus"`
	Reason    string `json:"reason,omitempty"`
	Comment   string `json:"comment,omitempty"`
}

// WorkOrderSetAssigneeInput sets the work order assignee using the new AssignableInput pattern.
type WorkOrderSetAssigneeInput struct {
	WorkOrderID string          `json:"workOrderId"`
	Assignee    AssignableInput `json:"assignee"`
}

// WorkOrderAddAttachmentInput adds an attachment to a work order.
type WorkOrderAddAttachmentInput struct {
	WorkOrderID string `json:"workOrderId"`
	FileName    string `json:"fileName"`
	MimeType    string `json:"mimeType"`
	Size        *int   `json:"size,omitempty"`
}

// --- Inspection Inputs ---

// InspectionCreateInput is the input for creating a new inspection.
type InspectionCreateInput struct {
	LocationID   string `json:"locationId"`
	TemplateID   string `json:"templateId"`
	ScheduledFor string `json:"scheduledFor"`
	AssignedToID string `json:"assignedToID,omitempty"`
	DueBy        string `json:"dueBy,omitempty"`
	Expires      *bool  `json:"expires,omitempty"`
}

// InspectionSetAssigneeInput sets the inspection assignee.
type InspectionSetAssigneeInput struct {
	InspectionID string `json:"inspectionId"`
	UserID       string `json:"userId"`
}

// InspectionSetDueByInput sets the due date for an inspection.
type InspectionSetDueByInput struct {
	InspectionID string `json:"inspectionId"`
	DueBy        string `json:"dueBy"`
	Expires      bool   `json:"expires"`
}

// InspectionSetHeaderFieldInput updates a header field on an inspection.
type InspectionSetHeaderFieldInput struct {
	InspectionID string `json:"inspectionId"`
	Label        string `json:"label"`
	Value        string `json:"value,omitempty"`
}

// InspectionSetFooterFieldInput updates a footer field on an inspection.
type InspectionSetFooterFieldInput struct {
	InspectionID string `json:"inspectionId"`
	Label        string `json:"label"`
	Value        string `json:"value,omitempty"`
}

// InspectionSetItemNotesInput sets notes on an inspection item.
type InspectionSetItemNotesInput struct {
	InspectionID string `json:"inspectionId"`
	SectionName  string `json:"sectionName"`
	ItemName     string `json:"itemName"`
	Notes        string `json:"notes,omitempty"`
}

// InspectionRatingInput represents a single rating on an inspection item.
type InspectionRatingInput struct {
	Key   string   `json:"key"`
	Score *float64 `json:"score,omitempty"`
	Value string   `json:"value,omitempty"`
}

// InspectionRateItemInput rates an item in an inspection.
type InspectionRateItemInput struct {
	InspectionID string                `json:"inspectionId"`
	SectionName  string                `json:"sectionName"`
	ItemName     string                `json:"itemName"`
	Rating       InspectionRatingInput `json:"rating"`
}

// InspectionAddSectionInput adds a section to an inspection.
type InspectionAddSectionInput struct {
	InspectionID string `json:"inspectionId"`
	Name         string `json:"name"`
}

// InspectionDeleteSectionInput deletes a section from an inspection.
type InspectionDeleteSectionInput struct {
	InspectionID string `json:"inspectionId"`
	SectionName  string `json:"sectionName"`
}

// InspectionDuplicateSectionInput duplicates a section in an inspection.
type InspectionDuplicateSectionInput struct {
	InspectionID string `json:"inspectionId"`
	SectionName  string `json:"sectionName"`
}

// InspectionRenameSectionInput renames a section in an inspection.
type InspectionRenameSectionInput struct {
	InspectionID   string `json:"inspectionId"`
	SectionName    string `json:"sectionName"`
	NewSectionName string `json:"newSectionName"`
}

// InspectionAddItemInput adds an item to a section in an inspection.
type InspectionAddItemInput struct {
	InspectionID  string `json:"inspectionId"`
	SectionName   string `json:"sectionName"`
	Name          string `json:"name"`
	RatingGroupID string `json:"ratingGroupId"`
	Info          string `json:"info,omitempty"`
}

// InspectionDeleteItemInput deletes an item from a section.
type InspectionDeleteItemInput struct {
	InspectionID string `json:"inspectionId"`
	SectionName  string `json:"sectionName"`
	ItemName     string `json:"itemName"`
}

// InspectionAddItemPhotoInput adds a photo to an inspection item.
type InspectionAddItemPhotoInput struct {
	InspectionID string `json:"inspectionID"`
	SectionName  string `json:"sectionName"`
	ItemName     string `json:"itemName"`
	MimeType     string `json:"mimeType"`
	Size         *int   `json:"size,omitempty"`
}

// InspectionRemoveItemPhotoInput removes a photo from an inspection item.
type InspectionRemoveItemPhotoInput struct {
	InspectionID string `json:"inspectionId"`
	PhotoID      string `json:"photoId"`
	SectionName  string `json:"sectionName"`
	ItemName     string `json:"itemName"`
}

// InspectionMoveItemPhotoInput moves a photo between items.
type InspectionMoveItemPhotoInput struct {
	InspectionID    string `json:"inspectionId"`
	PhotoID         string `json:"photoId"`
	FromSectionName string `json:"fromSectionName"`
	FromItemName    string `json:"fromItemName"`
	ToSectionName   string `json:"toSectionName"`
	ToItemName      string `json:"toItemName"`
}

// InspectionSendToGuestInput sends an inspection to a guest via email.
type InspectionSendToGuestInput struct {
	InspectionID string `json:"inspectionId"`
	Email        string `json:"email"`
	Name         string `json:"name,omitempty"`
	Message      string `json:"message,omitempty"`
	DueDate      string `json:"dueDate,omitempty"`
	Expires      *bool  `json:"expires,omitempty"`
}

// --- Project Inputs ---

// ProjectCreateInput is the input for creating a new project from a template.
type ProjectCreateInput struct {
	ProjectTemplateID    string `json:"projectTemplateId"`
	LocationID           string `json:"locationId"`
	StartAt              string `json:"startAt"`
	AssigneeID           string `json:"assigneeId,omitempty"`
	Priority             string `json:"priority,omitempty"`
	DueAt                string `json:"dueAt,omitempty"`
	AvailabilityTargetAt string `json:"availabilityTargetAt,omitempty"`
	Notes                string `json:"notes,omitempty"`
}

// ProjectSetAssigneeInput sets the project assignee. AssigneeID may be empty to unassign.
type ProjectSetAssigneeInput struct {
	ProjectID  string  `json:"projectId"`
	AssigneeID *string `json:"assigneeId"`
}

// --- User Inputs ---

// UserCreateInput is the input for creating a new user in an account.
type UserCreateInput struct {
	AccountID string   `json:"accountId"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	RoleID    []string `json:"roleId,omitempty"`
	ShortName string   `json:"shortName,omitempty"`
	Phone     string   `json:"phone,omitempty"`
	Message   string   `json:"message,omitempty"`
}

// --- Membership Inputs ---

// AccountMembershipCreateInput is the input for creating a new membership.
type AccountMembershipCreateInput struct {
	AccountID string   `json:"accountId"`
	UserID    string   `json:"userId"`
	RoleID    []string `json:"roleId,omitempty"`
}

// AccountMembershipActivateInput is the input for activating a membership.
type AccountMembershipActivateInput struct {
	AccountID string `json:"accountId"`
	UserID    string `json:"userId"`
}

// AccountMembershipDeactivateInput is the input for deactivating a membership.
type AccountMembershipDeactivateInput struct {
	AccountID string `json:"accountId"`
	UserID    string `json:"userId"`
}

// AccountMembershipSetRolesInput is the input for setting membership roles.
type AccountMembershipSetRolesInput struct {
	AccountID string   `json:"accountId"`
	UserID    string   `json:"userId"`
	RoleID    []string `json:"roleId,omitempty"`
}

// --- Property Access Inputs ---

// PropertyGrantUserAccessInput grants users access to a property.
type PropertyGrantUserAccessInput struct {
	PropertyID string   `json:"propertyId"`
	UserID     []string `json:"userId"`
}

// PropertyRevokeUserAccessInput revokes user access from a property.
type PropertyRevokeUserAccessInput struct {
	PropertyID string   `json:"propertyId"`
	UserID     []string `json:"userId"`
}

// PropertySetAccountWideAccessInput sets account-wide access on a property.
type PropertySetAccountWideAccessInput struct {
	PropertyID        string `json:"propertyId"`
	AccountWideAccess bool   `json:"accountWideAccess"`
}

// UserGrantPropertyAccessInput grants a user access to properties.
type UserGrantPropertyAccessInput struct {
	UserID     string   `json:"userId"`
	PropertyID []string `json:"propertyId"`
}

// UserRevokePropertyAccessInput revokes property access from a user.
type UserRevokePropertyAccessInput struct {
	UserID     string   `json:"userId"`
	PropertyID []string `json:"propertyId"`
}

// --- Role Inputs ---

// RoleCreateInput is the input for creating a new role in an account.
type RoleCreateInput struct {
	AccountID   string           `json:"accountId"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Permissions PermissionsInput `json:"permissions"`
}

// PermissionsInput holds the grant and revoke permission actions.
type PermissionsInput struct {
	Grant  []string `json:"grant,omitempty"`
	Revoke []string `json:"revoke,omitempty"`
}

// RoleSetNameInput is the input for updating a role's name.
type RoleSetNameInput struct {
	AccountID string `json:"accountId"`
	RoleID    string `json:"roleId"`
	Name      string `json:"name"`
}

// RoleSetDescriptionInput is the input for updating a role's description.
type RoleSetDescriptionInput struct {
	AccountID   string  `json:"accountId"`
	RoleID      string  `json:"roleId"`
	Description *string `json:"description"`
}

// RoleSetPermissionsInput is the input for updating a role's permissions.
type RoleSetPermissionsInput struct {
	AccountID   string           `json:"accountId"`
	RoleID      string           `json:"roleId"`
	Permissions PermissionsInput `json:"permissions"`
}

// InspectionAddItemPhotoResult is the multi-field response from addItemPhoto.
type InspectionAddItemPhotoResult struct {
	Inspection      Inspection      `json:"inspection"`
	InspectionPhoto InspectionPhoto `json:"inspectionPhoto"`
	SignedURL       string          `json:"signedURL"`
}

// --- Webhook Inputs ---

// WebhookCreateInput is the input for creating a new webhook.
type WebhookCreateInput struct {
	SubscriberID   string                      `json:"subscriberId"`
	SubscriberType string                      `json:"subscriberType"`
	URL            string                      `json:"url"`
	Subjects       []string                    `json:"subjects,omitempty"`
	Status         string                      `json:"status,omitempty"`
	Headers        []WebhookHeaderInput        `json:"headers,omitempty"`
	RateLimits     []WebhookRateLimitInput     `json:"rateLimits,omitempty"`
	RequestTimeout *WebhookRequestTimeoutInput `json:"requestTimeout,omitempty"`
}

// WebhookUpdateInput is the input for updating an existing webhook.
type WebhookUpdateInput struct {
	ID             string                      `json:"id"`
	URL            string                      `json:"url,omitempty"`
	Status         string                      `json:"status,omitempty"`
	Subjects       []string                    `json:"subjects,omitempty"`
	Headers        []WebhookHeaderInput        `json:"headers,omitempty"`
	RateLimits     []WebhookRateLimitInput     `json:"rateLimits,omitempty"`
	RequestTimeout *WebhookRequestTimeoutInput `json:"requestTimeout,omitempty"`
}

// WebhookHeaderInput represents a key-value header to include in webhook requests.
type WebhookHeaderInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// WebhookRateLimitInput configures a rate limit for the webhook.
type WebhookRateLimitInput struct {
	Period   string `json:"period"`
	Requests int    `json:"requests"`
}

// WebhookRequestTimeoutInput configures the request timeout for the webhook.
type WebhookRequestTimeoutInput struct {
	Seconds int `json:"seconds"`
}
