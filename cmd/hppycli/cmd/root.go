package cmd

import (
	"fmt"
	"os"

	"github.com/findingsimple/hppy-connect/internal/api"
	"github.com/findingsimple/hppy-connect/internal/config"
	"github.com/spf13/cobra"
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
	Long:  "Command-line interface for the HappyCo external GraphQL API.",
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

		if cfg.Email == "" || cfg.Password == "" || cfg.AccountID == "" {
			return fmt.Errorf(`missing required configuration

Run 'hppycli config init' to create a config file, or set environment variables:
  export HAPPYCO_EMAIL=your-email@example.com
  export HAPPYCO_PASSWORD=your-password
  export HAPPYCO_ACCOUNT_ID=your-account-id`)
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
}
