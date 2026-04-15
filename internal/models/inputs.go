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
