package api

import (
	"encoding/json"

	"github.com/findingsimple/hppy-connect/internal/models"
)

type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphqlError  `json:"errors,omitempty"`
}

type graphqlError struct {
	Message    string                 `json:"message"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

type pageInfo struct {
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	EndCursor       string `json:"endCursor"`
}

type connection[T any] struct {
	Count    int       `json:"count"`
	PageInfo pageInfo  `json:"pageInfo"`
	Edges    []edge[T] `json:"edges"`
}

type edge[T any] struct {
	Node   T      `json:"node"`
	Cursor string `json:"cursor"`
}

type accountResponse struct {
	Account models.Account `json:"account"`
}

type propertiesResponse struct {
	Account struct {
		Properties connection[models.Property] `json:"properties"`
	} `json:"account"`
}

type unitsResponse struct {
	Account struct {
		Properties struct {
			Edges []struct {
				Node struct {
					Units connection[models.Unit] `json:"units"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"properties"`
	} `json:"account"`
}

type workOrdersResponse struct {
	Account struct {
		WorkOrders connection[models.WorkOrder] `json:"workOrders"`
	} `json:"account"`
}

type inspectionsResponse struct {
	Account struct {
		Inspections connection[models.Inspection] `json:"inspections"`
	} `json:"account"`
}

type loginResponse struct {
	Login struct {
		Token                 string   `json:"token"`
		ExpiresAt             string   `json:"expiresAt"`
		AccessibleBusinessIds []string `json:"accessibleBusinessIds"`
	} `json:"login"`
}
