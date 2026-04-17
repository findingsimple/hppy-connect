package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/findingsimple/hppy-connect/internal/api"
	"github.com/findingsimple/hppy-connect/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/term"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

const serverInstructions = `HappyCo property management API — read and manage account properties, units, work orders, inspections, projects, users, roles, and webhooks. Includes 71 mutation tools for creating and modifying resources. Property ID is required for unit queries. Use resources for quick lookups; use tools for filtered/paginated queries.

Scope limitation: This server does not include list/query tools for users, roles, projects, webhooks, or templates. Ask the user to provide IDs for these entities when needed by mutation tools. Use list_members to find user IDs via account memberships.`

func main() {
	configPath := flag.String("config", "", "path to config file")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	// Resolve default config path
	cfgPath := *configPath
	if cfgPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			cfgPath = filepath.Join(home, ".hppycli.yaml")
		}
	}

	flags := map[string]string{}
	if *debug {
		flags["debug"] = "true"
	}

	cfg, err := config.LoadConfig(cfgPath, os.Getenv, flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nTo get started, create ~/.hppycli.yaml with:\n")
		fmt.Fprintf(os.Stderr, "  email: your@email.com\n")
		fmt.Fprintf(os.Stderr, "  password: your-password\n")
		fmt.Fprintf(os.Stderr, "  account_id: \"your-account-id\"\n")
		os.Exit(1)
	}

	if cfg.Email == "" || cfg.Password == "" || cfg.AccountID == "" {
		fmt.Fprintf(os.Stderr, "Error: email, password, and account_id are required\n")
		fmt.Fprintf(os.Stderr, "\nConfigure via ~/.hppycli.yaml or environment variables:\n")
		fmt.Fprintf(os.Stderr, "  HAPPYCO_EMAIL, HAPPYCO_PASSWORD, HAPPYCO_ACCOUNT_ID\n")
		os.Exit(1)
	}

	var opts []api.Option
	if cfg.Debug {
		opts = append(opts, api.WithDebug(true))
	}
	opts = append(opts, api.WithEndpoint(cfg.Endpoint))

	client, err := api.NewClient(cfg.Email, cfg.Password, cfg.AccountID, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating API client: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Auth is intentionally lazy — the API client authenticates on the first
	// tool/resource call and handles token refresh automatically. Eager auth
	// here would crash the MCP server on startup if credentials are temporarily
	// invalid (e.g. expired token, network blip), causing Claude Desktop to
	// show a cryptic "server disconnected" error instead of a actionable
	// per-tool error message.
	if cfg.Debug {
		log.Printf("[debug] hppymcp %s (commit=%s, built=%s) starting (auth deferred to first request)", version, commit, buildDate)
	}

	server := mcp.NewServer(
		&mcp.Implementation{Name: "hppymcp", Version: version},
		&mcp.ServerOptions{Instructions: serverInstructions},
	)

	registerTools(server, client, cfg.Debug)
	registerResources(server, client)
	registerPrompts(server)

	// If stdin is a terminal, the user is running this binary directly rather
	// than through an MCP client. The server will read JSON-RPC from stdin,
	// get nothing useful, and eventually exit with a confusing message. Print
	// a banner so they know this isn't a standalone CLI.
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(os.Stderr, "hppymcp is an MCP stdio server — it's not meant to be run directly.")
		fmt.Fprintln(os.Stderr, "Register it with your MCP client first:")
		fmt.Fprintln(os.Stderr, "    hppycli mcp setup --client claude         # Claude Code")
		fmt.Fprintln(os.Stderr, "    hppycli mcp setup --client claude-desktop # Claude Desktop")
		fmt.Fprintln(os.Stderr, "    hppycli mcp setup --client cursor         # Cursor")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Waiting for MCP messages on stdin (ctrl-D to exit)...")
	}

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
