package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConfigPathDefault(t *testing.T) {
	got := resolveConfigPathFrom("")
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".hppycli.yaml"), got)
}

func TestResolveConfigPathFlag(t *testing.T) {
	got := resolveConfigPathFrom("/custom/path/config.yaml")
	assert.Equal(t, "/custom/path/config.yaml", got)
}
