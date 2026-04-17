package models

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIDList(t *testing.T) {
	t.Run("happy path single ID", func(t *testing.T) {
		ids, err := ParseIDList("role-id", "abc123")
		assert.NoError(t, err)
		assert.Equal(t, []string{"abc123"}, ids)
	})

	t.Run("multiple IDs trimmed", func(t *testing.T) {
		ids, err := ParseIDList("role-id", " abc , def, ghi ")
		assert.NoError(t, err)
		assert.Equal(t, []string{"abc", "def", "ghi"}, ids)
	})

	t.Run("empty input is required error", func(t *testing.T) {
		ids, err := ParseIDList("role-id", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role-id is required")
		assert.Nil(t, ids)
	})

	t.Run("only commas is required error", func(t *testing.T) {
		ids, err := ParseIDList("role-id", ",,,")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role-id is required")
		assert.Nil(t, ids)
	})

	t.Run("invalid character rejected", func(t *testing.T) {
		ids, err := ParseIDList("role-id", "abc,bad id,def")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role-id")
		assert.Nil(t, ids)
	})

	t.Run("empty segments skipped", func(t *testing.T) {
		ids, err := ParseIDList("role-id", "abc,,def,")
		assert.NoError(t, err)
		assert.Equal(t, []string{"abc", "def"}, ids)
	})
}

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
		{"trailing semicolon rejected", "image/jpeg; charset=utf-8", true},
		{"trailing space rejected", "image/jpeg extra", true},
		{"double slash rejected", "image/jpeg/extra", true},
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

		// Embedded credentials rejected
		{"URL with credentials", "https://user:pass@example.com/webhook", true, "embedded credentials"},
		{"URL with user only", "https://user@example.com/webhook", true, "embedded credentials"},

		// IPv6 zone ID bypass prevention
		{"IPv6 link-local with zone ID", "https://[fe80::1%25eth0]/webhook", true, "internal/private"},
		{"IPv6 loopback with zone ID", "https://[::1%25lo]/webhook", true, "internal/private"},

		// IPv4-mapped IPv6 addresses
		{"IPv4-mapped loopback", "https://[::ffff:127.0.0.1]/webhook", true, "internal/private"},
		{"IPv4-mapped private", "https://[::ffff:10.0.0.1]/webhook", true, "internal/private"},

		// IPv6 private/link-local ranges
		{"IPv6 unique-local (fc00)", "https://[fc00::1]/webhook", true, "internal/private"},
		{"IPv6 link-local (fe80)", "https://[fe80::1]/webhook", true, "internal/private"},
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

func TestValidateWebhookSubjects(t *testing.T) {
	t.Run("valid subjects", func(t *testing.T) {
		subjects, err := ValidateWebhookSubjects("INSPECTIONS,WORK_ORDERS")
		assert.NoError(t, err)
		assert.Equal(t, []string{"INSPECTIONS", "WORK_ORDERS"}, subjects)
	})

	t.Run("lowercase normalised to uppercase", func(t *testing.T) {
		subjects, err := ValidateWebhookSubjects("inspections,work_orders")
		assert.NoError(t, err)
		assert.Equal(t, []string{"INSPECTIONS", "WORK_ORDERS"}, subjects)
	})

	t.Run("mixed case normalised", func(t *testing.T) {
		subjects, err := ValidateWebhookSubjects("Inspections,Work_Orders")
		assert.NoError(t, err)
		assert.Equal(t, []string{"INSPECTIONS", "WORK_ORDERS"}, subjects)
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		subjects, err := ValidateWebhookSubjects("")
		assert.NoError(t, err)
		assert.Nil(t, subjects)
	})

	t.Run("whitespace-only returns nil", func(t *testing.T) {
		subjects, err := ValidateWebhookSubjects("  ,  , ")
		assert.NoError(t, err)
		assert.Nil(t, subjects)
	})

	t.Run("invalid subject rejected", func(t *testing.T) {
		_, err := ValidateWebhookSubjects("INSPECTIONS,INVALID")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid webhook subject")
		assert.Contains(t, err.Error(), "INVALID")
	})

	t.Run("all valid subjects accepted", func(t *testing.T) {
		subjects, err := ValidateWebhookSubjects("INSPECTIONS,WORK_ORDERS,VENDORS,PLUGIN_SUBSCRIPTIONS")
		assert.NoError(t, err)
		assert.Len(t, subjects, 4)
	})

	t.Run("single subject", func(t *testing.T) {
		subjects, err := ValidateWebhookSubjects("INSPECTIONS")
		assert.NoError(t, err)
		assert.Equal(t, []string{"INSPECTIONS"}, subjects)
	})

	t.Run("spaces around commas trimmed", func(t *testing.T) {
		subjects, err := ValidateWebhookSubjects(" INSPECTIONS , WORK_ORDERS ")
		assert.NoError(t, err)
		assert.Equal(t, []string{"INSPECTIONS", "WORK_ORDERS"}, subjects)
	})
}

func TestValidateRatingScore(t *testing.T) {
	score := func(v float64) *float64 { return &v }

	tests := []struct {
		name    string
		value   *float64
		wantErr bool
		errMsg  string
	}{
		{"nil is valid", nil, false, ""},
		{"zero is valid", score(0), false, ""},
		{"positive is valid", score(5.5), false, ""},
		{"negative rejected", score(-1), true, "must not be negative"},
		{"NaN rejected", score(math.NaN()), true, "must be a finite number"},
		{"positive infinity rejected", score(math.Inf(1)), true, "must be a finite number"},
		{"negative infinity rejected", score(math.Inf(-1)), true, "must be a finite number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRatingScore(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePhotoSize(t *testing.T) {
	size := func(v int) *int { return &v }

	tests := []struct {
		name    string
		value   *int
		wantErr bool
	}{
		{"nil is valid", nil, false},
		{"positive is valid", size(1024), false},
		{"zero rejected", size(0), true},
		{"negative rejected", size(-1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhotoSize(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid email", "user@example.com", false},
		{"valid with display name", "John Doe <john@example.com>", false},
		{"no at sign", "notanemail", true},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"missing domain", "user@", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
