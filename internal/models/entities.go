package models

// User represents a HappyCo user account.
type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	ShortName string `json:"shortName"`
	Phone     string `json:"phone"`
}

// Role represents a permission role within an account.
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AccountMembership represents a user's membership in an account.
type AccountMembership struct {
	UserID    string `json:"userId"`
	AccountID string `json:"accountId"`
	IsActive  bool   `json:"isActive"`
	Roles     []Role `json:"roles"`
}

// Webhook represents a webhook subscription.
type Webhook struct {
	ID       string   `json:"id"`
	URL      string   `json:"url"`
	Status   string   `json:"status"`
	Subjects []string `json:"subjects"`
}

// Project represents a HappyCo project.
type Project struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Notes    string `json:"notes"`
	StartAt  string `json:"startAt"`
	DueAt    string `json:"dueAt"`
}

// WorkOrderAttachment represents a file attachment on a work order.
type WorkOrderAttachment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaType string `json:"mediaType"`
}

// WorkOrderTime represents a time entry on a work order.
type WorkOrderTime struct {
	ID       string `json:"id"`
	Duration string `json:"duration"`
}

// WorkOrderComment represents a comment on a work order.
type WorkOrderComment struct {
	ID   string `json:"id"`
	Body string `json:"body"`
}

// InspectionPhoto represents a photo attached to an inspection item.
type InspectionPhoto struct {
	ID       string `json:"id"`
	MimeType string `json:"mimeType"`
}

// InspectionGuestLink represents the result of sending an inspection to a guest.
type InspectionGuestLink struct {
	InspectionID string `json:"inspectionId"`
	Link         string `json:"link"`
}

// PropertyAccess represents the result of a property access mutation.
type PropertyAccess struct {
	PropertyID        string `json:"propertyId"`
	AccountWideAccess bool   `json:"accountWideAccess"`
}

// WorkOrderAddAttachmentResult is the multi-field response from addAttachment.
type WorkOrderAddAttachmentResult struct {
	WorkOrder  WorkOrder           `json:"workOrder"`
	Attachment WorkOrderAttachment `json:"attachment"`
	SignedURL  string              `json:"signedURL"`
}
