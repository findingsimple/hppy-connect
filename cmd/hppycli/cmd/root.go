package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/findingsimple/hppy-connect/internal/api"
	"github.com/findingsimple/hppy-connect/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile      string
	cfg          *config.Config
	apiClient    *api.Client
	outputFormat string

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
		if cmd.Name() == "help" || cmd.Name() == "version" || cmd.RunE == nil {
			return nil
		}

		// Validate output format early
		if !validOutputFormats[outputFormat] {
			return fmt.Errorf("invalid output format %q: valid options are text, json, csv, raw", outputFormat)
		}

		// Default config path
		configPath := cfgFile
		if configPath == "" {
			home, err := os.UserHomeDir()
			if err == nil {
				configPath = filepath.Join(home, ".hppycli.yaml")
			}
		}

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
		if cfg.Endpoint != "" {
			opts = append(opts, api.WithEndpoint(cfg.Endpoint))
		}
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
}

// outputData holds structured data for the printOutput helper.
type outputData struct {
	Headers []string
	Rows    [][]string
	Items   any
	Count   int
	RawJSON json.RawMessage
}

func printOutput(data outputData) error {
	switch outputFormat {
	case "json":
		wrapper := map[string]any{
			"count":    data.Count,
			"returned": len(data.Rows),
			"items":    data.Items,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(wrapper)

	case "csv":
		w := csv.NewWriter(os.Stdout)
		if err := w.Write(data.Headers); err != nil {
			return err
		}
		for _, row := range data.Rows {
			sanitized := make([]string, len(row))
			for i, cell := range row {
				sanitized[i] = sanitizeCSVCell(cell)
			}
			if err := w.Write(sanitized); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()

	case "raw":
		if data.RawJSON != nil {
			var buf bytes.Buffer
			if err := json.Indent(&buf, data.RawJSON, "", "  "); err != nil {
				os.Stdout.Write(data.RawJSON)
			} else {
				buf.WriteTo(os.Stdout)
			}
			fmt.Println()
		}
		return nil

	default: // "text"
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, strings.Join(data.Headers, "\t"))
		for _, row := range data.Rows {
			sanitized := make([]string, len(row))
			for i, cell := range row {
				sanitized[i] = sanitizeCell(cell)
			}
			fmt.Fprintln(w, strings.Join(sanitized, "\t"))
		}
		return w.Flush()
	}
}

// formatAddress formats an address as a single line.
func formatAddress(line1, line2, city, state, postalCode string) string {
	parts := []string{}
	if line1 != "" {
		parts = append(parts, line1)
	}
	if line2 != "" {
		parts = append(parts, line2)
	}
	cityState := []string{}
	if city != "" {
		cityState = append(cityState, city)
	}
	if state != "" {
		cityState = append(cityState, state)
	}
	if len(cityState) > 0 {
		parts = append(parts, strings.Join(cityState, ", "))
	}
	if postalCode != "" {
		parts = append(parts, postalCode)
	}
	return strings.Join(parts, ", ")
}

