package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/findingsimple/hppy-connect/internal/api"
	"github.com/findingsimple/hppy-connect/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a configuration file interactively",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := resolveConfigPath()

		reader := bufio.NewReader(os.Stdin)

		// Warn if config file already exists
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config file already exists at %s\n", configPath)
			fmt.Print("Overwrite? [y/N]: ")
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Prompt for email
		fmt.Print("Email: ")
		email, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading email: %w", err)
		}
		email = strings.TrimSpace(email)
		if email == "" {
			return fmt.Errorf("email is required")
		}

		// Prompt for password (no-echo)
		fmt.Print("Password: ")
		pwBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // newline after hidden input
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		password := strings.TrimSpace(string(pwBytes))
		if password == "" {
			return fmt.Errorf("password is required")
		}

		// Authenticate
		fmt.Println("Authenticating...")
		result, err := api.Login(cmd.Context(), email, password, api.DefaultEndpoint)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		if len(result.AccountIDs) == 0 {
			return fmt.Errorf("no accessible accounts found for this user")
		}

		// Select account ID
		accountID := result.AccountIDs[0]
		if len(result.AccountIDs) > 1 {
			fmt.Println("\nMultiple accounts found:")
			for i, id := range result.AccountIDs {
				fmt.Printf("  [%d] %s\n", i+1, id)
			}
			fmt.Printf("Select account (1-%d): ", len(result.AccountIDs))
			selection, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading selection: %w", err)
			}
			selection = strings.TrimSpace(selection)
			idx := 0
			if _, err := fmt.Sscanf(selection, "%d", &idx); err != nil || idx < 1 || idx > len(result.AccountIDs) {
				return fmt.Errorf("invalid selection %q: must be 1-%d", selection, len(result.AccountIDs))
			}
			accountID = result.AccountIDs[idx-1]
		} else {
			fmt.Printf("Account ID: %s\n", accountID)
		}

		// Write config file
		cfgData := config.Config{
			Email:     email,
			Password:  password,
			AccountID: accountID,
			Endpoint:  api.DefaultEndpoint,
		}

		data, err := yaml.Marshal(&cfgData)
		if err != nil {
			return fmt.Errorf("marshalling config: %w", err)
		}

		if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}

		if err := os.WriteFile(configPath, data, 0600); err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}

		fmt.Printf("\nConfiguration saved to %s\n", configPath)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := resolveConfigPath()

		loadedCfg, err := config.LoadConfig(configPath, os.Getenv, nil)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		maskedPw := ""
		if loadedCfg.Password != "" {
			maskedPw = "********"
		}

		fmt.Printf("Email:       %s\n", loadedCfg.Email)
		fmt.Printf("Password:    %s\n", maskedPw)
		fmt.Printf("Account ID:  %s\n", loadedCfg.AccountID)
		fmt.Printf("Endpoint:    %s\n", loadedCfg.Endpoint)
		fmt.Printf("Debug:       %t\n", loadedCfg.Debug)
		fmt.Printf("Config file: %s\n", configPath)
		return nil
	},
}

// resolveConfigPath returns the config file path from --config flag or the default.
func resolveConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".hppycli.yaml"
	}
	return filepath.Join(home, ".hppycli.yaml")
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
