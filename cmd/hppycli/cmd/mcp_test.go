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

func TestDetectMcpBinaryFallback(t *testing.T) {
	t.Setenv("GOPATH", "/nonexistent/gopath")
	t.Setenv("HOME", "/nonexistent/home")
	// detectMcpBinary should fall back to "hppymcp" when nothing is found
	got := detectMcpBinary()
	assert.Equal(t, "hppymcp", got)
}

func TestPrintClaudeConfig(t *testing.T) {
	var buf bytes.Buffer
	err := printClaudeConfig(&buf, "/usr/local/bin/hppymcp", "/home/user/.hppycli.yaml")
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Claude Code MCP settings")
	assert.Contains(t, output, "/usr/local/bin/hppymcp")
	assert.Contains(t, output, "/home/user/.hppycli.yaml")

	// Extract the JSON portion and validate it
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
	assert.Contains(t, parsed, "hppymcp")
}

func TestPrintCursorConfig(t *testing.T) {
	var buf bytes.Buffer
	err := printCursorConfig(&buf, "/usr/local/bin/hppymcp", "/home/user/.hppycli.yaml")
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Cursor MCP settings")
	assert.Contains(t, output, "/usr/local/bin/hppymcp")

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
	assert.Contains(t, mcpServers, "hppymcp")
}
