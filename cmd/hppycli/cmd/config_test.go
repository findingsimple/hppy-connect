package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveConfigPathDefault(t *testing.T) {
	// Reset cfgFile to ensure default path is used
	origCfgFile := cfgFile
	cfgFile = ""
	defer func() { cfgFile = origCfgFile }()

	got := resolveConfigPath()
	home, err := os.UserHomeDir()
	if err == nil {
		assert.Equal(t, filepath.Join(home, ".hppycli.yaml"), got)
	}
}

func TestResolveConfigPathFlag(t *testing.T) {
	origCfgFile := cfgFile
	cfgFile = "/custom/path/config.yaml"
	defer func() { cfgFile = origCfgFile }()

	got := resolveConfigPath()
	assert.Equal(t, "/custom/path/config.yaml", got)
}
