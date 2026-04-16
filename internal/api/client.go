// Package api provides a GraphQL client for the HappyCo external API,
// handling authentication, token refresh, pagination, and retry logic.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/findingsimple/hppy-connect/internal/models"
)

// DefaultEndpoint is the HappyCo external GraphQL API endpoint.
const DefaultEndpoint = "https://externalgraph.happyco.com"

const (
	pageSize        = 100
	defaultCap      = 1000
	hardMaxItems    = 50000 // defence-in-depth ceiling for unbounded pagination
	expiryBuffer    = 5 * time.Minute
	maxRetries      = 3
	maxResponseSize = 10 * 1024 * 1024 // 10 MB
)

// tokenState holds auth credentials atomically swapped as a unit.
type tokenState struct {
	token     string
	expiresAt time.Time
}

// apiError carries an error category to distinguish retryable from terminal errors.
type apiError struct {
	Category   string // auth_failed, not_found, invalid_input, rate_limited, api_error
	Message    string
	Retryable  bool
	RetryAfter time.Duration // from Retry-After header on 429 responses
}

func (e *apiError) Error() string {
	return fmt.Sprintf("%s: %s", e.Category, e.Message)
}

func errAuth(msg string) error     { return &apiError{Category: "auth_failed", Message: msg} }
func errAPI(msg string) error      { return &apiError{Category: "api_error", Message: msg, Retryable: true} }
func errAPIFatal(msg string) error { return &apiError{Category: "api_error", Message: msg} }
func errRateLimited(msg string) error {
	return &apiError{Category: "rate_limited", Message: msg, Retryable: true}
}
func errRateLimitedWithRetryAfter(msg, retryAfterHeader string) error {
	e := &apiError{Category: "rate_limited", Message: msg, Retryable: true}
	if retryAfterHeader != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(retryAfterHeader)); err == nil && secs > 0 {
			e.RetryAfter = time.Duration(secs) * time.Second
		}
	}
	return e
}

const loginCooldown = 30 * time.Second

// Client is a GraphQL API client for the HappyCo external graph.
type Client struct {
	endpoint       string
	httpClient     *http.Client
	email          string
	password       string
	accountID      string
	authState      atomic.Pointer[tokenState]
	mu             sync.Mutex
	debug          bool
	allowInsecure  bool // testing only — bypasses HTTPS requirement
	retryBackoff   []time.Duration
	lastLoginFail  time.Time // protected by mu — prevents login hammering
	lastLoginError error     // protected by mu — cached error during cooldown
}

// Option configures a Client.
type Option func(*Client)

// WithDebug enables debug logging.
func WithDebug(debug bool) Option {
	return func(c *Client) { c.debug = debug }
}

// WithEndpoint overrides the default GraphQL endpoint.
// Validation (HTTPS requirement) is deferred to NewClient.
func WithEndpoint(endpoint string) Option {
	return func(c *Client) { c.endpoint = endpoint }
}

// WithToken pre-seeds the auth state with a token from a prior Login call.
// This avoids a redundant login when a token is already available (e.g. during
// interactive account selection after api.Login).
func WithToken(token string, expiresAt time.Time) Option {
	return func(c *Client) {
		c.authState.Store(&tokenState{token: token, expiresAt: expiresAt})
	}
}

// withRetryBackoff overrides retry backoff durations (for testing).
func withRetryBackoff(backoff []time.Duration) Option {
	return func(c *Client) { c.retryBackoff = backoff }
}

// withInsecureHTTP opts out of the HTTPS requirement (for testing only).
func withInsecureHTTP() Option {
	return func(c *Client) { c.allowInsecure = true }
}

// NewClient creates a new API client. Returns an error if the endpoint
// is not HTTPS (credentials are transmitted over this connection).
func NewClient(email, password, accountID string, opts ...Option) (*Client, error) {
	c := &Client{
		endpoint:     DefaultEndpoint,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		email:        email,
		password:     password,
		accountID:    accountID,
		retryBackoff: []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second},
	}
	c.authState.Store(&tokenState{})
	for _, opt := range opts {
		opt(c)
	}

	if !c.allowInsecure {
		u, err := url.Parse(c.endpoint)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return nil, fmt.Errorf("endpoint %q must be a valid https:// URL (credentials are transmitted)", c.endpoint)
		}
	}

	return c, nil
}

// EnsureAuth is a public wrapper around ensureAuth for health checks.
func (c *Client) EnsureAuth(ctx context.Context) error {
	_, err := c.ensureAuth(ctx)
	return err
}

func (c *Client) getAuth() tokenState {
	return *c.authState.Load()
}

// ensureAuth returns a valid tokenState, re-authenticating if needed.
// Returning the state directly eliminates the TOCTOU gap between
// checking auth and reading the token in doQuery.
func (c *Client) ensureAuth(ctx context.Context) (tokenState, error) {
	// Fast path: atomic read — no lock needed
	if auth := c.getAuth(); time.Now().Before(auth.expiresAt.Add(-expiryBuffer)) {
		return auth, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-check after acquiring lock (another goroutine may have refreshed)
	if auth := c.getAuth(); time.Now().Before(auth.expiresAt.Add(-expiryBuffer)) {
		return auth, nil
	}

	// Prevent login hammering after recent failure
	if !c.lastLoginFail.IsZero() && time.Since(c.lastLoginFail) < loginCooldown {
		return tokenState{}, c.lastLoginError
	}

	if err := c.login(ctx); err != nil {
		// Only apply cooldown for permanent failures (bad credentials, rejected input).
		// Transient errors (network timeouts, 5xx, rate limiting) should allow immediate retry.
		var ae *apiError
		if errors.As(err, &ae) && !ae.Retryable {
			c.lastLoginFail = time.Now()
			c.lastLoginError = err
		}
		return tokenState{}, err
	}
	c.lastLoginFail = time.Time{}
	c.lastLoginError = nil
	return c.getAuth(), nil
}

// invalidateAuth clears the cached token so the next ensureAuth call re-authenticates.
func (c *Client) invalidateAuth() {
	c.authState.Store(&tokenState{})
}

func (c *Client) login(ctx context.Context) error {
	result, err := doLogin(ctx, c.httpClient, c.endpoint, c.email, c.password, c.debug)
	if err != nil {
		return err
	}

	c.authState.Store(&tokenState{
		token:     result.Token,
		expiresAt: result.ExpiresAt,
	})

	return nil
}

// LoginResult contains the data returned by a successful login.
type LoginResult struct {
	Token      string
	ExpiresAt  time.Time
	AccountIDs []string // accessibleBusinessIds
}

// Login authenticates with the HappyCo API and returns account IDs.
// This is a standalone function for use during initial setup (config init)
// where a full Client is not yet configured.
func Login(ctx context.Context, email, password, endpoint string) (*LoginResult, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, errAuth(fmt.Sprintf("endpoint %q must be a valid https:// URL (credentials are transmitted)", endpoint))
	}

	return doLogin(ctx, &http.Client{Timeout: 30 * time.Second}, endpoint, email, password, false)
}

// doLogin is the shared login implementation used by both Client.login and the
// standalone Login function.
func doLogin(ctx context.Context, httpClient *http.Client, endpoint, email, password string, debug bool) (*LoginResult, error) {
	vars := map[string]interface{}{
		"input": map[string]string{
			"email":    email,
			"password": password,
		},
	}

	body, err := json.Marshal(map[string]interface{}{
		"query":     loginMutation,
		"variables": vars,
	})
	if err != nil {
		return nil, errAuth(fmt.Sprintf("marshalling login request: %v", err))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errAuth(fmt.Sprintf("creating login request: %v", err))
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errAPI(fmt.Sprintf("login request: %v", err))
	}
	defer resp.Body.Close()

	if debug {
		log.Printf("[debug] login POST %s status=%d duration=%s", endpoint, resp.StatusCode, time.Since(start))
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, errAPI(fmt.Sprintf("reading login response: %v", err))
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, errRateLimitedWithRetryAfter("login rate limited", resp.Header.Get("Retry-After"))
	}
	if resp.StatusCode >= 500 {
		return nil, errAPI(fmt.Sprintf("login returned HTTP %d", resp.StatusCode))
	}
	if resp.StatusCode >= 400 {
		return nil, errAuth(fmt.Sprintf("login returned HTTP %d", resp.StatusCode))
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, errAuth(fmt.Sprintf("parsing login response: %v", err))
	}
	if len(gqlResp.Errors) > 0 {
		return nil, errAuth(gqlResp.Errors[0].Message)
	}

	var loginData loginResponse
	if err := json.Unmarshal(gqlResp.Data, &loginData); err != nil {
		return nil, errAuth(fmt.Sprintf("parsing login data: %v", err))
	}

	ms, err := strconv.ParseInt(loginData.Login.ExpiresAt, 10, 64)
	if err != nil {
		return nil, errAuth(fmt.Sprintf("parsing expiresAt %q: %v", loginData.Login.ExpiresAt, err))
	}

	return &LoginResult{
		Token:      loginData.Login.Token,
		ExpiresAt:  time.Unix(0, ms*int64(time.Millisecond)),
		AccountIDs: loginData.Login.AccessibleBusinessIds,
	}, nil
}

func (c *Client) doQuery(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	auth, err := c.ensureAuth(ctx)
	if err != nil {
		return err
	}

	body, err := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return errAPIFatal(fmt.Sprintf("marshalling query: %v", err))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return errAPIFatal(fmt.Sprintf("creating request: %v", err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+auth.token)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errAPI(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return errAPI(fmt.Sprintf("reading response: %v", err))
	}

	if c.debug {
		log.Printf("[debug] POST %s status=%d duration=%s size=%d", c.endpoint, resp.StatusCode, time.Since(start), len(respBody))
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return errRateLimitedWithRetryAfter("API rate limited", resp.Header.Get("Retry-After"))
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		c.invalidateAuth()
		return errAuth(fmt.Sprintf("HTTP %d", resp.StatusCode))
	}
	if resp.StatusCode >= 500 {
		return errAPI(fmt.Sprintf("HTTP %d", resp.StatusCode))
	}
	if resp.StatusCode >= 400 {
		return errAPIFatal(fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return errAPIFatal(fmt.Sprintf("parsing response: %v", err))
	}
	if len(gqlResp.Errors) > 0 {
		return errAPIFatal(gqlResp.Errors[0].Message)
	}

	if len(gqlResp.Data) == 0 || string(gqlResp.Data) == "null" {
		return errAPIFatal("API returned null data")
	}

	if err := json.Unmarshal(gqlResp.Data, result); err != nil {
		return errAPIFatal(fmt.Sprintf("parsing data: %v", err))
	}

	return nil
}

// doMutation sends a mutation with a single auth-retry but no transient-error retry.
// If the request gets a 401, re-authenticates once and retries. This assumes the
// HappyCo gateway rejects auth failures before executing the mutation — if the
// gateway ever commits before returning 401, the retry could create duplicates.
// See CLAUDE.md "Known Limitations" item C for the full rationale.
//
// On network timeout, the error is returned without retry. The caller cannot know
// whether the server committed — this is inherent to non-idempotent HTTP operations.
//
// Use for non-idempotent operations: creates, adds, sends, duplicates.
func (c *Client) doMutation(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	err := c.doQuery(ctx, query, variables, result)
	if err == nil {
		return nil
	}
	var ae *apiError
	if errors.As(err, &ae) && ae.Category == "auth_failed" {
		if _, authErr := c.ensureAuth(ctx); authErr != nil {
			return err
		}
		return c.doQuery(ctx, query, variables, result)
	}
	return err
}

// doMutationIdempotent sends an idempotent mutation with full retry logic
// (auth retry + transient error retry with backoff).
// Use for set*, archive, delete, remove, start, complete, reopen, etc.
func (c *Client) doMutationIdempotent(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	return c.doQueryWithRetry(ctx, query, variables, result)
}

// GetAccount returns the account details with retry for transient errors.
func (c *Client) GetAccount(ctx context.Context) (*models.Account, error) {
	vars := map[string]interface{}{
		"accountId": c.accountID,
	}
	var resp accountResponse
	if err := c.doQueryWithRetry(ctx, getAccountQuery, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.Account, nil
}

// GetAccountByID returns account details for any accessible account ID.
// Unlike GetAccount (which uses the client's configured account ID), this
// queries an arbitrary account — used during interactive account selection.
func (c *Client) GetAccountByID(ctx context.Context, accountID string) (*models.Account, error) {
	vars := map[string]interface{}{
		"accountId": accountID,
	}
	var resp accountResponse
	if err := c.doQueryWithRetry(ctx, getAccountQuery, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.Account, nil
}

// doQueryWithRetry wraps doQuery with the same retry logic used by paginated fetches.
// Suitable for idempotent operations (reads, setters, state transitions).
func (c *Client) doQueryWithRetry(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	var lastErr error
	authRetried := false
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return errAPIFatal(fmt.Sprintf("context cancelled: %v", err))
		}

		err := c.doQuery(ctx, query, variables, result)
		if err == nil {
			return nil
		}
		lastErr = err

		var ae *apiError
		if errors.As(err, &ae) {
			if ae.Category == "auth_failed" && !authRetried {
				authRetried = true
				if _, authErr := c.ensureAuth(ctx); authErr != nil {
					return err
				}
				attempt--
				continue
			}
			if !ae.Retryable {
				return err
			}
		}

		if attempt < maxRetries-1 {
			idx := min(attempt, len(c.retryBackoff)-1)
			backoff := c.retryBackoff[idx]
			if ae != nil && ae.RetryAfter > backoff {
				backoff = ae.RetryAfter
			}
			jitter := time.Duration(rand.Int64N(int64(backoff/2))) - backoff/4
			select {
			case <-ctx.Done():
				return errAPIFatal(fmt.Sprintf("context cancelled: %v", ctx.Err()))
			case <-time.After(backoff + jitter):
			}
		}
	}
	return &apiError{
		Category: "api_error",
		Message:  fmt.Sprintf("query failed after %d retries: %v", maxRetries, lastErr),
	}
}

// ListProperties returns properties for the account.
func (c *Client) ListProperties(ctx context.Context, opts models.ListOptions) ([]models.Property, int, error) {
	return paginate(ctx, c, "properties", opts, func(vars map[string]interface{}) {
		vars["orderBy"] = []string{"NAME_ASC"}
		if filter := buildPropertiesFilter(opts); len(filter) > 0 {
			vars["filter"] = filter
		}
	}, func(vars map[string]interface{}) (*connection[models.Property], error) {
		var resp propertiesResponse
		if err := c.doQuery(ctx, listPropertiesQuery, vars, &resp); err != nil {
			return nil, err
		}
		return &resp.Account.Properties, nil
	})
}

// ListUnits returns units for a specific property.
func (c *Client) ListUnits(ctx context.Context, propertyID string, opts models.ListOptions) ([]models.Unit, int, error) {
	return paginate(ctx, c, "units", opts, func(vars map[string]interface{}) {
		vars["propertiesFilter"] = map[string]interface{}{"propertyId": []string{propertyID}}
	}, func(vars map[string]interface{}) (*connection[models.Unit], error) {
		var resp unitsResponse
		if err := c.doQuery(ctx, listUnitsQuery, vars, &resp); err != nil {
			return nil, err
		}
		// Units are nested: properties → edges → node → units
		if len(resp.Account.Properties.Edges) == 0 {
			return &connection[models.Unit]{}, nil
		}
		return &resp.Account.Properties.Edges[0].Node.Units, nil
	})
}

// ListWorkOrders returns work orders for the account.
func (c *Client) ListWorkOrders(ctx context.Context, opts models.ListOptions) ([]models.WorkOrder, int, error) {
	return paginate(ctx, c, "workOrders", opts, func(vars map[string]interface{}) {
		vars["orderBy"] = []string{"CREATED_AT_DESC"}
		if filter := buildLocationDateFilter(opts); len(filter) > 0 {
			vars["filter"] = filter
		}
	}, func(vars map[string]interface{}) (*connection[models.WorkOrder], error) {
		var resp workOrdersResponse
		if err := c.doQuery(ctx, listWorkOrdersQuery, vars, &resp); err != nil {
			return nil, err
		}
		return &resp.Account.WorkOrders, nil
	})
}

// ListInspections returns inspections for the account.
func (c *Client) ListInspections(ctx context.Context, opts models.ListOptions) ([]models.Inspection, int, error) {
	return paginate(ctx, c, "inspections", opts, func(vars map[string]interface{}) {
		vars["orderBy"] = []string{"CREATED_AT_DESC"}
		if filter := buildLocationDateFilter(opts); len(filter) > 0 {
			vars["filter"] = filter
		}
	}, func(vars map[string]interface{}) (*connection[models.Inspection], error) {
		var resp inspectionsResponse
		if err := c.doQuery(ctx, listInspectionsQuery, vars, &resp); err != nil {
			return nil, err
		}
		return &resp.Account.Inspections, nil
	})
}

// --- Work Order Mutations (19) ---

// WorkOrderCreate creates a new work order. Non-idempotent: auth-retry only.
func (c *Client) WorkOrderCreate(ctx context.Context, input models.WorkOrderCreateInput) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": input}
	var resp workOrderCreateResponse
	if err := c.doMutation(ctx, workOrderCreateMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderCreate, nil
}

// WorkOrderSetStatusAndSubStatus sets both status and sub-status. Idempotent.
func (c *Client) WorkOrderSetStatusAndSubStatus(ctx context.Context, input models.WorkOrderSetStatusAndSubStatusInput) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": input}
	var resp workOrderSetStatusAndSubStatusResponse
	if err := c.doMutationIdempotent(ctx, workOrderSetStatusAndSubStatusMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderSetStatusAndSubStatus, nil
}

// WorkOrderSetAssignee sets the work order assignee. Idempotent.
func (c *Client) WorkOrderSetAssignee(ctx context.Context, input models.WorkOrderSetAssigneeInput) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": input}
	var resp workOrderSetAssigneeResponse
	if err := c.doMutationIdempotent(ctx, workOrderSetAssigneeMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderSetAssignee, nil
}

// WorkOrderSetDescription sets the work order description. Idempotent.
func (c *Client) WorkOrderSetDescription(ctx context.Context, workOrderID, description string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId": workOrderID,
		"description": description,
	}}
	var resp workOrderSetDescriptionResponse
	if err := c.doMutationIdempotent(ctx, workOrderSetDescriptionMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderSetDescription, nil
}

// WorkOrderSetPriority sets the work order priority. Idempotent.
func (c *Client) WorkOrderSetPriority(ctx context.Context, workOrderID, priority string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId": workOrderID,
		"priority":    priority,
	}}
	var resp workOrderSetPriorityResponse
	if err := c.doMutationIdempotent(ctx, workOrderSetPriorityMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderSetPriority, nil
}

// WorkOrderSetScheduledFor sets when the work order is scheduled. Idempotent.
func (c *Client) WorkOrderSetScheduledFor(ctx context.Context, workOrderID, scheduledFor string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId":  workOrderID,
		"scheduledFor": scheduledFor,
	}}
	var resp workOrderSetScheduledForResponse
	if err := c.doMutationIdempotent(ctx, workOrderSetScheduledForMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderSetScheduledFor, nil
}

// WorkOrderSetLocation sets the work order location. Idempotent.
func (c *Client) WorkOrderSetLocation(ctx context.Context, workOrderID, locationID string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId": workOrderID,
		"locationId":  locationID,
	}}
	var resp workOrderSetLocationResponse
	if err := c.doMutationIdempotent(ctx, workOrderSetLocationMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderSetLocation, nil
}

// WorkOrderSetType sets the work order type. Idempotent.
func (c *Client) WorkOrderSetType(ctx context.Context, workOrderID, woType string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId":   workOrderID,
		"workOrderType": woType,
	}}
	var resp workOrderSetTypeResponse
	if err := c.doMutationIdempotent(ctx, workOrderSetTypeMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderSetType, nil
}

// WorkOrderSetEntryNotes sets the work order entry notes. Idempotent.
func (c *Client) WorkOrderSetEntryNotes(ctx context.Context, workOrderID, entryNotes string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId": workOrderID,
		"entryNotes":  entryNotes,
	}}
	var resp workOrderSetEntryNotesResponse
	if err := c.doMutationIdempotent(ctx, workOrderSetEntryNotesMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderSetEntryNotes, nil
}

// WorkOrderSetPermissionToEnter sets the permission to enter flag. Idempotent.
func (c *Client) WorkOrderSetPermissionToEnter(ctx context.Context, workOrderID string, permission bool) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId":       workOrderID,
		"permissionToEnter": permission,
	}}
	var resp workOrderSetPermissionToEnterResponse
	if err := c.doMutationIdempotent(ctx, workOrderSetPermissionToEnterMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderSetPermissionToEnter, nil
}

// WorkOrderSetResidentApprovedEntry sets the resident approved entry flag. Idempotent.
func (c *Client) WorkOrderSetResidentApprovedEntry(ctx context.Context, workOrderID string, approved bool) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId":           workOrderID,
		"residentApprovedEntry": approved,
	}}
	var resp workOrderSetResidentApprovedEntryResponse
	if err := c.doMutationIdempotent(ctx, workOrderSetResidentApprovedEntryMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderSetResidentApprovedEntry, nil
}

// WorkOrderSetUnitEntered sets the unit entered flag. Idempotent.
func (c *Client) WorkOrderSetUnitEntered(ctx context.Context, workOrderID string, unitEntered bool) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId": workOrderID,
		"unitEntered": unitEntered,
	}}
	var resp workOrderSetUnitEnteredResponse
	if err := c.doMutationIdempotent(ctx, workOrderSetUnitEnteredMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderSetUnitEntered, nil
}

// WorkOrderArchive archives a work order. Idempotent.
func (c *Client) WorkOrderArchive(ctx context.Context, workOrderID string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId": workOrderID,
	}}
	var resp workOrderArchiveResponse
	if err := c.doMutationIdempotent(ctx, workOrderArchiveMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderArchive, nil
}

// WorkOrderAddComment adds a comment to a work order. Non-idempotent: auth-retry only.
func (c *Client) WorkOrderAddComment(ctx context.Context, workOrderID, comment string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId": workOrderID,
		"comment":     comment,
	}}
	var resp workOrderAddCommentResponse
	if err := c.doMutation(ctx, workOrderAddCommentMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderAddComment, nil
}

// WorkOrderAddTime adds time spent on a work order. Non-idempotent: auth-retry only.
func (c *Client) WorkOrderAddTime(ctx context.Context, workOrderID, duration string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId": workOrderID,
		"duration":    duration,
	}}
	var resp workOrderAddTimeResponse
	if err := c.doMutation(ctx, workOrderAddTimeMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderAddTime, nil
}

// WorkOrderAddAttachment adds an attachment to a work order. Non-idempotent: auth-retry only.
// Returns the updated work order, attachment metadata, and a signed upload URL.
func (c *Client) WorkOrderAddAttachment(ctx context.Context, input models.WorkOrderAddAttachmentInput) (*models.WorkOrderAddAttachmentResult, error) {
	vars := map[string]interface{}{"input": input}
	var resp workOrderAddAttachmentResponse
	if err := c.doMutation(ctx, workOrderAddAttachmentMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderAddAttachment, nil
}

// WorkOrderRemoveAttachment removes an attachment. Idempotent.
func (c *Client) WorkOrderRemoveAttachment(ctx context.Context, workOrderID, attachmentID string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId":  workOrderID,
		"attachmentId": attachmentID,
	}}
	var resp workOrderRemoveAttachmentResponse
	if err := c.doMutationIdempotent(ctx, workOrderRemoveAttachmentMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderRemoveAttachment, nil
}

// WorkOrderStartTimer starts the timer. Idempotent.
func (c *Client) WorkOrderStartTimer(ctx context.Context, workOrderID, startedAt string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId": workOrderID,
		"startedAt":   startedAt,
	}}
	var resp workOrderStartTimerResponse
	if err := c.doMutationIdempotent(ctx, workOrderStartTimerMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderStartTimer, nil
}

// WorkOrderStopTimer stops the timer. Idempotent.
func (c *Client) WorkOrderStopTimer(ctx context.Context, workOrderID, stoppedAt string) (*models.WorkOrder, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"workOrderId": workOrderID,
		"stoppedAt":   stoppedAt,
	}}
	var resp workOrderStopTimerResponse
	if err := c.doMutationIdempotent(ctx, workOrderStopTimerMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WorkOrderStopTimer, nil
}

// --- Inspection Mutations (24) ---

// InspectionCreate creates a new inspection. Non-idempotent: auth-retry only.
func (c *Client) InspectionCreate(ctx context.Context, input models.InspectionCreateInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionCreateResponse
	if err := c.doMutation(ctx, inspectionCreateMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionCreate, nil
}

// InspectionStart starts an inspection. Idempotent.
func (c *Client) InspectionStart(ctx context.Context, inspectionID string) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"inspectionId": inspectionID,
	}}
	var resp inspectionStartResponse
	if err := c.doMutationIdempotent(ctx, inspectionStartMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionStart, nil
}

// InspectionComplete marks an inspection as complete. Idempotent.
func (c *Client) InspectionComplete(ctx context.Context, inspectionID string) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"inspectionId": inspectionID,
	}}
	var resp inspectionCompleteResponse
	if err := c.doMutationIdempotent(ctx, inspectionCompleteMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionComplete, nil
}

// InspectionReopen reopens a completed inspection. Idempotent.
func (c *Client) InspectionReopen(ctx context.Context, inspectionID string) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"inspectionId": inspectionID,
	}}
	var resp inspectionReopenResponse
	if err := c.doMutationIdempotent(ctx, inspectionReopenMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionReopen, nil
}

// InspectionArchive archives an inspection. Idempotent.
func (c *Client) InspectionArchive(ctx context.Context, inspectionID string) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"inspectionId": inspectionID,
	}}
	var resp inspectionArchiveResponse
	if err := c.doMutationIdempotent(ctx, inspectionArchiveMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionArchive, nil
}

// InspectionExpire expires an inspection. Idempotent.
func (c *Client) InspectionExpire(ctx context.Context, inspectionID string) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"inspectionId": inspectionID,
	}}
	var resp inspectionExpireResponse
	if err := c.doMutationIdempotent(ctx, inspectionExpireMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionExpire, nil
}

// InspectionUnexpire unexpires an inspection. Idempotent.
func (c *Client) InspectionUnexpire(ctx context.Context, inspectionID string) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"inspectionId": inspectionID,
	}}
	var resp inspectionUnexpireResponse
	if err := c.doMutationIdempotent(ctx, inspectionUnexpireMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionUnexpire, nil
}

// InspectionSetAssignee assigns a user to an inspection. Idempotent.
func (c *Client) InspectionSetAssignee(ctx context.Context, input models.InspectionSetAssigneeInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionSetAssigneeResponse
	if err := c.doMutationIdempotent(ctx, inspectionSetAssigneeMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionSetAssignee, nil
}

// InspectionSetDueBy sets the due date for an inspection. Idempotent.
func (c *Client) InspectionSetDueBy(ctx context.Context, input models.InspectionSetDueByInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionSetDueByResponse
	if err := c.doMutationIdempotent(ctx, inspectionSetDueByMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionSetDueBy, nil
}

// InspectionSetScheduledFor sets the scheduled date for an inspection. Idempotent.
func (c *Client) InspectionSetScheduledFor(ctx context.Context, inspectionID, scheduledFor string) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"inspectionId": inspectionID,
		"scheduledFor": scheduledFor,
	}}
	var resp inspectionSetScheduledForResponse
	if err := c.doMutationIdempotent(ctx, inspectionSetScheduledForMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionSetScheduledFor, nil
}

// InspectionSetHeaderField updates a header field. Idempotent.
func (c *Client) InspectionSetHeaderField(ctx context.Context, input models.InspectionSetHeaderFieldInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionSetHeaderFieldResponse
	if err := c.doMutationIdempotent(ctx, inspectionSetHeaderFieldMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionSetHeaderField, nil
}

// InspectionSetFooterField updates a footer field. Idempotent.
func (c *Client) InspectionSetFooterField(ctx context.Context, input models.InspectionSetFooterFieldInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionSetFooterFieldResponse
	if err := c.doMutationIdempotent(ctx, inspectionSetFooterFieldMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionSetFooterField, nil
}

// InspectionSetItemNotes sets notes on an inspection item. Idempotent.
func (c *Client) InspectionSetItemNotes(ctx context.Context, input models.InspectionSetItemNotesInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionSetItemNotesResponse
	if err := c.doMutationIdempotent(ctx, inspectionSetItemNotesMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionSetItemNotes, nil
}

// InspectionRateItem rates an item in an inspection. Idempotent.
func (c *Client) InspectionRateItem(ctx context.Context, input models.InspectionRateItemInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionRateItemResponse
	if err := c.doMutationIdempotent(ctx, inspectionRateItemMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionRateItem, nil
}

// InspectionAddSection adds a section to an inspection. Non-idempotent: auth-retry only.
func (c *Client) InspectionAddSection(ctx context.Context, input models.InspectionAddSectionInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionAddSectionResponse
	if err := c.doMutation(ctx, inspectionAddSectionMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionAddSection, nil
}

// InspectionDeleteSection deletes a section from an inspection. Idempotent.
func (c *Client) InspectionDeleteSection(ctx context.Context, input models.InspectionDeleteSectionInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionDeleteSectionResponse
	if err := c.doMutationIdempotent(ctx, inspectionDeleteSectionMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionDeleteSection, nil
}

// InspectionDuplicateSection duplicates a section. Non-idempotent: auth-retry only.
func (c *Client) InspectionDuplicateSection(ctx context.Context, input models.InspectionDuplicateSectionInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionDuplicateSectionResponse
	if err := c.doMutation(ctx, inspectionDuplicateSectionMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionDuplicateSection, nil
}

// InspectionRenameSection renames a section. Idempotent.
func (c *Client) InspectionRenameSection(ctx context.Context, input models.InspectionRenameSectionInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionRenameSectionResponse
	if err := c.doMutationIdempotent(ctx, inspectionRenameSectionMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionRenameSection, nil
}

// InspectionAddItem adds an item to a section. Non-idempotent: auth-retry only.
func (c *Client) InspectionAddItem(ctx context.Context, input models.InspectionAddItemInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionAddItemResponse
	if err := c.doMutation(ctx, inspectionAddItemMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionAddItem, nil
}

// InspectionDeleteItem deletes an item from a section. Idempotent.
func (c *Client) InspectionDeleteItem(ctx context.Context, input models.InspectionDeleteItemInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionDeleteItemResponse
	if err := c.doMutationIdempotent(ctx, inspectionDeleteItemMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionDeleteItem, nil
}

// InspectionAddItemPhoto adds a photo to an inspection item. Non-idempotent: auth-retry only.
// Returns the updated inspection, photo metadata, and a signed upload URL.
func (c *Client) InspectionAddItemPhoto(ctx context.Context, input models.InspectionAddItemPhotoInput) (*models.InspectionAddItemPhotoResult, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionAddItemPhotoResponse
	if err := c.doMutation(ctx, inspectionAddItemPhotoMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionAddItemPhoto, nil
}

// InspectionRemoveItemPhoto removes a photo from an item. Idempotent.
func (c *Client) InspectionRemoveItemPhoto(ctx context.Context, input models.InspectionRemoveItemPhotoInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionRemoveItemPhotoResponse
	if err := c.doMutationIdempotent(ctx, inspectionRemoveItemPhotoMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionRemoveItemPhoto, nil
}

// InspectionMoveItemPhoto moves a photo between items. Idempotent.
func (c *Client) InspectionMoveItemPhoto(ctx context.Context, input models.InspectionMoveItemPhotoInput) (*models.Inspection, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionMoveItemPhotoResponse
	if err := c.doMutationIdempotent(ctx, inspectionMoveItemPhotoMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionMoveItemPhoto, nil
}

// InspectionSendToGuest sends an inspection to a guest. Non-idempotent: auth-retry only.
func (c *Client) InspectionSendToGuest(ctx context.Context, input models.InspectionSendToGuestInput) (*models.InspectionGuestLink, error) {
	vars := map[string]interface{}{"input": input}
	var resp inspectionSendToGuestResponse
	if err := c.doMutation(ctx, inspectionSendToGuestMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.InspectionSendToGuest, nil
}

// --- Project Mutations (8) ---

// ProjectCreate creates a new project from a template. Non-idempotent: auth-retry only.
func (c *Client) ProjectCreate(ctx context.Context, input models.ProjectCreateInput) (*models.Project, error) {
	vars := map[string]interface{}{"input": input}
	var resp projectCreateResponse
	if err := c.doMutation(ctx, projectCreateMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.ProjectCreate, nil
}

// ProjectSetAssignee sets or clears the project assignee. Idempotent.
func (c *Client) ProjectSetAssignee(ctx context.Context, input models.ProjectSetAssigneeInput) (*models.Project, error) {
	vars := map[string]interface{}{"input": input}
	var resp projectSetAssigneeResponse
	if err := c.doMutationIdempotent(ctx, projectSetAssigneeMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.ProjectSetAssignee, nil
}

// ProjectSetNotes sets the project notes. Idempotent.
func (c *Client) ProjectSetNotes(ctx context.Context, projectID, notes string) (*models.Project, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"projectId": projectID,
		"notes":     notes,
	}}
	var resp projectSetNotesResponse
	if err := c.doMutationIdempotent(ctx, projectSetNotesMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.ProjectSetNotes, nil
}

// ProjectSetDueAt sets the project due date. Idempotent.
func (c *Client) ProjectSetDueAt(ctx context.Context, projectID, dueAt string) (*models.Project, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"projectId": projectID,
		"dueAt":     dueAt,
	}}
	var resp projectSetDueAtResponse
	if err := c.doMutationIdempotent(ctx, projectSetDueAtMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.ProjectSetDueAt, nil
}

// ProjectSetStartAt sets the project start date. Idempotent.
func (c *Client) ProjectSetStartAt(ctx context.Context, projectID, startAt string) (*models.Project, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"projectId": projectID,
		"startAt":   startAt,
	}}
	var resp projectSetStartAtResponse
	if err := c.doMutationIdempotent(ctx, projectSetStartAtMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.ProjectSetStartAt, nil
}

// ProjectSetPriority sets the project priority. Idempotent.
func (c *Client) ProjectSetPriority(ctx context.Context, projectID, priority string) (*models.Project, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"projectId": projectID,
		"priority":  priority,
	}}
	var resp projectSetPriorityResponse
	if err := c.doMutationIdempotent(ctx, projectSetPriorityMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.ProjectSetPriority, nil
}

// ProjectSetOnHold sets the project on-hold status. Idempotent.
func (c *Client) ProjectSetOnHold(ctx context.Context, projectID string, onHold bool) (*models.Project, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"projectId": projectID,
		"onHold":    onHold,
	}}
	var resp projectSetOnHoldResponse
	if err := c.doMutationIdempotent(ctx, projectSetOnHoldMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.ProjectSetOnHold, nil
}

// ProjectSetAvailabilityTargetAt sets the project availability target date. Idempotent.
// Pass empty string to clear the date.
func (c *Client) ProjectSetAvailabilityTargetAt(ctx context.Context, projectID string, availabilityTargetAt *string) (*models.Project, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"projectId":            projectID,
		"availabilityTargetAt": availabilityTargetAt,
	}}
	var resp projectSetAvailabilityTargetAtResponse
	if err := c.doMutationIdempotent(ctx, projectSetAvailabilityTargetAtMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.ProjectSetAvailabilityTargetAt, nil
}

// --- User Mutations (5) ---

// UserCreate creates a new user in an account. Non-idempotent: auth-retry only.
func (c *Client) UserCreate(ctx context.Context, input models.UserCreateInput) (*models.User, error) {
	vars := map[string]interface{}{"input": input}
	var resp userCreateResponse
	if err := c.doMutation(ctx, userCreateMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.UserCreate, nil
}

// UserSetEmail sets a user's email. Idempotent.
func (c *Client) UserSetEmail(ctx context.Context, userID, email string) (*models.User, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"userId": userID,
		"email":  email,
	}}
	var resp userSetEmailResponse
	if err := c.doMutationIdempotent(ctx, userSetEmailMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.UserSetEmail, nil
}

// UserSetName sets a user's full name. Idempotent.
func (c *Client) UserSetName(ctx context.Context, userID, name string) (*models.User, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"userId": userID,
		"name":   name,
	}}
	var resp userSetNameResponse
	if err := c.doMutationIdempotent(ctx, userSetNameMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.UserSetName, nil
}

// UserSetShortName sets a user's short name. Pass empty string to clear (derives from name). Idempotent.
func (c *Client) UserSetShortName(ctx context.Context, userID string, shortName *string) (*models.User, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"userId":    userID,
		"shortName": shortName,
	}}
	var resp userSetShortNameResponse
	if err := c.doMutationIdempotent(ctx, userSetShortNameMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.UserSetShortName, nil
}

// UserSetPhone sets a user's phone number. Pass nil to remove. Idempotent.
func (c *Client) UserSetPhone(ctx context.Context, userID string, phone *string) (*models.User, error) {
	vars := map[string]interface{}{"input": map[string]interface{}{
		"userId": userID,
		"phone":  phone,
	}}
	var resp userSetPhoneResponse
	if err := c.doMutationIdempotent(ctx, userSetPhoneMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.UserSetPhone, nil
}

// --- Membership Mutations (4) ---

// AccountMembershipCreate creates a new membership. Non-idempotent: auth-retry only.
func (c *Client) AccountMembershipCreate(ctx context.Context, input models.AccountMembershipCreateInput) (*models.AccountMembership, error) {
	vars := map[string]interface{}{"input": input}
	var resp accountMembershipCreateResponse
	if err := c.doMutation(ctx, accountMembershipCreateMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.AccountMembershipCreate, nil
}

// AccountMembershipActivate activates a membership. Idempotent.
func (c *Client) AccountMembershipActivate(ctx context.Context, input models.AccountMembershipActivateInput) (*models.AccountMembership, error) {
	vars := map[string]interface{}{"input": input}
	var resp accountMembershipActivateResponse
	if err := c.doMutationIdempotent(ctx, accountMembershipActivateMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.AccountMembershipActivate, nil
}

// AccountMembershipDeactivate deactivates a membership. Idempotent.
func (c *Client) AccountMembershipDeactivate(ctx context.Context, input models.AccountMembershipDeactivateInput) (*models.AccountMembership, error) {
	vars := map[string]interface{}{"input": input}
	var resp accountMembershipDeactivateResponse
	if err := c.doMutationIdempotent(ctx, accountMembershipDeactivateMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.AccountMembershipDeactivate, nil
}

// AccountMembershipSetRoles sets the roles on a membership. Idempotent.
func (c *Client) AccountMembershipSetRoles(ctx context.Context, input models.AccountMembershipSetRolesInput) (*models.AccountMembership, error) {
	vars := map[string]interface{}{"input": input}
	var resp accountMembershipSetRolesResponse
	if err := c.doMutationIdempotent(ctx, accountMembershipSetRolesMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.AccountMembershipSetRoles, nil
}

// --- Property Access Mutations (3) ---

// PropertyGrantUserAccess grants users access to a property. Idempotent.
func (c *Client) PropertyGrantUserAccess(ctx context.Context, input models.PropertyGrantUserAccessInput) (*models.PropertyAccess, error) {
	vars := map[string]interface{}{"input": input}
	var resp propertyGrantUserAccessResponse
	if err := c.doMutationIdempotent(ctx, propertyGrantUserAccessMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.PropertyGrantUserAccess, nil
}

// PropertyRevokeUserAccess revokes user access from a property. Idempotent.
func (c *Client) PropertyRevokeUserAccess(ctx context.Context, input models.PropertyRevokeUserAccessInput) (*models.PropertyAccess, error) {
	vars := map[string]interface{}{"input": input}
	var resp propertyRevokeUserAccessResponse
	if err := c.doMutationIdempotent(ctx, propertyRevokeUserAccessMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.PropertyRevokeUserAccess, nil
}

// PropertySetAccountWideAccess sets account-wide access on a property. Idempotent.
func (c *Client) PropertySetAccountWideAccess(ctx context.Context, input models.PropertySetAccountWideAccessInput) (*models.PropertyAccess, error) {
	vars := map[string]interface{}{"input": input}
	var resp propertySetAccountWideAccessResponse
	if err := c.doMutationIdempotent(ctx, propertySetAccountWideAccessMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.PropertySetAccountWideAccess, nil
}

// --- User Property Access Mutations (2) ---

// UserGrantPropertyAccess grants a user access to properties. Idempotent.
func (c *Client) UserGrantPropertyAccess(ctx context.Context, input models.UserGrantPropertyAccessInput) (*models.User, error) {
	vars := map[string]interface{}{"input": input}
	var resp userGrantPropertyAccessResponse
	if err := c.doMutationIdempotent(ctx, userGrantPropertyAccessMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.UserGrantPropertyAccess, nil
}

// UserRevokePropertyAccess revokes property access from a user. Idempotent.
func (c *Client) UserRevokePropertyAccess(ctx context.Context, input models.UserRevokePropertyAccessInput) (*models.User, error) {
	vars := map[string]interface{}{"input": input}
	var resp userRevokePropertyAccessResponse
	if err := c.doMutationIdempotent(ctx, userRevokePropertyAccessMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.UserRevokePropertyAccess, nil
}

// --- Role Mutations (4) ---

// RoleCreate creates a new role in an account. Non-idempotent.
func (c *Client) RoleCreate(ctx context.Context, input models.RoleCreateInput) (*models.Role, error) {
	vars := map[string]interface{}{"input": input}
	var resp roleCreateResponse
	if err := c.doMutation(ctx, roleCreateMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.RoleCreate, nil
}

// RoleSetName updates a role's name. Idempotent.
func (c *Client) RoleSetName(ctx context.Context, input models.RoleSetNameInput) (*models.Role, error) {
	vars := map[string]interface{}{"input": input}
	var resp roleSetNameResponse
	if err := c.doMutationIdempotent(ctx, roleSetNameMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.RoleSetName, nil
}

// RoleSetDescription updates a role's description. Idempotent.
func (c *Client) RoleSetDescription(ctx context.Context, input models.RoleSetDescriptionInput) (*models.Role, error) {
	vars := map[string]interface{}{"input": input}
	var resp roleSetDescriptionResponse
	if err := c.doMutationIdempotent(ctx, roleSetDescriptionMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.RoleSetDescription, nil
}

// RoleSetPermissions updates a role's permissions. Idempotent.
func (c *Client) RoleSetPermissions(ctx context.Context, input models.RoleSetPermissionsInput) (*models.Role, error) {
	vars := map[string]interface{}{"input": input}
	var resp roleSetPermissionsResponse
	if err := c.doMutationIdempotent(ctx, roleSetPermissionsMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.RoleSetPermissions, nil
}

// --- Webhook Mutations (2) ---

// WebhookCreate creates a new webhook. Non-idempotent.
func (c *Client) WebhookCreate(ctx context.Context, input models.WebhookCreateInput) (*models.Webhook, error) {
	vars := map[string]interface{}{"input": input}
	var resp webhookCreateResponse
	if err := c.doMutation(ctx, webhookCreateMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WebhookCreate, nil
}

// WebhookUpdate updates an existing webhook. Idempotent.
func (c *Client) WebhookUpdate(ctx context.Context, input models.WebhookUpdateInput) (*models.Webhook, error) {
	vars := map[string]interface{}{"input": input}
	var resp webhookUpdateResponse
	if err := c.doMutationIdempotent(ctx, webhookUpdateMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.WebhookUpdate, nil
}

// paginate is a generic pagination loop shared by all List* methods.
func paginate[T any](
	ctx context.Context,
	c *Client,
	resource string,
	opts models.ListOptions,
	prepareVars func(map[string]interface{}),
	fetchPage func(map[string]interface{}) (*connection[T], error),
) ([]T, int, error) {
	cap := effectiveCap(opts.Limit)
	initialCap := pageSize
	if cap > 0 && cap < initialCap {
		initialCap = cap
	}
	allItems := make([]T, 0, initialCap)
	var totalCount int
	cursor := ""

	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, 0, errAPIFatal(fmt.Sprintf("context cancelled: %v", err))
		}

		vars := map[string]interface{}{
			"accountId": c.accountID,
			"first":     pageSize,
		}
		if cursor != "" {
			vars["after"] = cursor
		}
		prepareVars(vars)

		conn, err := fetchPageWithRetry(ctx, c, page, vars, fetchPage)
		if err != nil {
			return nil, 0, err
		}

		totalCount = conn.Count
		for _, e := range conn.Edges {
			allItems = append(allItems, e.Node)
		}

		if c.debug {
			log.Printf("[debug] %s page %d: fetched %d, total so far %d/%d", resource, page, len(conn.Edges), len(allItems), totalCount)
		}

		if cap > 0 && len(allItems) >= cap {
			allItems = allItems[:cap]
			if c.debug && totalCount > cap {
				log.Printf("[debug] %s: returning %d of %d total items (cap applied)", resource, cap, totalCount)
			}
			return allItems, totalCount, nil
		}

		// Defence-in-depth: hard ceiling prevents unbounded memory growth
		// for direct API callers that pass no cap (limit < 0).
		if len(allItems) >= hardMaxItems {
			if c.debug {
				log.Printf("[debug] %s: hard ceiling reached (%d items), stopping pagination", resource, hardMaxItems)
			}
			return allItems[:hardMaxItems], totalCount, nil
		}

		if !conn.PageInfo.HasNextPage {
			return allItems, totalCount, nil
		}

		// Detect stuck cursor: if the server returns the same endCursor twice,
		// pagination is not advancing and would loop until hardMaxItems.
		if conn.PageInfo.EndCursor == cursor {
			return allItems, totalCount, errAPIFatal("pagination stuck: server returned same cursor")
		}
		cursor = conn.PageInfo.EndCursor
	}
}

// fetchPageWithRetry wraps a page fetch with retries for transient errors.
// On auth failures (401/403), it attempts a single re-authentication before giving up,
// preventing mid-pagination token expiry from discarding all fetched pages.
func fetchPageWithRetry[T any](
	ctx context.Context,
	c *Client,
	page int,
	vars map[string]interface{},
	fetchPage func(map[string]interface{}) (*connection[T], error),
) (*connection[T], error) {
	var lastErr error
	authRetried := false
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, errAPIFatal(fmt.Sprintf("context cancelled: %v", err))
		}

		conn, err := fetchPage(vars)
		if err == nil {
			return conn, nil
		}
		lastErr = err

		var ae *apiError
		if errors.As(err, &ae) {
			// On auth failure, attempt a single re-auth before giving up.
			// This handles mid-pagination token expiry gracefully.
			if ae.Category == "auth_failed" && !authRetried {
				authRetried = true
				if c.debug {
					log.Printf("[debug] page %d: auth failed, attempting re-authentication", page)
				}
				if _, authErr := c.ensureAuth(ctx); authErr != nil {
					return nil, err // return the original auth error
				}
				// Retry the same page with the fresh token (don't count as a retry attempt)
				attempt--
				continue
			}

			// Only retry transient errors
			if !ae.Retryable {
				return nil, err
			}
		}

		if c.debug {
			log.Printf("[debug] page %d attempt %d failed: %v", page, attempt+1, err)
		}

		if attempt < maxRetries-1 {
			idx := min(attempt, len(c.retryBackoff)-1)
			backoff := c.retryBackoff[idx]
			// Honour Retry-After header from 429 responses
			if ae != nil && ae.RetryAfter > backoff {
				backoff = ae.RetryAfter
			}
			// Add jitter: ±25% of backoff
			jitter := time.Duration(rand.Int64N(int64(backoff/2))) - backoff/4
			select {
			case <-ctx.Done():
				return nil, errAPIFatal(fmt.Sprintf("context cancelled: %v", ctx.Err()))
			case <-time.After(backoff + jitter):
			}
		}
	}
	return nil, &apiError{
		Category:  "api_error",
		Message:   fmt.Sprintf("page %d failed after %d retries: %v", page, maxRetries, lastErr),
		Retryable: false,
	}
}

// effectiveCap translates a user-supplied limit into a pagination cap.
// 0 = default cap (1,000); negative = no cap (fetch all); positive = use as cap.
// Note: the CLI layer rejects negative limits. Negative values are intentionally
// supported here for programmatic callers (e.g., MCP server) that want all results.
func effectiveCap(limit int) int {
	if limit == 0 {
		return defaultCap
	}
	if limit < 0 {
		return 0 // no cap
	}
	return limit
}

// GetAccountRaw returns the raw GraphQL response for the account query with retry.
func (c *Client) GetAccountRaw(ctx context.Context) (json.RawMessage, error) {
	vars := map[string]interface{}{"accountId": c.accountID}
	var raw json.RawMessage
	return raw, c.doQueryWithRetry(ctx, getAccountQuery, vars, &raw)
}

// ListPropertiesRaw returns the raw GraphQL response for the first page of properties.
func (c *Client) ListPropertiesRaw(ctx context.Context, opts models.ListOptions) (json.RawMessage, error) {
	vars := map[string]interface{}{
		"accountId": c.accountID,
		"first":     rawPageFirst(opts.Limit),
		"orderBy":   []string{"NAME_ASC"},
	}
	if filter := buildPropertiesFilter(opts); len(filter) > 0 {
		vars["filter"] = filter
	}
	var raw json.RawMessage
	return raw, c.doQueryWithRetry(ctx, listPropertiesQuery, vars, &raw)
}

// ListUnitsRaw returns the raw GraphQL response for the first page of units.
func (c *Client) ListUnitsRaw(ctx context.Context, propertyID string, limit int) (json.RawMessage, error) {
	vars := map[string]interface{}{
		"accountId":        c.accountID,
		"first":            rawPageFirst(limit),
		"propertiesFilter": map[string]interface{}{"propertyId": []string{propertyID}},
	}
	var raw json.RawMessage
	return raw, c.doQueryWithRetry(ctx, listUnitsQuery, vars, &raw)
}

// ListWorkOrdersRaw returns the raw GraphQL response for the first page of work orders.
func (c *Client) ListWorkOrdersRaw(ctx context.Context, opts models.ListOptions) (json.RawMessage, error) {
	return c.listRawWithDateFilter(ctx, listWorkOrdersQuery, opts)
}

// ListInspectionsRaw returns the raw GraphQL response for the first page of inspections.
func (c *Client) ListInspectionsRaw(ctx context.Context, opts models.ListOptions) (json.RawMessage, error) {
	return c.listRawWithDateFilter(ctx, listInspectionsQuery, opts)
}

// listRawWithDateFilter is the shared implementation for raw queries that use
// CREATED_AT_DESC ordering and location/date/status filters.
func (c *Client) listRawWithDateFilter(ctx context.Context, query string, opts models.ListOptions) (json.RawMessage, error) {
	vars := map[string]interface{}{
		"accountId": c.accountID,
		"first":     rawPageFirst(opts.Limit),
		"orderBy":   []string{"CREATED_AT_DESC"},
	}
	if filter := buildLocationDateFilter(opts); len(filter) > 0 {
		vars["filter"] = filter
	}
	var raw json.RawMessage
	return raw, c.doQueryWithRetry(ctx, query, vars, &raw)
}

// rawPageFirst returns the page size for raw queries, honouring the limit if set.
func rawPageFirst(limit int) int {
	if limit > 0 && limit < pageSize {
		return limit
	}
	return pageSize
}

func buildPropertiesFilter(opts models.ListOptions) map[string]interface{} {
	f := map[string]interface{}{}
	if opts.Search != "" {
		f["search"] = opts.Search
	}
	if opts.LocationID != "" {
		f["propertyId"] = []string{opts.LocationID}
	}
	return f
}

func buildLocationDateFilter(opts models.ListOptions) map[string]interface{} {
	f := map[string]interface{}{}
	if opts.LocationID != "" {
		f["locationId"] = []string{opts.LocationID}
	}
	if len(opts.Status) > 0 {
		f["status"] = opts.Status
	}
	if opts.CreatedAfter != nil {
		f["createdAfter"] = opts.CreatedAfter.Format(time.RFC3339)
	}
	if opts.CreatedBefore != nil {
		f["createdBefore"] = opts.CreatedBefore.Format(time.RFC3339)
	}
	return f
}
