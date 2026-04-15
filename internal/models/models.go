package models

import "time"

type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Address struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	PostalCode string `json:"postalCode"`
}

type Property struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	CreatedAt string  `json:"createdAt"`
	Address   Address `json:"address"`
}

type Unit struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WorkOrder struct {
	ID                string             `json:"id"`
	Status            string             `json:"status"`
	SubStatus         string             `json:"subStatus"`
	Description       string             `json:"description"`
	Summary           string             `json:"summary"`
	Priority          string             `json:"priority"`
	CreatedAt         string             `json:"createdAt"`
	UpdatedAt         string             `json:"updatedAt"`
	ScheduledFor      string             `json:"scheduledFor"`
	AssignedTo        *Assignee          `json:"assignedTo"`
	Location          *Location          `json:"locationV2"`
	InspectionDetails *InspectionDetails `json:"inspectionDetails"`
}

// Assignee represents WorkOrderAssignee (interface) or InspectionAssignee (interface).
// WorkOrderAssignee has possibleTypes: WorkOrderAssigneeUser, WorkOrderAssigneeVendor.
// InspectionAssignee has possibleTypes: InspectionAssigneeUser.
// Queries MUST use inline fragments.
type Assignee struct {
	Typename string `json:"__typename"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Type     string `json:"type"`
}

// Location represents the locationV2 field on WorkOrder/Inspection.
type Location struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Property *Property `json:"property"`
}

type InspectionDetails struct {
	Inspection *Inspection `json:"inspection"`
}

// InspectionTemplate represents the templateV2 field on Inspection.
type InspectionTemplate struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Inspection struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Status         string              `json:"status"`
	StartedAt      string              `json:"startedAt"`
	EndedAt        string              `json:"endedAt"`
	ScheduledFor   string              `json:"scheduledFor"`
	DueBy          string              `json:"dueBy"`
	Score          *float64            `json:"score"`
	PotentialScore *float64            `json:"potentialScore"`
	AssignedTo     *Assignee           `json:"assignedTo"`
	Location       *Location           `json:"locationV2"`
	TemplateV2     *InspectionTemplate `json:"templateV2"`
}

type ListOptions struct {
	Limit         int
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	LocationID    string
	Status        []string
	Search        string
}
