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
  hppymcp/         # MCP server binary (stdio transport)
    main.go        # Entry point — server setup
    tools.go       # MCP tool handlers
    resources.go   # MCP resource handlers
    prompts.go     # MCP prompt definitions
internal/
  api/             # GraphQL client (shared by both binaries)
    client.go      # HTTP client, auth, pagination
    queries.go     # GraphQL query strings
    responses.go   # Generic response/connection types
  config/          # YAML config loading + env var overrides
  models/          # Domain model structs
```

Both binaries are thin frontends over shared logic in `internal/`.

## Reference Documentation

`.scratch/` (gitignored) contains contextual information and copies of documentation, including HappyCo API docs that are not available via Context7.

`.scratch/tasks/` contains the implementation briefs used to build each feature area.

## Context7

Always use Context7 for library/API documentation, code generation, and setup/configuration steps proactively.

HappyCo API documentation is **not** available on Context7 — use `.scratch/` or web search instead.

Gitlab cli **IS** available for reference (https://context7.com/websites/gitlab_cli).

JIRA cli **IS** available for reference (https://context7.com/ankitpokhrel/jira-cli).
