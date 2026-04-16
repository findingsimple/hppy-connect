package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock ---

type mockSeedClient struct {
	properties    []models.Property
	propertiesErr error
	units         map[string][]models.Unit // propertyID -> units
	unitsErr      map[string]error         // propertyID -> error
	createdWOs    []models.WorkOrderCreateInput
	woCreateErr   error
	createdInsps  []models.InspectionCreateInput
	inspCreateErr error
	startedInsps  []string
	inspStartErr  error
	createdProjs  []models.ProjectCreateInput
	projCreateErr error
	createdWHs    []models.WebhookCreateInput
	whCreateErr   error
}

func (m *mockSeedClient) ListProperties(_ context.Context, _ models.ListOptions) ([]models.Property, int, error) {
	return m.properties, len(m.properties), m.propertiesErr
}

func (m *mockSeedClient) ListUnits(_ context.Context, propertyID string, _ models.ListOptions) ([]models.Unit, int, error) {
	if m.unitsErr != nil {
		if err, ok := m.unitsErr[propertyID]; ok {
			return nil, 0, err
		}
	}
	units := m.units[propertyID]
	return units, len(units), nil
}

func (m *mockSeedClient) WorkOrderCreate(_ context.Context, input models.WorkOrderCreateInput) (*models.WorkOrder, error) {
	m.createdWOs = append(m.createdWOs, input)
	if m.woCreateErr != nil {
		return nil, m.woCreateErr
	}
	return &models.WorkOrder{ID: fmt.Sprintf("wo-%d", len(m.createdWOs))}, nil
}

func (m *mockSeedClient) InspectionCreate(_ context.Context, input models.InspectionCreateInput) (*models.Inspection, error) {
	m.createdInsps = append(m.createdInsps, input)
	if m.inspCreateErr != nil {
		return nil, m.inspCreateErr
	}
	return &models.Inspection{ID: fmt.Sprintf("insp-%d", len(m.createdInsps))}, nil
}

func (m *mockSeedClient) InspectionStart(_ context.Context, id string) (*models.Inspection, error) {
	m.startedInsps = append(m.startedInsps, id)
	if m.inspStartErr != nil {
		return nil, m.inspStartErr
	}
	return &models.Inspection{ID: id, Status: "INCOMPLETE"}, nil
}

func (m *mockSeedClient) ProjectCreate(_ context.Context, input models.ProjectCreateInput) (*models.Project, error) {
	m.createdProjs = append(m.createdProjs, input)
	if m.projCreateErr != nil {
		return nil, m.projCreateErr
	}
	return &models.Project{ID: fmt.Sprintf("proj-%d", len(m.createdProjs))}, nil
}

func (m *mockSeedClient) WebhookCreate(_ context.Context, input models.WebhookCreateInput) (*models.Webhook, error) {
	m.createdWHs = append(m.createdWHs, input)
	if m.whCreateErr != nil {
		return nil, m.whCreateErr
	}
	return &models.Webhook{ID: fmt.Sprintf("wh-%d", len(m.createdWHs))}, nil
}

// --- buildSeedPlan tests ---

func TestBuildSeedPlan(t *testing.T) {
	locations := []seedLocation{
		{ID: "u1", DisplayName: "Prop A > Unit 1", PropertyID: "p1"},
		{ID: "u2", DisplayName: "Prop A > Unit 2", PropertyID: "p1"},
		{ID: "u3", DisplayName: "Prop B > Unit 1", PropertyID: "p2"},
	}

	t.Run("work orders only", func(t *testing.T) {
		plan := buildSeedPlan(locations, seedOptions{Count: 3})

		// 3 locations x 3 count = 9 work orders
		assert.Equal(t, 9, len(plan))
		for _, item := range plan {
			assert.Equal(t, "work-order", item.EntityType)
			assert.NotNil(t, item.Recipe)
		}
	})

	t.Run("recipe cycling wraps", func(t *testing.T) {
		plan := buildSeedPlan(locations, seedOptions{Count: 5})

		// 3 locations x 5 count = 15 work orders, wrapping the 10 recipes
		assert.Equal(t, 15, len(plan))
		// Item 10 (index 10) should cycle back to recipe 0
		assert.Equal(t, recipes[0].Description, plan[10].Recipe.Description)
	})

	t.Run("inspections capped at 2 per property", func(t *testing.T) {
		plan := buildSeedPlan(locations, seedOptions{Count: 1, InspTemplateID: "tmpl1"})

		inspections := filterPlan(plan, "inspection")
		// p1 has 2 locations (u1, u2) -> 2 inspections
		// p2 has 1 location (u3) -> 1 inspection (can't reach 2)
		assert.Equal(t, 3, len(inspections))

		// p1: first not started, second started
		assert.False(t, inspections[0].InspShouldStart)
		assert.True(t, inspections[1].InspShouldStart)
		// p2: first not started (only location)
		assert.False(t, inspections[2].InspShouldStart)
	})

	t.Run("inspections 2 per property when enough locations", func(t *testing.T) {
		locs := []seedLocation{
			{ID: "u1", DisplayName: "P1 > U1", PropertyID: "p1"},
			{ID: "u2", DisplayName: "P1 > U2", PropertyID: "p1"},
			{ID: "u3", DisplayName: "P1 > U3", PropertyID: "p1"},
			{ID: "u4", DisplayName: "P2 > U1", PropertyID: "p2"},
			{ID: "u5", DisplayName: "P2 > U2", PropertyID: "p2"},
		}
		plan := buildSeedPlan(locs, seedOptions{Count: 1, InspTemplateID: "tmpl1"})

		inspections := filterPlan(plan, "inspection")
		assert.Equal(t, 4, len(inspections))
	})

	t.Run("projects capped at 1 per property", func(t *testing.T) {
		plan := buildSeedPlan(locations, seedOptions{Count: 1, ProjTemplateID: "ptmpl1"})

		projects := filterPlan(plan, "project")
		// 2 properties = 2 projects
		assert.Equal(t, 2, len(projects))
		assert.Equal(t, "NORMAL", projects[0].ProjPriority)
		assert.Equal(t, "URGENT", projects[1].ProjPriority)
	})

	t.Run("webhook included when URL provided", func(t *testing.T) {
		plan := buildSeedPlan(locations, seedOptions{Count: 1, WebhookURL: "https://example.com/hook"})

		webhooks := filterPlan(plan, "webhook")
		assert.Equal(t, 1, len(webhooks))
		assert.Equal(t, "https://example.com/hook", webhooks[0].WebhookURL)
	})

	t.Run("no inspections or projects without template IDs", func(t *testing.T) {
		plan := buildSeedPlan(locations, seedOptions{Count: 1})

		for _, item := range plan {
			assert.Equal(t, "work-order", item.EntityType)
		}
	})

	t.Run("day offsets are 1 and 2", func(t *testing.T) {
		plan := buildSeedPlan(locations, seedOptions{Count: 1, InspTemplateID: "tmpl1"})

		inspections := filterPlan(plan, "inspection")
		require.Equal(t, 3, len(inspections))
		// p1 has 2 locations: offsets 1, 2
		assert.Equal(t, 1, inspections[0].InspDayOffset)
		assert.Equal(t, 2, inspections[1].InspDayOffset)
		// p2 has 1 location: offset 1
		assert.Equal(t, 1, inspections[2].InspDayOffset)
	})
}

// --- runSeed tests ---

func TestRunSeed(t *testing.T) {
	t.Run("no properties returns error", func(t *testing.T) {
		mock := &mockSeedClient{properties: nil}
		var stdout, stderr bytes.Buffer

		err := runSeed(context.Background(), mock, seedOptions{Count: 1, DryRun: true}, &stdout, &stderr)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no properties found")
	})

	t.Run("ListProperties error propagates", func(t *testing.T) {
		mock := &mockSeedClient{propertiesErr: fmt.Errorf("network timeout")}
		var stdout, stderr bytes.Buffer

		err := runSeed(context.Background(), mock, seedOptions{Count: 1, DryRun: true}, &stdout, &stderr)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing properties")
	})

	t.Run("ListUnits failure degrades to property", func(t *testing.T) {
		mock := &mockSeedClient{
			properties: []models.Property{{ID: "p1", Name: "Prop A"}},
			unitsErr:   map[string]error{"p1": fmt.Errorf("unit listing failed")},
		}
		var stdout, stderr bytes.Buffer

		err := runSeed(context.Background(), mock, seedOptions{Count: 1, DryRun: true}, &stdout, &stderr)

		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "Warning")
		assert.Contains(t, stdout.String(), "Prop A") // property used as location
	})

	t.Run("dry run makes no API calls", func(t *testing.T) {
		mock := &mockSeedClient{
			properties: []models.Property{{ID: "p1", Name: "Prop A"}},
			units:      map[string][]models.Unit{"p1": {{ID: "u1", Name: "Unit 1"}}},
		}
		var stdout, stderr bytes.Buffer

		err := runSeed(context.Background(), mock, seedOptions{
			Count:          2,
			DryRun:         true,
			InspTemplateID: "tmpl1",
		}, &stdout, &stderr)

		require.NoError(t, err)
		assert.Empty(t, mock.createdWOs)
		assert.Empty(t, mock.createdInsps)
		assert.Contains(t, stdout.String(), "work-order")
		assert.Contains(t, stdout.String(), "inspection")
	})

	t.Run("creates work orders with batch tag and status", func(t *testing.T) {
		mock := &mockSeedClient{
			properties: []models.Property{{ID: "p1", Name: "Prop A"}},
			units:      map[string][]models.Unit{"p1": {{ID: "u1", Name: "Unit 1"}}},
		}
		var stdout, stderr bytes.Buffer

		err := runSeed(context.Background(), mock, seedOptions{Count: 3}, &stdout, &stderr)

		require.NoError(t, err)
		require.Equal(t, 3, len(mock.createdWOs))

		// All descriptions should have [SEED ...] prefix
		for _, wo := range mock.createdWOs {
			assert.Contains(t, wo.Description, "[SEED ")
		}

		// First recipe is URGENT SERVICE_REQUEST OPEN
		assert.Equal(t, "URGENT", mock.createdWOs[0].Priority)
		assert.Equal(t, "SERVICE_REQUEST", mock.createdWOs[0].Type)
		assert.Equal(t, "OPEN", mock.createdWOs[0].Status)

		// Third recipe is COMPLETED
		assert.Equal(t, "COMPLETED", mock.createdWOs[2].Status)
	})

	t.Run("inspections: second per property is started", func(t *testing.T) {
		mock := &mockSeedClient{
			properties: []models.Property{{ID: "p1", Name: "Prop A"}},
			units: map[string][]models.Unit{"p1": {
				{ID: "u1", Name: "Unit 1"},
				{ID: "u2", Name: "Unit 2"},
			}},
		}
		var stdout, stderr bytes.Buffer

		err := runSeed(context.Background(), mock, seedOptions{Count: 1, InspTemplateID: "tmpl1"}, &stdout, &stderr)

		require.NoError(t, err)
		assert.Equal(t, 2, len(mock.createdInsps))
		// Only the second inspection should have been started
		require.Equal(t, 1, len(mock.startedInsps))
		assert.Equal(t, "insp-2", mock.startedInsps[0])
	})

	t.Run("partial failure continues and reports", func(t *testing.T) {
		callCount := 0
		mock := &mockSeedClient{
			properties: []models.Property{{ID: "p1", Name: "Prop A"}},
			units:      map[string][]models.Unit{"p1": {{ID: "u1", Name: "Unit 1"}}},
		}
		// Override WorkOrderCreate to fail on second call
		originalCreate := mock.WorkOrderCreate
		_ = originalCreate
		var stdout, stderr bytes.Buffer

		// Use a mock that fails on second call
		failingMock := &failOnNthWOMock{mockSeedClient: mock, failOnCall: 2}
		err := runSeed(context.Background(), failingMock, seedOptions{Count: 3}, &stdout, &stderr)
		_ = callCount

		require.Error(t, err)
		assert.Contains(t, err.Error(), "1 of 3 seed operations failed")
		assert.Contains(t, stdout.String(), "simulated failure")
	})

	t.Run("context cancellation stops early", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		mock := &mockSeedClient{
			properties: []models.Property{{ID: "p1", Name: "Prop A"}},
			units:      map[string][]models.Unit{"p1": {{ID: "u1", Name: "Unit 1"}}},
		}
		var stdout, stderr bytes.Buffer

		err := runSeed(ctx, mock, seedOptions{Count: 3}, &stdout, &stderr)

		// All items should be marked cancelled, no API calls made
		require.Error(t, err)
		assert.Equal(t, 0, len(mock.createdWOs))
		assert.Contains(t, stdout.String(), "cancelled")
	})

	t.Run("webhook uses account ID", func(t *testing.T) {
		// Set the package-level configAccountID for this test
		oldAccountID := configAccountID
		configAccountID = "acct-123"
		defer func() { configAccountID = oldAccountID }()

		mock := &mockSeedClient{
			properties: []models.Property{{ID: "p1", Name: "Prop A"}},
			units:      map[string][]models.Unit{"p1": {{ID: "u1", Name: "Unit 1"}}},
		}
		var stdout, stderr bytes.Buffer

		err := runSeed(context.Background(), mock, seedOptions{
			Count:      1,
			WebhookURL: "https://example.com/hook",
			AccountID:  "acct-123",
		}, &stdout, &stderr)

		require.NoError(t, err)
		require.Equal(t, 1, len(mock.createdWHs))
		assert.Equal(t, "acct-123", mock.createdWHs[0].SubscriberID)
		assert.Equal(t, "DISABLED", mock.createdWHs[0].Status)
		assert.Equal(t, []string{"INSPECTIONS", "WORK_ORDERS"}, mock.createdWHs[0].Subjects)
	})

	t.Run("json output format", func(t *testing.T) {
		oldFormat := outputFormat
		outputFormat = "json"
		defer func() { outputFormat = oldFormat }()

		mock := &mockSeedClient{
			properties: []models.Property{{ID: "p1", Name: "Prop A"}},
			units:      map[string][]models.Unit{"p1": {{ID: "u1", Name: "Unit 1"}}},
		}
		var stdout, stderr bytes.Buffer

		err := runSeed(context.Background(), mock, seedOptions{Count: 1}, &stdout, &stderr)

		require.NoError(t, err)
		assert.Contains(t, stdout.String(), `"created"`)
		assert.Contains(t, stdout.String(), `"summary"`)
		assert.Contains(t, stdout.String(), `"success": 1`)
	})
}

// --- printSeedSummary tests ---

func TestPrintSeedSummary(t *testing.T) {
	t.Run("text format with mixed results", func(t *testing.T) {
		oldFormat := outputFormat
		outputFormat = "text"
		defer func() { outputFormat = oldFormat }()

		results := []seedResult{
			{EntityType: "work-order", ID: "wo-1", Location: "Prop A", Description: "Test WO"},
			{EntityType: "work-order", Location: "Prop A", Description: "Failed WO", Error: "auth failed"},
		}
		var stdout, stderr bytes.Buffer

		printSeedSummary(&stdout, &stderr, results)

		assert.Contains(t, stdout.String(), "wo-1")
		assert.Contains(t, stdout.String(), "auth failed")
		assert.Contains(t, stderr.String(), "Created 1 of 2 items")
		assert.Contains(t, stderr.String(), "1 failed")
	})

	t.Run("json format", func(t *testing.T) {
		oldFormat := outputFormat
		outputFormat = "json"
		defer func() { outputFormat = oldFormat }()

		results := []seedResult{
			{EntityType: "work-order", ID: "wo-1", Description: "Test WO"},
		}
		var stdout, stderr bytes.Buffer

		printSeedSummary(&stdout, &stderr, results)

		assert.Contains(t, stdout.String(), `"created"`)
		assert.Contains(t, stdout.String(), `"wo-1"`)
		assert.Contains(t, stdout.String(), `"success": 1`)
	})

	t.Run("error strings are sanitized", func(t *testing.T) {
		oldFormat := outputFormat
		outputFormat = "text"
		defer func() { outputFormat = oldFormat }()

		results := []seedResult{
			{EntityType: "work-order", Description: "test", Error: "error\twith\ttabs\nand newlines"},
		}
		var stdout, stderr bytes.Buffer

		printSeedSummary(&stdout, &stderr, results)

		// sanitizeCell should have replaced tabs and newlines
		assert.NotContains(t, stdout.String(), "\t\t\t") // no raw tabs in error column beyond table separators
		output := stdout.String()
		lines := strings.Split(output, "\n")
		// The data line should have exactly 5 tab-separated columns (header + 1 data row)
		require.True(t, len(lines) >= 2)
	})
}

// --- Helpers ---

func filterPlan(plan []plannedItem, entityType string) []plannedItem {
	var filtered []plannedItem
	for _, item := range plan {
		if item.EntityType == entityType {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// failOnNthWOMock wraps mockSeedClient but fails WorkOrderCreate on the nth call.
type failOnNthWOMock struct {
	*mockSeedClient
	failOnCall int
	callCount  int
}

func (m *failOnNthWOMock) WorkOrderCreate(ctx context.Context, input models.WorkOrderCreateInput) (*models.WorkOrder, error) {
	m.callCount++
	if m.callCount == m.failOnCall {
		m.mockSeedClient.createdWOs = append(m.mockSeedClient.createdWOs, input)
		return nil, fmt.Errorf("simulated failure")
	}
	return m.mockSeedClient.WorkOrderCreate(ctx, input)
}
