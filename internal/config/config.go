// Package config handles YAML config file loading with environment variable
// and CLI flag overrides, enforcing secure file permissions.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the unified application configuration loaded from YAML, env vars, and flags.
type Config struct {
	Email     string `yaml:"email" json:"email"`
	Password  string `yaml:"password" json:"-"`
	AccountID string `yaml:"account_id" json:"account_id"`
	Endpoint  string `yaml:"endpoint" json:"endpoint"`
	Debug     bool   `yaml:"debug" json:"debug"`
}

// String returns a redacted representation safe for logging.
func (c Config) String() string {
	pw := ""
	if c.Password != "" {
		pw = "****"
	}
	return fmt.Sprintf("Config{Email:%q Password:%s AccountID:%q Endpoint:%q Debug:%t}",
		c.Email, pw, c.AccountID, c.Endpoint, c.Debug)
}

// LoadConfig loads configuration with precedence: flags > env vars > config file > defaults.
// envFunc is injectable for testing (production passes os.Getenv).
// flags map allows flag overrides without Cobra dependency.
func LoadConfig(filePath string, envFunc func(string) string, flags map[string]string) (*Config, error) {
	cfg := &Config{
		Endpoint: "https://externalgraph.happyco.com",
		Debug:    false,
	}

	// Load from file if it exists
	if filePath != "" {
		info, err := os.Stat(filePath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("reading config file: %w", err)
			}
			// File doesn't exist — skip, env vars may provide everything
		} else {
			if mode := info.Mode().Perm(); mode&0077 != 0 {
				return nil, fmt.Errorf("config file %s has permissions %04o; must be 0600 or stricter (contains credentials)", filePath, mode)
			}
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("reading config file: %w", err)
			}
			if len(bytes.TrimSpace(data)) > 0 {
				dec := yaml.NewDecoder(bytes.NewReader(data))
				dec.KnownFields(true)
				if err := dec.Decode(cfg); err != nil {
					return nil, fmt.Errorf("parsing config file: %w", err)
				}
			}
		}
	}

	// Apply env var overrides
	if v := envFunc("HAPPYCO_EMAIL"); v != "" {
		cfg.Email = v
	}
	if v := envFunc("HAPPYCO_PASSWORD"); v != "" {
		cfg.Password = v
	}
	if v := envFunc("HAPPYCO_ACCOUNT_ID"); v != "" {
		cfg.AccountID = v
	}
	if v := envFunc("HAPPYCO_ENDPOINT"); v != "" {
		cfg.Endpoint = v
	}
	if v := envFunc("HAPPYCO_DEBUG"); v != "" {
		cfg.Debug = parseBool(v)
	}

	// Apply flag overrides (no "password" — flags are visible in ps output)
	if flags != nil {
		if v, ok := flags["email"]; ok {
			cfg.Email = v
		}
		if v, ok := flags["account_id"]; ok {
			cfg.AccountID = v
		}
		if v, ok := flags["endpoint"]; ok {
			cfg.Endpoint = v
		}
		if v, ok := flags["debug"]; ok {
			cfg.Debug = parseBool(v)
		}
	}

	// Validate endpoint is HTTPS (credentials are sent over this connection)
	if !strings.HasPrefix(cfg.Endpoint, "https://") {
		return nil, fmt.Errorf("endpoint %q must use https:// (credentials are transmitted)", cfg.Endpoint)
	}

	return cfg, nil
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1"
}
