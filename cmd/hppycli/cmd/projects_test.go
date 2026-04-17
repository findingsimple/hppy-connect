package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// CLI handler validation tests for the projects domain. Cover the pre-API
// validation paths via a fresh cobra wrapper around the production RunE —
// no API client mock needed.

func runProjectsCreate(args ...string) error {
	wrapper := &cobra.Command{Use: "create-test", RunE: projectsCreateCmd.RunE}
	wrapper.Flags().String("template-id", "", "")
	wrapper.Flags().String("location-id", "", "")
	wrapper.Flags().String("start-at", "", "")
	wrapper.Flags().String("assignee-id", "", "")
	wrapper.Flags().String("priority", "", "")
	wrapper.Flags().String("due-at", "", "")
	wrapper.Flags().String("availability-target-at", "", "")
	wrapper.Flags().String("notes", "", "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestProjectsCreate_RequiresTemplateID(t *testing.T) {
	err := runProjectsCreate("--location-id=loc1", "--start-at=2026-05-01T00:00:00Z")
	assert.ErrorContains(t, err, "--template-id is required")
}

func TestProjectsCreate_RequiresLocationID(t *testing.T) {
	err := runProjectsCreate("--template-id=tpl1", "--start-at=2026-05-01T00:00:00Z")
	assert.ErrorContains(t, err, "--location-id is required")
}

func TestProjectsCreate_RequiresStartAt(t *testing.T) {
	err := runProjectsCreate("--template-id=tpl1", "--location-id=loc1")
	assert.ErrorContains(t, err, "--start-at is required")
}

func TestProjectsCreate_RejectsInvalidTimestamp(t *testing.T) {
	err := runProjectsCreate("--template-id=tpl1", "--location-id=loc1", "--start-at=not-a-date")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RFC3339")
}

func TestProjectsCreate_RejectsInvalidPriority(t *testing.T) {
	err := runProjectsCreate("--template-id=tpl1", "--location-id=loc1", "--start-at=2026-05-01T00:00:00Z", "--priority=HIGH")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "priority")
}

func TestProjectsCreate_RejectsInvalidIDChars(t *testing.T) {
	err := runProjectsCreate("--template-id=../bad", "--location-id=loc1", "--start-at=2026-05-01T00:00:00Z")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template-id")
}

func TestProjectsCreate_RejectsOversizeNotes(t *testing.T) {
	huge := strings.Repeat("x", 100_001)
	err := runProjectsCreate("--template-id=tpl1", "--location-id=loc1", "--start-at=2026-05-01T00:00:00Z", "--notes="+huge)
	assert.ErrorContains(t, err, "notes")
}

func runProjectsSetAssignee(args ...string) error {
	wrapper := &cobra.Command{Use: "set-assignee-test", RunE: projectsSetAssigneeCmd.RunE}
	wrapper.Flags().String("id", "", "")
	wrapper.Flags().String("assignee-id", "", "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestProjectsSetAssignee_RequiresID(t *testing.T) {
	err := runProjectsSetAssignee()
	assert.ErrorContains(t, err, "--id is required")
}

func TestProjectsSetAssignee_RejectsInvalidAssigneeID(t *testing.T) {
	err := runProjectsSetAssignee("--id=proj1", "--assignee-id=../bad")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "assignee-id")
}
