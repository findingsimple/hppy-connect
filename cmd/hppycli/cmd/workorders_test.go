package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// CLI handler validation tests for the workorders domain. The most
// frequently-used mutation surface; covers create + set-status validation
// (rounds 1+2 added free-text validation on `comment`; round 1 added enum
// validation; this locks the wiring).

func runWorkOrdersCreate(args ...string) error {
	wrapper := &cobra.Command{Use: "create-test", RunE: workordersCreateCmd.RunE}
	wrapper.Flags().String("location-id", "", "")
	wrapper.Flags().String("description", "", "")
	wrapper.Flags().String("priority", "", "")
	wrapper.Flags().String("status", "", "")
	wrapper.Flags().String("type", "", "")
	wrapper.Flags().String("scheduled-for", "", "")
	wrapper.Flags().String("entry-notes", "", "")
	wrapper.Flags().Bool("permission-to-enter", false, "")
	wrapper.Flags().String("assignee-id", "", "")
	wrapper.Flags().String("assignee-type", "", "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestWorkOrdersCreate_RequiresLocationID(t *testing.T) {
	err := runWorkOrdersCreate("--description=Fix leak")
	assert.ErrorContains(t, err, "--location-id is required")
}

func TestWorkOrdersCreate_RejectsInvalidLocationID(t *testing.T) {
	err := runWorkOrdersCreate("--location-id=../bad")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "location-id")
}

func TestWorkOrdersCreate_RejectsInvalidPriority(t *testing.T) {
	err := runWorkOrdersCreate("--location-id=loc1", "--priority=HIGH")
	assert.ErrorContains(t, err, "priority")
}

func TestWorkOrdersCreate_RejectsInvalidStatus(t *testing.T) {
	err := runWorkOrdersCreate("--location-id=loc1", "--status=BOGUS")
	assert.ErrorContains(t, err, "status")
}

func TestWorkOrdersCreate_RejectsInvalidType(t *testing.T) {
	err := runWorkOrdersCreate("--location-id=loc1", "--type=NOT_A_TYPE")
	assert.ErrorContains(t, err, "type")
}

func TestWorkOrdersCreate_RejectsInvalidTimestamp(t *testing.T) {
	err := runWorkOrdersCreate("--location-id=loc1", "--scheduled-for=not-a-date")
	assert.ErrorContains(t, err, "scheduled-for")
}

func TestWorkOrdersCreate_RejectsOversizeDescription(t *testing.T) {
	huge := strings.Repeat("x", 100_001)
	err := runWorkOrdersCreate("--location-id=loc1", "--description="+huge)
	assert.ErrorContains(t, err, "description")
}

func TestWorkOrdersCreate_RejectsOversizeEntryNotes(t *testing.T) {
	huge := strings.Repeat("x", 100_001)
	err := runWorkOrdersCreate("--location-id=loc1", "--entry-notes="+huge)
	assert.ErrorContains(t, err, "entry-notes")
}

func TestWorkOrdersCreate_RejectsInvalidAssigneeID(t *testing.T) {
	err := runWorkOrdersCreate("--location-id=loc1", "--assignee-id=bad id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "assignee-id")
}

func TestWorkOrdersCreate_RejectsInvalidAssigneeType(t *testing.T) {
	err := runWorkOrdersCreate("--location-id=loc1", "--assignee-id=u1", "--assignee-type=NEITHER")
	assert.ErrorContains(t, err, "assignee-type")
}

func runWorkOrdersSetStatus(args ...string) error {
	wrapper := &cobra.Command{Use: "set-status-test", RunE: workordersSetStatusCmd.RunE}
	wrapper.Flags().String("id", "", "")
	wrapper.Flags().String("status", "", "")
	wrapper.Flags().String("sub-status", "", "")
	wrapper.Flags().String("comment", "", "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestWorkOrdersSetStatus_RequiresID(t *testing.T) {
	err := runWorkOrdersSetStatus("--status=COMPLETED")
	assert.ErrorContains(t, err, "--id is required")
}

func TestWorkOrdersSetStatus_RequiresStatus(t *testing.T) {
	err := runWorkOrdersSetStatus("--id=wo1")
	assert.ErrorContains(t, err, "--status is required")
}

func TestWorkOrdersSetStatus_RejectsInvalidStatus(t *testing.T) {
	err := runWorkOrdersSetStatus("--id=wo1", "--status=BOGUS")
	assert.ErrorContains(t, err, "status")
}

func TestWorkOrdersSetStatus_RejectsInvalidSubStatus(t *testing.T) {
	err := runWorkOrdersSetStatus("--id=wo1", "--status=COMPLETED", "--sub-status=NOPE")
	assert.ErrorContains(t, err, "sub-status")
}

func TestWorkOrdersSetStatus_RejectsOversizeComment(t *testing.T) {
	// Round 1 P1-A added ValidateFreeText on comment in both CLI and MCP.
	huge := strings.Repeat("x", 100_001)
	err := runWorkOrdersSetStatus("--id=wo1", "--status=COMPLETED", "--comment="+huge)
	assert.ErrorContains(t, err, "comment")
}
