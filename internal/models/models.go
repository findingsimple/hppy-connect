package models

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Valid status values per domain.
var (
	ValidWorkOrderStatuses  = map[string]bool{"OPEN": true, "ON_HOLD": true, "COMPLETED": true}
	ValidInspectionStatuses = map[string]bool{"COMPLETE": true, "EXPIRED": true, "INCOMPLETE": true, "SCHEDULED": true}
)

// ValidateStatus checks that a status value is allowed, normalises it to uppercase,
// and returns the validated status slice. Returns an error listing valid options if invalid.
func ValidateStatus(status string, validStatuses map[string]bool) ([]string, error) {
	if status == "" {
		return nil, nil
	}
	upper := strings.ToUpper(status)
	if !validStatuses[upper] {
		allowed := make([]string, 0, len(validStatuses))
		for k := range validStatuses {
			allowed = append(allowed, k)
		}
		sort.Strings(allowed)
		return nil, fmt.Errorf("invalid status %q — must be one of: %s", status, strings.Join(allowed, ", "))
	}
	return []string{upper}, nil
}

// ValidateDateRange checks that after is strictly before before, if both are set.
// Equal dates are rejected since they represent an empty range.
func ValidateDateRange(after, before *time.Time) error {
	if after != nil && before != nil && !after.Before(*before) {
		return fmt.Errorf("created_after must be before created_before")
	}
	return nil
}

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
