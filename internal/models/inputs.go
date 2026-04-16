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

// InspectionAddItemPhotoResult is the multi-field response from addItemPhoto.
type InspectionAddItemPhotoResult struct {
	Inspection      Inspection      `json:"inspection"`
	InspectionPhoto InspectionPhoto `json:"inspectionPhoto"`
	SignedURL       string          `json:"signedURL"`
}
