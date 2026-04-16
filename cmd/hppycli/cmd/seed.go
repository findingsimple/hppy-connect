package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/spf13/cobra"
)

// seedClient defines the API surface used by the seed command.
// Mirrors the MCP server's composed-interface pattern for testability.
type seedClient interface {
	ListProperties(ctx context.Context, opts models.ListOptions) ([]models.Property, int, error)
	ListUnits(ctx context.Context, propertyID string, opts models.ListOptions) ([]models.Unit, int, error)
	WorkOrderCreate(ctx context.Context, input models.WorkOrderCreateInput) (*models.WorkOrder, error)
	InspectionCreate(ctx context.Context, input models.InspectionCreateInput) (*models.Inspection, error)
	InspectionStart(ctx context.Context, inspectionID string) (*models.Inspection, error)
	ProjectCreate(ctx context.Context, input models.ProjectCreateInput) (*models.Project, error)
	WebhookCreate(ctx context.Context, input models.WebhookCreateInput) (*models.Webhook, error)
}

// seedResult tracks a single seed operation outcome.
type seedResult struct {
	EntityType  string `json:"type"`
	ID          string `json:"id,omitempty"`
	Location    string `json:"location,omitempty"`
	Description string `json:"description"`
	Error       string `json:"error,omitempty"`
}

// seedLocation holds a discovered property or unit for seeding.
type seedLocation struct {
	ID          string
	DisplayName string
	PropertyID  string
}

// seedOptions holds parsed flags for the seed command.
type seedOptions struct {
	InspTemplateID string
	ProjTemplateID string
	WebhookURL     string
	Count          int
	DryRun         bool
	AccountID      string
}

// workOrderRecipe defines a template for creating a work order.
type workOrderRecipe struct {
	Description string
	Priority    string
	Type        string
	Status      string
}

var recipes = []workOrderRecipe{
	{"Leaking faucet in kitchen", "URGENT", "SERVICE_REQUEST", "OPEN"},
	{"Replace HVAC filter", "NORMAL", "PREVENTATIVE_MAINTENANCE", "OPEN"},
	{"Repaint hallway walls", "NORMAL", "CURB_APPEAL", "COMPLETED"},
	{"Fire extinguisher inspection overdue", "URGENT", "LIFE_SAFETY", "OPEN"},
	{"Unit turnover prep - deep clean", "NORMAL", "TURN", "ON_HOLD"},
	{"Fix broken window latch", "NORMAL", "SERVICE_REQUEST", "OPEN"},
	{"Annual roof inspection", "NORMAL", "REGULATORY", "COMPLETED"},
	{"Replace lobby carpet", "URGENT", "CAPITAL_IMPROVEMENT", "OPEN"},
	{"Snow removal equipment check", "NORMAL", "SEASONAL_MAINTENANCE", "OPEN"},
	{"Pest control treatment", "NORMAL", "INCIDENT", "OPEN"},
}

// plannedItem represents a single entity to be created during seeding.
type plannedItem struct {
	EntityType      string
	LocationID      string
	DisplayName     string
	Description     string // for display
	Recipe          *workOrderRecipe
	InspTemplateID  string
	InspDayOffset   int
	InspShouldStart bool
	ProjTemplateID  string
	ProjPriority    string
	WebhookURL      string
}

const maxSeedCount = 50

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Populate account with test data",
	Long: `Seed your HappyCo account with realistic test data for exercising
the CLI and MCP server. Auto-discovers properties and units, then creates
work orders, inspections, projects, and webhooks.

Properties and units cannot be created via the API — they must already exist
in your account. Inspection and project templates must be provided by ID.`,
	Example: `  # Preview what would be created
  hppycli seed --dry-run

  # Create work orders only (3 per location)
  hppycli seed --yes

  # Full seed with inspections and projects
  hppycli seed --inspection-template-id=tmpl123 --project-template-id=ptmpl456 --yes

  # More work orders per location
  hppycli seed --count=5 --yes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		stdout := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()

		// Parse flags.
		inspTemplateID, _ := cmd.Flags().GetString("inspection-template-id")
		projTemplateID, _ := cmd.Flags().GetString("project-template-id")
		webhookURL, _ := cmd.Flags().GetString("webhook-url")
		count, _ := cmd.Flags().GetInt("count")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if count < 1 || count > maxSeedCount {
			return fmt.Errorf("--count must be between 1 and %d", maxSeedCount)
		}
		if inspTemplateID != "" {
			if err := models.ValidateID("inspection-template-id", inspTemplateID); err != nil {
				return err
			}
		}
		if projTemplateID != "" {
			if err := models.ValidateID("project-template-id", projTemplateID); err != nil {
				return err
			}
		}
		if webhookURL != "" {
			if err := models.ValidateWebhookURL(webhookURL); err != nil {
				return err
			}
		}

		// Resolve account ID for webhook (validates via resolveAccountID).
		var accountID string
		if webhookURL != "" {
			var err error
			accountID, err = resolveAccountID(cmd)
			if err != nil {
				return fmt.Errorf("--webhook-url requires a valid account ID: %w", err)
			}
		}

		// Confirm before proceeding.
		if !dryRun {
			if err := confirmAction(cmd, "seed test data into your HappyCo account", os.Stdin, stderr); err != nil {
				return err
			}
		}

		opts := seedOptions{
			InspTemplateID: inspTemplateID,
			ProjTemplateID: projTemplateID,
			WebhookURL:     webhookURL,
			Count:          count,
			DryRun:         dryRun,
			AccountID:      accountID,
		}

		return runSeed(ctx, apiClient, opts, stdout, stderr)
	},
}

// runSeed contains the core seed logic, separated for testability.
func runSeed(ctx context.Context, client seedClient, opts seedOptions, stdout io.Writer, stderr io.Writer) error {
	// Discover properties and units.
	fmt.Fprintln(stderr, "Discovering properties and units...")
	properties, _, err := client.ListProperties(ctx, models.ListOptions{Limit: 3})
	if err != nil {
		return fmt.Errorf("listing properties: %w", err)
	}
	if len(properties) == 0 {
		return fmt.Errorf("no properties found — seed requires at least one property in your account")
	}

	var locations []seedLocation
	for _, p := range properties {
		units, _, err := client.ListUnits(ctx, p.ID, models.ListOptions{Limit: 5})
		if err != nil {
			fmt.Fprintf(stderr, "Warning: could not list units for property %s: %v\n", p.Name, err)
			locations = append(locations, seedLocation{
				ID:          p.ID,
				DisplayName: p.Name,
				PropertyID:  p.ID,
			})
			continue
		}
		if len(units) == 0 {
			locations = append(locations, seedLocation{
				ID:          p.ID,
				DisplayName: p.Name,
				PropertyID:  p.ID,
			})
		} else {
			for _, u := range units {
				locations = append(locations, seedLocation{
					ID:          u.ID,
					DisplayName: p.Name + " > " + u.Name,
					PropertyID:  p.ID,
				})
			}
		}
	}

	fmt.Fprintf(stderr, "Found %d locations across %d properties\n", len(locations), len(properties))

	// Build the plan once — used for both dry-run display and execution.
	plan := buildSeedPlan(locations, opts)

	// Dry-run: print plan and exit.
	if opts.DryRun {
		fmt.Fprintf(stderr, "\nDry run — %d items would be created:\n\n", len(plan))
		w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TYPE\tLOCATION\tDESCRIPTION")
		for _, p := range plan {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				sanitizeCell(p.EntityType), sanitizeCell(p.DisplayName), sanitizeCell(p.Description))
		}
		return w.Flush()
	}

	// Execute the plan.
	batchTag := time.Now().Format("2006-01-02T15:04")
	results := executeSeedPlan(ctx, client, plan, opts, batchTag, stderr)

	// Print summary.
	fmt.Fprintln(stderr)
	printSeedSummary(stdout, stderr, results)

	failCount := 0
	for _, r := range results {
		if r.Error != "" {
			failCount++
		}
	}
	if failCount > 0 {
		return fmt.Errorf("%d of %d seed operations failed", failCount, len(results))
	}
	return nil
}

// buildSeedPlan creates the full list of items to seed from discovered locations and options.
// This single plan is used for both dry-run display and execution, preventing drift.
func buildSeedPlan(locations []seedLocation, opts seedOptions) []plannedItem {
	var plan []plannedItem

	// Work orders: cycle recipes across locations.
	recipeIdx := 0
	for _, loc := range locations {
		for i := 0; i < opts.Count; i++ {
			r := recipes[recipeIdx%len(recipes)]
			plan = append(plan, plannedItem{
				EntityType:  "work-order",
				LocationID:  loc.ID,
				DisplayName: loc.DisplayName,
				Description: fmt.Sprintf("%s [%s, %s → %s]", r.Description, r.Priority, r.Type, r.Status),
				Recipe:      &recipes[recipeIdx%len(recipes)],
			})
			recipeIdx++
		}
	}

	// Inspections: 2 per property (on different locations).
	if opts.InspTemplateID != "" {
		seen := map[string]int{}
		for _, loc := range locations {
			if seen[loc.PropertyID] >= 2 {
				continue
			}
			dayOffset := seen[loc.PropertyID] + 1
			shouldStart := seen[loc.PropertyID] == 1
			status := "SCHEDULED"
			if shouldStart {
				status = "→ INCOMPLETE (started)"
			}
			plan = append(plan, plannedItem{
				EntityType:      "inspection",
				LocationID:      loc.ID,
				DisplayName:     loc.DisplayName,
				Description:     fmt.Sprintf("Scheduled +%dd, %s", dayOffset, status),
				InspTemplateID:  opts.InspTemplateID,
				InspDayOffset:   dayOffset,
				InspShouldStart: shouldStart,
			})
			seen[loc.PropertyID]++
		}
	}

	// Projects: 1 per property.
	if opts.ProjTemplateID != "" {
		seen := map[string]bool{}
		priorityIdx := 0
		for _, loc := range locations {
			if seen[loc.PropertyID] {
				continue
			}
			priority := "NORMAL"
			if priorityIdx%2 == 1 {
				priority = "URGENT"
			}
			plan = append(plan, plannedItem{
				EntityType:     "project",
				LocationID:     loc.ID,
				DisplayName:    loc.DisplayName,
				Description:    fmt.Sprintf("Priority: %s", priority),
				ProjTemplateID: opts.ProjTemplateID,
				ProjPriority:   priority,
			})
			seen[loc.PropertyID] = true
			priorityIdx++
		}
	}

	// Webhook.
	if opts.WebhookURL != "" {
		plan = append(plan, plannedItem{
			EntityType:  "webhook",
			DisplayName: "-",
			Description: fmt.Sprintf("%s (DISABLED)", truncateString(opts.WebhookURL, 50)),
			WebhookURL:  opts.WebhookURL,
		})
	}

	return plan
}

// executeSeedPlan runs through the plan, creating entities and collecting results.
func executeSeedPlan(ctx context.Context, client seedClient, plan []plannedItem, opts seedOptions, batchTag string, stderr io.Writer) []seedResult {
	var results []seedResult
	now := time.Now()

	for _, item := range plan {
		if ctx.Err() != nil {
			results = append(results, seedResult{
				EntityType:  item.EntityType,
				Location:    item.DisplayName,
				Description: item.Description,
				Error:       "cancelled",
			})
			continue
		}

		switch item.EntityType {
		case "work-order":
			r := item.Recipe
			desc := fmt.Sprintf("[SEED %s] %s", batchTag, r.Description)
			input := models.WorkOrderCreateInput{
				LocationID:  item.LocationID,
				Description: desc,
				Priority:    r.Priority,
				Type:        r.Type,
				Status:      r.Status,
			}

			wo, err := client.WorkOrderCreate(ctx, input)
			if err != nil {
				results = append(results, seedResult{
					EntityType:  "work-order",
					Location:    item.DisplayName,
					Description: r.Description,
					Error:       err.Error(),
				})
			} else {
				results = append(results, seedResult{
					EntityType:  "work-order",
					ID:          wo.ID,
					Location:    item.DisplayName,
					Description: fmt.Sprintf("%s → %s", r.Description, r.Status),
				})
			}

		case "inspection":
			scheduledFor := now.AddDate(0, 0, item.InspDayOffset).Format(time.RFC3339)
			input := models.InspectionCreateInput{
				LocationID:   item.LocationID,
				TemplateID:   item.InspTemplateID,
				ScheduledFor: scheduledFor,
			}

			insp, err := client.InspectionCreate(ctx, input)
			if err != nil {
				results = append(results, seedResult{
					EntityType:  "inspection",
					Location:    item.DisplayName,
					Description: fmt.Sprintf("template=%s", item.InspTemplateID),
					Error:       err.Error(),
				})
				continue
			}

			result := seedResult{
				EntityType:  "inspection",
				ID:          insp.ID,
				Location:    item.DisplayName,
				Description: fmt.Sprintf("Scheduled +%dd", item.InspDayOffset),
			}

			if item.InspShouldStart {
				_, startErr := client.InspectionStart(ctx, insp.ID)
				if startErr != nil {
					result.Description += fmt.Sprintf(" (start failed: %s)", startErr.Error())
				} else {
					result.Description += " → INCOMPLETE"
				}
			}

			results = append(results, result)

		case "project":
			input := models.ProjectCreateInput{
				ProjectTemplateID: item.ProjTemplateID,
				LocationID:        item.LocationID,
				StartAt:           now.Format(time.RFC3339),
				Priority:          item.ProjPriority,
			}

			proj, err := client.ProjectCreate(ctx, input)
			if err != nil {
				results = append(results, seedResult{
					EntityType:  "project",
					Location:    item.DisplayName,
					Description: fmt.Sprintf("template=%s", item.ProjTemplateID),
					Error:       err.Error(),
				})
			} else {
				results = append(results, seedResult{
					EntityType:  "project",
					ID:          proj.ID,
					Location:    item.DisplayName,
					Description: fmt.Sprintf("Priority: %s", item.ProjPriority),
				})
			}

		case "webhook":
			input := models.WebhookCreateInput{
				SubscriberID:   opts.AccountID,
				SubscriberType: "ACCOUNT",
				URL:            item.WebhookURL,
				Subjects:       []string{"INSPECTIONS", "WORK_ORDERS"},
				Status:         "DISABLED",
			}

			wh, err := client.WebhookCreate(ctx, input)
			if err != nil {
				results = append(results, seedResult{
					EntityType:  "webhook",
					Description: truncateString(item.WebhookURL, 50),
					Error:       err.Error(),
				})
			} else {
				results = append(results, seedResult{
					EntityType:  "webhook",
					ID:          wh.ID,
					Description: fmt.Sprintf("%s (DISABLED)", truncateString(item.WebhookURL, 50)),
				})
			}
		}

		fmt.Fprintf(stderr, ".")
	}

	return results
}

// printSeedSummary outputs the results table and counts.
func printSeedSummary(stdout io.Writer, stderr io.Writer, results []seedResult) {
	successCount := 0
	failCount := 0

	switch outputFormat {
	case "json":
		var created, failed []seedResult
		for _, r := range results {
			if r.Error != "" {
				failCount++
				failed = append(failed, r)
			} else {
				successCount++
				created = append(created, r)
			}
		}
		wrapper := map[string]any{
			"created": created,
			"failed":  failed,
			"summary": map[string]int{
				"total":   len(results),
				"success": successCount,
				"failed":  failCount,
			},
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		enc.Encode(wrapper)

	default:
		w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TYPE\tID\tLOCATION\tDESCRIPTION\tERROR")
		for _, r := range results {
			errStr := ""
			if r.Error != "" {
				errStr = truncateString(r.Error, 60)
				failCount++
			} else {
				successCount++
			}
			id := r.ID
			if id == "" {
				id = "-"
			}
			loc := r.Location
			if loc == "" {
				loc = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				r.EntityType, id, sanitizeCell(loc),
				sanitizeCell(truncateString(r.Description, 50)),
				sanitizeCell(errStr))
		}
		w.Flush()
	}

	total := successCount + failCount
	fmt.Fprintf(stderr, "\nCreated %d of %d items", successCount, total)
	if failCount > 0 {
		fmt.Fprintf(stderr, " (%d failed)", failCount)
	}
	fmt.Fprintln(stderr)
}

func init() {
	seedCmd.Flags().String("inspection-template-id", "", "inspection template ID (enables inspection seeding)")
	seedCmd.Flags().String("project-template-id", "", "project template ID (enables project seeding)")
	seedCmd.Flags().String("webhook-url", "", "webhook endpoint URL (enables webhook seeding; must be HTTPS)")
	seedCmd.Flags().Int("count", 3, "work orders to create per location (max 50)")
	seedCmd.Flags().Bool("dry-run", false, "show what would be created without making API calls")
	seedCmd.Flags().Bool("yes", false, "skip confirmation prompt")
}
