package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty string passes", "", false},
		{"simple numeric", "12345", false},
		{"UUID-style", "abc-123-def-456", false},
		{"alphanumeric with underscores", "wo_123_abc", false},
		{"path traversal rejected", "../../etc/passwd", true},
		{"spaces rejected", "wo 123", true},
		{"slashes rejected", "wo/123", true},
		{"semicolons rejected", "wo;drop", true},
		{"quotes rejected", `wo"123`, true},
		{"single char valid", "a", false},
		{"mixed case with dashes", "WorkOrder-ABC-123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateID("test_field", tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid characters")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFileName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"simple filename", "photo.jpg", false},
		{"with spaces", "my photo.jpg", false},
		{"forward slash rejected", "path/file.jpg", true},
		{"backslash rejected", "path\\file.jpg", true},
		{"dot rejected", ".", true},
		{"dotdot rejected", "..", true},
		{"hidden file allowed", ".gitignore", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileName(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMIMEType(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"image/jpeg", "image/jpeg", false},
		{"application/pdf", "application/pdf", false},
		{"text/plain", "text/plain", false},
		{"application/vnd.ms-excel", "application/vnd.ms-excel", false},
		{"missing subtype", "image", true},
		{"empty string", "", true},
		{"just slash", "/", true},
		{"no type", "/jpeg", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMIMEType(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"valid RFC3339", "2026-05-01T09:00:00Z", false},
		{"valid with offset", "2026-05-01T09:00:00+10:00", false},
		{"date only rejected", "2026-05-01", true},
		{"garbage rejected", "not-a-date", true},
		{"unix timestamp rejected", "1714550400", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimestamp("test_field", tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"hours only", "PT1H", false},
		{"minutes only", "PT30M", false},
		{"seconds only", "PT45S", false},
		{"hours and minutes", "PT1H30M", false},
		{"all components", "PT2H15M30S", false},
		{"fractional seconds", "PT1.5S", false},
		{"bare PT rejected", "PT", true},
		{"no PT prefix rejected", "1H30M", true},
		{"garbage rejected", "not-a-duration", true},
		{"P without T rejected", "P1D", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDuration(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFreeText(t *testing.T) {
	assert.NoError(t, ValidateFreeText("desc", "short text"))
	assert.NoError(t, ValidateFreeText("desc", ""))

	longText := make([]byte, MaxFreeTextLength+1)
	for i := range longText {
		longText[i] = 'a'
	}
	err := ValidateFreeText("desc", string(longText))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length")
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{"valid HTTPS URL", "https://example.com/webhook", false, ""},
		{"valid HTTPS with path and port", "https://hooks.example.com:8443/v1/events", false, ""},

		// Scheme enforcement
		{"HTTP rejected", "http://example.com/webhook", true, "must use HTTPS"},
		{"FTP rejected", "ftp://example.com/file", true, "must use HTTPS"},
		{"no scheme", "example.com/webhook", true, "must use HTTPS"},
		{"empty string", "", true, "must use HTTPS"},

		// Private/internal IP rejection
		{"loopback IPv4", "https://127.0.0.1/webhook", true, "internal/private"},
		{"loopback IPv6", "https://[::1]/webhook", true, "internal/private"},
		{"private 10.x", "https://10.0.0.1/webhook", true, "internal/private"},
		{"private 172.16.x", "https://172.16.0.1/webhook", true, "internal/private"},
		{"private 192.168.x", "https://192.168.1.1/webhook", true, "internal/private"},
		{"link-local IPv4", "https://169.254.1.1/webhook", true, "internal/private"},
		{"unspecified 0.0.0.0", "https://0.0.0.0/webhook", true, "internal/private"},

		// Cloud metadata endpoint rejection
		{"AWS metadata IP", "https://169.254.169.254/latest/meta-data", true, "dangerous hostname"},
		{"GCP metadata hostname", "https://metadata.google.internal/computeMetadata", true, "dangerous hostname"},
		{"GCP metadata mixed case", "https://Metadata.Google.Internal/computeMetadata", true, "dangerous hostname"},

		// Hostname-based blocking
		{"localhost rejected", "https://localhost/webhook", true, "dangerous hostname"},
		{"localhost with port rejected", "https://localhost:8443/webhook", true, "dangerous hostname"},

		// Public IPs should pass
		{"public IP", "https://203.0.113.1/webhook", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
