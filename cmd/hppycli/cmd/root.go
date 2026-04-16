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

var (
	cfgFile      string
	cfg          *config.Config
	apiClient    *api.Client
	outputFormat string

	// configAccountID is populated from the config file after loading.
	// Mutation commands that need --account-id can default to this value.
	configAccountID string

	versionStr string
	commitStr  string
	buildStr   string
)

var validOutputFormats = map[string]bool{
	"text": true, "json": true, "csv": true, "raw": true,
}

// SetVersionInfo sets version metadata from build-time ldflags.
func SetVersionInfo(version, commit, buildDate string) {
	versionStr = version
	commitStr = commit
	buildStr = buildDate
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("hppycli %s (commit: %s, built: %s)\n", version, commit, buildDate))
}

var rootCmd = &cobra.Command{
	Use:   "hppycli",
	Short: "HappyCo CLI tool",
	Long: `Command-line interface for the HappyCo external GraphQL API.

Getting started:
  1. Run 'hppycli config init' to create a config file
  2. Or set HAPPYCO_EMAIL, HAPPYCO_PASSWORD, HAPPYCO_ACCOUNT_ID environment variables
  3. Run 'hppycli account' to verify your connection

Available domains: account, properties, units (read), work orders, inspections,
projects, users, memberships, roles, webhooks (read + write).

Utilities: seed (populate test data), config, completion, mcp, version.

Use 'hppycli [command] --help' for details on any command.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for commands that don't need the API client.
		// Convention: all leaf commands that call the API must use RunE (not Run).
		// Parent/grouping commands (RunE == nil) just show help and skip auth.
		// Use full command path to avoid collisions with generic names like "init" or "show".
		skipPaths := map[string]bool{
			"hppycli help":        true,
			"hppycli version":     true,
			"hppycli completion":  true,
			"hppycli config init": true,
			"hppycli config show": true,
			"hppycli mcp setup":   true,
		}
		if skipPaths[cmd.CommandPath()] || cmd.RunE == nil {
			return nil
		}

		// Validate output format early
		if !validOutputFormats[outputFormat] {
			return fmt.Errorf("invalid output format %q: valid options are text, json, csv, raw", outputFormat)
		}

		configPath := resolveConfigPath()

		flags := make(map[string]string)
		if cmd.Flags().Changed("email") {
			v, _ := cmd.Flags().GetString("email")
			flags["email"] = v
		}
		if cmd.Flags().Changed("account-id") {
			v, _ := cmd.Flags().GetString("account-id")
			flags["account_id"] = v
		}
		if cmd.Flags().Changed("endpoint") {
			v, _ := cmd.Flags().GetString("endpoint")
			flags["endpoint"] = v
		}
		if cmd.Flags().Changed("debug") {
			v, _ := cmd.Flags().GetBool("debug")
			flags["debug"] = fmt.Sprintf("%t", v)
		}

		var err error
		cfg, err = config.LoadConfig(configPath, os.Getenv, flags)
		if err != nil {
			return err
		}

		configAccountID = cfg.AccountID

		if cfg.Email == "" || cfg.Password == "" {
			return fmt.Errorf(`missing required configuration (email and password)

Run 'hppycli config init' to create a config file, or set environment variables:
  export HAPPYCO_EMAIL=your-email@example.com
  export HAPPYCO_PASSWORD=your-password
  export HAPPYCO_ACCOUNT_ID=your-account-id`)
		}

		// If account ID is missing, try to resolve it interactively.
		if cfg.AccountID == "" {
			accountID, err := resolveAccountInteractive(cmd, cfg)
			if err != nil {
				return err
			}
			cfg.AccountID = accountID
			configAccountID = accountID
		}

		var opts []api.Option
		if cfg.Debug {
			opts = append(opts, api.WithDebug(true))
		}
		opts = append(opts, api.WithEndpoint(cfg.Endpoint))
		apiClient, err = api.NewClient(cfg.Email, cfg.Password, cfg.AccountID, opts...)
		if err != nil {
			return err
		}

		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// resolveAccountInteractive authenticates, discovers accessible accounts,
// and prompts the user to select one if multiple are available.
// Requires an interactive terminal — fails with a clear message otherwise.
func resolveAccountInteractive(cmd *cobra.Command, cfg *config.Config) (string, error) {
	if !term.IsTerminal(int(syscall.Stdin)) {
		return "", fmt.Errorf(`account ID is required but not configured

Set it via one of:
  hppycli config init                          # interactive setup
  --account-id=<id>                            # CLI flag
  export HAPPYCO_ACCOUNT_ID=your-account-id    # environment variable`)
	}

	stderr := cmd.ErrOrStderr()
	fmt.Fprintln(stderr, "Account ID not configured — discovering accessible accounts...")

	result, err := api.Login(cmd.Context(), cfg.Email, cfg.Password, cfg.Endpoint)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}
	if len(result.AccountIDs) == 0 {
		return "", fmt.Errorf("no accessible accounts found for this user")
	}

	choices := resolveAccountNames(cmd.Context(), result.AccountIDs, cfg.Email, cfg.Password, cfg.Endpoint, result.Token, result.ExpiresAt)

	// Single buffered reader for all stdin prompts — prevents data loss from
	// mixing buffered and unbuffered reads on the same file descriptor.
	stdinReader := bufio.NewReader(os.Stdin)
	accountID, err := selectAccount(choices, stdinReader, stderr)
	if err != nil {
		return "", err
	}

	// Auto-save for single accounts (no choice was made, so no need to confirm).
	// Prompt for multiple accounts.
	configPath := resolveConfigPath()
	if len(result.AccountIDs) == 1 {
		if err := saveAccountToConfig(configPath, accountID); err != nil {
			fmt.Fprintf(stderr, "Warning: could not save to config: %v\n", err)
		} else {
			fmt.Fprintf(stderr, "Saved account ID to %s\n", configPath)
		}
	} else {
		fmt.Fprintf(stderr, "\nSave account ID %s to %s? [Y/n] ", accountID, configPath)
		line, readErr := stdinReader.ReadString('\n')
		response := strings.TrimSpace(line)
		if readErr != nil && response == "" {
			// EOF or read error — don't save without explicit consent.
			fmt.Fprintln(stderr)
		} else if response == "" || response == "y" || response == "Y" || response == "yes" {
			if err := saveAccountToConfig(configPath, accountID); err != nil {
				fmt.Fprintf(stderr, "Warning: could not save to config: %v\n", err)
			} else {
				fmt.Fprintf(stderr, "Saved.\n")
			}
		}
	}

	return accountID, nil
}

// saveAccountToConfig reads the existing config file, updates the account_id,
// and writes it back atomically. Creates the file if it does not exist.
// Returns an error if the existing file contains invalid YAML (to prevent
// silently destroying other config fields like email and password).
func saveAccountToConfig(configPath, accountID string) error {
	data := make(map[string]interface{})
	existing, err := os.ReadFile(configPath)
	if err == nil {
		if err := yaml.Unmarshal(existing, &data); err != nil {
			return fmt.Errorf("existing config has invalid YAML: %w", err)
		}
	}
	data["account_id"] = accountID

	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	// Atomic write: temp file in same directory + rename prevents partial writes
	// and ensures 0600 permissions even if the existing file was loosened.
	dir := filepath.Dir(configPath)
	tmp, err := os.CreateTemp(dir, ".hppycli-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, configPath)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.hppycli.yaml)")
	rootCmd.PersistentFlags().String("email", "", "HappyCo email")
	rootCmd.PersistentFlags().String("account-id", "", "HappyCo account ID")
	rootCmd.PersistentFlags().String("endpoint", "", "GraphQL endpoint URL")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "text", "output format: text, json, csv, raw")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging to stderr")

	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(propertiesCmd)
	rootCmd.AddCommand(unitsCmd)
	rootCmd.AddCommand(workordersCmd)
	rootCmd.AddCommand(inspectionsCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(membershipsCmd)
	rootCmd.AddCommand(rolesCmd)
	rootCmd.AddCommand(webhooksCmd)
	rootCmd.AddCommand(seedCmd)
}
