package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// CLI handler validation tests for the roles domain. Roles are a privilege-
// escalation surface — bad role-id parsing or skipped validation could
// silently grant admin. Cover the pre-API validation paths.

func runRolesCreate(args ...string) error {
	wrapper := &cobra.Command{Use: "create-test", RunE: rolesCreateCmd.RunE}
	wrapper.Flags().String("account-id", "acct-1", "")
	wrapper.Flags().String("name", "", "")
	wrapper.Flags().String("description", "", "")
	wrapper.Flags().String("grant", "", "")
	wrapper.Flags().String("revoke", "", "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestRolesCreate_RequiresName(t *testing.T) {
	err := runRolesCreate("--grant=inspection:inspection.create")
	assert.ErrorContains(t, err, "--name is required")
}

func TestRolesCreate_RequiresGrantOrRevoke(t *testing.T) {
	// Neither --grant nor --revoke given — should reject (privilege-escalation safety).
	err := runRolesCreate("--name=Inspector")
	assert.ErrorContains(t, err, "--grant is required")
}

func TestRolesCreate_RejectsOversizeName(t *testing.T) {
	huge := strings.Repeat("x", 100_001)
	err := runRolesCreate("--name="+huge, "--grant=inspection:inspection.create")
	assert.ErrorContains(t, err, "name")
}

func TestRolesCreate_RejectsOversizeDescription(t *testing.T) {
	huge := strings.Repeat("x", 100_001)
	err := runRolesCreate("--name=Inspector", "--description="+huge, "--grant=inspection:inspection.create")
	assert.ErrorContains(t, err, "description")
}

func runRolesSetPermissions(args ...string) error {
	wrapper := &cobra.Command{Use: "set-permissions-test", RunE: rolesSetPermissionsCmd.RunE}
	wrapper.Flags().String("account-id", "acct-1", "")
	wrapper.Flags().String("id", "", "")
	wrapper.Flags().String("grant", "", "")
	wrapper.Flags().String("revoke", "", "")
	wrapper.Flags().Bool("yes", false, "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestRolesSetPermissions_RequiresID(t *testing.T) {
	err := runRolesSetPermissions("--grant=inspection:inspection.create")
	assert.ErrorContains(t, err, "--id is required")
}

func TestRolesSetPermissions_RequiresGrantOrRevoke(t *testing.T) {
	// Calling without grant OR revoke is a no-op; reject early.
	err := runRolesSetPermissions("--id=role1")
	assert.ErrorContains(t, err, "at least one of --grant or --revoke is required")
}

func TestRolesSetPermissions_RejectsInvalidID(t *testing.T) {
	err := runRolesSetPermissions("--id=../bad", "--grant=inspection:inspection.create", "--yes")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id")
}
