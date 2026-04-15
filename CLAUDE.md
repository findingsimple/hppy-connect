# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This repository (`~/hppy-connect`) is a project built during a 2-day HappyCo AI Hackathon (April 2026).

## Naming Conventions

| Thing       | Name       |
|-------------|------------|
| Repository  | hppy-connect |
| CLI command | hppycli    |
| MCP server  | hppymcp    |

## Build Commands

```bash
make build     # Build both hppycli and hppymcp to bin/
make test      # Run all tests (go test ./... -v -count=1)
make lint      # Run go fmt and go vet
make cover     # Generate coverage report (coverage.html)
make clean     # Remove bin/, coverage files
make install   # Install both binaries to $GOPATH/bin
```

## Architecture

```
cmd/
  hppycli/         # Cobra CLI binary (thin frontend)
    main.go        # Entry point — version injection via ldflags
    cmd/           # Cobra command definitions
      root.go      # Global flags, config loading, API client init
      helpers.go   # Shared output formatting, flag parsing, validation
      workorders.go / inspections.go / properties.go / units.go / account.go
      mcp.go       # `mcp setup` — generates MCP client config JSON
  hppymcp/         # MCP server binary (stdio transport)
    main.go        # Entry point — server setup
    tools.go       # MCP tool handlers + input validation + buildListOpts
    resources.go   # MCP resource handlers (account, property details)
    prompts.go     # MCP prompt definitions (property_summary, maintenance_report)
internal/
  api/             # GraphQL client (shared by both binaries)
    client.go      # HTTP client, auth, pagination, retry logic
    queries.go     # GraphQL query strings
    responses.go   # Generic response/connection types
  config/          # YAML config loading + env var overrides
  models/          # Domain model structs + shared validation (ValidateStatus, ValidateDateRange)
```

Both binaries are thin frontends over shared logic in `internal/`.

## Key Design Decisions

### Authentication & Token Management
- Tokens are cached in memory via atomic swap (`tokenState` struct). Double-checked locking ensures only one goroutine refreshes at a time.
- If a token expires mid-pagination, `fetchPageWithRetry` detects the 401 and re-authenticates once before retrying the failed page. This is tracked via `authRetried` flag per fetch call.
- Login cooldown (30s) only triggers on **permanent** failures (bad credentials). Transient errors (network failures, 500s) do not set the cooldown, so retries can proceed immediately. This distinction is made via `apiError.Retryable`.

### Pagination
- Relay-style cursor pagination with configurable limits. Default cap is 1000 items; hard ceiling is 50,000 (`hardMaxItems`) as defence-in-depth against runaway loops.
- Page size is fixed at 100 (server-enforced maximum).
- The MCP server uses a channel-based semaphore (`sem`) to limit concurrent pagination loops to 3.

### Shared Validation
- `models.ValidateStatus()` and `models.ValidateDateRange()` are the single source of truth, called by both CLI (`parseListFlags`) and MCP (`buildListOpts`).
- Status maps (`ValidWorkOrderStatuses`, `ValidInspectionStatuses`) are exported from `models` and referenced (not copied) by both callers.

### Error Handling in MCP
- `toolError()` logs full error details to stderr but only returns a sanitised category to the MCP client (e.g. "auth_failed: Authentication failed"). Internal details like URLs and HTTP status codes are never exposed.
- `toolInputError()` returns validation errors with an `invalid_input:` prefix.

### Config Precedence
- 3-layer: CLI flags > environment variables > config file > defaults.
- Config file is YAML at `~/.hppycli.yaml` with chmod 600 enforcement.
- No `--password` flag exists by design — passwords should never appear in process lists.

## API Gotchas

These are runtime-verified behaviours of the HappyCo External GraphQL API that differ from what you might expect:

- **`accessibleCustomerIds` can be empty** — always use `account(accountId)` with `accessibleBusinessIds`, not the Customer path.
- **`template` vs `templateV2`** on Inspection — `template` is a JSON scalar (no subfields). Use `templateV2` for structured access.
- **`endedAt` not `completedAt`** on Inspection.
- **`WorkOrderAssignee` and `InspectionAssignee` are interfaces** — must use inline fragments (`... on WorkOrderAssigneeUser { id name }`).
- **`expiresAt` is milliseconds** — string of millisecond unix timestamp, not seconds.
- **Auth errors return double error objects** from the federated gateway.
- **Max 100 items per page** enforced server-side.
- **Empty `endCursor`** — 0-item collections return `""` not null.

Full API reference is in `.scratch/API Runtime Findings.md`.

## Testing Patterns

- **No global state mutation in tests.** Output functions accept `io.Writer`; tests pass `bytes.Buffer` instead of replacing `os.Stdout`.
- **Channel-based synchronisation** for concurrent tests (e.g. `TestConcurrentAuth` uses `goroutinesReady` and `loginLatch` channels instead of `time.Sleep`).
- **Mock interface for MCP tests** — `apiClient` interface in `tools.go` allows tests to mock the API client without importing `internal/api`.
- Tests use `require` for preconditions and `assert` for the actual checks (testify convention).

## Reference Documentation

`.scratch/` (gitignored) contains contextual information and copies of documentation, including HappyCo API docs that are not available via Context7.

`.scratch/tasks/` contains the implementation briefs used to build each feature area.

## Context7

Always use Context7 for library/API documentation, code generation, and setup/configuration steps proactively.

HappyCo API documentation is **not** available on Context7 — use `.scratch/` or web search instead.

Gitlab cli **IS** available for reference (https://context7.com/websites/gitlab_cli).

JIRA cli **IS** available for reference (https://context7.com/ankitpokhrel/jira-cli).
