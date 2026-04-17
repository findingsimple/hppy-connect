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

		// Prompt and authenticate, with up to 3 attempts on auth failure.
		// Avoids the failure mode where a typo in the password aborts the
		// whole flow and forces the user to re-run the command.
		const maxAttempts = 3
		var email, password string
		var result *api.LoginResult

		for attempt := 1; ; attempt++ {
			// Prompt for email (allow re-use of previous on retry).
			if email == "" {
				fmt.Print("Email: ")
				v, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading email: %w", err)
				}
				email = strings.TrimSpace(v)
				if email == "" {
					return fmt.Errorf("email is required")
				}
			} else {
				fmt.Printf("Email [%s] (Enter to keep): ", email)
				v, _ := reader.ReadString('\n')
				v = strings.TrimSpace(v)
				if v != "" {
					email = v
				}
			}

			// Prompt for password (no-echo, always re-prompt — never reuse).
			fmt.Print("Password: ")
			pwBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("reading password: %w", err)
			}
			// Do not TrimSpace — passwords may legitimately contain leading
			// or trailing whitespace, and silently stripping causes confusing
			// auth failures the user can't diagnose.
			password = string(pwBytes)
			if password == "" {
				return fmt.Errorf("password is required")
			}

			fmt.Println("Authenticating...")
			r, err := api.Login(cmd.Context(), email, password, api.DefaultEndpoint)
			if err == nil {
				result = r
				break
			}

			fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
			if attempt >= maxAttempts {
				return fmt.Errorf("authentication failed after %d attempts", maxAttempts)
			}
			fmt.Print("Try again? [Y/n]: ")
			ans, _ := reader.ReadString('\n')
			ans = strings.TrimSpace(strings.ToLower(ans))
			if ans == "n" || ans == "no" {
				return fmt.Errorf("aborted")
			}
		}

		if len(result.AccountIDs) == 0 {
			return fmt.Errorf("no accessible accounts found for this user")
		}

		choices := resolveAccountNames(cmd.Context(), result.AccountIDs, email, password, api.DefaultEndpoint, result.Token, result.ExpiresAt)

		accountID, err := selectAccount(choices, reader, os.Stderr)
		if err != nil {
			return err
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

		// atomicWriteConfig defeats symlink-swap TOCTOU between Stat and Write.
		if err := atomicWriteConfig(configPath, data); err != nil {
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
	return resolveConfigPathFrom(cfgFile)
}

// resolveConfigPathFrom returns flagValue if non-empty, otherwise the default path.
func resolveConfigPathFrom(flagValue string) string {
	if flagValue != "" {
		return flagValue
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
