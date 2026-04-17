package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// CLI handler validation tests for the memberships domain. set-roles is a
// privilege-escalation surface — bad role-id parsing or skipped validation
// could silently grant admin to a user. Cover the pre-API validation paths.

func runMembershipsCreate(args ...string) error {
	wrapper := &cobra.Command{Use: "create-test", RunE: membershipsCreateCmd.RunE}
	wrapper.Flags().String("account-id", "acct-1", "")
	wrapper.Flags().String("user-id", "", "")
	wrapper.Flags().String("role-id", "", "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestMembershipsCreate_RequiresUserID(t *testing.T) {
	err := runMembershipsCreate()
	assert.ErrorContains(t, err, "--user-id is required")
}

func TestMembershipsCreate_RejectsInvalidUserID(t *testing.T) {
	err := runMembershipsCreate("--user-id=../bad")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user-id")
}

func TestMembershipsCreate_RejectsInvalidRoleID(t *testing.T) {
	err := runMembershipsCreate("--user-id=u1", "--role-id=role1,bad id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role-id")
}

func runMembershipsSetRoles(args ...string) error {
	wrapper := &cobra.Command{Use: "set-roles-test", RunE: membershipsSetRolesCmd.RunE}
	wrapper.Flags().String("account-id", "acct-1", "")
	wrapper.Flags().String("user-id", "", "")
	wrapper.Flags().String("role-id", "", "")
	wrapper.Flags().Bool("yes", false, "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestMembershipsSetRoles_RequiresUserID(t *testing.T) {
	err := runMembershipsSetRoles("--role-id=role1")
	assert.ErrorContains(t, err, "--user-id is required")
}

func TestMembershipsSetRoles_RejectsInvalidUserID(t *testing.T) {
	err := runMembershipsSetRoles("--user-id=../bad", "--yes")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user-id")
}

func TestMembershipsSetRoles_RejectsInvalidRoleID(t *testing.T) {
	err := runMembershipsSetRoles("--user-id=u1", "--role-id=role1,bad id", "--yes")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role-id")
}

func runMembershipsActivate(args ...string) error {
	wrapper := &cobra.Command{Use: "activate-test", RunE: membershipsActivateCmd.RunE}
	wrapper.Flags().String("account-id", "acct-1", "")
	wrapper.Flags().String("user-id", "", "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestMembershipsActivate_RequiresUserID(t *testing.T) {
	err := runMembershipsActivate()
	assert.ErrorContains(t, err, "--user-id is required")
}

func runMembershipsDeactivate(args ...string) error {
	wrapper := &cobra.Command{Use: "deactivate-test", RunE: membershipsDeactivateCmd.RunE}
	wrapper.Flags().String("account-id", "acct-1", "")
	wrapper.Flags().String("user-id", "", "")
	wrapper.Flags().Bool("yes", false, "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestMembershipsDeactivate_RequiresUserID(t *testing.T) {
	err := runMembershipsDeactivate("--yes")
	assert.ErrorContains(t, err, "--user-id is required")
}

func TestMembershipsDeactivate_RejectsInvalidUserID(t *testing.T) {
	err := runMembershipsDeactivate("--user-id=../bad", "--yes")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user-id")
}
