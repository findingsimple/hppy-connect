package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Manage webhook subscriptions",
}

var webhooksCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new webhook subscription",
	Example: `  hppycli webhooks create --subscriber-id=acct123 --subscriber-type=ACCOUNT --url=https://example.com/webhook --subjects=INSPECTIONS,WORK_ORDERS`,
	RunE: func(cmd *cobra.Command, args []string) error {
		subscriberID, _ := cmd.Flags().GetString("subscriber-id")
		if subscriberID == "" {
			return fmt.Errorf("--subscriber-id is required")
		}
		if err := models.ValidateID("subscriber-id", subscriberID); err != nil {
			return err
		}

		subscriberType, _ := cmd.Flags().GetString("subscriber-type")
		if subscriberType == "" {
			return fmt.Errorf("--subscriber-type is required")
		}
		upperType := strings.ToUpper(subscriberType)
		if !models.ValidWebhookSubscriberTypes[upperType] {
			return fmt.Errorf("--subscriber-type must be ACCOUNT or PLUGIN")
		}

		webhookURL := strings.TrimSpace(mustGetString(cmd, "url"))
		if webhookURL == "" {
			return fmt.Errorf("--url is required")
		}
		if err := models.ValidateWebhookURL(webhookURL); err != nil {
			return err
		}

		input := models.WebhookCreateInput{
			SubscriberID:   subscriberID,
			SubscriberType: upperType,
			URL:            webhookURL,
		}

		if v, _ := cmd.Flags().GetString("subjects"); v != "" {
			subjects, err := models.ValidateWebhookSubjects(v)
			if err != nil {
				return err
			}
			if len(subjects) > 0 {
				input.Subjects = subjects
			}
		}

		if v, _ := cmd.Flags().GetString("status"); v != "" {
			upper := strings.ToUpper(v)
			if !models.ValidWebhookStatuses[upper] {
				return fmt.Errorf("--status must be ENABLED or DISABLED")
			}
			input.Status = upper
		}

		webhook, err := apiClient.WebhookCreate(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("creating webhook: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, webhook)
	},
}

var webhooksUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update an existing webhook",
	Example: `  hppycli webhooks update --id=wh123 --url=https://example.com/new-webhook --status=ENABLED`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		if err := models.ValidateID("id", id); err != nil {
			return err
		}

		input := models.WebhookUpdateInput{
			ID: id,
		}

		hasUpdate := false

		if v := strings.TrimSpace(mustGetString(cmd, "url")); v != "" {
			if err := models.ValidateWebhookURL(v); err != nil {
				return err
			}
			input.URL = v
			hasUpdate = true
		}

		if v, _ := cmd.Flags().GetString("status"); v != "" {
			upper := strings.ToUpper(v)
			if !models.ValidWebhookStatuses[upper] {
				return fmt.Errorf("--status must be ENABLED or DISABLED")
			}
			input.Status = upper
			hasUpdate = true
		}

		if v, _ := cmd.Flags().GetString("subjects"); v != "" {
			subjects, err := models.ValidateWebhookSubjects(v)
			if err != nil {
				return err
			}
			if len(subjects) > 0 {
				input.Subjects = subjects
				hasUpdate = true
			}
		}

		if !hasUpdate {
			return fmt.Errorf("at least one of --url, --status, or --subjects is required")
		}

		webhook, err := apiClient.WebhookUpdate(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("updating webhook: %w", err)
		}
		return printMutationResult(cmd, os.Stdout, webhook)
	},
}

// mustGetString is a helper to avoid the error return from GetString for flags
// that are guaranteed to be registered.
func mustGetString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func init() {
	// Create
	webhooksCreateCmd.Flags().String("subscriber-id", "", "subscriber ID — account or plugin ID (required)")
	webhooksCreateCmd.Flags().String("subscriber-type", "", "subscriber type: ACCOUNT or PLUGIN (required)")
	webhooksCreateCmd.Flags().String("url", "", "webhook endpoint URL — must be HTTPS (required)")
	webhooksCreateCmd.Flags().String("subjects", "", "comma-separated subjects: INSPECTIONS, WORK_ORDERS, VENDORS, PLUGIN_SUBSCRIPTIONS")
	webhooksCreateCmd.Flags().String("status", "", "initial status: ENABLED or DISABLED (default DISABLED)")
	webhooksCmd.AddCommand(webhooksCreateCmd)

	// Update
	webhooksUpdateCmd.Flags().String("id", "", "webhook ID (required)")
	webhooksUpdateCmd.Flags().String("url", "", "new webhook endpoint URL — must be HTTPS")
	webhooksUpdateCmd.Flags().String("status", "", "new status: ENABLED or DISABLED")
	webhooksUpdateCmd.Flags().String("subjects", "", "comma-separated subjects: INSPECTIONS, WORK_ORDERS, VENDORS, PLUGIN_SUBSCRIPTIONS")
	webhooksCmd.AddCommand(webhooksUpdateCmd)
}
