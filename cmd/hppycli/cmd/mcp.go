package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server integration helpers",
}

var mcpSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Generate MCP server configuration for AI clients",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _ := cmd.Flags().GetString("client")

		binaryPath := detectMcpBinary()
		configPath := resolveConfigPath()

		switch client {
		case "claude":
			return printClaudeConfig(os.Stdout, binaryPath, configPath)
		case "cursor":
			return printCursorConfig(os.Stdout, binaryPath, configPath)
		default:
			return fmt.Errorf("unsupported --client %q: valid options are claude, cursor", client)
		}
	},
}

func detectMcpBinary() string {
	// Check $GOPATH/bin/hppymcp
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		candidate := filepath.Join(gopath, "bin", "hppymcp")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Check ~/go/bin/hppymcp (default GOPATH)
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, "go", "bin", "hppymcp")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Check ./bin/hppymcp
	if candidate, err := filepath.Abs("./bin/hppymcp"); err == nil {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Check PATH
	if path, err := exec.LookPath("hppymcp"); err == nil {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}

	// Fallback — assume PATH
	return "hppymcp"
}

func printClaudeConfig(w io.Writer, binaryPath, configPath string) error {
	cfg := map[string]interface{}{
		"hppymcp": map[string]interface{}{
			"command": binaryPath,
			"args":    []string{"--config", configPath},
		},
	}

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config JSON: %w", err)
	}

	fmt.Fprintln(w, "Add the following to your Claude Code MCP settings")
	fmt.Fprintln(w, "(~/.claude/settings.json → mcpServers):")
	fmt.Fprintln(w)
	fmt.Fprintln(w, string(out))
	return nil
}

func printCursorConfig(w io.Writer, binaryPath, configPath string) error {
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"hppymcp": map[string]interface{}{
				"command": binaryPath,
				"args":    []string{"--config", configPath},
			},
		},
	}

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config JSON: %w", err)
	}

	fmt.Fprintln(w, "Add the following to your Cursor MCP settings")
	fmt.Fprintln(w, "(.cursor/mcp.json):")
	fmt.Fprintln(w)
	fmt.Fprintln(w, string(out))
	return nil
}

func init() {
	mcpSetupCmd.Flags().String("client", "claude", "AI client: claude, cursor")
	mcpCmd.AddCommand(mcpSetupCmd)
	rootCmd.AddCommand(mcpCmd)
}
