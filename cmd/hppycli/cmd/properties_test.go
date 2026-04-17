package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// CLI handler validation tests for the properties domain. Property access
// is a high-blast-radius mutation surface (account-wide access flag, multi-user
// grant/revoke). Cover the pre-API validation paths.

func runPropertiesGrantAccess(args ...string) error {
	wrapper := &cobra.Command{Use: "grant-access-test", RunE: propertiesGrantAccessCmd.RunE}
	wrapper.Flags().String("id", "", "")
	wrapper.Flags().String("user-id", "", "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestPropertiesGrantAccess_RequiresID(t *testing.T) {
	err := runPropertiesGrantAccess("--user-id=u1")
	assert.ErrorContains(t, err, "--id is required")
}

func TestPropertiesGrantAccess_RequiresUserID(t *testing.T) {
	err := runPropertiesGrantAccess("--id=prop1")
	assert.ErrorContains(t, err, "--user-id is required")
}

func TestPropertiesGrantAccess_RejectsInvalidUserID(t *testing.T) {
	err := runPropertiesGrantAccess("--id=prop1", "--user-id=u1,bad id,u3")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user-id")
}

func runPropertiesRevokeAccess(args ...string) error {
	wrapper := &cobra.Command{Use: "revoke-access-test", RunE: propertiesRevokeAccessCmd.RunE}
	wrapper.Flags().String("id", "", "")
	wrapper.Flags().String("user-id", "", "")
	wrapper.Flags().Bool("yes", false, "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestPropertiesRevokeAccess_RequiresID(t *testing.T) {
	err := runPropertiesRevokeAccess("--user-id=u1")
	assert.ErrorContains(t, err, "--id is required")
}

func TestPropertiesRevokeAccess_RejectsInvalidUserID(t *testing.T) {
	err := runPropertiesRevokeAccess("--id=prop1", "--user-id=../bad", "--yes")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user-id")
}

func runPropertiesSetAccountWide(args ...string) error {
	wrapper := &cobra.Command{Use: "set-aw-test", RunE: propertiesSetAccountWideAccessCmd.RunE}
	wrapper.Flags().String("id", "", "")
	wrapper.Flags().Bool("account-wide-access", false, "")
	wrapper.Flags().Bool("yes", false, "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestPropertiesSetAccountWide_RequiresID(t *testing.T) {
	err := runPropertiesSetAccountWide("--account-wide-access=true")
	assert.ErrorContains(t, err, "--id is required")
}

func TestPropertiesSetAccountWide_RequiresFlagExplicit(t *testing.T) {
	// --account-wide-access must be explicitly set; default-false would
	// silently DENY access (round 2 P1 — privilege-meaningful flag).
	err := runPropertiesSetAccountWide("--id=prop1")
	assert.ErrorContains(t, err, "--account-wide-access is required")
}
