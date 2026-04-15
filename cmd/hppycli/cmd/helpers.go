package cmd

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

// truncateString truncates s to max runes, appending "..." if truncated.
func truncateString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

// sanitizeCell replaces tabs and newlines with spaces so tabwriter output stays aligned.
func sanitizeCell(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

// sanitizeCSVCell prefixes cells that start with formula-trigger characters to prevent
// CSV injection when opened in spreadsheet software.
func sanitizeCSVCell(s string) string {
	if len(s) > 0 && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@') {
		return "'" + s
	}
	return s
}

// formatLocation returns a display string for a location, showing "Property > Unit"
// when the location name differs from the property name.
func formatLocation(loc *models.Location) string {
	if loc == nil || loc.Property == nil {
		return ""
	}
	if loc.Name != "" && loc.Name != loc.Property.Name {
		return loc.Property.Name + " > " + loc.Name
	}
	return loc.Property.Name
}

// formatScore formats an inspection score as "score/potential" or just "score".
// Returns "" for nil scores and handles NaN/Inf gracefully.
func formatScore(score, potential *float64) string {
	if score == nil || math.IsNaN(*score) || math.IsInf(*score, 0) {
		return ""
	}
	if potential != nil && !math.IsNaN(*potential) && !math.IsInf(*potential, 0) {
		return fmt.Sprintf("%g/%g", *score, *potential)
	}
	return fmt.Sprintf("%g", *score)
}

// parseDate tries RFC3339 first, then falls back to YYYY-MM-DD in local timezone.
// Returns a clear error if neither format works, and validates that YYYY-MM-DD dates
// are real calendar dates (e.g. rejects Feb 30).
func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		if t.Format("2006-01-02") != s {
			return time.Time{}, fmt.Errorf("%q is not a valid calendar date", s)
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("%q is not a valid date (expected RFC3339 or YYYY-MM-DD)", s)
}

// addListFlags registers the common filter flags used by workorders and inspections.
func addListFlags(cmd *cobra.Command, statusHelp string) {
	cmd.Flags().String("property-id", "", "filter by property ID")
	cmd.Flags().String("unit-id", "", "filter by unit ID (mutually exclusive with --property-id)")
	cmd.Flags().String("status", "", statusHelp)
	cmd.Flags().String("created-after", "", "filter by creation date (RFC3339 or YYYY-MM-DD)")
	cmd.Flags().String("created-before", "", "filter by creation date (RFC3339 or YYYY-MM-DD)")
	cmd.Flags().Int("limit", 0, "maximum number of items to return (0 = default cap)")
	cmd.MarkFlagsMutuallyExclusive("property-id", "unit-id")
}

// parseListFlags reads the common filter flags and builds a ListOptions.
func parseListFlags(cmd *cobra.Command, validStatuses map[string]bool) (models.ListOptions, error) {
	propertyID, _ := cmd.Flags().GetString("property-id")
	unitID, _ := cmd.Flags().GetString("unit-id")
	status, _ := cmd.Flags().GetString("status")
	createdAfter, _ := cmd.Flags().GetString("created-after")
	createdBefore, _ := cmd.Flags().GetString("created-before")
	limit, _ := cmd.Flags().GetInt("limit")

	if limit < 0 {
		return models.ListOptions{}, fmt.Errorf("--limit must be a non-negative integer")
	}

	opts := models.ListOptions{Limit: limit}

	if unitID != "" {
		opts.LocationID = unitID
	} else if propertyID != "" {
		opts.LocationID = propertyID
	}

	if status != "" {
		upper := strings.ToUpper(status)
		if !validStatuses[upper] {
			valid := make([]string, 0, len(validStatuses))
			for k := range validStatuses {
				valid = append(valid, k)
			}
			sort.Strings(valid)
			return models.ListOptions{}, fmt.Errorf("invalid --status %q: valid options are %s", status, strings.Join(valid, ", "))
		}
		opts.Status = []string{upper}
	}

	if createdAfter != "" {
		t, err := parseDate(createdAfter)
		if err != nil {
			return models.ListOptions{}, fmt.Errorf("invalid --created-after: %w", err)
		}
		opts.CreatedAfter = &t
	}
	if createdBefore != "" {
		t, err := parseDate(createdBefore)
		if err != nil {
			return models.ListOptions{}, fmt.Errorf("invalid --created-before: %w", err)
		}
		opts.CreatedBefore = &t
	}

	if opts.CreatedAfter != nil && opts.CreatedBefore != nil && !opts.CreatedAfter.Before(*opts.CreatedBefore) {
		return models.ListOptions{}, fmt.Errorf("--created-after (%s) must be before --created-before (%s)", createdAfter, createdBefore)
	}

	return opts, nil
}
