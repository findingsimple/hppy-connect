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
	"sync"
	"sync/atomic"
	"time"

	"github.com/findingsimple/hppy-connect/internal/models"
)

// DefaultEndpoint is the HappyCo external GraphQL API endpoint.
const DefaultEndpoint = "https://externalgraph.happyco.com"

const (
	pageSize = 100
	defaultCap      = 1000
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
	Category  string // auth_failed, not_found, invalid_input, rate_limited, api_error
	Message   string
	Retryable bool
}

func (e *apiError) Error() string {
	return fmt.Sprintf("%s: %s", e.Category, e.Message)
}

func errAuth(msg string) error       { return &apiError{Category: "auth_failed", Message: msg} }
func errAPI(msg string) error         { return &apiError{Category: "api_error", Message: msg, Retryable: true} }
func errAPIFatal(msg string) error    { return &apiError{Category: "api_error", Message: msg} }
func errRateLimited(msg string) error { return &apiError{Category: "rate_limited", Message: msg, Retryable: true} }

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
	allowInsecure  bool          // testing only — bypasses HTTPS requirement
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
		c.lastLoginFail = time.Now()
		c.lastLoginError = err
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
		return nil, errAuth(fmt.Sprintf("login request: %v", err))
	}
	defer resp.Body.Close()

	if debug {
		log.Printf("[debug] login POST %s status=%d duration=%s", endpoint, resp.StatusCode, time.Since(start))
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, errAuth(fmt.Sprintf("reading login response: %v", err))
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, errRateLimited("login rate limited")
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
		return errRateLimited("API rate limited")
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		c.invalidateAuth()
		// TODO: re-authenticate and retry once instead of aborting, so long-running
		// paginations survive token expiry mid-loop.
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

	if err := json.Unmarshal(gqlResp.Data, result); err != nil {
		return errAPIFatal(fmt.Sprintf("parsing data: %v", err))
	}

	return nil
}

// GetAccount returns the account details.
func (c *Client) GetAccount(ctx context.Context) (*models.Account, error) {
	vars := map[string]interface{}{
		"accountId": c.accountID,
	}
	var resp accountResponse
	if err := c.doQuery(ctx, getAccountQuery, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.Account, nil
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
	allItems := make([]T, 0)
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

		if !conn.PageInfo.HasNextPage {
			return allItems, totalCount, nil
		}
		cursor = conn.PageInfo.EndCursor
	}
}

// fetchPageWithRetry wraps a page fetch with retries for transient errors.
func fetchPageWithRetry[T any](
	ctx context.Context,
	c *Client,
	page int,
	vars map[string]interface{},
	fetchPage func(map[string]interface{}) (*connection[T], error),
) (*connection[T], error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, errAPIFatal(fmt.Sprintf("context cancelled: %v", err))
		}

		conn, err := fetchPage(vars)
		if err == nil {
			return conn, nil
		}
		lastErr = err

		// Only retry transient errors
		var ae *apiError
		if errors.As(err, &ae) && !ae.Retryable {
			return nil, err
		}

		if c.debug {
			log.Printf("[debug] page %d attempt %d failed: %v", page, attempt+1, err)
		}

		if attempt < maxRetries-1 {
			idx := min(attempt, len(c.retryBackoff)-1)
			backoff := c.retryBackoff[idx]
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

// GetAccountRaw returns the raw GraphQL response for the account query.
func (c *Client) GetAccountRaw(ctx context.Context) (json.RawMessage, error) {
	vars := map[string]interface{}{"accountId": c.accountID}
	var raw json.RawMessage
	return raw, c.doQuery(ctx, getAccountQuery, vars, &raw)
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
	return raw, c.doQuery(ctx, listPropertiesQuery, vars, &raw)
}

// ListUnitsRaw returns the raw GraphQL response for the first page of units.
func (c *Client) ListUnitsRaw(ctx context.Context, propertyID string, limit int) (json.RawMessage, error) {
	vars := map[string]interface{}{
		"accountId":        c.accountID,
		"first":            rawPageFirst(limit),
		"propertiesFilter": map[string]interface{}{"propertyId": []string{propertyID}},
	}
	var raw json.RawMessage
	return raw, c.doQuery(ctx, listUnitsQuery, vars, &raw)
}

// ListWorkOrdersRaw returns the raw GraphQL response for the first page of work orders.
func (c *Client) ListWorkOrdersRaw(ctx context.Context, opts models.ListOptions) (json.RawMessage, error) {
	vars := map[string]interface{}{
		"accountId": c.accountID,
		"first":     rawPageFirst(opts.Limit),
		"orderBy":   []string{"CREATED_AT_DESC"},
	}
	if filter := buildLocationDateFilter(opts); len(filter) > 0 {
		vars["filter"] = filter
	}
	var raw json.RawMessage
	return raw, c.doQuery(ctx, listWorkOrdersQuery, vars, &raw)
}

// ListInspectionsRaw returns the raw GraphQL response for the first page of inspections.
func (c *Client) ListInspectionsRaw(ctx context.Context, opts models.ListOptions) (json.RawMessage, error) {
	vars := map[string]interface{}{
		"accountId": c.accountID,
		"first":     rawPageFirst(opts.Limit),
		"orderBy":   []string{"CREATED_AT_DESC"},
	}
	if filter := buildLocationDateFilter(opts); len(filter) > 0 {
		vars["filter"] = filter
	}
	var raw json.RawMessage
	return raw, c.doQuery(ctx, listInspectionsQuery, vars, &raw)
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
