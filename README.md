<div align="center">

<img src="docs/images/HappyCo-Logo-Green-Circle.svg" alt="HappyCo" width="80" />

# hppy-connect

**The HappyCo platform, from your terminal and your AI.**

A CLI and MCP server that gives developers and AI assistants full access to the HappyCo External GraphQL API — properties, inspections, work orders, projects, users, roles, webhooks, and more.

Built during the 2-day HappyCo AI Hackathon (April 2026).

[Installation](#installation) | [Getting Started](#quick-start) | [CLI Commands](#cli-commands) | [MCP Server](#mcp-server-setup) | [Architecture](#architecture)

</div>

---

### `hppycli` — Command Line

A full-featured CLI for managing your HappyCo account. Query properties and inspections, create work orders, manage users and roles, seed test data — all from the terminal with tab completion, multiple output formats, and scriptable JSON output.

<img src="docs/images/cli-demo.gif" alt="hppycli demo" width="100%" />

### `hppymcp` — AI Assistant

An MCP server that exposes the same capabilities to Claude Code, Cursor, and other AI clients. One command to connect, 77 tools across 8 domains. Your AI can read, create, and manage HappyCo data with built-in safety guards on destructive operations.

```bash
hppycli mcp setup --client claude
```

Both binaries share the same internal Go API client and configuration — one codebase, two interfaces.

---

## Possible Use Cases

### Morning Maintenance Triage

A maintenance lead reviews the day's queue, re-prioritises urgent items, and assigns work — all through conversation.

> "Show me all open work orders for Sunset Apartments. Which ones are urgent? Reassign the water heater one to Maria and schedule it for today."

Tools: `list_properties` → `list_work_orders` → `list_members` → `work_order_set_assignee` → `work_order_set_priority` → `work_order_set_scheduled_for` → `work_order_add_comment`

### Emergency After-Hours Intake

A resident calls about a burst pipe at 10pm. One prompt creates the work order, assigns the on-call vendor, sets entry permissions, and documents everything.

> "Create an urgent work order at unit 4B in Sunset Apartments for a burst pipe under the kitchen sink. Assign it to PlumbFast, set permission to enter, and add entry notes that the resident gave verbal permission and the key is in lockbox #3."

Tools: `list_properties` → `list_units` → `work_order_create` (with priority, assignee, entry permissions) → `work_order_add_comment`

### Unit Turn / Make-Ready Coordination

A resident is moving out. The coordinator sets up the full turnover workflow in a single conversation — inspection, project, and individual work orders.

> "Unit 12A at Oakwood has a move-out on May 1st. Schedule a move-out inspection for May 2nd. Create a turn project due May 15th with availability target May 16th. Then create work orders for painting, carpet cleaning, and appliance inspection — all as TURN type, assigned to the maintenance lead."

Tools: `list_units` → `inspection_create` → `project_create` → `project_set_availability_target_at` → `work_order_create` ×3 → `work_order_set_assignee` ×3 → `work_order_set_scheduled_for` ×3

### New Employee Onboarding

A new maintenance tech starts Monday. Create their account, assign the right role, and grant access to their properties.

> "Create a user for Carlos Ramirez, email carlos@example.com. Give him the Maintenance Tech role and grant access to Sunset Apartments, Oakwood Terrace, and Riverside Commons."

Tools: `user_create` → `membership_create` → `membership_set_roles` → `list_properties` → `user_grant_property_access` ×3

### Inspection Compliance Check

Company policy requires monthly inspections. Identify which properties are behind and bulk-schedule to close the gaps.

> "List all inspections completed in the last 30 days for each property. Which properties are behind based on their unit count? Schedule inspections for the missing units across the next two weeks."

Tools: `list_properties` → `list_units` (per property) → `list_inspections` (per property) → Claude calculates gaps → `inspection_create` ×N → `inspection_set_assignee` ×N

### Weekly Portfolio Report

Before the weekly ops meeting, get a cross-property snapshot of maintenance health.

> "Give me a maintenance report for all properties covering the last 30 days. Include open work order counts, recently completed work, and inspection completion rates."

Tools: `list_properties` → `list_work_orders` (per property) → `list_inspections` (per property) → Claude synthesizes report

### Work Order Lifecycle with Time Tracking

Track a complex repair from start to finish — timer, progress notes, parts hold, and completion.

> "Start the timer on the HVAC work order. Add a comment that the compressor has been removed and the new part arrives tomorrow. Put it on hold."

> *Next day:* "Take the HVAC work order off hold, stop the timer, log 4 hours 30 minutes, and mark it completed with a note that the compressor is replaced and tested."

Tools: `work_order_start_timer` → `work_order_add_comment` → `work_order_set_status` (ON_HOLD) → `work_order_set_status` (OPEN) → `work_order_stop_timer` → `work_order_add_time` → `work_order_add_comment` → `work_order_set_status` (COMPLETED)

### Integration Setup

Connect HappyCo to your internal systems with webhook subscriptions.

> "Set up a webhook that sends inspection and work order events to https://hooks.ourcompany.com/happyco."

Tools: `get_account` → `webhook_create`

---

### Seed Test Data

Want to try these workflows? The `seed` command populates your account with realistic test data — work orders, inspections, projects, and webhooks spread across your properties.

```bash
# Preview what will be created
hppycli seed --dry-run

# Create 10 entities of each type
hppycli seed --count 10 --yes

# With specific templates
hppycli seed --inspection-template-id=tmpl123 --project-template-id=ptmpl456 --yes

# Get created entity IDs as JSON (useful for scripting demos)
hppycli seed --count 5 --output json --yes
```

Seeded entities are tagged with `[SEED <timestamp>]` in their descriptions for easy identification and cleanup.

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

The shared API client in `internal/` handles authentication, Relay-style cursor pagination, retries with exponential backoff, mid-pagination auth recovery, and write mutations with appropriate retry semantics. Both binaries are thin frontends over this shared logic.

## Installation

### Requirements

- **Go 1.26+** — [install Go](https://go.dev/doc/install)
- `$HOME/go/bin` must be on your `PATH` for installed binaries to work:
  ```bash
  # Add to ~/.zshrc or ~/.bashrc
  export PATH="$HOME/go/bin:$PATH"
  ```

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

   This prompts for your HappyCo email and password, authenticates, discovers your accessible accounts, and writes `~/.hppycli.yaml` (chmod 600). If you have multiple accounts, you'll be prompted to select one.

2. Enable tab completion (optional):

   ```bash
   # Zsh (add to ~/.zshrc)
   source <(hppycli completion zsh)

   # Bash (add to ~/.bashrc)
   source <(hppycli completion bash)
   ```

3. Verify your credentials:

   ```bash
   hppycli account
   ```

4. List properties:

   ```bash
   hppycli properties list
   ```

> **Note:** If `account_id` is not set in your config, the CLI will automatically authenticate and discover your accessible accounts on first use (interactive terminal required). Single accounts are auto-saved; multiple accounts prompt you to select one and offer to save the choice.

## CLI Commands

### Queries

| Command | Description |
|---------|-------------|
| `hppycli account` | Show account details |
| `hppycli properties list` | List properties |
| `hppycli units list --property-id <id>` | List units for a property |
| `hppycli inspections list` | List inspections |
| `hppycli workorders list` | List work orders |

### Work Order Mutations (19)

| Command | Description |
|---------|-------------|
| `workorders create` | Create a work order |
| `workorders set-status` | Set status and optional sub-status |
| `workorders set-assignee` | Assign to a user or vendor |
| `workorders set-description` | Update description |
| `workorders set-priority` | Set priority (NORMAL/URGENT) |
| `workorders set-scheduled-for` | Set scheduled date |
| `workorders set-location` | Change location |
| `workorders set-type` | Set work order type |
| `workorders set-entry-notes` | Set entry notes |
| `workorders set-permission-to-enter` | Set permission to enter flag |
| `workorders set-resident-approved-entry` | Set resident approved entry flag |
| `workorders set-unit-entered` | Set unit entered flag |
| `workorders archive` | Archive a work order |
| `workorders add-comment` | Add a comment |
| `workorders add-time` | Log time (ISO 8601 duration) |
| `workorders add-attachment` | Add an attachment (returns signed upload URL) |
| `workorders remove-attachment` | Remove an attachment |
| `workorders start-timer` | Start a time tracking timer |
| `workorders stop-timer` | Stop a time tracking timer |

### Inspection Mutations (24)

| Command | Description |
|---------|-------------|
| `inspections create` | Create an inspection |
| `inspections start` | Start an inspection |
| `inspections complete` | Complete an inspection |
| `inspections reopen` | Reopen a completed inspection |
| `inspections archive` | Archive an inspection |
| `inspections expire` | Expire an inspection |
| `inspections unexpire` | Unexpire an inspection |
| `inspections set-assignee` | Set assignee |
| `inspections set-due-by` | Set due date and expiry |
| `inspections set-scheduled-for` | Set scheduled date |
| `inspections set-header-field` | Set a header field value |
| `inspections set-footer-field` | Set a footer field value |
| `inspections set-item-notes` | Set notes on an item |
| `inspections rate-item` | Rate an inspection item |
| `inspections add-section` | Add a new section |
| `inspections delete-section` | Delete a section |
| `inspections duplicate-section` | Duplicate a section |
| `inspections rename-section` | Rename a section |
| `inspections add-item` | Add an item to a section |
| `inspections delete-item` | Delete an item |
| `inspections add-item-photo` | Add a photo to an item (returns signed upload URL) |
| `inspections remove-item-photo` | Remove a photo from an item |
| `inspections move-item-photo` | Move a photo between items |
| `inspections send-to-guest` | Send inspection to a guest via email |

### Project Mutations (8)

| Command | Description |
|---------|-------------|
| `projects create` | Create a project |
| `projects set-assignee` | Set or clear assignee |
| `projects set-notes` | Set project notes |
| `projects set-due-at` | Set due date |
| `projects set-start-at` | Set start date |
| `projects set-priority` | Set priority |
| `projects set-on-hold` | Set on-hold status |
| `projects set-availability-target-at` | Set availability target date |

### User Mutations (7)

| Command | Description |
|---------|-------------|
| `users create` | Create a user and optionally assign a role |
| `users set-email` | Update user email |
| `users set-name` | Update user name |
| `users set-short-name` | Set or clear short name |
| `users set-phone` | Set or clear phone number |
| `users grant-property-access` | Grant a user access to properties |
| `users revoke-property-access` | Revoke a user's property access |

### Memberships (5)

| Command | Description |
|---------|-------------|
| `memberships list` | List account memberships (`--search`, `--include-inactive`, `--limit`) |
| `memberships create` | Create an account membership |
| `memberships activate` | Activate a membership |
| `memberships deactivate` | Deactivate a membership |
| `memberships set-roles` | Set roles on a membership |

### Property Access Mutations (3)

| Command | Description |
|---------|-------------|
| `properties grant-access` | Grant users access to a property |
| `properties revoke-access` | Revoke users' property access |
| `properties set-account-wide-access` | Set account-wide access on a property |

### Role Mutations (4)

| Command | Description |
|---------|-------------|
| `roles create` | Create a role with permissions |
| `roles set-name` | Update role name |
| `roles set-description` | Set or clear role description |
| `roles set-permissions` | Modify role permissions |

### Webhook Mutations (2)

| Command | Description |
|---------|-------------|
| `webhooks create` | Create a webhook subscription |
| `webhooks update` | Update a webhook's URL, status, or subjects |

### Utility Commands

| Command | Description |
|---------|-------------|
| `hppycli seed` | Populate account with test data (work orders, inspections, projects, webhooks) |
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

### Destructive Operations

Commands that archive, delete, revoke access, or change permissions prompt for confirmation before executing. Use `--yes` to skip the prompt (e.g. in scripts):

```bash
hppycli workorders archive --id=abc123          # prompts: "About to archive work order. Continue? [y/N]"
hppycli workorders archive --id=abc123 --yes    # skips confirmation
```

The `--yes` flag is available on: `seed`, `workorders archive`, `workorders remove-attachment`, `inspections archive`, `inspections expire`, `inspections delete-section`, `inspections delete-item`, `inspections remove-item-photo`, `memberships deactivate`, `memberships set-roles`, `properties revoke-access`, `properties set-account-wide-access`, `users revoke-property-access`, and `roles set-permissions`.

### Mutation Output

All mutation commands output JSON (formatted with indentation). If `--output text` is set, a note is printed to stderr and JSON is still used.

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

# Create a work order and then assign it (chaining with jq)
ID=$(hppycli workorders create --location-id=225393 --description="Fix leak in unit 4B" | jq -r '.id')
hppycli workorders set-assignee --id=$ID --assignee-id=U123 --assignee-type=USER
hppycli workorders set-priority --id=$ID --priority=URGENT

# Create an inspection from a template
hppycli inspections create --location-id=225393 --template-id=c71sn3-a-0-928286 --scheduled-for=2026-05-01T00:00:00Z

# Create a project and set it on hold
ID=$(hppycli projects create --template-id=TPL1 --location-id=225393 --start-at=2026-05-01T00:00:00Z | jq -r '.id')
hppycli projects set-on-hold --id=$ID --on-hold=true

# User management
hppycli users create --email=new@example.com --name="New User" --role-id=ROLE1
hppycli roles create --name="Inspector" --grant=inspection:inspection.create,inspection:inspection.view

# Webhook setup
hppycli webhooks create --subscriber-id=54522 --subscriber-type=ACCOUNT --url=https://hooks.example.com/happyco --subjects=INSPECTIONS,WORK_ORDERS

# Seed test data (preview first, then create)
hppycli seed --dry-run
hppycli seed --yes
hppycli seed --inspection-template-id=tmpl123 --project-template-id=ptmpl456 --yes
hppycli seed --count=5 --output json --yes
```

## MCP Server Setup

### Claude Code

1. Register the MCP server:
   ```bash
   hppycli mcp setup --client claude
   # Then run the command it outputs, e.g.:
   claude mcp add --transport stdio --scope user hppymcp -- hppymcp --config ~/.hppycli.yaml
   ```
2. Restart Claude Code. Ask "What HappyCo account am I connected to?" to verify.

### Claude Desktop

1. Open your config file at `~/Library/Application Support/Claude/claude_desktop_config.json`
2. Add the `mcpServers` block:
   ```json
   {
     "mcpServers": {
       "hppymcp": {
         "command": "/path/to/hppymcp",
         "args": ["--config", "/path/to/.hppycli.yaml"]
       }
     }
   }
   ```
3. Restart Claude Desktop. The hppymcp tools should appear in the MCP tool picker.

> **Note:** Claude Desktop launches MCP servers outside your shell environment, so `~/.hppycli.yaml` must contain `email`, `password`, and `account_id` directly — environment variables won't be available.

### Cursor

1. Generate the config snippet:
   ```bash
   hppycli mcp setup --client cursor
   ```
2. Add the output to `.cursor/mcp.json` in your project root (or globally in `~/.cursor/mcp.json`).
3. Restart Cursor. The hppymcp server should appear in your MCP server list.

The server binary (`hppymcp`) runs via stdio transport and reads the same `~/.hppycli.yaml` config file.

### Things to try

Once connected, try asking your AI assistant:

- "List all properties in my HappyCo account"
- "Show me open work orders for property 225393"
- "Create a work order for a leaking faucet in unit 4B"
- "Generate a maintenance report for the last 30 days"
- "Who are the users in my account and what roles do they have?"

### MCP Tools (77 total: 6 read + 71 mutation)

#### Read Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `get_account` | Get authenticated account info (name, ID) | None |
| `list_properties` | List all properties with name, address, creation date | `limit` |
| `list_units` | List units within a property | `property_id` (required), `limit` |
| `list_members` | List account members (users with memberships) | `search`, `include_inactive`, `limit` |
| `list_work_orders` | List work orders with optional filters | `property_id`, `unit_id`, `status`, `created_after`, `created_before`, `limit` |
| `list_inspections` | List inspections with optional filters | `property_id`, `unit_id`, `status`, `created_after`, `created_before`, `limit` |

Date parameters use ISO 8601 format (e.g. `2026-01-15T00:00:00Z`). Status values are the same as the CLI (see [Status Values](#status-values)).

#### Mutation Tools (71 total)

Mutation tools follow the naming pattern `{domain}_{action}` in snake_case. All ID parameters are validated against a safe character set.

| Domain | Tools | Count |
|--------|-------|-------|
| Work Orders | `work_order_create`, `work_order_set_status`, `work_order_set_assignee`, `work_order_set_description`, `work_order_set_priority`, `work_order_set_scheduled_for`, `work_order_set_location`, `work_order_set_type`, `work_order_set_entry_notes`, `work_order_set_permission_to_enter`, `work_order_set_resident_approved_entry`, `work_order_set_unit_entered`, `work_order_archive`, `work_order_add_comment`, `work_order_add_time`, `work_order_add_attachment`, `work_order_remove_attachment`, `work_order_start_timer`, `work_order_stop_timer` | 19 |
| Inspections | `inspection_create`, `inspection_start`, `inspection_complete`, `inspection_reopen`, `inspection_archive`, `inspection_expire`, `inspection_unexpire`, `inspection_set_assignee`, `inspection_set_due_by`, `inspection_set_scheduled_for`, `inspection_set_header_field`, `inspection_set_footer_field`, `inspection_set_item_notes`, `inspection_rate_item`, `inspection_add_section`, `inspection_delete_section`, `inspection_duplicate_section`, `inspection_rename_section`, `inspection_add_item`, `inspection_delete_item`, `inspection_add_item_photo`, `inspection_remove_item_photo`, `inspection_move_item_photo`, `inspection_send_to_guest` | 24 |
| Projects | `project_create`, `project_set_assignee`, `project_set_notes`, `project_set_due_at`, `project_set_start_at`, `project_set_priority`, `project_set_on_hold`, `project_set_availability_target_at` | 8 |
| Users | `user_create`, `user_set_email`, `user_set_name`, `user_set_short_name`, `user_set_phone`, `user_grant_property_access`, `user_revoke_property_access` | 7 |
| Memberships | `membership_create`, `membership_activate`, `membership_deactivate`, `membership_set_roles` | 4 |
| Properties | `property_grant_access`, `property_revoke_access`, `property_set_account_wide_access` | 3 |
| Roles | `role_create`, `role_set_name`, `role_set_description`, `role_set_permissions` | 4 |
| Webhooks | `webhook_create`, `webhook_update` | 2 |

Destructive mutation tools (archive, delete, revoke, etc.) are annotated with `DestructiveHint` so MCP clients can gate them with human-in-the-loop confirmation.

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
- Webhook URLs are validated: must use HTTPS, no private/internal IPs, no cloud metadata endpoints
- Destructive CLI operations require interactive confirmation (bypass with `--yes`)
- Destructive MCP tools are annotated with `DestructiveHint` for client-side gating

## Limitations

- **Hackathon scope** — built in 2 days, not production-hardened
- **Account login only** — requires non-SSO admin credentials (no OAuth/SSO support)
- **No plugin auth** — single auth mechanism (email/password); Plugin domain mutations are excluded (see CLAUDE.md for rationale)
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

Run `hppycli config init` or set the environment variables `HAPPYCO_EMAIL` and `HAPPYCO_PASSWORD`. The `HAPPYCO_ACCOUNT_ID` is optional — if omitted, the CLI will discover your accessible accounts and prompt you to select one (interactive terminal required).

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
