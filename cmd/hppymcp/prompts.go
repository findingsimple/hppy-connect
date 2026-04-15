package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxDaysBack = 365

// validateDaysBack validates that a days_back string is a positive integer (max 365), returning the sanitised value.
func validateDaysBack(s string) (string, error) {
	if s == "" {
		return "30", nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return "", fmt.Errorf("days_back must be a positive integer (max %d)", maxDaysBack)
	}
	if n > maxDaysBack {
		return "", fmt.Errorf("days_back must be at most %d", maxDaysBack)
	}
	return strconv.Itoa(n), nil
}

// requirePropertyID extracts and validates property_id from prompt arguments.
func requirePropertyID(args map[string]string) (string, error) {
	id := args["property_id"]
	if id == "" {
		return "", fmt.Errorf("property_id is required")
	}
	if err := models.ValidateID("property_id", id); err != nil {
		return "", err
	}
	return id, nil
}

func registerPrompts(server *mcp.Server) {
	server.AddPrompt(
		&mcp.Prompt{
			Name:        "property_summary",
			Description: "Summarise all units and open work orders for a given property",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "property_id",
					Description: "The property ID to summarise",
					Required:    true,
				},
			},
		},
		func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			propertyID, err := requirePropertyID(req.Params.Arguments)
			if err != nil {
				return nil, err
			}

			return &mcp.GetPromptResult{
				Description: fmt.Sprintf("Summary of property %s", propertyID),
				Messages: []*mcp.PromptMessage{
					{
						Role: "user",
						Content: &mcp.TextContent{
							Text: fmt.Sprintf(`Please provide a summary for property %s. Follow these steps:

1. Use the get_account tool to confirm the account context.
2. Use the list_units tool with property_id "%s" to get all units.
3. Use the list_work_orders tool with property_id "%s" and status "OPEN" to get open work orders.
4. Summarise the property including:
   - Total number of units
   - Number of open work orders
   - A brief overview of the open work orders (priority, description, assigned to)`, propertyID, propertyID, propertyID),
						},
					},
				},
			}, nil
		},
	)

	server.AddPrompt(
		&mcp.Prompt{
			Name:        "maintenance_report",
			Description: "Generate a maintenance status report including open work orders and recent inspections",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "property_id",
					Description: "The property ID to report on",
					Required:    true,
				},
				{
					Name:        "days_back",
					Description: "Number of days to look back (default: 30)",
					Required:    false,
				},
			},
		},
		func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			propertyID, err := requirePropertyID(req.Params.Arguments)
			if err != nil {
				return nil, err
			}

			daysBack, err := validateDaysBack(req.Params.Arguments["days_back"])
			if err != nil {
				return nil, err
			}

			return &mcp.GetPromptResult{
				Description: fmt.Sprintf("Maintenance report for property %s (last %s days)", propertyID, daysBack),
				Messages: []*mcp.PromptMessage{
					{
						Role: "user",
						Content: &mcp.TextContent{
							Text: fmt.Sprintf(`Generate a maintenance status report for property %s covering the last %s days. Follow these steps:

1. Use the get_account tool to confirm the account context.
2. Use the list_work_orders tool with property_id "%s" and status "OPEN" to get all open work orders.
3. Use the list_work_orders tool with property_id "%s" and status "COMPLETED" to get recently completed work orders.
4. Use the list_inspections tool with property_id "%s" to get recent inspections.
5. Generate a maintenance report including:
   - Open work orders: count, priority breakdown, oldest open item
   - Recently completed work orders: count, average time to completion
   - Inspections: count by status, average scores
   - Any urgent items requiring immediate attention`, propertyID, daysBack, propertyID, propertyID, propertyID),
						},
					},
				},
			}, nil
		},
	)
}
