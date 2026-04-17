package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
		apply, _ := cmd.Flags().GetBool("apply")

		binaryPath := detectMcpBinary()
		configPath := resolveConfigPath()

		switch client {
		case "claude":
			if apply {
				return applyClaudeConfig(cmd, binaryPath, configPath)
			}
			return printClaudeConfig(os.Stdout, binaryPath, configPath)
		case "claude-desktop":
			if apply {
				return fmt.Errorf("--apply is not supported for claude-desktop (config is JSON, not a CLI command); paste the printed JSON manually")
			}
			return printClaudeDesktopConfig(os.Stdout, binaryPath, configPath)
		case "cursor":
			if apply {
				return fmt.Errorf("--apply is not supported for cursor (config is JSON, not a CLI command); paste the printed JSON manually")
			}
			return printCursorConfig(os.Stdout, binaryPath, configPath)
		default:
			return fmt.Errorf("unsupported --client %q: valid options are claude, claude-desktop, cursor", client)
		}
	},
}

// applyClaudeConfig executes the `claude mcp add` command directly so the user
// doesn't have to copy-paste. Requires the `claude` CLI to be on PATH.
//
// Prints the resolved `claude` path BEFORE executing so the user can spot a
// PATH-shadowing situation (a malicious `claude` binary earlier in PATH would
// otherwise run silently with the absolute config path as an argument). If the
// resolved path looks wrong, ctrl-C before it runs.
func applyClaudeConfig(cmd *cobra.Command, binaryPath, configPath string) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("--apply requires the `claude` CLI on your PATH (https://docs.claude.com/en/docs/claude-code): %w", err)
	}
	args := []string{"mcp", "add", "--transport", "stdio", "--scope", "user", "hppymcp", "--", binaryPath, "--config", configPath}
	c := exec.CommandContext(cmd.Context(), claudePath, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	fmt.Fprintf(os.Stdout, "Resolved `claude` to: %s\n", claudePath)
	fmt.Fprintln(os.Stdout, "(if that path looks wrong — e.g. a shim from a recently-installed npm package — ctrl-C now)")
	fmt.Fprintf(os.Stdout, "\nRunning: %s %s\n\n", claudePath, strings.Join(args, " "))
	return c.Run()
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

// shellQuote wraps a string in single quotes with proper escaping for POSIX shells.
// Interior single quotes are replaced with the standard '\” sequence.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func printClaudeConfig(w io.Writer, binaryPath, configPath string) error {
	fmt.Fprintln(w, "Run the following command to register hppymcp with Claude Code:")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  claude mcp add --transport stdio --scope user hppymcp -- %s --config %s\n", shellQuote(binaryPath), shellQuote(configPath))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Or re-run with --apply to execute it for you:")
	fmt.Fprintln(w, "  hppycli mcp setup --client claude --apply")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Then restart Claude Code and ask Claude to call the `get_account` tool to verify.")
	return nil
}

func printClaudeDesktopConfig(w io.Writer, binaryPath, configPath string) error {
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

	fmt.Fprintln(w, "Add the following to your Claude Desktop config")
	fmt.Fprintln(w, "(~/Library/Application Support/Claude/claude_desktop_config.json on macOS,")
	fmt.Fprintln(w, " %APPDATA%\\Claude\\claude_desktop_config.json on Windows):")
	fmt.Fprintln(w)
	fmt.Fprintln(w, string(out))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "To allow the list_members tool to return user email addresses, add an `env`")
	fmt.Fprintln(w, "block to the hppymcp entry above (JSON doesn't allow comments — paste the")
	fmt.Fprintln(w, "snippet below into the same hppymcp object):")
	fmt.Fprintln(w)
	fmt.Fprintln(w, `      "env": { "HPPYMCP_ALLOW_EMAIL_DISCLOSURE": "1" }`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Then restart Claude Desktop. The hppymcp tools will appear in the MCP picker.")
	fmt.Fprintln(w, "Note: Claude Desktop launches MCP servers outside your shell, so the config")
	fmt.Fprintln(w, "file must contain email, password, and account_id directly — environment")
	fmt.Fprintln(w, "variables (including HPPYMCP_ALLOW_EMAIL_DISCLOSURE) must be in the env")
	fmt.Fprintln(w, "block above, not in your shell.")
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
	mcpSetupCmd.Flags().String("client", "claude", "AI client: claude (Claude Code CLI), claude-desktop, cursor")
	mcpSetupCmd.Flags().Bool("apply", false, "for --client=claude only: actually run `claude mcp add` instead of printing it")
	mcpCmd.AddCommand(mcpSetupCmd)
	rootCmd.AddCommand(mcpCmd)
}
