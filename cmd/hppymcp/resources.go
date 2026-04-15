package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerResources(server *mcp.Server, client apiClient) {
	server.AddResource(
		&mcp.Resource{
			URI:         "happyco://account",
			Name:        "Account info",
			Description: "Returns authenticated account's name and ID",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			account, err := client.GetAccount(ctx)
			if err != nil {
				log.Printf("[error] resource account: %v", err)
				return nil, fmt.Errorf("failed to retrieve account information")
			}
			data, err := json.MarshalIndent(account, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to format account data")
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:  "happyco://account",
					Text: string(data),
				}},
			}, nil
		},
	)

	server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "happyco://properties/{property_id}",
			Name:        "Property details",
			Description: "Returns property name, address, and unit count",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			propertyID := extractPropertyID(req.Params.URI)
			if propertyID == "" {
				return nil, fmt.Errorf("missing property ID in URI")
			}
			if err := models.ValidateID("property_id", propertyID); err != nil {
				return nil, fmt.Errorf("invalid property ID format")
			}

			if err := acquireSem(ctx, sem); err != nil {
				return nil, fmt.Errorf("failed to retrieve property")
			}
			defer releaseSem(sem)

			opts := models.ListOptions{LocationID: propertyID, Limit: 1}
			properties, _, err := client.ListProperties(ctx, opts)
			if err != nil {
				log.Printf("[error] resource property %s: %v", propertyID, err)
				return nil, fmt.Errorf("failed to retrieve property")
			}
			if len(properties) == 0 {
				return nil, fmt.Errorf("property not found")
			}

			unitOpts := models.ListOptions{Limit: 1}
			_, unitCount, err := client.ListUnits(ctx, propertyID, unitOpts)
			if err != nil {
				log.Printf("[error] resource property %s units: %v", propertyID, err)
				return nil, fmt.Errorf("failed to retrieve unit count")
			}

			result := map[string]any{
				"id":         properties[0].ID,
				"name":       properties[0].Name,
				"created_at": properties[0].CreatedAt,
				"address":    properties[0].Address,
				"unit_count": unitCount,
			}

			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to format property data")
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:  req.Params.URI,
					Text: string(data),
				}},
			}, nil
		},
	)
}

// extractPropertyID parses a property ID from a happyco://properties/{id} URI.
func extractPropertyID(uri string) string {
	const prefix = "happyco://properties/"
	if strings.HasPrefix(uri, prefix) && len(uri) > len(prefix) {
		return uri[len(prefix):]
	}
	return ""
}
