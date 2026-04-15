package cmd

import (
	"math"
	"testing"
	"time"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- truncateString ---

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"short string unchanged", "hello", 60, "hello"},
		{"exact length unchanged", "abcdef", 6, "abcdef"},
		{"truncated with ellipsis", "abcdefghij", 8, "abcde..."},
		{"empty string", "", 60, ""},
		{"unicode preserved", "こんにちは世界です", 6, "こんに..."},
		{"emoji preserved", "🏠🏢🏗️🔧🔨💡🛠️", 5, "🏠🏢..."},
		{"max 0 returns empty", "hello", 0, ""},
		{"max 1 returns first rune", "hello", 1, "h"},
		{"max 2 returns first two runes", "hello", 2, "he"},
		{"max 3 returns first three runes", "hello", 3, "hel"},
		{"max 4 truncates with ellipsis", "hello world", 4, "h..."},
		{"short string with small max", "hi", 1, "h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, truncateString(tt.s, tt.max))
		})
	}
}

// --- sanitizeCell ---

func TestSanitizeCell(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"no special chars", "hello world", "hello world"},
		{"tab replaced", "hello\tworld", "hello world"},
		{"newline replaced", "hello\nworld", "hello world"},
		{"carriage return replaced", "hello\rworld", "hello world"},
		{"mixed whitespace", "a\tb\nc\rd", "a b c d"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeCell(tt.s))
		})
	}
}

// --- sanitizeCSVCell ---

func TestSanitizeCSVCell(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"normal text", "hello", "hello"},
		{"equals prefix", "=SUM(A1)", "'=SUM(A1)"},
		{"plus prefix", "+1 555-1234", "'+1 555-1234"},
		{"minus prefix", "-5 degrees", "'-5 degrees"},
		{"at prefix", "@mention", "'@mention"},
		{"empty string", "", ""},
		{"safe number", "42", "42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeCSVCell(tt.s))
		})
	}
}

// --- formatLocation ---

func TestFormatLocation(t *testing.T) {
	tests := []struct {
		name string
		loc  *models.Location
		want string
	}{
		{"nil location", nil, ""},
		{"nil property", &models.Location{Name: "Unit A"}, ""},
		{"property only", &models.Location{
			Name:     "Sunset Apartments",
			Property: &models.Property{Name: "Sunset Apartments"},
		}, "Sunset Apartments"},
		{"property and unit", &models.Location{
			Name:     "Unit 3B",
			Property: &models.Property{Name: "Sunset Apartments"},
		}, "Sunset Apartments > Unit 3B"},
		{"empty location name", &models.Location{
			Name:     "",
			Property: &models.Property{Name: "Sunset Apartments"},
		}, "Sunset Apartments"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatLocation(tt.loc))
		})
	}
}

// --- formatScore ---

func TestFormatScore(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	tests := []struct {
		name      string
		score     *float64
		potential *float64
		want      string
	}{
		{"nil score", nil, nil, ""},
		{"score only", f(85), nil, "85"},
		{"score and potential", f(85), f(100), "85/100"},
		{"fractional score", f(85.7), f(100), "85.7/100"},
		{"zero score", f(0), f(100), "0/100"},
		{"NaN score", f(math.NaN()), f(100), ""},
		{"Inf score", f(math.Inf(1)), nil, ""},
		{"score with NaN potential", f(85), f(math.NaN()), "85"},
		{"score with Inf potential", f(85), f(math.Inf(1)), "85"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatScore(tt.score, tt.potential))
		})
	}
}

// --- parseDate ---

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, got time.Time)
	}{
		{
			name:  "RFC3339",
			input: "2026-04-15T10:30:00Z",
			check: func(t *testing.T, got time.Time) {
				assert.Equal(t, 2026, got.Year())
				assert.Equal(t, time.April, got.Month())
				assert.Equal(t, 15, got.Day())
			},
		},
		{
			name:  "RFC3339 with offset",
			input: "2026-04-15T10:30:00+10:00",
			check: func(t *testing.T, got time.Time) {
				assert.Equal(t, 2026, got.Year())
			},
		},
		{
			name:  "YYYY-MM-DD",
			input: "2026-04-15",
			check: func(t *testing.T, got time.Time) {
				assert.Equal(t, 2026, got.Year())
				assert.Equal(t, time.April, got.Month())
				assert.Equal(t, 15, got.Day())
			},
		},
		{
			name:    "invalid format",
			input:   "April 15, 2026",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid date Feb 30",
			input:   "2026-02-30",
			wantErr: true,
		},
		{
			name:    "invalid date Feb 29 non-leap",
			input:   "2026-02-29",
			wantErr: true,
		},
		{
			name:  "valid date Feb 29 leap year",
			input: "2028-02-29",
			check: func(t *testing.T, got time.Time) {
				assert.Equal(t, 29, got.Day())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// --- parseListFlags ---

func newTestCmd() *cobra.Command {
	return &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
}

func TestParseListFlagsStatusValidation(t *testing.T) {
	valid := map[string]bool{"OPEN": true, "ON_HOLD": true, "COMPLETED": true}

	tests := []struct {
		name    string
		status  string
		wantErr bool
		wantVal string
	}{
		{"valid uppercase", "OPEN", false, "OPEN"},
		{"valid lowercase auto-uppercased", "open", false, "OPEN"},
		{"valid mixed case", "On_Hold", false, "ON_HOLD"},
		{"invalid status", "INVALID", true, ""},
		{"empty is fine", "", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestCmd()
			addListFlags(cmd, "test statuses")

			args := []string{}
			if tt.status != "" {
				args = append(args, "--status", tt.status)
			}
			cmd.SetArgs(args)
			require.NoError(t, cmd.ParseFlags(args))

			opts, err := parseListFlags(cmd, valid)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid --status")
				return
			}
			require.NoError(t, err)
			if tt.wantVal != "" {
				require.Len(t, opts.Status, 1)
				assert.Equal(t, tt.wantVal, opts.Status[0])
			}
		})
	}
}

func TestParseListFlagsStatusErrorIsSorted(t *testing.T) {
	valid := map[string]bool{"OPEN": true, "ON_HOLD": true, "COMPLETED": true}

	cmd := newTestCmd()
	addListFlags(cmd, "test")
	args := []string{"--status", "BOGUS"}
	require.NoError(t, cmd.ParseFlags(args))

	_, err := parseListFlags(cmd, valid)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "COMPLETED, ON_HOLD, OPEN")
}

func TestParseListFlagsMutuallyExclusiveLocation(t *testing.T) {
	cmd := newTestCmd()
	addListFlags(cmd, "test")

	// Cobra's MarkFlagsMutuallyExclusive is enforced at Execute time,
	// not ParseFlags time. Verify property-id works alone.
	args := []string{"--property-id", "prop-1"}
	require.NoError(t, cmd.ParseFlags(args))
	opts, err := parseListFlags(cmd, map[string]bool{})
	require.NoError(t, err)
	assert.Equal(t, "prop-1", opts.LocationID)

	// And unit-id works alone.
	cmd2 := newTestCmd()
	addListFlags(cmd2, "test")
	args2 := []string{"--unit-id", "unit-2"}
	require.NoError(t, cmd2.ParseFlags(args2))
	opts2, err := parseListFlags(cmd2, map[string]bool{})
	require.NoError(t, err)
	assert.Equal(t, "unit-2", opts2.LocationID)
}

func TestParseListFlagsDateRangeValidation(t *testing.T) {
	cmd := newTestCmd()
	addListFlags(cmd, "test")

	args := []string{"--created-after", "2026-04-15", "--created-before", "2026-04-01"}
	require.NoError(t, cmd.ParseFlags(args))

	_, err := parseListFlags(cmd, map[string]bool{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be before")
}

func TestParseListFlagsEqualDatesRejected(t *testing.T) {
	cmd := newTestCmd()
	addListFlags(cmd, "test")

	args := []string{"--created-after", "2026-04-15", "--created-before", "2026-04-15"}
	require.NoError(t, cmd.ParseFlags(args))

	_, err := parseListFlags(cmd, map[string]bool{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be before")
}

func TestParseListFlagsNegativeLimit(t *testing.T) {
	cmd := newTestCmd()
	addListFlags(cmd, "test")

	args := []string{"--limit", "-5"}
	require.NoError(t, cmd.ParseFlags(args))

	_, err := parseListFlags(cmd, map[string]bool{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit must be a non-negative integer")
}
