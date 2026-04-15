package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/findingsimple/hppy-connect/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fastBackoff = []time.Duration{1 * time.Millisecond, 1 * time.Millisecond, 1 * time.Millisecond}

// newTestServer creates a test server and a client pointing at it with a pre-set valid token.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient("test@example.com", "password", "12345",
		withInsecureHTTP(),
		WithEndpoint(srv.URL),
		withRetryBackoff(fastBackoff),
	)
	require.NoError(t, err)
	// Pre-set a valid token so doQuery doesn't trigger login
	c.authState.Store(&tokenState{
		token:     "test-token",
		expiresAt: time.Now().Add(1 * time.Hour),
	})
	return srv, c
}

func loginSuccessHandler() http.HandlerFunc {
	expiresAt := fmt.Sprintf("%d", time.Now().Add(1*time.Hour).UnixMilli())
	return func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"login": map[string]interface{}{
					"token":                 "jwt-token-123",
					"expiresAt":             expiresAt,
					"accessibleBusinessIds": []string{"12345"},
				},
			},
		})
	}
}

// gqlResponse builds a successful GraphQL response body.
func gqlResponse(data interface{}) map[string]interface{} {
	return map[string]interface{}{"data": data}
}

// gqlErrorResponse builds a GraphQL error response body.
func gqlErrorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"data":   nil,
		"errors": []map[string]interface{}{{"message": message}},
	}
}

func propertiesPage(count, n int, hasNext bool, cursor string) map[string]interface{} {
	return gqlResponse(map[string]interface{}{
		"account": map[string]interface{}{
			"properties": map[string]interface{}{
				"count":    count,
				"pageInfo": map[string]interface{}{"hasNextPage": hasNext, "endCursor": cursor},
				"edges":    makePropertyEdges(n),
			},
		},
	})
}

// --- Auth Tests ---

func TestLoginSuccess(t *testing.T) {
	srv := httptest.NewServer(loginSuccessHandler())
	defer srv.Close()

	c, err := NewClient("test@example.com", "password", "12345", withInsecureHTTP(), WithEndpoint(srv.URL))
	require.NoError(t, err)
	err = c.EnsureAuth(context.Background())
	require.NoError(t, err)

	state := c.getAuth()
	assert.Equal(t, "jwt-token-123", state.token)
	assert.True(t, state.expiresAt.After(time.Now()))
}

func TestLoginFailureGraphqlError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlErrorResponse("Invalid credentials"))
	}))
	defer srv.Close()

	c, err := NewClient("bad@example.com", "wrong", "12345", withInsecureHTTP(), WithEndpoint(srv.URL))
	require.NoError(t, err)
	err = c.EnsureAuth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth_failed")
}

func TestLoginFailureHttp429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c, err := NewClient("test@example.com", "password", "12345", withInsecureHTTP(), WithEndpoint(srv.URL))
	require.NoError(t, err)
	err = c.EnsureAuth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate_limited")
}

func TestLoginFailureHttp500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, err := NewClient("test@example.com", "password", "12345", withInsecureHTTP(), WithEndpoint(srv.URL))
	require.NoError(t, err)
	err = c.EnsureAuth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_error")
}

func TestLoginFailureHttp400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c, err := NewClient("test@example.com", "password", "12345", withInsecureHTTP(), WithEndpoint(srv.URL))
	require.NoError(t, err)
	err = c.EnsureAuth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth_failed")
	assert.Contains(t, err.Error(), "400")
}

func TestTokenRefresh(t *testing.T) {
	var loginCount int32
	expiresAt := fmt.Sprintf("%d", time.Now().Add(1*time.Hour).UnixMilli())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&loginCount, 1)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"login": map[string]interface{}{
					"token":                 "refreshed-token",
					"expiresAt":             expiresAt,
					"accessibleBusinessIds": []string{"12345"},
				},
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient("test@example.com", "password", "12345", withInsecureHTTP(), WithEndpoint(srv.URL))
	require.NoError(t, err)
	// Set expired token
	c.authState.Store(&tokenState{token: "old-token", expiresAt: time.Now().Add(-1 * time.Minute)})

	err = c.EnsureAuth(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "refreshed-token", c.getAuth().token)
	assert.Equal(t, int32(1), atomic.LoadInt32(&loginCount))
}

func TestConcurrentAuth(t *testing.T) {
	var loginCount int32
	expiresAt := fmt.Sprintf("%d", time.Now().Add(1*time.Hour).UnixMilli())

	// Use a gate channel to ensure all goroutines are ready before any can proceed,
	// and a latch to hold the login response until all goroutines have started.
	goroutinesReady := make(chan struct{})
	loginLatch := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&loginCount, 1)
		// Wait for all goroutines to have been launched — guarantees they all
		// contend on the same login rather than racing against startup.
		<-loginLatch
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"login": map[string]interface{}{
					"token":                 "concurrent-token",
					"expiresAt":             expiresAt,
					"accessibleBusinessIds": []string{"12345"},
				},
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient("test@example.com", "password", "12345", withInsecureHTTP(), WithEndpoint(srv.URL))
	require.NoError(t, err)

	const n = 10
	var wg sync.WaitGroup
	var started int32
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Signal that this goroutine is ready
			if atomic.AddInt32(&started, 1) == n {
				close(goroutinesReady)
			}
			<-goroutinesReady // wait for all goroutines
			err := c.EnsureAuth(context.Background())
			assert.NoError(t, err)
		}()
	}

	// Once all goroutines are ready, release the login handler
	<-goroutinesReady
	close(loginLatch)
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&loginCount), "double-checked locking should result in exactly 1 login")
}

// --- Pagination Tests ---

func TestPaginationLimitLessThan100(t *testing.T) {
	var requestCount int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		json.NewEncoder(w).Encode(propertiesPage(50, 50, false, ""))
	})

	items, count, err := c.ListProperties(context.Background(), models.ListOptions{Limit: 30})
	require.NoError(t, err)
	assert.Equal(t, 30, len(items))
	assert.Equal(t, 50, count)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
}

func TestPaginationLimit100NoSecondRequest(t *testing.T) {
	var requestCount int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		json.NewEncoder(w).Encode(propertiesPage(200, 100, true, "cursor100"))
	})

	items, count, err := c.ListProperties(context.Background(), models.ListOptions{Limit: 100})
	require.NoError(t, err)
	assert.Equal(t, 100, len(items))
	assert.Equal(t, 200, count)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount), "should NOT fetch page 2")
}

func TestPaginationDefaultCap(t *testing.T) {
	var pageNum int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		p := atomic.AddInt32(&pageNum, 1)
		hasNext := p < 15
		json.NewEncoder(w).Encode(propertiesPage(1500, 100, hasNext, fmt.Sprintf("c%d", p)))
	})

	items, count, err := c.ListProperties(context.Background(), models.ListOptions{Limit: 0})
	require.NoError(t, err)
	assert.Equal(t, 1000, len(items), "default cap of 1000")
	assert.Equal(t, 1500, count)
}

func TestPaginationNoCap(t *testing.T) {
	var pageNum int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		p := atomic.AddInt32(&pageNum, 1)
		hasNext := p < 3
		json.NewEncoder(w).Encode(propertiesPage(250, 100, hasNext, fmt.Sprintf("c%d", p)))
	})

	items, count, err := c.ListProperties(context.Background(), models.ListOptions{Limit: -1})
	require.NoError(t, err)
	assert.Equal(t, 300, len(items)) // 3 pages * 100
	assert.Equal(t, 250, count)
}

func TestPaginationEmptyResult(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(propertiesPage(0, 0, false, ""))
	})

	items, count, err := c.ListProperties(context.Background(), models.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(items))
	assert.Equal(t, 0, count)
}

func TestPaginationFinalPagePartial(t *testing.T) {
	var pageNum int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		p := atomic.AddInt32(&pageNum, 1)
		if p == 1 {
			json.NewEncoder(w).Encode(propertiesPage(150, 100, true, "c1"))
		} else {
			json.NewEncoder(w).Encode(propertiesPage(150, 50, false, ""))
		}
	})

	items, count, err := c.ListProperties(context.Background(), models.ListOptions{Limit: -1})
	require.NoError(t, err)
	assert.Equal(t, 150, len(items))
	assert.Equal(t, 150, count)
}

func TestPaginationCursorForwarding(t *testing.T) {
	var pageNum int32
	var receivedCursors []string
	var mu sync.Mutex

	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		p := atomic.AddInt32(&pageNum, 1)

		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		mu.Lock()
		cursor, _ := body.Variables["after"].(string)
		receivedCursors = append(receivedCursors, cursor)
		mu.Unlock()

		remaining := 250 - int(p-1)*100
		n := 100
		if remaining < 100 {
			n = remaining
		}
		hasNext := p < 3
		endCursor := fmt.Sprintf("cursor-page-%d", p)
		json.NewEncoder(w).Encode(propertiesPage(250, n, hasNext, endCursor))
	})

	items, count, err := c.ListProperties(context.Background(), models.ListOptions{Limit: 250})
	require.NoError(t, err)
	assert.Equal(t, 250, len(items))
	assert.Equal(t, 250, count)

	// Verify cursor was forwarded correctly on each page
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 3, len(receivedCursors))
	assert.Equal(t, "", receivedCursors[0], "page 1 should have no cursor")
	assert.Equal(t, "cursor-page-1", receivedCursors[1], "page 2 should use cursor from page 1")
	assert.Equal(t, "cursor-page-2", receivedCursors[2], "page 3 should use cursor from page 2")
}

// --- Resilience Tests ---

func TestRetrySuccessAfterTransientFailure(t *testing.T) {
	var attempt int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempt, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(propertiesPage(5, 5, false, ""))
	})

	items, count, err := c.ListProperties(context.Background(), models.ListOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 5, len(items))
	assert.Equal(t, 5, count)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempt))
}

func TestAbortAfter3ConsecutiveFailures(t *testing.T) {
	var attempt int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempt, 1)
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, _, err := c.ListProperties(context.Background(), models.ListOptions{Limit: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 3 retries")
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempt))
}

func TestNoRetryOnNonTransientError(t *testing.T) {
	var attempt int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempt, 1)
		json.NewEncoder(w).Encode(gqlErrorResponse("Field 'bogus' not found"))
	})

	_, _, err := c.ListProperties(context.Background(), models.ListOptions{Limit: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Field 'bogus' not found")
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempt), "should not retry non-transient errors")
}

func TestNoRetryOnHttp400(t *testing.T) {
	var attempt int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempt, 1)
		w.WriteHeader(http.StatusBadRequest)
	})

	_, _, err := c.ListProperties(context.Background(), models.ListOptions{Limit: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempt), "should not retry 400")
}

func TestContextCancellationStopsPagination(t *testing.T) {
	// Cancel the context after page 1 completes but before page 2's request.
	// The handler signals when page 1 is done; we cancel before the pagination
	// loop checks ctx.Err() at the top of the next iteration.
	page1Done := make(chan struct{})
	var pageNum int32

	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		p := atomic.AddInt32(&pageNum, 1)
		json.NewEncoder(w).Encode(propertiesPage(500, 100, true, fmt.Sprintf("c%d", p)))
		if p == 1 {
			close(page1Done)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, _, err := c.ListProperties(ctx, models.ListOptions{Limit: -1})
		errCh <- err
	}()

	// Wait for page 1 to complete, then cancel before page 2 starts
	<-page1Done
	cancel()

	err := <-errCh
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "context cancelled") || strings.Contains(err.Error(), "context canceled"),
		"expected context cancellation error, got: %v", err,
	)
}

// --- HTTP Status Code Tests ---

func TestHttp401InvalidatesAuthAndReturnsError(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := c.GetAccount(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth_failed")
	assert.Contains(t, err.Error(), "401")

	// Verify token was invalidated
	state := c.getAuth()
	assert.True(t, state.expiresAt.IsZero(), "token should be invalidated after 401")
}

func TestHttp403ReturnsAuthError(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	_, err := c.GetAccount(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth_failed")
	assert.Contains(t, err.Error(), "403")
}

func TestHttp400ReturnsNonRetryableError(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	_, err := c.GetAccount(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_error")
	assert.Contains(t, err.Error(), "400")
}

func TestHttp5xxReturnsRetryableApiError(t *testing.T) {
	var attempts int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.GetAccount(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_error")
	// GetAccount retries transient errors; after exhausting retries, the error is terminal.
	assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(2), "should retry on 5xx")
}

func TestHttp429ReturnsRateLimited(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := c.GetAccount(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate_limited")
}

func TestGraphqlErrorResponseNotRetried(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlErrorResponse("Something went wrong"))
	})

	_, err := c.GetAccount(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_error")
	assert.Contains(t, err.Error(), "Something went wrong")
	var ae *apiError
	require.True(t, errors.As(err, &ae))
	assert.False(t, ae.Retryable)
}

// --- Query Method Tests ---

func TestGetAccount(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"id":   "12345",
				"name": "Test Account",
			},
		}))
	})

	acct, err := c.GetAccount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "12345", acct.ID)
	assert.Equal(t, "Test Account", acct.Name)
}

func TestListPropertiesWithFilters(t *testing.T) {
	var receivedFilter map[string]interface{}
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFilter, _ = body.Variables["filter"].(map[string]interface{})

		json.NewEncoder(w).Encode(propertiesPage(0, 0, false, ""))
	})

	_, _, err := c.ListProperties(context.Background(), models.ListOptions{
		Search:     "sunset",
		LocationID: "prop-99",
	})
	require.NoError(t, err)
	require.NotNil(t, receivedFilter, "filter should be present in request")
	assert.Equal(t, "sunset", receivedFilter["search"])
	propIDs, ok := receivedFilter["propertyId"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, "prop-99", propIDs[0])
}

func TestListUnitsHappyPath(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"properties": map[string]interface{}{
					"edges": []interface{}{
						map[string]interface{}{
							"node": map[string]interface{}{
								"units": map[string]interface{}{
									"count":    2,
									"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
									"edges": []interface{}{
										map[string]interface{}{"cursor": "u1", "node": map[string]interface{}{"id": "unit-1", "name": "Unit A"}},
										map[string]interface{}{"cursor": "u2", "node": map[string]interface{}{"id": "unit-2", "name": "Unit B"}},
									},
								},
							},
						},
					},
				},
			},
		}))
	})

	units, count, err := c.ListUnits(context.Background(), "prop-1", models.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(units))
	assert.Equal(t, 2, count)
	assert.Equal(t, "unit-1", units[0].ID)
	assert.Equal(t, "Unit B", units[1].Name)
}

func TestListUnitsEmptyProperty(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"properties": map[string]interface{}{
					"edges": []interface{}{},
				},
			},
		}))
	})

	units, count, err := c.ListUnits(context.Background(), "nonexistent", models.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(units))
	assert.Equal(t, 0, count)
}

func TestListWorkOrdersHappyPath(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"workOrders": map[string]interface{}{
					"count":    1,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges": []interface{}{
						map[string]interface{}{"cursor": "w1", "node": map[string]interface{}{
							"id": "wo-1", "status": "OPEN", "description": "Fix leak", "summary": "Leak in unit 3",
							"priority": "URGENT", "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-02T00:00:00Z",
						}},
					},
				},
			},
		}))
	})

	wos, count, err := c.ListWorkOrders(context.Background(), models.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, len(wos))
	assert.Equal(t, 1, count)
	assert.Equal(t, "wo-1", wos[0].ID)
	assert.Equal(t, "OPEN", wos[0].Status)
	assert.Equal(t, "URGENT", wos[0].Priority)
}

func TestListWorkOrdersWithFilters(t *testing.T) {
	var receivedFilter map[string]interface{}
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFilter, _ = body.Variables["filter"].(map[string]interface{})

		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"workOrders": map[string]interface{}{
					"count":    0,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges":    []interface{}{},
				},
			},
		}))
	})

	after := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := c.ListWorkOrders(context.Background(), models.ListOptions{
		LocationID:   "prop-123",
		Status:       []string{"OPEN", "ON_HOLD"},
		CreatedAfter: &after,
	})
	require.NoError(t, err)

	require.NotNil(t, receivedFilter, "filter should be present in request")

	locIDs, ok := receivedFilter["locationId"].([]interface{})
	require.True(t, ok, "locationId should be an array")
	assert.Equal(t, "prop-123", locIDs[0])

	statuses, ok := receivedFilter["status"].([]interface{})
	require.True(t, ok, "status should be an array")
	assert.Equal(t, []interface{}{"OPEN", "ON_HOLD"}, statuses)

	assert.Equal(t, "2026-01-01T00:00:00Z", receivedFilter["createdAfter"])
}

func TestListInspectionsHappyPath(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"inspections": map[string]interface{}{
					"count":    1,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges": []interface{}{
						map[string]interface{}{"cursor": "i1", "node": map[string]interface{}{
							"id": "insp-1", "name": "Move-in Inspection", "status": "COMPLETE",
							"startedAt": "2026-01-01T10:00:00Z", "endedAt": "2026-01-01T11:00:00Z",
							"score": 95.0, "potentialScore": 100.0,
							"templateV2": map[string]interface{}{"id": "tmpl-1", "name": "Move-in Template"},
						}},
					},
				},
			},
		}))
	})

	inspections, count, err := c.ListInspections(context.Background(), models.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, len(inspections))
	assert.Equal(t, 1, count)
	assert.Equal(t, "insp-1", inspections[0].ID)
	assert.Equal(t, "COMPLETE", inspections[0].Status)
	require.NotNil(t, inspections[0].Score)
	assert.Equal(t, 95.0, *inspections[0].Score)
}

func TestListInspectionsWithFilters(t *testing.T) {
	var receivedFilter map[string]interface{}
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFilter, _ = body.Variables["filter"].(map[string]interface{})

		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"inspections": map[string]interface{}{
					"count":    0,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges":    []interface{}{},
				},
			},
		}))
	})

	after := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := c.ListInspections(context.Background(), models.ListOptions{
		LocationID:    "prop-456",
		Status:        []string{"COMPLETE", "INCOMPLETE"},
		CreatedAfter:  &after,
		CreatedBefore: &before,
	})
	require.NoError(t, err)

	require.NotNil(t, receivedFilter, "filter should be present in request")

	locIDs, ok := receivedFilter["locationId"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, "prop-456", locIDs[0])

	statuses, ok := receivedFilter["status"].([]interface{})
	require.True(t, ok, "status should be an array")
	assert.Equal(t, []interface{}{"COMPLETE", "INCOMPLETE"}, statuses)

	assert.Equal(t, "2026-01-01T00:00:00Z", receivedFilter["createdAfter"])
	assert.Equal(t, "2026-06-01T00:00:00Z", receivedFilter["createdBefore"])
}

// --- Debug Logging Security ---

func TestDebugNeverLogsCredentials(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(nil)

	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer "))
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{"id": "12345", "name": "Test"},
		}))
	})
	c.debug = true

	_, err := c.GetAccount(context.Background())
	require.NoError(t, err)

	logOutput := logBuf.String()
	assert.NotContains(t, logOutput, "password", "log output must not contain password")
	assert.NotContains(t, logOutput, "test-token", "log output must not contain token value")
	assert.NotContains(t, logOutput, "test@example.com", "log output must not contain email")
	assert.Contains(t, logOutput, "[debug]", "debug logging should produce output")
}

// --- effectiveCap ---

func TestEffectiveCap(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"zero returns default cap", 0, 1000},
		{"negative returns no cap", -1, 0},
		{"positive returns limit", 50, 50},
		{"large positive returns limit", 5000, 5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, effectiveCap(tt.limit))
		})
	}
}

// --- Raw Query Tests ---

func TestGetAccountRaw(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"id":   "12345",
				"name": "Test Account",
			},
		}))
	})

	raw, err := c.GetAccountRaw(context.Background())
	require.NoError(t, err)
	require.NotNil(t, raw, "raw response should not be nil")

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &parsed))
	acct := parsed["account"].(map[string]interface{})
	assert.Equal(t, "12345", acct["id"])
	assert.Equal(t, "Test Account", acct["name"])
}

func TestListPropertiesRaw(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(propertiesPage(5, 5, false, ""))
	})

	raw, err := c.ListPropertiesRaw(context.Background(), models.ListOptions{})
	require.NoError(t, err)
	require.NotNil(t, raw, "raw response should not be nil")

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &parsed))
	acct := parsed["account"].(map[string]interface{})
	props := acct["properties"].(map[string]interface{})
	assert.Equal(t, float64(5), props["count"])
}

func TestListPropertiesRawHonoursLimit(t *testing.T) {
	var receivedFirst float64
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFirst, _ = body.Variables["first"].(float64)
		json.NewEncoder(w).Encode(propertiesPage(100, 10, false, ""))
	})

	_, err := c.ListPropertiesRaw(context.Background(), models.ListOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, float64(10), receivedFirst, "raw query should use limit as page size")
}

func TestListUnitsRaw(t *testing.T) {
	var receivedFilter map[string]interface{}
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		pf, _ := body.Variables["propertiesFilter"].(map[string]interface{})
		receivedFilter = pf

		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"properties": map[string]interface{}{
					"edges": []interface{}{
						map[string]interface{}{
							"node": map[string]interface{}{
								"units": map[string]interface{}{
									"count":    2,
									"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
									"edges": []interface{}{
										map[string]interface{}{"cursor": "u1", "node": map[string]interface{}{"id": "unit-1", "name": "Unit A"}},
									},
								},
							},
						},
					},
				},
			},
		}))
	})

	raw, err := c.ListUnitsRaw(context.Background(), "prop-1", 0)
	require.NoError(t, err)
	require.NotNil(t, raw, "raw response should not be nil")

	// Verify propertyId was forwarded correctly
	require.NotNil(t, receivedFilter)
	propIDs, ok := receivedFilter["propertyId"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, "prop-1", propIDs[0])

	// Verify raw response contains expected data
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &parsed))
	acct := parsed["account"].(map[string]interface{})
	require.NotNil(t, acct["properties"])
}

func TestListUnitsRawHonoursLimit(t *testing.T) {
	var receivedFirst float64
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFirst, _ = body.Variables["first"].(float64)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"properties": map[string]interface{}{
					"edges": []interface{}{
						map[string]interface{}{
							"node": map[string]interface{}{
								"units": map[string]interface{}{
									"count":    50,
									"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
									"edges":    []interface{}{},
								},
							},
						},
					},
				},
			},
		}))
	})

	_, err := c.ListUnitsRaw(context.Background(), "prop-1", 5)
	require.NoError(t, err)
	assert.Equal(t, float64(5), receivedFirst, "raw query should use limit as page size")
}

// --- Login Cooldown Tests ---

func TestLoginCooldownPreventsHammering(t *testing.T) {
	var loginCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&loginCount, 1)
		json.NewEncoder(w).Encode(gqlErrorResponse("Invalid credentials"))
	}))
	defer srv.Close()

	c, err := NewClient("bad@example.com", "wrong", "12345", withInsecureHTTP(), WithEndpoint(srv.URL))
	require.NoError(t, err)

	// First attempt should actually call login
	err1 := c.EnsureAuth(context.Background())
	require.Error(t, err1)

	// Second attempt within cooldown should return cached error without calling login
	err2 := c.EnsureAuth(context.Background())
	require.Error(t, err2)

	assert.Equal(t, int32(1), atomic.LoadInt32(&loginCount), "login should only be called once within cooldown")
}

// --- Empty Result JSON Marshal ---

func TestPaginationEmptyResultMarshalEmptyArray(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(propertiesPage(0, 0, false, ""))
	})

	items, _, err := c.ListProperties(context.Background(), models.ListOptions{})
	require.NoError(t, err)

	data, err := json.Marshal(items)
	require.NoError(t, err)
	assert.Equal(t, "[]", string(data), "empty result should marshal to [] not null")
}

// --- rawPageFirst ---

func TestRawPageFirst(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"zero returns pageSize", 0, pageSize},
		{"negative returns pageSize", -1, pageSize},
		{"small limit returns limit", 10, 10},
		{"limit at boundary returns pageSize", pageSize, pageSize},
		{"limit above boundary returns pageSize", 200, pageSize},
		{"limit one below boundary returns limit", pageSize - 1, pageSize - 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, rawPageFirst(tt.limit))
		})
	}
}

// --- HTTPS Enforcement ---

func TestNewClientRejectsHTTPEndpoint(t *testing.T) {
	_, err := NewClient("test@example.com", "password", "12345",
		WithEndpoint("http://example.com"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "https://")
}

func TestNewClientRejectsMalformedEndpoint(t *testing.T) {
	_, err := NewClient("test@example.com", "password", "12345",
		WithEndpoint("not-a-url"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "https://")
}

func TestNewClientRejectsEmptyEndpoint(t *testing.T) {
	_, err := NewClient("test@example.com", "password", "12345",
		WithEndpoint(""),
	)
	require.Error(t, err)
}

func TestNewClientAcceptsHTTPSEndpoint(t *testing.T) {
	c, err := NewClient("test@example.com", "password", "12345",
		WithEndpoint("https://externalgraph.happyco.com"),
	)
	require.NoError(t, err)
	assert.Equal(t, "https://externalgraph.happyco.com", c.endpoint)
}

func TestNewClientInsecureHTTPBypasses(t *testing.T) {
	c, err := NewClient("test@example.com", "password", "12345",
		withInsecureHTTP(),
		WithEndpoint("http://localhost:8080"),
	)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", c.endpoint)
}

// --- Login Cooldown Expiry ---

func TestLoginCooldownExpiresAndRetries(t *testing.T) {
	var loginCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&loginCount, 1)
		json.NewEncoder(w).Encode(gqlErrorResponse("Invalid credentials"))
	}))
	defer srv.Close()

	c, err := NewClient("bad@example.com", "wrong", "12345", withInsecureHTTP(), WithEndpoint(srv.URL))
	require.NoError(t, err)

	// First attempt — hits the server
	err1 := c.EnsureAuth(context.Background())
	require.Error(t, err1)
	assert.Equal(t, int32(1), atomic.LoadInt32(&loginCount))

	// Simulate cooldown expiry by backdating lastLoginFail
	c.mu.Lock()
	c.lastLoginFail = time.Now().Add(-loginCooldown - time.Second)
	c.mu.Unlock()

	// Second attempt — cooldown expired, should hit the server again
	err2 := c.EnsureAuth(context.Background())
	require.Error(t, err2)
	assert.Equal(t, int32(2), atomic.LoadInt32(&loginCount), "should retry login after cooldown expires")
}

// --- Raw Work Orders & Inspections ---

func TestListWorkOrdersRaw(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"workOrders": map[string]interface{}{
					"count":    2,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges": []interface{}{
						map[string]interface{}{"cursor": "w1", "node": map[string]interface{}{
							"id": "wo-1", "status": "OPEN", "description": "Fix leak",
						}},
					},
				},
			},
		}))
	})

	raw, err := c.ListWorkOrdersRaw(context.Background(), models.ListOptions{})
	require.NoError(t, err)
	require.NotNil(t, raw, "raw response should not be nil")

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &parsed))
	acct := parsed["account"].(map[string]interface{})
	wos := acct["workOrders"].(map[string]interface{})
	assert.Equal(t, float64(2), wos["count"])
}

func TestListWorkOrdersRawHonoursLimit(t *testing.T) {
	var receivedFirst float64
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFirst, _ = body.Variables["first"].(float64)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"workOrders": map[string]interface{}{
					"count":    50,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges":    []interface{}{},
				},
			},
		}))
	})

	_, err := c.ListWorkOrdersRaw(context.Background(), models.ListOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, float64(10), receivedFirst, "raw query should use limit as page size")
}

func TestListWorkOrdersRawForwardsFilters(t *testing.T) {
	var receivedFilter map[string]interface{}
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFilter, _ = body.Variables["filter"].(map[string]interface{})
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"workOrders": map[string]interface{}{
					"count":    0,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges":    []interface{}{},
				},
			},
		}))
	})

	before := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err := c.ListWorkOrdersRaw(context.Background(), models.ListOptions{
		LocationID:    "prop-99",
		Status:        []string{"OPEN"},
		CreatedBefore: &before,
	})
	require.NoError(t, err)
	require.NotNil(t, receivedFilter)

	locIDs, ok := receivedFilter["locationId"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, "prop-99", locIDs[0])

	statuses, ok := receivedFilter["status"].([]interface{})
	require.True(t, ok, "status should be an array")
	assert.Equal(t, []interface{}{"OPEN"}, statuses)

	assert.Equal(t, "2026-06-01T00:00:00Z", receivedFilter["createdBefore"])
}

func TestListInspectionsRaw(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"inspections": map[string]interface{}{
					"count":    3,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges": []interface{}{
						map[string]interface{}{"cursor": "i1", "node": map[string]interface{}{
							"id": "insp-1", "name": "Test Inspection", "status": "COMPLETE",
						}},
					},
				},
			},
		}))
	})

	raw, err := c.ListInspectionsRaw(context.Background(), models.ListOptions{})
	require.NoError(t, err)
	require.NotNil(t, raw, "raw response should not be nil")

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &parsed))
	acct := parsed["account"].(map[string]interface{})
	insps := acct["inspections"].(map[string]interface{})
	assert.Equal(t, float64(3), insps["count"])
}

func TestListInspectionsRawHonoursLimit(t *testing.T) {
	var receivedFirst float64
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFirst, _ = body.Variables["first"].(float64)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"inspections": map[string]interface{}{
					"count":    50,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges":    []interface{}{},
				},
			},
		}))
	})

	_, err := c.ListInspectionsRaw(context.Background(), models.ListOptions{Limit: 25})
	require.NoError(t, err)
	assert.Equal(t, float64(25), receivedFirst, "raw query should use limit as page size")
}

func TestListInspectionsRawForwardsFilters(t *testing.T) {
	var receivedFilter map[string]interface{}
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFilter, _ = body.Variables["filter"].(map[string]interface{})
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"inspections": map[string]interface{}{
					"count":    0,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges":    []interface{}{},
				},
			},
		}))
	})

	after := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	_, err := c.ListInspectionsRaw(context.Background(), models.ListOptions{
		LocationID:   "prop-55",
		Status:       []string{"COMPLETE", "INCOMPLETE"},
		CreatedAfter: &after,
	})
	require.NoError(t, err)
	require.NotNil(t, receivedFilter)

	locIDs, ok := receivedFilter["locationId"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, "prop-55", locIDs[0])

	statuses, ok := receivedFilter["status"].([]interface{})
	require.True(t, ok, "status should be an array")
	assert.Equal(t, []interface{}{"COMPLETE", "INCOMPLETE"}, statuses)

	assert.Equal(t, "2026-03-01T00:00:00Z", receivedFilter["createdAfter"])
}

// --- Standalone Login (doLogin / Login) ---

func TestStandaloneLoginSuccess(t *testing.T) {
	expiresAt := fmt.Sprintf("%d", time.Now().Add(1*time.Hour).UnixMilli())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"login": map[string]interface{}{
					"token":                 "jwt-abc",
					"expiresAt":             expiresAt,
					"accessibleBusinessIds": []string{"111", "222"},
				},
			},
		})
	}))
	defer srv.Close()

	// Login() enforces HTTPS, so test doLogin directly via the internal path
	result, err := doLogin(context.Background(), srv.Client(), srv.URL, "user@example.com", "pass", false)
	require.NoError(t, err)
	assert.Equal(t, "jwt-abc", result.Token)
	assert.Equal(t, []string{"111", "222"}, result.AccountIDs)
	assert.False(t, result.ExpiresAt.IsZero())
}

func TestStandaloneLoginHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := doLogin(context.Background(), srv.Client(), srv.URL, "user@example.com", "bad", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth_failed")
	assert.Contains(t, err.Error(), "401")
}

func TestStandaloneLoginGraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":   nil,
			"errors": []map[string]interface{}{{"message": "invalid credentials"}},
		})
	}))
	defer srv.Close()

	_, err := doLogin(context.Background(), srv.Client(), srv.URL, "user@example.com", "bad", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestStandaloneLoginMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := doLogin(context.Background(), srv.Client(), srv.URL, "user@example.com", "pass", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing login response")
}

func TestStandaloneLoginInvalidExpiresAt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"login": map[string]interface{}{
					"token":                 "jwt-abc",
					"expiresAt":             "not-a-number",
					"accessibleBusinessIds": []string{"111"},
				},
			},
		})
	}))
	defer srv.Close()

	_, err := doLogin(context.Background(), srv.Client(), srv.URL, "user@example.com", "pass", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing expiresAt")
}

func TestStandaloneLoginRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := doLogin(context.Background(), srv.Client(), srv.URL, "user@example.com", "pass", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate_limited")
}

func TestStandaloneLoginServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := doLogin(context.Background(), srv.Client(), srv.URL, "user@example.com", "pass", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_error")
}

func TestStandaloneLoginPublicRejectsHTTP(t *testing.T) {
	_, err := Login(context.Background(), "user@example.com", "pass", "http://insecure.example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "https://")
}

func TestStandaloneLoginPublicDefaultEndpoint(t *testing.T) {
	// With empty endpoint, Login should use DefaultEndpoint — not fail on HTTPS validation.
	// It will either fail on network or reach the real API and get an auth error.
	_, err := Login(context.Background(), "user@example.com", "pass", "")
	require.Error(t, err)
	// Should NOT be an HTTPS validation error
	assert.NotContains(t, err.Error(), "must be a valid https://")
}

// --- Empty Credentials ---

func TestNewClientEmptyCredentialsAccepted(t *testing.T) {
	// NewClient doesn't validate credentials — that's deferred to login.
	// Verify it doesn't panic or error on empty strings.
	c, err := NewClient("", "", "12345", withInsecureHTTP(), WithEndpoint("http://localhost"))
	require.NoError(t, err)
	assert.Equal(t, "", c.email)
	assert.Equal(t, "", c.password)
}

func TestNewClientEmptyCredentialsFailAtAuth(t *testing.T) {
	// Verify that empty credentials produce a clear error at auth time.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlErrorResponse("email is required"))
	}))
	defer srv.Close()

	c, err := NewClient("", "", "12345", withInsecureHTTP(), WithEndpoint(srv.URL))
	require.NoError(t, err)

	err = c.EnsureAuth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth_failed")
}

// --- Malformed Query Response ---

func TestDoQueryMalformedJSON(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json at all"))
	})

	_, err := c.GetAccount(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing response")

	var ae *apiError
	require.True(t, errors.As(err, &ae))
	assert.False(t, ae.Retryable, "malformed JSON should not be retried")
}

// --- Auth Retry During Pagination ---

func TestAuthRetryDuringPagination(t *testing.T) {
	// Simulate token expiry mid-pagination: page 1 succeeds, page 2 gets 401,
	// re-auth succeeds, then page 2 retry succeeds.
	var requestCount int32
	var loginCount int32
	expiresAt := fmt.Sprintf("%d", time.Now().Add(1*time.Hour).UnixMilli())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query string `json:"query"`
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		json.Unmarshal(bodyBytes, &body)

		// Login requests
		if strings.Contains(body.Query, "login") {
			atomic.AddInt32(&loginCount, 1)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"login": map[string]interface{}{
						"token":                 "refreshed-token",
						"expiresAt":             expiresAt,
						"accessibleBusinessIds": []string{"12345"},
					},
				},
			})
			return
		}

		// Data requests
		n := atomic.AddInt32(&requestCount, 1)
		if n == 2 {
			// Second data request (page 2, first attempt) returns 401
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if n == 1 {
			// Page 1
			json.NewEncoder(w).Encode(propertiesPage(200, 100, true, "cursor1"))
		} else {
			// Page 2 retry after re-auth
			json.NewEncoder(w).Encode(propertiesPage(200, 100, false, ""))
		}
	}))
	defer srv.Close()

	c, err := NewClient("test@example.com", "password", "12345",
		withInsecureHTTP(), WithEndpoint(srv.URL), withRetryBackoff(fastBackoff))
	require.NoError(t, err)
	c.authState.Store(&tokenState{token: "original-token", expiresAt: time.Now().Add(1 * time.Hour)})

	items, count, err := c.ListProperties(context.Background(), models.ListOptions{Limit: 200})
	require.NoError(t, err)
	assert.Equal(t, 200, len(items), "should recover all items after mid-pagination auth retry")
	assert.Equal(t, 200, count)
	assert.Equal(t, int32(1), atomic.LoadInt32(&loginCount), "should re-authenticate exactly once")
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount), "page1 + page2(fail) + page2(retry)")
}

// --- Login Cooldown: Transient Errors Skip Cooldown ---

func TestLoginCooldownSkippedForTransientErrors(t *testing.T) {
	var loginCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&loginCount, 1)
		if n == 1 {
			// First attempt: transient 500 error
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Second attempt: also fails but we just want to verify it was called
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, err := NewClient("test@example.com", "password", "12345", withInsecureHTTP(), WithEndpoint(srv.URL))
	require.NoError(t, err)

	// First attempt — transient 500
	err1 := c.EnsureAuth(context.Background())
	require.Error(t, err1)

	// Second attempt — should NOT be blocked by cooldown since 500 is transient
	err2 := c.EnsureAuth(context.Background())
	require.Error(t, err2)
	assert.Equal(t, int32(2), atomic.LoadInt32(&loginCount),
		"transient login failures should not trigger cooldown — second attempt should hit the server")
}

// --- hardMaxItems ---

func TestHardMaxItems(t *testing.T) {
	assert.Equal(t, 50000, hardMaxItems, "hard ceiling should be 50000")
}

// TestHardMaxItemsBehavioural verifies that pagination actually stops at the hard ceiling
// even when hasNextPage is true and no cap is set.
func TestHardMaxItemsBehavioural(t *testing.T) {
	// Use a large page size to hit the ceiling faster in the test.
	// The server always says hasNextPage=true with 100 items per page.
	var pagesFetched int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&pagesFetched, 1)
		json.NewEncoder(w).Encode(propertiesPage(999999, pageSize, true, fmt.Sprintf("cursor-%d", pagesFetched)))
	})

	// limit < 0 means "fetch all" — only hardMaxItems should stop it
	items, _, err := c.ListProperties(context.Background(), models.ListOptions{Limit: -1})
	require.NoError(t, err)
	assert.Equal(t, hardMaxItems, len(items), "should stop at hardMaxItems")
}

// --- Stuck Cursor Detection ---

func TestPaginationStuckCursorDetected(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Always return hasNextPage=true with the same cursor
		json.NewEncoder(w).Encode(propertiesPage(200, 10, true, "stuck-cursor"))
	})

	_, _, err := c.ListProperties(context.Background(), models.ListOptions{Limit: -1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pagination stuck")
}

// --- Null Data Guard ---

func TestNullDataReturnsError(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": nil,
		})
	})

	_, err := c.GetAccount(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "null data")
}

// --- doQueryWithRetry ---

func TestDoQueryWithRetryRetriesTransientErrors(t *testing.T) {
	var attempts int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{"id": "12345", "name": "Test"},
		}))
	})

	acct, err := c.GetAccount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "12345", acct.ID)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts), "should retry twice before succeeding")
}

// --- Retry-After Header ---

func TestRetryAfterHeaderParsed(t *testing.T) {
	var attempts int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{"id": "12345", "name": "Test"},
		}))
	})

	acct, err := c.GetAccount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "12345", acct.ID)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
}

// --- Helpers ---

func makePropertyEdges(n int) []interface{} {
	edges := make([]interface{}, n)
	for i := 0; i < n; i++ {
		edges[i] = map[string]interface{}{
			"cursor": fmt.Sprintf("cursor-%d", i),
			"node": map[string]interface{}{
				"id":        fmt.Sprintf("prop-%d", i),
				"name":      fmt.Sprintf("Property %d", i),
				"createdAt": "2026-01-01T00:00:00Z",
				"address": map[string]interface{}{
					"line1": "123 Main St", "line2": "",
					"city": "San Francisco", "state": "CA",
					"country": "US", "postalCode": "94102",
				},
			},
		}
	}
	return edges
}
