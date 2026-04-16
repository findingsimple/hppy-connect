# hppy-connect demo — HappyCo AI Hackathon 2026
# Run: vhs demo.tape

Output demo.gif
Output demo.mp4

# Terminal settings
Set Shell "zsh"
Set FontFamily "JetBrains Mono"
Set FontSize 16
Set Width 1400
Set Height 800
Set Padding 20
Set Framerate 24
Set TypingSpeed 50ms
Set Theme "Catppuccin Mocha"

# ─── Intro ───

Type "# hppy-connect: CLI + MCP server for the HappyCo API"
Enter
Type "# Built during the 2-day AI Hackathon (April 2026)"
Enter
Sleep 2s
Enter

# ─── 1. Version & Help ───

Type "hppycli version"
Enter
Sleep 1.5s

Type "hppycli --help"
Enter
Sleep 4s

# ─── 2. Account & Properties ───

Type "# Let's explore the HappyCo account"
Enter
Sleep 1s

Type "hppycli account"
Enter
Sleep 2.5s

Type "hppycli properties list"
Enter
Sleep 2.5s

# ─── 3. Seed test data ───

Type "# Seed realistic test data — preview first with --dry-run"
Enter
Sleep 1.5s

Type "hppycli seed --dry-run"
Enter
Sleep 3s

Type "# Now create the work orders for real"
Enter
Sleep 1s

Type "hppycli seed --yes"
Enter
Sleep 5s

# ─── 4. Query the seeded data ───

Type "# Query the work orders we just created"
Enter
Sleep 1s

Type "hppycli workorders list"
Enter
Sleep 3s

Type "# Filter by status"
Enter
Sleep 1s

Type "hppycli workorders list --status open"
Enter
Sleep 3s

# ─── 5. Output formats ───

Type "# Multiple output formats: text, json, csv, raw"
Enter
Sleep 1s

Type "hppycli workorders list --output json"
Enter
Sleep 3s

Type "hppycli workorders list --output csv"
Enter
Sleep 3s

# ─── 6. MCP Server Setup ───

Type "# One command to connect the MCP server to Claude Code"
Enter
Sleep 1.5s

Type "hppycli mcp setup --client claude"
Enter
Sleep 3s

Type "# The MCP server exposes 71 tools across 8 domains"
Enter
Type "# so an LLM can read and manage your HappyCo data"
Enter
Sleep 3s

# ─── Outro ───

Enter
Type "# hppy-connect — CLI + MCP for HappyCo"
Enter
Type "# github.com/findingsimple/hppy-connect"
Enter
Sleep 4s
