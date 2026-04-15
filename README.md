# hppy-connect

CLI and MCP server for the HappyCo platform API. Built during the 2-day HappyCo AI Hackathon (April 2026).

**hppycli** — a command-line tool for querying properties, units, inspections, and work orders.
**hppymcp** — a Model Context Protocol server that exposes the same data to AI assistants (Claude Code, Cursor, etc.).

Both binaries share the same internal API client and configuration.

## Architecture

```
                    ┌─────────────┐
                    │  HappyCo    │
                    │  GraphQL    │
                    │  API        │
                    └──────▲──────┘
                           │
                    ┌──────┴──────┐
                    │  Shared Go  │
                    │  API Client │
                    │  (internal) │
                    └──┬──────┬───┘
                       │      │
              ┌────────┘      └────────┐
              │                        │
        ┌─────▼─────┐          ┌──────▼──────┐
        │  hppycli   │          │   hppymcp   │
        │  (Cobra)   │          │   (MCP)     │
        │            │          │             │
        │ Developers │          │ AI Clients  │
        │ Scripts    │          │ Claude      │
        │ CI/CD      │          │ Cursor      │
        └────────────┘          └─────────────┘
```

The shared API client in `internal/` handles authentication, Relay-style cursor pagination, retries with exponential backoff, and mid-pagination auth recovery. Both binaries are thin frontends over this shared logic.

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
| `--email` | HappyCo email (overrides config file) |
| `--account-id` | HappyCo account ID (overrides config file) |
| `--endpoint` | GraphQL endpoint URL (overrides config file) |
| `--debug` | Enable debug logging to stderr |

### List Filters

The `inspections list` and `workorders list` commands support these filter flags:

| Flag | Description |
|------|-------------|
| `--limit` | Maximum number of results (0 = default cap of 1000) |
| `--status` | Filter by status (see [Status Values](#status-values) below) |
| `--property-id` | Filter by property ID |
| `--unit-id` | Filter by unit ID (mutually exclusive with `--property-id`) |
| `--created-after` | Filter by creation date (RFC 3339 or YYYY-MM-DD) |
| `--created-before` | Filter by creation date (RFC 3339 or YYYY-MM-DD) |

The `properties list` command only supports `--limit`.

### Status Values

| Entity | Valid Statuses |
|--------|---------------|
| Work Orders | `OPEN`, `ON_HOLD`, `COMPLETED` |
| Inspections | `COMPLETE`, `EXPIRED`, `INCOMPLETE`, `SCHEDULED` |

Status values are case-insensitive (e.g. `--status open` is normalised to `OPEN`).

### Examples

```bash
# List open work orders for a property
hppycli workorders list --property-id 225393 --status OPEN

# Export inspections to CSV
hppycli inspections list --output csv > inspections.csv

# JSON output piped to jq
hppycli workorders list --output json | jq '.items | length'

# Filter by date range
hppycli inspections list --created-after 2026-01-01 --created-before 2026-04-01

# Raw GraphQL response for debugging
hppycli workorders list --output raw
```

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

| Tool | Description | Parameters |
|------|-------------|------------|
| `get_account` | Get authenticated account info (name, ID) | None |
| `list_properties` | List all properties with name, address, creation date | `limit` |
| `list_units` | List units within a property | `property_id` (required), `limit` |
| `list_work_orders` | List work orders with optional filters | `property_id`, `unit_id`, `status`, `created_after`, `created_before`, `limit` |
| `list_inspections` | List inspections with optional filters | `property_id`, `unit_id`, `status`, `created_after`, `created_before`, `limit` |

Date parameters use ISO 8601 format (e.g. `2026-01-15T00:00:00Z`). Status values are the same as the CLI (see [Status Values](#status-values)).

### MCP Resources

| URI | Description |
|-----|-------------|
| `happyco://account` | Current account information (name, ID) |
| `happyco://properties/{property_id}` | Property details including address and unit count |

### MCP Prompts

| Prompt | Description | Parameters |
|--------|-------------|------------|
| `property_summary` | Summarise units and open work orders for a property | `property_id` (required) |
| `maintenance_report` | Generate a maintenance status report with work orders and inspections | `property_id` (required), `days_back` (default: 30, max: 365) |

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
| `json` | Structured JSON with `count`, `returned`, and `items` fields |
| `csv` | CSV with headers (sanitised against formula injection) |
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
- MCP server sanitises error messages — internal details (URLs, HTTP codes) are never exposed to AI clients
- ID parameters are validated against a safe character set before use in API calls
- CSV output is sanitised against formula injection (cells starting with `=`, `+`, `-`, `@`)

## Limitations

- **Hackathon scope** — built in 2 days, not production-hardened
- **Account login only** — requires non-SSO admin credentials (no OAuth/SSO support)
- **No plugin auth** — single auth mechanism (email/password)
- **Read-only** — CLI and MCP server only query data (no create/update/delete)
- **No offline mode** — requires network access to the HappyCo API

## Troubleshooting

### Authentication failures

The client authenticates on first use and caches the token in memory (valid for ~1 hour). If a token expires mid-session, the client automatically re-authenticates — including during pagination. Permanent auth failures (bad credentials) trigger a 30-second cooldown to avoid hammering the API; transient errors (network issues, 500s) do not.

```bash
# Verify credentials work
hppycli account

# Check your config
hppycli config show

# Enable debug logging for request details
hppycli workorders list --debug
```

### "missing required configuration"

Run `hppycli config init` or set the environment variables `HAPPYCO_EMAIL`, `HAPPYCO_PASSWORD`, and `HAPPYCO_ACCOUNT_ID`.

### Empty results

- Check that `--property-id` or `--unit-id` values are correct
- Verify the status filter matches the entity type (e.g. `OPEN` is a work order status, not an inspection status)
- Try without filters to confirm data exists: `hppycli workorders list`

### MCP server not responding

- Verify the binary path in your MCP client config matches the actual location of `hppymcp`
- Check that `~/.hppycli.yaml` exists and has valid credentials
- Run `hppymcp` directly to see any startup errors: `hppymcp 2>/tmp/hppymcp.log`
- Set `debug: true` in config to get detailed logging on stderr

## Development

```bash
make build     # Build both binaries
make test      # Run all tests
make cover     # Generate coverage report
make lint      # Run go fmt and go vet
make clean     # Remove build artifacts
make install   # Install binaries to $GOPATH/bin
```
