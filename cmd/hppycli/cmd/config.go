package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
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

// loginFn matches api.Login's signature so tests can swap in a stub.
type loginFn func(ctx context.Context, email, password, endpoint string) (*api.LoginResult, error)

// passwordReader returns the (no-echo) password the user typed. In production
// this reads from the controlling terminal; tests pass a fake.
type passwordReader func() (string, error)

// configInitMaxAttempts caps the auth-retry loop in `config init`. Three is
// the right number for a typo-recovery loop — beyond that, the user should
// give up and check their actual credentials elsewhere.
const configInitMaxAttempts = 3

// authenticateWithRetry runs the prompt-and-login loop used by `config init`.
// Extracted so the retry/exhaustion behaviour is testable without touching
// os.Stdin or syscall.Stdin. Returns the resolved email + password (so the
// caller can persist them), the LoginResult, or an error after exhaustion
// or explicit user abort.
func authenticateWithRetry(ctx context.Context, reader *bufio.Reader, stderr io.Writer, login loginFn, readPassword passwordReader, endpoint string) (email, password string, result *api.LoginResult, err error) {
	for attempt := 1; ; attempt++ {
		// Email — first attempt prompts; subsequent attempts allow keep-or-replace.
		if email == "" {
			fmt.Fprint(stderr, "Email: ")
			v, rerr := reader.ReadString('\n')
			if rerr != nil {
				return "", "", nil, fmt.Errorf("reading email: %w", rerr)
			}
			email = strings.TrimSpace(v)
			if email == "" {
				return "", "", nil, fmt.Errorf("email is required")
			}
		} else {
			fmt.Fprintf(stderr, "Email [%s] (Enter to keep): ", email)
			v, _ := reader.ReadString('\n')
			v = strings.TrimSpace(v)
			if v != "" {
				email = v
			}
		}

		// Password — always re-prompt; never reuse, never trim (whitespace
		// can be legitimate).
		fmt.Fprint(stderr, "Password: ")
		pw, perr := readPassword()
		fmt.Fprintln(stderr)
		if perr != nil {
			return "", "", nil, fmt.Errorf("reading password: %w", perr)
		}
		password = pw
		if password == "" {
			return "", "", nil, fmt.Errorf("password is required")
		}

		fmt.Fprintln(stderr, "Authenticating...")
		r, lerr := login(ctx, email, password, endpoint)
		if lerr == nil {
			return email, password, r, nil
		}

		fmt.Fprintf(stderr, "Authentication failed: %v\n", lerr)
		if attempt >= configInitMaxAttempts {
			return "", "", nil, fmt.Errorf("authentication failed after %d attempts", configInitMaxAttempts)
		}
		fmt.Fprint(stderr, "Try again? [Y/n]: ")
		ans, _ := reader.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans == "n" || ans == "no" {
			return "", "", nil, fmt.Errorf("aborted")
		}
	}
}

// readTerminalPassword is the production passwordReader. Reads no-echo from
// the controlling terminal.
func readTerminalPassword() (string, error) {
	pwBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	return string(pwBytes), nil
}

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

		// Prompt and authenticate, with up to configInitMaxAttempts on auth
		// failure. Avoids the failure mode where a typo aborts the whole flow.
		email, password, result, err := authenticateWithRetry(
			cmd.Context(), reader, os.Stderr,
			api.Login, readTerminalPassword, api.DefaultEndpoint,
		)
		if err != nil {
			return err
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
