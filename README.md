# hppy-connect

CLI and MCP server for the HappyCo platform API. Built during the 2-day HappyCo AI Hackathon (April 2026).

**hppycli** — a command-line tool for querying properties, units, inspections, and work orders.
**hppymcp** — a Model Context Protocol server that exposes the same data to AI assistants (Claude Code, Cursor, etc.).

Both binaries share the same internal API client and configuration.

## Installation

### From source

```bash
git clone https://github.com/findingsimple/hppy-connect.git
cd hppy-connect
make build
```

Binaries are written to `bin/hppycli` and `bin/hppymcp`. To install to `$GOPATH/bin`:

```bash
make install
```

### go install

```bash
go install github.com/findingsimple/hppy-connect/cmd/hppycli@latest
go install github.com/findingsimple/hppy-connect/cmd/hppymcp@latest
```

## Quick Start

1. Create a configuration file:

   ```bash
   hppycli config init
   ```

   This prompts for your HappyCo email, password, and account ID, then writes `~/.hppycli.yaml` (chmod 600).

2. Verify your credentials:

   ```bash
   hppycli account
   ```

3. List properties:

   ```bash
   hppycli properties list
   ```

## CLI Commands

| Command | Description |
|---------|-------------|
| `hppycli account` | Show account details |
| `hppycli properties list` | List properties |
| `hppycli units list --property-id <id>` | List units for a property |
| `hppycli inspections list` | List inspections |
| `hppycli workorders list` | List work orders |
| `hppycli config init` | Create config file interactively |
| `hppycli config show` | Display current configuration (password masked) |
| `hppycli mcp setup` | Generate MCP server config for AI clients |
| `hppycli completion <shell>` | Generate shell completion scripts |
| `hppycli version` | Print version information |

### Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Path to config file (default `~/.hppycli.yaml`) |
| `--output` | Output format: `text`, `json`, `csv`, `raw` (default `text`) |

### List Filters

The `inspections list` and `workorders list` commands support these filter flags:

| Flag | Description |
|------|-------------|
| `--limit` | Maximum number of results (0 = default cap) |
| `--status` | Filter by status (single value) |
| `--property-id` | Filter by property ID |
| `--unit-id` | Filter by unit ID (mutually exclusive with `--property-id`) |
| `--created-after` | Filter by creation date (RFC 3339 or YYYY-MM-DD) |
| `--created-before` | Filter by creation date (RFC 3339 or YYYY-MM-DD) |

The `properties list` command only supports `--limit`.

## MCP Server Setup

Generate configuration for your AI client:

```bash
# Claude Code / Claude Desktop
hppycli mcp setup --client claude

# Cursor
hppycli mcp setup --client cursor
```

The output is a JSON snippet to add to your client's MCP configuration. The server binary (`hppymcp`) runs via stdio transport and reads the same `~/.hppycli.yaml` config file.

### MCP Tools

The MCP server exposes tools for listing properties, units, inspections, and work orders, plus retrieving account details.

### MCP Resources

- `happyco://account` — current account information
- `happyco://properties/{property_id}` — property details including address and unit count

### MCP Prompts

- `property_summary` — summarise units and open work orders for a property
- `maintenance_report` — generate a maintenance status report with work orders and inspections

## Configuration

Config file location: `~/.hppycli.yaml`

```yaml
email: admin@example.com
password: secret
account_id: "54522"
endpoint: https://externalgraph.happyco.com
debug: false
```

### Environment Variables

Environment variables override config file values:

| Variable | Config Key |
|----------|------------|
| `HAPPYCO_EMAIL` | `email` |
| `HAPPYCO_PASSWORD` | `password` |
| `HAPPYCO_ACCOUNT_ID` | `account_id` |
| `HAPPYCO_ENDPOINT` | `endpoint` |
| `HAPPYCO_DEBUG` | `debug` |

### Precedence

Flags > Environment variables > Config file > Defaults

## Output Formats

| Format | Description |
|--------|-------------|
| `text` | Human-readable table (default) |
| `json` | JSON array |
| `csv` | CSV with headers |
| `raw` | Raw GraphQL JSON response |

```bash
hppycli properties list --output json
hppycli workorders list --output csv
```

## Security

- Config file is created with **chmod 600** (owner read/write only)
- No `--password` CLI flag — passwords are never visible in process lists
- Credentials are stored in plaintext in the config file (accepted trade-off for hackathon scope)
- Debug logging only outputs request metadata (URL, status, duration, size) — credentials and auth headers are never included
- Auth tokens are stored in memory only (not persisted to disk)
- All API communication uses HTTPS
- `config show` masks the password in output

## Limitations

- **Hackathon scope** — built in 2 days, not production-hardened
- **Account login only** — requires non-SSO admin credentials (no OAuth/SSO support)
- **No plugin auth** — single auth mechanism (email/password)
- **No token refresh** — re-authenticates on each command invocation
- **Read-only** — CLI and MCP server only query data (no create/update/delete)
- **No offline mode** — requires network access to the HappyCo API

## Development

```bash
make build     # Build both binaries
make test      # Run all tests
make cover     # Generate coverage report
make lint      # Run go fmt and go vet
make clean     # Remove build artifacts
make install   # Install binaries to $GOPATH/bin
```
