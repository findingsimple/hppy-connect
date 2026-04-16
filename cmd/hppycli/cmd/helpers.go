package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/findingsimple/hppy-connect/internal/api"
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
// CSV injection when opened in spreadsheet software. Covers characters that Excel,
// Google Sheets, and LibreOffice interpret as formula or DDE triggers.
func sanitizeCSVCell(s string) string {
	if len(s) > 0 && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@' || s[0] == '\t' || s[0] == '\r' || s[0] == '|') {
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

// validateLimit checks that a limit value is non-negative.
func validateLimit(limit int) error {
	if limit < 0 {
		return fmt.Errorf("--limit must be a non-negative integer")
	}
	return nil
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

	if err := validateLimit(limit); err != nil {
		return models.ListOptions{}, err
	}

	opts := models.ListOptions{Limit: limit}

	if unitID != "" {
		opts.LocationID = unitID
	} else if propertyID != "" {
		opts.LocationID = propertyID
	}

	statuses, err := models.ValidateStatus(status, validStatuses)
	if err != nil {
		return models.ListOptions{}, fmt.Errorf("invalid --status: %w", err)
	}
	opts.Status = statuses

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

	if err := models.ValidateDateRange(opts.CreatedAfter, opts.CreatedBefore); err != nil {
		return models.ListOptions{}, fmt.Errorf("--%s", err)
	}

	return opts, nil
}

// outputData holds structured data for the printOutput helper.
type outputData struct {
	Headers []string
	Rows    [][]string
	Items   any
	Count   int
	RawJSON json.RawMessage
}

func printOutput(data outputData) error {
	switch outputFormat {
	case "json":
		wrapper := map[string]any{
			"count":    data.Count,
			"returned": len(data.Rows),
			"items":    data.Items,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(wrapper)

	case "csv":
		w := csv.NewWriter(os.Stdout)
		if err := w.Write(data.Headers); err != nil {
			return err
		}
		for _, row := range data.Rows {
			sanitized := make([]string, len(row))
			for i, cell := range row {
				sanitized[i] = sanitizeCSVCell(cell)
			}
			if err := w.Write(sanitized); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()

	case "raw":
		if data.RawJSON != nil {
			var buf bytes.Buffer
			if err := json.Indent(&buf, data.RawJSON, "", "  "); err != nil {
				os.Stdout.Write(data.RawJSON)
			} else {
				buf.WriteTo(os.Stdout)
			}
			fmt.Println()
		}
		return nil

	default: // "text"
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, strings.Join(data.Headers, "\t"))
		for _, row := range data.Rows {
			sanitized := make([]string, len(row))
			for i, cell := range row {
				sanitized[i] = sanitizeCell(cell)
			}
			fmt.Fprintln(w, strings.Join(sanitized, "\t"))
		}
		return w.Flush()
	}
}

// confirmAction prompts the user for y/n confirmation on destructive operations.
// Skipped when --yes flag is set. Accepts io.Reader and io.Writer for testability
// (pass os.Stdin/os.Stderr in production).
func confirmAction(cmd *cobra.Command, action string, input io.Reader, output io.Writer) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if yes {
		return nil
	}
	fmt.Fprintf(output, "About to %s. Continue? [y/N] ", action)
	scanner := bufio.NewScanner(input)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response == "y" || response == "yes" {
			return nil
		}
	}
	return fmt.Errorf("aborted")
}

// resolveAccountID resolves the account ID from the --account-id flag, falling back to
// the config file's account_id. Returns an error if neither is set or the ID is invalid.
func resolveAccountID(cmd *cobra.Command) (string, error) {
	accountID, _ := cmd.Flags().GetString("account-id")
	if accountID == "" {
		accountID = configAccountID
	}
	if accountID == "" {
		return "", fmt.Errorf("--account-id is required (or set account_id in config)")
	}
	if err := models.ValidateID("account-id", accountID); err != nil {
		return "", err
	}
	return accountID, nil
}

// parseIDList splits a comma-separated string into validated IDs.
// Returns an error if any ID fails validation. Empty segments are skipped.
func parseIDList(flagName, raw string) ([]string, error) {
	var ids []string
	for _, p := range strings.Split(raw, ",") {
		id := strings.TrimSpace(p)
		if id == "" {
			continue
		}
		if err := models.ValidateID(flagName, id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// printMutationResult outputs a mutation result as indented JSON to the given writer.
// If --output text is explicitly set, warns on stderr that mutation output is always JSON.
func printMutationResult(cmd *cobra.Command, w io.Writer, result any) error {
	if f, _ := cmd.Flags().GetString("output"); f == "text" && cmd.Flags().Changed("output") {
		fmt.Fprintln(os.Stderr, "note: mutation output is always JSON")
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// formatAddress formats an address as a single line.
func formatAddress(line1, line2, city, state, postalCode string) string {
	parts := []string{}
	if line1 != "" {
		parts = append(parts, line1)
	}
	if line2 != "" {
		parts = append(parts, line2)
	}
	cityState := []string{}
	if city != "" {
		cityState = append(cityState, city)
	}
	if state != "" {
		cityState = append(cityState, state)
	}
	if len(cityState) > 0 {
		parts = append(parts, strings.Join(cityState, ", "))
	}
	if postalCode != "" {
		parts = append(parts, postalCode)
	}
	return strings.Join(parts, ", ")
}

// accountChoice represents an account available for selection.
type accountChoice struct {
	ID   string
	Name string
}

// maxDisplayAccounts caps the numbered list shown during account selection.
// Accounts beyond this limit are still selectable by number.
const maxDisplayAccounts = 20

// selectAccount prompts the user to choose from multiple accounts.
// If there is only one account, it is returned without prompting.
// The input must be a *bufio.Reader so callers can reuse the same buffered
// reader for subsequent prompts (mixing buffered/unbuffered reads on the same
// file descriptor causes data loss).
func selectAccount(accounts []accountChoice, input *bufio.Reader, output io.Writer) (string, error) {
	if len(accounts) == 0 {
		return "", fmt.Errorf("no accessible accounts found")
	}
	if len(accounts) == 1 {
		a := accounts[0]
		if a.Name != "" {
			fmt.Fprintf(output, "Account: %s (%s)\n", a.Name, a.ID)
		} else {
			fmt.Fprintf(output, "Account: %s\n", a.ID)
		}
		return a.ID, nil
	}

	fmt.Fprintln(output, "\nMultiple accounts found:")
	displayCount := len(accounts)
	if displayCount > maxDisplayAccounts {
		displayCount = maxDisplayAccounts
	}
	for i := 0; i < displayCount; i++ {
		a := accounts[i]
		if a.Name != "" {
			fmt.Fprintf(output, "  [%d] %s (%s)\n", i+1, a.Name, a.ID)
		} else {
			fmt.Fprintf(output, "  [%d] %s\n", i+1, a.ID)
		}
	}
	if len(accounts) > maxDisplayAccounts {
		fmt.Fprintf(output, "  ... and %d more (use --account-id to specify directly)\n", len(accounts)-maxDisplayAccounts)
	}
	fmt.Fprintf(output, "Select account (1-%d): ", len(accounts))

	line, err := input.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("no input received")
	}
	selection := strings.TrimSpace(line)
	idx, err := strconv.Atoi(selection)
	if err != nil || idx < 1 || idx > len(accounts) {
		return "", fmt.Errorf("invalid selection %q: must be 1-%d", selection, len(accounts))
	}
	return accounts[idx-1].ID, nil
}

// resolveAccountNames creates account choices with resolved display names.
// Creates a temporary API client pre-seeded with the login token to look up
// names. Falls back to ID-only choices if name resolution fails for any account.
func resolveAccountNames(ctx context.Context, accountIDs []string, email, password, endpoint, token string, expiresAt time.Time) []accountChoice {
	choices := make([]accountChoice, len(accountIDs))
	for i, id := range accountIDs {
		choices[i] = accountChoice{ID: id}
	}

	tempClient, err := api.NewClient(email, password, accountIDs[0],
		api.WithEndpoint(endpoint),
		api.WithToken(token, expiresAt),
	)
	if err != nil {
		return choices
	}

	for i, id := range accountIDs {
		acct, err := tempClient.GetAccountByID(ctx, id)
		if err == nil && acct.Name != "" {
			choices[i].Name = acct.Name
		}
	}
	return choices
}
