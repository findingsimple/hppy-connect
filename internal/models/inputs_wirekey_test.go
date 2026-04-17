package models

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelInputJSONWireKeyAnomalies pins the JSON tag casing for input
// structs that deliberately diverge from the project's lowercase-id convention.
//
// The HappyCo GraphQL schema is internally inconsistent across input types:
// most fields use lowercase "id" (e.g. "inspectionId", "workOrderId"), but a
// handful require capital "ID" or PascalCase. When these tags drift, the API
// silently drops the variable and the mutation appears to "succeed" with
// missing data — exactly the burn pattern that prompted this test.
//
// If the wire format changes intentionally, update both the struct tag *and*
// the expected key in this test together.
func TestModelInputJSONWireKeyAnomalies(t *testing.T) {
	t.Run("InspectionAddItemPhotoInput uses inspectionID with capital ID", func(t *testing.T) {
		input := InspectionAddItemPhotoInput{
			InspectionID: "insp-1",
			SectionName:  "Bedroom",
			ItemName:     "Walls",
			MimeType:     "image/jpeg",
		}
		got := marshalToMap(t, input)

		_, hasCapital := got["inspectionID"]
		_, hasLowercase := got["inspectionId"]
		assert.True(t, hasCapital, "InspectionAddItemPhotoInput must marshal as 'inspectionID' (capital ID); got keys: %v", keysOf(got))
		assert.False(t, hasLowercase, "InspectionAddItemPhotoInput must NOT marshal as 'inspectionId' (lowercase) — the API silently drops it; got keys: %v", keysOf(got))
		assert.Equal(t, "insp-1", got["inspectionID"])
	})

	t.Run("InspectionCreateInput uses assignedToID with capital ID", func(t *testing.T) {
		input := InspectionCreateInput{
			LocationID:   "loc-1",
			TemplateID:   "tpl-1",
			ScheduledFor: "2026-06-01T00:00:00Z",
			AssignedToID: "user-1",
		}
		got := marshalToMap(t, input)

		_, hasCapital := got["assignedToID"]
		_, hasLowercase := got["assignedToId"]
		assert.True(t, hasCapital, "InspectionCreateInput must marshal AssignedToID as 'assignedToID' (capital ID); got keys: %v", keysOf(got))
		assert.False(t, hasLowercase, "InspectionCreateInput must NOT marshal as 'assignedToId' (lowercase d); got keys: %v", keysOf(got))
		assert.Equal(t, "user-1", got["assignedToID"])
		// Sanity: peer fields stay lowercase.
		assert.Equal(t, "loc-1", got["locationId"])
		assert.Equal(t, "tpl-1", got["templateId"])
	})

	t.Run("WorkOrderCreateInput uses lowercase 'type' for Type field", func(t *testing.T) {
		// `Type` is a Go reserved-feeling field name and could plausibly be
		// renamed/recased in a refactor. Pin the wire key.
		input := WorkOrderCreateInput{
			LocationID: "loc-1",
			Type:       "SERVICE_REQUEST",
			Status:     "OPEN",
		}
		got := marshalToMap(t, input)

		_, hasLowerType := got["type"]
		_, hasUpperType := got["Type"]
		assert.True(t, hasLowerType, "WorkOrderCreateInput must marshal Type as 'type' (lowercase); got keys: %v", keysOf(got))
		assert.False(t, hasUpperType, "WorkOrderCreateInput must NOT marshal as 'Type' (capital); got keys: %v", keysOf(got))
		assert.Equal(t, "SERVICE_REQUEST", got["type"])
		assert.Equal(t, "OPEN", got["status"])
	})

	t.Run("WorkOrderSetStatusAndSubStatusInput uses PascalCase Status and SubStatus", func(t *testing.T) {
		input := WorkOrderSetStatusAndSubStatusInput{
			WorkOrderID: "wo-1",
			Status:      WorkOrderStatusInput{Status: "COMPLETED"},
			SubStatus:   WorkOrderSubStatusInput{SubStatus: "UNKNOWN"},
		}
		got := marshalToMap(t, input)

		_, hasPascalStatus := got["Status"]
		_, hasLowerStatus := got["status"]
		_, hasPascalSub := got["SubStatus"]
		_, hasLowerSub := got["subStatus"]
		assert.True(t, hasPascalStatus, "Status field must marshal as 'Status' (capital S); got keys: %v", keysOf(got))
		assert.False(t, hasLowerStatus, "Status field must NOT marshal as 'status' (lowercase); got keys: %v", keysOf(got))
		assert.True(t, hasPascalSub, "SubStatus field must marshal as 'SubStatus' (capital S); got keys: %v", keysOf(got))
		assert.False(t, hasLowerSub, "SubStatus field must NOT marshal as 'subStatus' (lowercase); got keys: %v", keysOf(got))
		assert.Equal(t, "wo-1", got["workOrderId"])
	})
}

// TestNoUnintendedCapitalIDDrift catches the case where a developer adds a new
// input field with capital ID. The intentional anomalies above are listed in
// allowedCapitalIDFields; everything else is rejected.
//
// This is a guard rail, not an exhaustive audit — covers the structs in this
// file. If you genuinely need a new capital-ID field, add it to allowedCapitalIDFields
// AND add an explicit assertion to TestModelInputJSONWireKeyAnomalies above.
func TestNoUnintendedCapitalIDDrift(t *testing.T) {
	allowedCapitalIDFields := map[string]bool{
		"InspectionAddItemPhotoInput.inspectionID": true,
		"InspectionCreateInput.assignedToID":       true,
	}

	cases := []struct {
		name    string
		marshal func() map[string]any
	}{
		{"InspectionSetAssigneeInput", func() map[string]any {
			return marshalToMapInline(InspectionSetAssigneeInput{InspectionID: "x", UserID: "y"})
		}},
		{"InspectionSetDueByInput", func() map[string]any {
			return marshalToMapInline(InspectionSetDueByInput{InspectionID: "x", DueBy: "y"})
		}},
		{"InspectionRateItemInput", func() map[string]any {
			return marshalToMapInline(InspectionRateItemInput{InspectionID: "x"})
		}},
		{"InspectionAddSectionInput", func() map[string]any {
			return marshalToMapInline(InspectionAddSectionInput{InspectionID: "x", Name: "y"})
		}},
		{"InspectionRemoveItemPhotoInput", func() map[string]any {
			return marshalToMapInline(InspectionRemoveItemPhotoInput{InspectionID: "x", PhotoID: "y"})
		}},
		{"InspectionMoveItemPhotoInput", func() map[string]any {
			return marshalToMapInline(InspectionMoveItemPhotoInput{InspectionID: "x", PhotoID: "y"})
		}},
		{"InspectionSendToGuestInput", func() map[string]any {
			return marshalToMapInline(InspectionSendToGuestInput{InspectionID: "x", Email: "y@z"})
		}},
		{"WorkOrderCreateInput", func() map[string]any {
			return marshalToMapInline(WorkOrderCreateInput{LocationID: "x"})
		}},
		{"WorkOrderSetAssigneeInput", func() map[string]any {
			return marshalToMapInline(WorkOrderSetAssigneeInput{WorkOrderID: "x"})
		}},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := c.marshal()
			for k := range got {
				if strings.HasSuffix(k, "ID") {
					qualified := c.name + "." + k
					assert.True(t, allowedCapitalIDFields[qualified],
						"unexpected capital-ID field %s — most input fields use lowercase 'Id'. If this is intentional, add %q to allowedCapitalIDFields AND add an explicit assertion in TestModelInputJSONWireKeyAnomalies.",
						qualified, qualified)
				}
			}
		})
	}
}

func marshalToMap(t *testing.T, v any) map[string]any {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

func marshalToMapInline(v any) map[string]any {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		panic(err)
	}
	return out
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
