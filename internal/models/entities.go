package models

// User represents a HappyCo user account.
type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	ShortName string `json:"shortName"`
	Phone     string `json:"phone"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Role represents a permission role within an account.
type Role struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	CreatedAt   string       `json:"createdAt"`
	UpdatedAt   string       `json:"updatedAt"`
	ArchivedAt  string       `json:"archivedAt"`
	Permissions []Permission `json:"permissions"`
}

// Permission represents a single granted permission action.
type Permission struct {
	Action      string `json:"action"`
	Description string `json:"description"`
}

// AccountMembership represents a user's membership in an account.
// The GraphQL API returns nested user/account objects; this struct matches that shape.
type AccountMembership struct {
	IsActive      bool             `json:"isActive"`
	Account       *Account         `json:"account"`
	User          *User            `json:"user"`
	CreatedAt     string           `json:"createdAt"`
	UpdatedAt     string           `json:"updatedAt"`
	InactivatedAt string           `json:"inactivatedAt"`
	Roles         *MembershipRoles `json:"roles"`
}

// MembershipRoles wraps the paginated roles connection on AccountMembership.
type MembershipRoles struct {
	Nodes []Role `json:"nodes"`
}

// Webhook represents a webhook subscription.
type Webhook struct {
	ID             string             `json:"id"`
	URL            string             `json:"url"`
	Status         string             `json:"status"`
	Subjects       []string           `json:"subjects"`
	CreatedAt      string             `json:"createdAt"`
	UpdatedAt      string             `json:"updatedAt"`
	Subscriber     *WebhookSubscriber `json:"subscriber"`
	SigningSecret  string             `json:"signingSecret"`
	RateLimits     []WebhookRateLimit `json:"rateLimits"`
	RequestTimeout *WebhookTimeout    `json:"requestTimeout"`
}

// WebhookSubscriber identifies who owns the webhook.
type WebhookSubscriber struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// WebhookRateLimit represents a rate limit configured on a webhook.
type WebhookRateLimit struct {
	Period   string `json:"period"`
	Requests int    `json:"requests"`
}

// WebhookTimeout represents the request timeout configured on a webhook.
type WebhookTimeout struct {
	Seconds int `json:"seconds"`
}

// Project represents a HappyCo project.
type Project struct {
	ID                   string    `json:"id"`
	Status               string    `json:"status"`
	Priority             string    `json:"priority"`
	Notes                string    `json:"notes"`
	StartAt              string    `json:"start"`
	DueAt                string    `json:"dueAt"`
	AvailabilityTargetAt string    `json:"availabilityTargetAt"`
	HeldAt               string    `json:"heldAt"`
	CreatedAt            string    `json:"createdAt"`
	UpdatedAt            string    `json:"updatedAt"`
	Assignee             *User     `json:"assignee"`
	Location             *Location `json:"location"`
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
// The GraphQL API returns PropertyV2 with "id" (not "propertyId").
type PropertyAccess struct {
	PropertyID        string `json:"id"`
	AccountWideAccess bool   `json:"accountWideAccess"`
}

// WorkOrderAddAttachmentResult is the multi-field response from addAttachment.
type WorkOrderAddAttachmentResult struct {
	WorkOrder  WorkOrder           `json:"workOrder"`
	Attachment WorkOrderAttachment `json:"attachment"`
	SignedURL  string              `json:"signedURL"`
}
