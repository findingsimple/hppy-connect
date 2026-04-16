package cmd

import (
	"fmt"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Show account details",
	Long:  "Display the account name and ID for the configured account. Unlike other commands, account is a single action (there is only one account per config).",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if outputFormat == "raw" {
			raw, err := apiClient.GetAccountRaw(ctx)
			if err != nil {
				return fmt.Errorf("fetching account: %w", err)
			}
			return printOutput(outputData{RawJSON: raw})
		}

		account, err := apiClient.GetAccount(ctx)
		if err != nil {
			return fmt.Errorf("fetching account: %w", err)
		}

		return printOutput(outputData{
			Headers: []string{"ID", "NAME"},
			Rows:    [][]string{{account.ID, account.Name}},
			Items:   []models.Account{*account},
			Count:   1,
		})
	},
}
