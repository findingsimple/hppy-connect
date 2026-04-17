package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// CLI handler validation tests for the webhooks domain. Webhook URL
// validation is the only client-side defence against SSRF/private-address
// posting (the API-side validates too, but the CLI is the only safety net
// when an operator hand-types the URL). Cover the wiring carefully.

func runWebhooksCreate(args ...string) error {
	wrapper := &cobra.Command{Use: "create-test", RunE: webhooksCreateCmd.RunE}
	wrapper.Flags().String("subscriber-id", "", "")
	wrapper.Flags().String("subscriber-type", "", "")
	wrapper.Flags().String("url", "", "")
	wrapper.Flags().String("subjects", "", "")
	wrapper.Flags().String("status", "", "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestWebhooksCreate_RequiresSubscriberID(t *testing.T) {
	err := runWebhooksCreate("--subscriber-type=ACCOUNT", "--url=https://example.com/wh")
	assert.ErrorContains(t, err, "--subscriber-id is required")
}

func TestWebhooksCreate_RequiresSubscriberType(t *testing.T) {
	err := runWebhooksCreate("--subscriber-id=acct1", "--url=https://example.com/wh")
	assert.ErrorContains(t, err, "--subscriber-type is required")
}

func TestWebhooksCreate_RejectsInvalidSubscriberType(t *testing.T) {
	err := runWebhooksCreate("--subscriber-id=acct1", "--subscriber-type=INVALID", "--url=https://example.com/wh")
	assert.ErrorContains(t, err, "subscriber-type")
}

func TestWebhooksCreate_RequiresURL(t *testing.T) {
	err := runWebhooksCreate("--subscriber-id=acct1", "--subscriber-type=ACCOUNT")
	assert.ErrorContains(t, err, "--url is required")
}

func TestWebhooksCreate_WhitespaceOnlyURLRejected(t *testing.T) {
	err := runWebhooksCreate("--subscriber-id=acct1", "--subscriber-type=ACCOUNT", "--url=   ")
	assert.ErrorContains(t, err, "--url is required")
}

// Critical: webhook URL validation wiring. ValidateWebhookURL has its own
// extensive tests in models — these confirm the CLI handler actually invokes
// it for both create and update, and surfaces a meaningful error.
func TestWebhooksCreate_RejectsHTTP(t *testing.T) {
	err := runWebhooksCreate("--subscriber-id=acct1", "--subscriber-type=ACCOUNT", "--url=http://example.com/wh")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTPS")
}

func TestWebhooksCreate_RejectsCredentialsInURL(t *testing.T) {
	err := runWebhooksCreate("--subscriber-id=acct1", "--subscriber-type=ACCOUNT", "--url=https://user:pass@example.com/wh")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credentials")
}

func TestWebhooksCreate_RejectsPrivateIP(t *testing.T) {
	err := runWebhooksCreate("--subscriber-id=acct1", "--subscriber-type=ACCOUNT", "--url=https://192.168.1.1/wh")
	assert.Error(t, err)
}

func TestWebhooksCreate_RejectsInvalidSubject(t *testing.T) {
	err := runWebhooksCreate("--subscriber-id=acct1", "--subscriber-type=ACCOUNT", "--url=https://example.com/wh", "--subjects=BAD_SUBJECT")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook subject")
}

func TestWebhooksCreate_RejectsInvalidStatus(t *testing.T) {
	err := runWebhooksCreate("--subscriber-id=acct1", "--subscriber-type=ACCOUNT", "--url=https://example.com/wh", "--status=INVALID")
	assert.ErrorContains(t, err, "--status must be ENABLED or DISABLED")
}

func TestWebhooksCreate_RejectsInvalidSubscriberIDChars(t *testing.T) {
	err := runWebhooksCreate("--subscriber-id=../bad", "--subscriber-type=ACCOUNT", "--url=https://example.com/wh")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscriber-id")
}

func runWebhooksUpdate(args ...string) error {
	wrapper := &cobra.Command{Use: "update-test", RunE: webhooksUpdateCmd.RunE}
	wrapper.Flags().String("id", "", "")
	wrapper.Flags().String("url", "", "")
	wrapper.Flags().String("status", "", "")
	wrapper.Flags().String("subjects", "", "")
	wrapper.SetOut(&bytes.Buffer{})
	wrapper.SetErr(&bytes.Buffer{})
	wrapper.SetArgs(args)
	return wrapper.Execute()
}

func TestWebhooksUpdate_RequiresID(t *testing.T) {
	err := runWebhooksUpdate("--url=https://example.com/new")
	assert.ErrorContains(t, err, "--id is required")
}

func TestWebhooksUpdate_RequiresAtLeastOneField(t *testing.T) {
	err := runWebhooksUpdate("--id=wh1")
	assert.ErrorContains(t, err, "at least one of --url, --status, or --subjects")
}

func TestWebhooksUpdate_RejectsHTTPInUpdate(t *testing.T) {
	err := runWebhooksUpdate("--id=wh1", "--url=http://example.com/new")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTPS")
}

func TestWebhooksUpdate_RejectsInvalidStatus(t *testing.T) {
	err := runWebhooksUpdate("--id=wh1", "--status=INVALID")
	assert.ErrorContains(t, err, "--status must be ENABLED or DISABLED")
}

func TestWebhooksUpdate_RejectsInvalidSubject(t *testing.T) {
	err := runWebhooksUpdate("--id=wh1", "--subjects=BAD_SUBJECT")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook subject")
}
