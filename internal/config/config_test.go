package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noEnv(string) string { return "" }

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		envFunc  func(string) string
		flags    map[string]string
		expected *Config
	}{
		{
			name: "file only",
			yaml: "email: file@test.com\npassword: filepass\naccount_id: \"100\"\nendpoint: https://custom.api.com\ndebug: true\n",
			envFunc: noEnv,
			expected: &Config{
				Email:     "file@test.com",
				Password:  "filepass",
				AccountID: "100",
				Endpoint:  "https://custom.api.com",
				Debug:     true,
			},
		},
		{
			name: "env overrides file",
			yaml: "email: file@test.com\npassword: filepass\n",
			envFunc: func(key string) string {
				if key == "HAPPYCO_EMAIL" {
					return "env@test.com"
				}
				return ""
			},
			expected: &Config{
				Email:    "env@test.com",
				Password: "filepass",
				Endpoint: "https://externalgraph.happyco.com",
			},
		},
		{
			name: "flag overrides env and file",
			yaml: "email: file@test.com\n",
			envFunc: func(key string) string {
				if key == "HAPPYCO_EMAIL" {
					return "env@test.com"
				}
				return ""
			},
			flags: map[string]string{"email": "flag@test.com"},
			expected: &Config{
				Email:    "flag@test.com",
				Endpoint: "https://externalgraph.happyco.com",
			},
		},
		{
			name: "missing file with env fallback",
			yaml: "",
			envFunc: func(key string) string {
				switch key {
				case "HAPPYCO_EMAIL":
					return "env@test.com"
				case "HAPPYCO_PASSWORD":
					return "envpass"
				case "HAPPYCO_ACCOUNT_ID":
					return "200"
				}
				return ""
			},
			expected: &Config{
				Email:     "env@test.com",
				Password:  "envpass",
				AccountID: "200",
				Endpoint:  "https://externalgraph.happyco.com",
			},
		},
		{
			name:    "empty file with defaults",
			yaml:    "\n",
			envFunc: noEnv,
			expected: &Config{
				Endpoint: "https://externalgraph.happyco.com",
				Debug:    false,
			},
		},
		{
			name: "all three layers - flag wins",
			yaml: "email: file@test.com\npassword: filepass\naccount_id: \"100\"\nendpoint: https://file.api.com\ndebug: false\n",
			envFunc: func(key string) string {
				switch key {
				case "HAPPYCO_EMAIL":
					return "env@test.com"
				case "HAPPYCO_PASSWORD":
					return "envpass"
				case "HAPPYCO_ACCOUNT_ID":
					return "200"
				case "HAPPYCO_ENDPOINT":
					return "https://env.api.com"
				case "HAPPYCO_DEBUG":
					return "1"
				}
				return ""
			},
			flags: map[string]string{
				"email":      "flag@test.com",
				"account_id": "300",
			},
			expected: &Config{
				Email:     "flag@test.com",
				Password:  "envpass",
				AccountID: "300",
				Endpoint:  "https://env.api.com",
				Debug:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			if tt.yaml != "" {
				filePath = writeYAML(t, tt.yaml)
			} else {
				filePath = "/nonexistent/path/config.yaml"
			}

			cfg, err := LoadConfig(filePath, tt.envFunc, tt.flags)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg)
		})
	}
}

func TestLoadConfig_MalformedYAML(t *testing.T) {
	path := writeYAML(t, "email: [invalid\n  yaml: {broken")
	cfg, err := LoadConfig(path, noEnv, nil)
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config file")
}

func TestLoadConfig_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("email: test@test.com\n"), 0600))
	require.NoError(t, os.Chmod(path, 0000))
	t.Cleanup(func() { os.Chmod(path, 0600) })

	cfg, err := LoadConfig(path, noEnv, nil)
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading config file")
}

func TestLoadConfig_WorldReadableFileRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("email: test@test.com\n"), 0644))

	cfg, err := LoadConfig(path, noEnv, nil)
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permissions")
}

func TestLoadConfig_PasswordFlagIgnored(t *testing.T) {
	// Password must never come from CLI flags (visible in ps output).
	// Verify that a flags["password"] value is ignored.
	cfg, err := LoadConfig("", func(key string) string {
		if key == "HAPPYCO_PASSWORD" {
			return "envpass"
		}
		return ""
	}, map[string]string{"password": "flagpass"})
	require.NoError(t, err)
	assert.Equal(t, "envpass", cfg.Password, "password flag should be ignored; env var should win")
}

func TestLoadConfig_EmptyFilePath(t *testing.T) {
	cfg, err := LoadConfig("", noEnv, nil)
	require.NoError(t, err)
	assert.Equal(t, "https://externalgraph.happyco.com", cfg.Endpoint)
	assert.Equal(t, "", cfg.Email)
}

func TestLoadConfig_DebugFalseOverridesTrue(t *testing.T) {
	path := writeYAML(t, "debug: true\n")
	cfg, err := LoadConfig(path, func(key string) string {
		if key == "HAPPYCO_DEBUG" {
			return "false"
		}
		return ""
	}, nil)
	require.NoError(t, err)
	assert.False(t, cfg.Debug, "env var 'false' should override file 'true'")
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{" true ", true},
		{"1", true},
		{" 1 ", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"", false},
		{"yes", false},
		{"no", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseBool(tt.input))
		})
	}
}

func TestLoadConfig_HttpEndpointRejected(t *testing.T) {
	cfg, err := LoadConfig("", func(key string) string {
		if key == "HAPPYCO_ENDPOINT" {
			return "http://insecure.example.com"
		}
		return ""
	}, nil)
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "https://")
}

func TestConfig_StringRedactsPassword(t *testing.T) {
	cfg := Config{
		Email:    "user@test.com",
		Password: "secret123",
		Endpoint: "https://example.com",
	}
	s := cfg.String()
	assert.NotContains(t, s, "secret123")
	assert.Contains(t, s, "****")
	assert.Contains(t, s, "user@test.com")
}

func TestConfig_JSONOmitsPassword(t *testing.T) {
	cfg := Config{
		Email:    "user@test.com",
		Password: "secret123",
		Endpoint: "https://example.com",
	}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "secret123")
	assert.NotContains(t, string(data), "password")
	assert.Contains(t, string(data), "user@test.com")
}
