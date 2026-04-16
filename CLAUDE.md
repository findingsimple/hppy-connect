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
      helpers.go   # Shared output formatting, flag parsing, validation, confirmAction
      workorders.go / inspections.go / properties.go / units.go / account.go
      projects.go / users.go / memberships.go / roles.go / webhooks.go
      seed.go      # `seed` — populate account with test data (seedClient interface for testability)
      config.go / completion.go / version.go  # Utility commands
      mcp.go       # `mcp setup` — generates MCP client registration commands
  hppymcp/         # MCP server binary (stdio transport)
    main.go        # Entry point — server setup
    tools.go       # MCP read tool handlers + composed apiClient interface
    tools_mutations.go  # MCP mutation tool handlers (71 tools across 8 domains)
    resources.go   # MCP resource handlers (account, property details)
    prompts.go     # MCP prompt definitions (property_summary, maintenance_report)
internal/
  api/             # GraphQL client (shared by both binaries)
    client.go      # HTTP client, auth, pagination, retry logic, mutation methods
    queries.go     # GraphQL query strings (reads)
    mutations.go   # GraphQL mutation strings (writes)
    responses.go   # Generic response/connection types
    mutation_responses.go  # Mutation-specific response structs
  config/          # YAML config loading + env var overrides
  models/          # Domain model structs + shared validation
    models.go      # Entity types (WorkOrder, Inspection, etc.) + validation maps
    inputs.go      # Mutation input structs (WorkOrderCreateInput, etc.)
    entities.go    # Additional entity types for mutation responses (User, Role, Webhook, etc.)
    validation.go  # Webhook URL validation (ValidateWebhookURL)
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

### Mutation Retry Semantics
- `doMutation` — single auth-retry on 401, no transient error retry. Used for non-idempotent operations (creates, adds) to prevent duplicates.
- `doMutationIdempotent` — full retry with backoff (same as reads). Used for idempotent setters, archives, deletes, state transitions.
- Exception: `InspectionDuplicateSection` uses `doMutation` despite the "duplicate" name — it creates a new copy on each call.

### Composed API Client Interface
- The MCP server's `apiClient` interface is composed from domain-specific sub-interfaces (`workOrderClient`, `inspectionClient`, `projectClient`, etc.).
- Test mocks only need to implement the sub-interface their test uses (embed a no-op base struct and override specific methods).
- This prevents adding a new mutation from breaking every existing mock.
- The `seed` command uses the same pattern: `seedClient` interface in `seed.go` defines the 7 API methods it needs. Tests mock this interface with `mockSeedClient`.

### Seed Command
- `hppycli seed` populates a HappyCo account with realistic test data for exercising both CLI and MCP server.
- Auto-discovers up to 3 properties and 5 units each. Creates work orders (cycling through 10 recipes), inspections, projects, and webhooks.
- Work orders are created with their target status directly (not created-then-transitioned) to minimise API calls.
- Descriptions are tagged with `[SEED <timestamp>]` (e.g. `[SEED 2026-04-16T15:04]`) for batch identification and future cleanup.
- `--count` is capped at 50 (`maxSeedCount`) as defence-in-depth against runaway creation.
- Plan is built once (`buildSeedPlan`) and iterated for both `--dry-run` display and execution — prevents drift between what's shown and what's created.
- Core logic is in `runSeed()` which accepts `seedClient` interface and `io.Writer` parameters for full testability.
- Supports `--output json` for programmatic access to created entity IDs.

### Plugins Exclusion
- The Plugin domain (5 mutations) is excluded: wrong audience (integration partners, not property managers), secrets in process lists (`pluginLogin`), and disproportionate complexity (`setPluginData` nested input shape).

### Config Precedence
- 3-layer: CLI flags > environment variables > config file > defaults.
- Config file is YAML at `~/.hppycli.yaml` with chmod 600 enforcement.
- No `--password` flag exists by design — passwords should never appear in process lists.
- Commands requiring `--account-id` (users, memberships, roles) default to the config file's `account_id` value.

### Interactive Account Selection
- If `account_id` is missing from config/env/flags, the CLI authenticates and discovers accessible accounts via `api.Login()`.
- Account names are resolved via `resolveAccountNames()` (shared helper using a temporary client pre-seeded with `WithToken`), used by both `config init` and `PersistentPreRunE`.
- For multiple accounts, an interactive picker is shown (capped at 20 displayed; all remain selectable by number). After selection, the user is offered to save the account ID to config.
- Single accounts are auto-selected and auto-saved to the config file without prompting.
- Non-interactive terminals (piped stdin) fail with a clear error directing users to `config init`, `--account-id`, or env vars.
- `saveAccountToConfig` uses atomic write (temp file + rename) and always enforces 0600 permissions. Rejects invalid YAML in existing config files to prevent silent data loss.

## Known Limitations & Accepted Trade-offs

These were identified during security and resilience review and consciously accepted.

### A. DNS Rebinding in Webhook URL Validation
`ValidateWebhookURL` checks hostnames at parse time only. A public domain that later re-resolves to a private IP (169.254.169.254, 127.0.0.1) would bypass the check. **Accepted because:** this client validates before sending to the HappyCo API — the actual HTTP request to the webhook URL is made server-side by HappyCo's infrastructure, not by this client. The inline comment at `validation.go:197-199` documents this.

### B. Hex/Octal/Decimal IP Encoding in Webhook URLs
Hostnames like `0x7f000001` (hex 127.0.0.1) or `2130706433` (decimal) bypass `net.ParseIP` (returns nil), skipping private IP checks. **Accepted because:** same as (A) — server-side concern. The HappyCo API performs its own validation on webhook delivery.

### C. Non-Idempotent Mutation Auth-Retry Assumption
`doMutation` retries once on 401 after re-authentication. If the federated gateway returns 401 *after* the downstream service has already committed the mutation, a duplicate could be created. **Accepted because:** the HappyCo gateway rejects auth failures before executing mutations (verified at runtime). The assumption is documented in the `doMutation` comment at `client.go:373-377`. `InspectionSendToGuest` is the highest-risk case since it triggers an email — noted here for awareness.

### D. Ambiguous State on Network Timeout for Non-Idempotent Mutations
If a non-idempotent mutation times out, the client returns an error but the server may have committed the change. The error message does not indicate possible partial success. **Accepted because:** this is inherent to non-idempotent HTTP operations. The correct behaviour is to fail and let the user check, which is what happens. Improving the error message ("operation may have been executed — check before retrying") is a nice-to-have.

### E. MCP Mutations Bypass Concurrency Semaphore
The semaphore (`sem`) limits concurrent pagination loops to 3 but mutation tools are not gated. A misbehaving MCP client could fire many mutations concurrently. **Accepted because:** mutations are single HTTP requests (no pagination memory concern), and the upstream HappyCo API has its own rate limiting. For `InspectionSendToGuest` specifically, rapid concurrent calls could send duplicate emails — but the MCP client (typically an LLM) is unlikely to do this without user intent.

### F. GraphQL Error Messages Forwarded to CLI Users
The CLI returns GraphQL error messages verbatim from `gqlResp.Errors[0].Message`. If the HappyCo API returns verbose errors with internal details, these would be shown to CLI users. **Accepted because:** the MCP path sanitises errors through `toolError()`/`sanitiseErrorCategory()`, and the CLI is a developer tool where full error context is useful. The API is external (HappyCo-controlled) so error content is predictable.

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
- **Mock interface for MCP tests** — `apiClient` composed interface in `tools.go` allows tests to mock the API client without importing `internal/api`. Mutation tests use domain sub-interfaces with no-op base structs.
- Tests use `require` for preconditions and `assert` for the actual checks (testify convention).
- **Mutation client tests** use `httptest.Server` to verify GraphQL mutation strings, variable marshalling, and response unmarshalling.
- **`confirmAction` tests** use `io.Reader` injection (matching the project's testable I/O pattern).
- **Seed command tests** use `mockSeedClient` (implements `seedClient` interface) with `bytes.Buffer` for stdout/stderr. Tests cover plan building, discovery failures, partial failures, context cancellation, and JSON output.

## Reference Documentation

`.scratch/` (gitignored) contains contextual information and copies of documentation, including HappyCo API docs that are not available via Context7.

`.scratch/tasks/` contains the implementation briefs used to build each feature area.

## Context7

Always use Context7 for library/API documentation, code generation, and setup/configuration steps proactively.

HappyCo API documentation is **not** available on Context7 — use `.scratch/` or web search instead.

Gitlab cli **IS** available for reference (https://context7.com/websites/gitlab_cli).

JIRA cli **IS** available for reference (https://context7.com/ankitpokhrel/jira-cli).
