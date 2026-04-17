package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// These tests cover input validation in `users create` (P0-3 confirmation
// gate + P1-B phone validation from the 2026-04-17 review pass). They drive
// usersCreateCmd's RunE through a fresh cobra wrapper so they exercise the
// pre-API code paths only — no API client mock is needed.
//
// The goal is to catch a future refactor that drops a validator. The
// confirmation prompt itself reads from os.Stdin so isn't covered here;
// confirmAction's own behaviour is exercised in helpers_test.go.

func runUsersCreateRunE(args ...string) error {
	wrapper := &cobra.Command{Use: "create-test", RunE: usersCreateCmd.RunE}
	wrapper.Flags().String("account-id", "acct-1", "")
	wrapper.Flags().String("email", "", "")
	wrapper.Flags().String("name", "", "")
	wrapper.Flags().String("role-id", "", "")
	wrapper.Flags().String("short-name", "", "")
	wrapper.Flags().String("phone", "", "")
	wrapper.Flags().String("message", "", "")
	wrapper.Flags().Bool("yes", false, "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})

	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestUsersCreate_RejectsMissingEmail(t *testing.T) {
	err := runUsersCreateRunE("--name=Jane Doe")
	assert.ErrorContains(t, err, "--email is required")
}

func TestUsersCreate_RejectsInvalidEmail(t *testing.T) {
	err := runUsersCreateRunE("--email=notanemail", "--name=Jane Doe")
	assert.Error(t, err)
	if err != nil {
		assert.NotContains(t, err.Error(), "--email is required",
			"validator should fire after presence check")
	}
}

func TestUsersCreate_RejectsMissingName(t *testing.T) {
	err := runUsersCreateRunE("--email=a@b.co")
	assert.ErrorContains(t, err, "--name is required")
}

func TestUsersCreate_RejectsOversizePhone(t *testing.T) {
	// P1-B: phone validation must reject values over MaxPhoneLength (50 bytes).
	longPhone := strings.Repeat("9", 100)
	err := runUsersCreateRunE("--email=a@b.co", "--name=Jane", "--phone="+longPhone)
	assert.ErrorContains(t, err, "phone")
}

func TestUsersCreate_RejectsInvalidRoleID(t *testing.T) {
	err := runUsersCreateRunE("--email=a@b.co", "--name=Jane", "--role-id=bad id")
	assert.Error(t, err)
}
