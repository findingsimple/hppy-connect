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
	workOrders  []models.WorkOrder
	woTotal     int
	inspections []models.Inspection
	inspTotal   int
	err         error // returned by all methods when set

	// Capture call args for verification.
	lastListOpts   models.ListOptions
	lastPropertyID string
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
		assert.Contains(t, text, `"properties": []`)
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
	woStatuses := validWorkOrderStatuses

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
			statuses:   validInspectionStatuses,
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
			wantErr:      "(e.g. 2026-01-15T00:00:00Z)",
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
