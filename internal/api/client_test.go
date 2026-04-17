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

func ptrFloat64(f float64) *float64 { return &f }

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

func TestListMembersHappyPath(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"memberships": map[string]interface{}{
					"count":    2,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges": []interface{}{
						map[string]interface{}{"cursor": "m1", "node": map[string]interface{}{
							"isActive":  true,
							"createdAt": "2026-01-01T00:00:00Z",
							"account":   map[string]interface{}{"id": "acct-1", "name": "Test Account"},
							"user":      map[string]interface{}{"id": "u1", "name": "Alice", "email": "alice@example.com", "shortName": "A"},
							"roles":     map[string]interface{}{"nodes": []interface{}{map[string]interface{}{"id": "r1", "name": "Admin"}}},
						}},
						map[string]interface{}{"cursor": "m2", "node": map[string]interface{}{
							"isActive":      false,
							"createdAt":     "2026-02-01T00:00:00Z",
							"inactivatedAt": "2026-03-01T00:00:00Z",
							"user":          map[string]interface{}{"id": "u2", "name": "Bob", "email": "bob@example.com"},
						}},
					},
				},
			},
		}))
	})

	members, count, err := c.ListMembers(context.Background(), models.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	require.Len(t, members, 2)

	assert.True(t, members[0].IsActive)
	require.NotNil(t, members[0].User)
	assert.Equal(t, "u1", members[0].User.ID)
	assert.Equal(t, "Alice", members[0].User.Name)
	assert.Equal(t, "alice@example.com", members[0].User.Email)
	assert.Equal(t, "A", members[0].User.ShortName)
	require.NotNil(t, members[0].Account)
	assert.Equal(t, "acct-1", members[0].Account.ID)
	require.NotNil(t, members[0].Roles)
	require.Len(t, members[0].Roles.Nodes, 1)
	assert.Equal(t, "r1", members[0].Roles.Nodes[0].ID)
	assert.Equal(t, "Admin", members[0].Roles.Nodes[0].Name)

	assert.False(t, members[1].IsActive)
	assert.Equal(t, "2026-03-01T00:00:00Z", members[1].InactivatedAt)
	require.NotNil(t, members[1].User)
	assert.Equal(t, "u2", members[1].User.ID)
}

func TestListMembersWithFilters(t *testing.T) {
	var receivedFilter map[string]interface{}
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFilter, _ = body.Variables["filter"].(map[string]interface{})

		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"memberships": map[string]interface{}{
					"count":    0,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges":    []interface{}{},
				},
			},
		}))
	})

	_, _, err := c.ListMembers(context.Background(), models.ListOptions{
		Search:          "alice",
		IncludeInactive: true,
	})
	require.NoError(t, err)
	require.NotNil(t, receivedFilter, "filter should be present in request")
	assert.Equal(t, "alice", receivedFilter["search"])
	assert.Equal(t, true, receivedFilter["includeInactive"])
}

func TestListMembersNoFilterWhenDefault(t *testing.T) {
	var receivedVars map[string]interface{}
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedVars = body.Variables

		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"memberships": map[string]interface{}{
					"count":    0,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges":    []interface{}{},
				},
			},
		}))
	})

	_, _, err := c.ListMembers(context.Background(), models.ListOptions{})
	require.NoError(t, err)
	_, hasFilter := receivedVars["filter"]
	assert.False(t, hasFilter, "filter should not be sent when no search or include_inactive set")
}

func TestListMembersNilUser(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"memberships": map[string]interface{}{
					"count":    1,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges": []interface{}{
						map[string]interface{}{"cursor": "m1", "node": map[string]interface{}{
							"isActive":  true,
							"createdAt": "2026-01-01T00:00:00Z",
							"user":      nil,
						}},
					},
				},
			},
		}))
	})

	members, count, err := c.ListMembers(context.Background(), models.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, members, 1)
	assert.Nil(t, members[0].User)
	assert.True(t, members[0].IsActive)
}

func TestListMembersRaw(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"memberships": map[string]interface{}{
					"count":    3,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges":    []interface{}{},
				},
			},
		}))
	})

	raw, err := c.ListMembersRaw(context.Background(), models.ListOptions{})
	require.NoError(t, err)
	require.NotNil(t, raw, "raw response should not be nil")

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &parsed))
	acct := parsed["account"].(map[string]interface{})
	memberships := acct["memberships"].(map[string]interface{})
	assert.Equal(t, float64(3), memberships["count"])
}

func TestListMembersRawHonoursLimit(t *testing.T) {
	var receivedFirst float64
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFirst, _ = body.Variables["first"].(float64)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"memberships": map[string]interface{}{
					"count":    100,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges":    []interface{}{},
				},
			},
		}))
	})

	_, err := c.ListMembersRaw(context.Background(), models.ListOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, float64(10), receivedFirst, "raw query should use limit as page size")
}

func TestListMembersRawForwardsFilters(t *testing.T) {
	var receivedFilter map[string]interface{}
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedFilter, _ = body.Variables["filter"].(map[string]interface{})

		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"account": map[string]interface{}{
				"memberships": map[string]interface{}{
					"count":    0,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					"edges":    []interface{}{},
				},
			},
		}))
	})

	_, err := c.ListMembersRaw(context.Background(), models.ListOptions{
		Search:          "bob",
		IncludeInactive: true,
	})
	require.NoError(t, err)
	require.NotNil(t, receivedFilter)
	assert.Equal(t, "bob", receivedFilter["search"])
	assert.Equal(t, true, receivedFilter["includeInactive"])
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

// TestPaginationPageCeilingTrips verifies that pagination stops when the page
// count ceiling is reached, even if the per-page edge count is zero (so the
// item ceiling never trips). Defends against the malicious-server case where
// hasNextPage=true and the cursor advances on every page but no items arrive.
func TestPaginationPageCeilingTrips(t *testing.T) {
	var pagesFetched int32
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&pagesFetched, 1)
		// Zero edges, hasNextPage=true, advancing cursor each page.
		json.NewEncoder(w).Encode(propertiesPage(0, 0, true, fmt.Sprintf("cursor-%d", n)))
	})

	items, _, err := c.ListProperties(context.Background(), models.ListOptions{Limit: -1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pagination ceiling")
	assert.Empty(t, items, "no items expected when ceiling trips with zero-edge pages")
	assert.Equal(t, int32(hardMaxPages), atomic.LoadInt32(&pagesFetched),
		"should fetch exactly hardMaxPages before tripping the ceiling")
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

// --- doMutation tests ---

func TestDoMutationNoRetryOnTransient500(t *testing.T) {
	var requestCount int32
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	})
	_ = srv

	var result json.RawMessage
	err := c.doMutation(context.Background(), "mutation { test }", nil, &result)
	require.Error(t, err)

	var ae *apiError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, "api_error", ae.Category)
	// Should NOT have retried — only 1 request
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
}

func TestDoMutationAuthRetryOn401(t *testing.T) {
	var mutationAttempts int32
	var loginAttempts int32
	expiresAt := fmt.Sprintf("%d", time.Now().Add(1*time.Hour).UnixMilli())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)

		if strings.Contains(bodyStr, "login") {
			atomic.AddInt32(&loginAttempts, 1)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"login": map[string]interface{}{
						"token":                 "fresh-token",
						"expiresAt":             expiresAt,
						"accessibleBusinessIds": []string{"12345"},
					},
				},
			})
			return
		}

		attempt := atomic.AddInt32(&mutationAttempts, 1)
		if attempt == 1 {
			// First mutation attempt: return 401 to trigger auth-retry
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Second mutation attempt (after re-auth): succeed
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"testMutation": map[string]interface{}{"id": "123"},
		}))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("test@example.com", "password", "12345",
		withInsecureHTTP(),
		WithEndpoint(srv.URL),
		withRetryBackoff(fastBackoff),
	)
	require.NoError(t, err)
	// Pre-set a valid token
	c.authState.Store(&tokenState{
		token:     "about-to-expire-token",
		expiresAt: time.Now().Add(1 * time.Hour),
	})

	var result json.RawMessage
	err = c.doMutation(context.Background(), "mutation { testMutation { id } }", nil, &result)
	require.NoError(t, err)
	assert.Contains(t, string(result), "123")
	// Verify the retry path was exercised: 2 mutation attempts + 1 login
	assert.Equal(t, int32(2), atomic.LoadInt32(&mutationAttempts), "expected exactly 2 mutation attempts (initial + retry)")
	assert.Equal(t, int32(1), atomic.LoadInt32(&loginAttempts), "expected exactly 1 re-auth login")
}

func TestDoMutationNoRetryOnFatalError(t *testing.T) {
	var requestCount int32
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusBadRequest)
	})
	_ = srv

	var result json.RawMessage
	err := c.doMutation(context.Background(), "mutation { test }", nil, &result)
	require.Error(t, err)
	// 400 is a fatal error (not auth, not retryable) — should be 1 request only
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
}

func TestWorkOrderCreateRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderCreate": map[string]interface{}{
				"id":          "wo-new-123",
				"status":      "OPEN",
				"priority":    "URGENT",
				"description": "Fix leak",
			},
		}))
	})
	_ = srv

	input := models.WorkOrderCreateInput{
		LocationID:  "loc-456",
		Description: "Fix leak",
		Priority:    "URGENT",
	}
	wo, err := c.WorkOrderCreate(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "wo-new-123", wo.ID)
	assert.Equal(t, "OPEN", wo.Status)
	assert.Equal(t, "URGENT", wo.Priority)
	assert.Equal(t, "Fix leak", wo.Description)

	// Verify the mutation string was sent
	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderCreate")

	// Verify the input variables were sent
	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "loc-456", inputVar["locationId"])
	assert.Equal(t, "Fix leak", inputVar["description"])
	assert.Equal(t, "URGENT", inputVar["priority"])
}

func TestWorkOrderSetPriorityRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderSetPriority": map[string]interface{}{
				"id":       "wo-123",
				"status":   "OPEN",
				"priority": "URGENT",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderSetPriority(context.Background(), "wo-123", "URGENT")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)
	assert.Equal(t, "URGENT", wo.Priority)

	// Verify the mutation string was sent
	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderSetPriority")

	// Verify the input variables were sent
	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "URGENT", inputVar["priority"])
}

func TestWorkOrderAddAttachmentRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderAddAttachment": map[string]interface{}{
				"workOrder": map[string]interface{}{
					"id":     "wo-123",
					"status": "OPEN",
				},
				"attachment": map[string]interface{}{
					"id":        "att-456",
					"name":      "photo.jpg",
					"mediaType": "image/jpeg",
				},
				"signedURL": "https://storage.example.com/upload/att-456",
			},
		}))
	})
	_ = srv

	input := models.WorkOrderAddAttachmentInput{
		WorkOrderID: "wo-123",
		FileName:    "photo.jpg",
		MimeType:    "image/jpeg",
	}
	result, err := c.WorkOrderAddAttachment(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "wo-123", result.WorkOrder.ID)
	assert.Equal(t, "att-456", result.Attachment.ID)
	assert.Equal(t, "photo.jpg", result.Attachment.Name)
	assert.Equal(t, "https://storage.example.com/upload/att-456", result.SignedURL)

	// Verify the mutation string was sent
	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderAddAttachment")

	// Verify the input variables were sent
	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "photo.jpg", inputVar["fileName"])
	assert.Equal(t, "image/jpeg", inputVar["mimeType"])
}

func TestWorkOrderSetStatusAndSubStatusRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderSetStatusAndSubStatus": map[string]interface{}{
				"id":        "wo-123",
				"status":    "COMPLETE",
				"subStatus": "DONE",
			},
		}))
	})
	_ = srv

	input := models.WorkOrderSetStatusAndSubStatusInput{
		WorkOrderID: "wo-123",
		Status:      models.WorkOrderStatusInput{Status: "COMPLETE"},
		SubStatus:   models.WorkOrderSubStatusInput{SubStatus: "DONE"},
	}
	wo, err := c.WorkOrderSetStatusAndSubStatus(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)
	assert.Equal(t, "COMPLETE", wo.Status)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderSetStatusAndSubStatus")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	statusVar, ok := inputVar["Status"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "COMPLETE", statusVar["status"])
	subStatusVar, ok := inputVar["SubStatus"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "DONE", subStatusVar["subStatus"])
}

func TestWorkOrderSetAssigneeRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderSetAssignee": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	input := models.WorkOrderSetAssigneeInput{
		WorkOrderID: "wo-123",
		Assignee: models.AssignableInput{
			AssigneeID:   "user-456",
			AssigneeType: "USER",
		},
	}
	wo, err := c.WorkOrderSetAssignee(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderSetAssignee")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assigneeVar, ok := inputVar["assignee"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "user-456", assigneeVar["assigneeId"])
	assert.Equal(t, "USER", assigneeVar["assigneeType"])
}

func TestWorkOrderSetDescriptionRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderSetDescription": map[string]interface{}{
				"id":          "wo-123",
				"status":      "OPEN",
				"description": "New description",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderSetDescription(context.Background(), "wo-123", "New description")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)
	assert.Equal(t, "New description", wo.Description)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderSetDescription")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "New description", inputVar["description"])
}

func TestWorkOrderSetScheduledForRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderSetScheduledFor": map[string]interface{}{
				"id":           "wo-123",
				"status":       "OPEN",
				"scheduledFor": "2026-05-01T09:00:00Z",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderSetScheduledFor(context.Background(), "wo-123", "2026-05-01T09:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderSetScheduledFor")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "2026-05-01T09:00:00Z", inputVar["scheduledFor"])
}

func TestWorkOrderSetLocationRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderSetLocation": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderSetLocation(context.Background(), "wo-123", "loc-789")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderSetLocation")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "loc-789", inputVar["locationId"])
}

func TestWorkOrderSetTypeRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderSetType": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderSetType(context.Background(), "wo-123", "MAINTENANCE")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderSetType")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "MAINTENANCE", inputVar["workOrderType"])
}

func TestWorkOrderSetEntryNotesRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderSetEntryNotes": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderSetEntryNotes(context.Background(), "wo-123", "Knock first")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderSetEntryNotes")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "Knock first", inputVar["entryNotes"])
}

func TestWorkOrderSetPermissionToEnterRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderSetPermissionToEnter": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderSetPermissionToEnter(context.Background(), "wo-123", true)
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderSetPermissionToEnter")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, true, inputVar["permissionToEnter"])
}

func TestWorkOrderSetResidentApprovedEntryRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderSetResidentApprovedEntry": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderSetResidentApprovedEntry(context.Background(), "wo-123", true)
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderSetResidentApprovedEntry")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, true, inputVar["residentApprovedEntry"])
}

func TestWorkOrderSetUnitEnteredRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderSetUnitEntered": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderSetUnitEntered(context.Background(), "wo-123", true)
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderSetUnitEntered")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, true, inputVar["unitEntered"])
}

func TestWorkOrderArchiveRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderArchive": map[string]interface{}{
				"id":     "wo-123",
				"status": "ARCHIVED",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderArchive(context.Background(), "wo-123")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)
	assert.Equal(t, "ARCHIVED", wo.Status)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderArchive")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, 1, len(inputVar), "archive input should only contain workOrderId")
}

func TestWorkOrderAddCommentRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderAddComment": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderAddComment(context.Background(), "wo-123", "Checked the unit")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderAddComment")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "Checked the unit", inputVar["comment"])
}

func TestWorkOrderAddTimeRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderAddTime": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderAddTime(context.Background(), "wo-123", "PT30M")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderAddTime")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "PT30M", inputVar["duration"])
}

func TestWorkOrderRemoveAttachmentRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderRemoveAttachment": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderRemoveAttachment(context.Background(), "wo-123", "att-456")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderRemoveAttachment")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "att-456", inputVar["attachmentId"])
}

func TestWorkOrderStartTimerRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderStartTimer": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderStartTimer(context.Background(), "wo-123", "2026-04-16T08:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderStartTimer")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "2026-04-16T08:00:00Z", inputVar["startedAt"])
}

func TestWorkOrderStopTimerRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"workOrderStopTimer": map[string]interface{}{
				"id":     "wo-123",
				"status": "OPEN",
			},
		}))
	})
	_ = srv

	wo, err := c.WorkOrderStopTimer(context.Background(), "wo-123", "2026-04-16T09:30:00Z")
	require.NoError(t, err)
	assert.Equal(t, "wo-123", wo.ID)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "workOrderStopTimer")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wo-123", inputVar["workOrderId"])
	assert.Equal(t, "2026-04-16T09:30:00Z", inputVar["stoppedAt"])
}

// --- Inspection Round-Trip Tests ---

func TestInspectionCreateRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"inspectionCreate": map[string]interface{}{
				"id":     "insp-new-1",
				"name":   "Move-in",
				"status": "SCHEDULED",
			},
		}))
	})
	_ = srv

	input := models.InspectionCreateInput{
		LocationID:   "loc-1",
		TemplateID:   "tpl-1",
		ScheduledFor: "2026-06-01T00:00:00Z",
		AssignedToID: "user-1",
	}
	insp, err := c.InspectionCreate(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "insp-new-1", insp.ID)
	assert.Equal(t, "SCHEDULED", insp.Status)

	query, ok := capturedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "inspectionCreate")

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "loc-1", inputVar["locationId"])
	assert.Equal(t, "tpl-1", inputVar["templateId"])
	assert.Equal(t, "2026-06-01T00:00:00Z", inputVar["scheduledFor"])
	assert.Equal(t, "user-1", inputVar["assignedToID"])
}

func TestInspectionRateItemRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"inspectionRateItem": map[string]interface{}{
				"id":     "insp-1",
				"status": "INCOMPLETE",
			},
		}))
	})
	_ = srv

	input := models.InspectionRateItemInput{
		InspectionID: "insp-1",
		SectionName:  "Bedroom",
		ItemName:     "Walls",
		Rating: models.InspectionRatingInput{
			Key:   "condition",
			Score: ptrFloat64(4.0),
			Value: "Good",
		},
	}
	insp, err := c.InspectionRateItem(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "insp-1", insp.ID)

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "insp-1", inputVar["inspectionId"])
	assert.Equal(t, "Bedroom", inputVar["sectionName"])
	assert.Equal(t, "Walls", inputVar["itemName"])
	rating, ok := inputVar["rating"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "condition", rating["key"])
	assert.Equal(t, float64(4), rating["score"])
	assert.Equal(t, "Good", rating["value"])
}

// --- Webhook Round-Trip Tests ---

func TestWebhookCreateRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"webhookCreate": map[string]interface{}{
				"id":     "wh-new-1",
				"url":    "https://example.com/hook",
				"status": "ENABLED",
			},
		}))
	})
	_ = srv

	input := models.WebhookCreateInput{
		SubscriberID:   "acct-1",
		SubscriberType: "ACCOUNT",
		URL:            "https://example.com/hook",
		Subjects:       []string{"WORK_ORDERS", "INSPECTIONS"},
		Status:         "ENABLED",
	}
	wh, err := c.WebhookCreate(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "wh-new-1", wh.ID)
	assert.Equal(t, "https://example.com/hook", wh.URL)

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "acct-1", inputVar["subscriberId"])
	assert.Equal(t, "ACCOUNT", inputVar["subscriberType"])
	assert.Equal(t, "https://example.com/hook", inputVar["url"])
	subjects, ok := inputVar["subjects"].([]interface{})
	require.True(t, ok)
	assert.Len(t, subjects, 2)
}

// --- Role Round-Trip Tests ---

func TestRoleCreateRoundTrip(t *testing.T) {
	var capturedBody map[string]interface{}
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"roleCreate": map[string]interface{}{
				"id":   "role-new-1",
				"name": "Inspector",
			},
		}))
	})
	_ = srv

	input := models.RoleCreateInput{
		AccountID:   "acct-1",
		Name:        "Inspector",
		Description: "Can perform inspections",
		Permissions: models.PermissionsInput{
			Grant:  []string{"inspection:inspection.create", "inspection:inspection.view"},
			Revoke: []string{"task:task.delete"},
		},
	}
	role, err := c.RoleCreate(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "role-new-1", role.ID)
	assert.Equal(t, "Inspector", role.Name)

	vars, ok := capturedBody["variables"].(map[string]interface{})
	require.True(t, ok)
	inputVar, ok := vars["input"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "acct-1", inputVar["accountId"])
	assert.Equal(t, "Inspector", inputVar["name"])
	assert.Equal(t, "Can perform inspections", inputVar["description"])
	perms, ok := inputVar["permissions"].(map[string]interface{})
	require.True(t, ok)
	grant, ok := perms["grant"].([]interface{})
	require.True(t, ok)
	assert.Len(t, grant, 2)
	revoke, ok := perms["revoke"].([]interface{})
	require.True(t, ok)
	assert.Len(t, revoke, 1)
}

// TestMutationRetryClassification verifies that non-idempotent mutations (Create,
// AddComment, AddTime, AddAttachment) do NOT retry on 500, while idempotent
// mutations (all set*, archive, remove, start, stop) DO retry.
func TestMutationRetryClassification(t *testing.T) {
	// nonIdempotent mutations should attempt exactly 1 request on a 500 error.
	// Idempotent mutations should attempt >1 request (retry).
	tests := []struct {
		name       string
		call       func(ctx context.Context, c *Client) error
		idempotent bool
	}{
		{"WorkOrderCreate", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderCreate(ctx, models.WorkOrderCreateInput{LocationID: "loc-1"})
			return err
		}, false},
		{"WorkOrderAddComment", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderAddComment(ctx, "wo-1", "text")
			return err
		}, false},
		{"WorkOrderAddTime", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderAddTime(ctx, "wo-1", "PT1H")
			return err
		}, false},
		{"WorkOrderAddAttachment", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderAddAttachment(ctx, models.WorkOrderAddAttachmentInput{WorkOrderID: "wo-1", FileName: "f", MimeType: "image/jpeg"})
			return err
		}, false},
		{"WorkOrderSetPriority", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderSetPriority(ctx, "wo-1", "URGENT")
			return err
		}, true},
		{"WorkOrderSetDescription", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderSetDescription(ctx, "wo-1", "desc")
			return err
		}, true},
		{"WorkOrderArchive", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderArchive(ctx, "wo-1")
			return err
		}, true},
		{"WorkOrderRemoveAttachment", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderRemoveAttachment(ctx, "wo-1", "att-1")
			return err
		}, true},
		{"WorkOrderStartTimer", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderStartTimer(ctx, "wo-1", "2026-01-01T00:00:00Z")
			return err
		}, true},
		{"WorkOrderStopTimer", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderStopTimer(ctx, "wo-1", "2026-01-01T01:00:00Z")
			return err
		}, true},
		{"WorkOrderSetStatusAndSubStatus", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderSetStatusAndSubStatus(ctx, models.WorkOrderSetStatusAndSubStatusInput{WorkOrderID: "wo-1", Status: models.WorkOrderStatusInput{Status: "COMPLETED"}})
			return err
		}, true},
		{"WorkOrderSetAssignee", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderSetAssignee(ctx, models.WorkOrderSetAssigneeInput{WorkOrderID: "wo-1"})
			return err
		}, true},
		{"WorkOrderSetScheduledFor", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderSetScheduledFor(ctx, "wo-1", "2026-01-01T00:00:00Z")
			return err
		}, true},
		{"WorkOrderSetLocation", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderSetLocation(ctx, "wo-1", "loc-1")
			return err
		}, true},
		{"WorkOrderSetType", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderSetType(ctx, "wo-1", "SERVICE_REQUEST")
			return err
		}, true},
		{"WorkOrderSetEntryNotes", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderSetEntryNotes(ctx, "wo-1", "notes")
			return err
		}, true},
		{"WorkOrderSetPermissionToEnter", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderSetPermissionToEnter(ctx, "wo-1", true)
			return err
		}, true},
		{"WorkOrderSetResidentApprovedEntry", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderSetResidentApprovedEntry(ctx, "wo-1", true)
			return err
		}, true},
		{"WorkOrderSetUnitEntered", func(ctx context.Context, c *Client) error {
			_, err := c.WorkOrderSetUnitEntered(ctx, "wo-1", true)
			return err
		}, true},

		// --- Inspection domain ---
		{"InspectionCreate", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionCreate(ctx, models.InspectionCreateInput{LocationID: "loc-1", TemplateID: "tpl-1", ScheduledFor: "2026-01-01T00:00:00Z"})
			return err
		}, false},
		{"InspectionAddSection", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionAddSection(ctx, models.InspectionAddSectionInput{InspectionID: "insp-1", Name: "s"})
			return err
		}, false},
		{"InspectionAddItem", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionAddItem(ctx, models.InspectionAddItemInput{InspectionID: "insp-1", SectionName: "s", Name: "i", RatingGroupID: "rg-1"})
			return err
		}, false},
		{"InspectionAddItemPhoto", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionAddItemPhoto(ctx, models.InspectionAddItemPhotoInput{InspectionID: "insp-1", SectionName: "s", ItemName: "i", MimeType: "image/jpeg"})
			return err
		}, false},
		{"InspectionDuplicateSection", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionDuplicateSection(ctx, models.InspectionDuplicateSectionInput{InspectionID: "insp-1", SectionName: "s"})
			return err
		}, false},
		{"InspectionSendToGuest", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionSendToGuest(ctx, models.InspectionSendToGuestInput{InspectionID: "insp-1", Email: "a@b.com"})
			return err
		}, false},
		{"InspectionStart", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionStart(ctx, "insp-1")
			return err
		}, true},
		{"InspectionComplete", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionComplete(ctx, "insp-1")
			return err
		}, true},
		{"InspectionArchive", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionArchive(ctx, "insp-1")
			return err
		}, true},
		{"InspectionSetAssignee", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionSetAssignee(ctx, models.InspectionSetAssigneeInput{InspectionID: "insp-1"})
			return err
		}, true},
		{"InspectionDeleteSection", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionDeleteSection(ctx, models.InspectionDeleteSectionInput{InspectionID: "insp-1", SectionName: "s"})
			return err
		}, true},
		{"InspectionRemoveItemPhoto", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionRemoveItemPhoto(ctx, models.InspectionRemoveItemPhotoInput{InspectionID: "insp-1", PhotoID: "p-1", SectionName: "s", ItemName: "i"})
			return err
		}, true},
		{"InspectionReopen", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionReopen(ctx, "insp-1")
			return err
		}, true},
		{"InspectionExpire", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionExpire(ctx, "insp-1")
			return err
		}, true},
		{"InspectionUnexpire", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionUnexpire(ctx, "insp-1")
			return err
		}, true},
		{"InspectionSetDueBy", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionSetDueBy(ctx, models.InspectionSetDueByInput{InspectionID: "insp-1", DueBy: "2026-01-01T00:00:00Z"})
			return err
		}, true},
		{"InspectionSetScheduledFor", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionSetScheduledFor(ctx, "insp-1", "2026-01-01T00:00:00Z")
			return err
		}, true},
		{"InspectionSetHeaderField", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionSetHeaderField(ctx, models.InspectionSetHeaderFieldInput{InspectionID: "insp-1", Label: "l", Value: "v"})
			return err
		}, true},
		{"InspectionSetFooterField", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionSetFooterField(ctx, models.InspectionSetFooterFieldInput{InspectionID: "insp-1", Label: "l", Value: "v"})
			return err
		}, true},
		{"InspectionSetItemNotes", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionSetItemNotes(ctx, models.InspectionSetItemNotesInput{InspectionID: "insp-1", SectionName: "s", ItemName: "i", Notes: "n"})
			return err
		}, true},
		{"InspectionRateItem", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionRateItem(ctx, models.InspectionRateItemInput{InspectionID: "insp-1", SectionName: "s", ItemName: "i", Rating: models.InspectionRatingInput{Key: "k"}})
			return err
		}, true},
		{"InspectionRenameSection", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionRenameSection(ctx, models.InspectionRenameSectionInput{InspectionID: "insp-1", SectionName: "s", NewSectionName: "n"})
			return err
		}, true},
		{"InspectionDeleteItem", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionDeleteItem(ctx, models.InspectionDeleteItemInput{InspectionID: "insp-1", SectionName: "s", ItemName: "i"})
			return err
		}, true},
		{"InspectionMoveItemPhoto", func(ctx context.Context, c *Client) error {
			_, err := c.InspectionMoveItemPhoto(ctx, models.InspectionMoveItemPhotoInput{InspectionID: "insp-1", PhotoID: "p-1", FromSectionName: "s1", FromItemName: "i1", ToSectionName: "s2", ToItemName: "i2"})
			return err
		}, true},

		// --- Project domain ---
		{"ProjectCreate", func(ctx context.Context, c *Client) error {
			_, err := c.ProjectCreate(ctx, models.ProjectCreateInput{ProjectTemplateID: "tpl-1", LocationID: "loc-1", StartAt: "2026-01-01T00:00:00Z"})
			return err
		}, false},
		{"ProjectSetPriority", func(ctx context.Context, c *Client) error {
			_, err := c.ProjectSetPriority(ctx, "proj-1", "HIGH")
			return err
		}, true},
		{"ProjectSetOnHold", func(ctx context.Context, c *Client) error {
			_, err := c.ProjectSetOnHold(ctx, "proj-1", true)
			return err
		}, true},
		{"ProjectSetAssignee", func(ctx context.Context, c *Client) error {
			_, err := c.ProjectSetAssignee(ctx, models.ProjectSetAssigneeInput{ProjectID: "proj-1"})
			return err
		}, true},
		{"ProjectSetNotes", func(ctx context.Context, c *Client) error {
			_, err := c.ProjectSetNotes(ctx, "proj-1", "notes")
			return err
		}, true},
		{"ProjectSetDueAt", func(ctx context.Context, c *Client) error {
			_, err := c.ProjectSetDueAt(ctx, "proj-1", "2026-01-01T00:00:00Z")
			return err
		}, true},
		{"ProjectSetStartAt", func(ctx context.Context, c *Client) error {
			_, err := c.ProjectSetStartAt(ctx, "proj-1", "2026-01-01T00:00:00Z")
			return err
		}, true},
		{"ProjectSetAvailabilityTargetAt", func(ctx context.Context, c *Client) error {
			at := "2026-01-01T00:00:00Z"
			_, err := c.ProjectSetAvailabilityTargetAt(ctx, "proj-1", &at)
			return err
		}, true},

		// --- User domain ---
		{"UserCreate", func(ctx context.Context, c *Client) error {
			_, err := c.UserCreate(ctx, models.UserCreateInput{AccountID: "acct-1", Email: "a@b.com", Name: "Test"})
			return err
		}, false},
		{"UserSetEmail", func(ctx context.Context, c *Client) error {
			_, err := c.UserSetEmail(ctx, "user-1", "a@b.com")
			return err
		}, true},
		{"UserSetName", func(ctx context.Context, c *Client) error {
			_, err := c.UserSetName(ctx, "user-1", "Name")
			return err
		}, true},
		{"UserSetShortName", func(ctx context.Context, c *Client) error {
			sn := "SN"
			_, err := c.UserSetShortName(ctx, "user-1", &sn)
			return err
		}, true},
		{"UserSetPhone", func(ctx context.Context, c *Client) error {
			ph := "555-0100"
			_, err := c.UserSetPhone(ctx, "user-1", &ph)
			return err
		}, true},

		// --- Membership domain ---
		{"AccountMembershipCreate", func(ctx context.Context, c *Client) error {
			_, err := c.AccountMembershipCreate(ctx, models.AccountMembershipCreateInput{AccountID: "acct-1", UserID: "user-1"})
			return err
		}, false},
		{"AccountMembershipActivate", func(ctx context.Context, c *Client) error {
			_, err := c.AccountMembershipActivate(ctx, models.AccountMembershipActivateInput{AccountID: "acct-1", UserID: "user-1"})
			return err
		}, true},
		{"AccountMembershipDeactivate", func(ctx context.Context, c *Client) error {
			_, err := c.AccountMembershipDeactivate(ctx, models.AccountMembershipDeactivateInput{AccountID: "acct-1", UserID: "user-1"})
			return err
		}, true},
		{"AccountMembershipSetRoles", func(ctx context.Context, c *Client) error {
			_, err := c.AccountMembershipSetRoles(ctx, models.AccountMembershipSetRolesInput{AccountID: "acct-1", UserID: "user-1"})
			return err
		}, true},

		// --- Property Access domain ---
		{"PropertyGrantUserAccess", func(ctx context.Context, c *Client) error {
			_, err := c.PropertyGrantUserAccess(ctx, models.PropertyGrantUserAccessInput{PropertyID: "prop-1", UserID: []string{"u-1"}})
			return err
		}, true},
		{"PropertyRevokeUserAccess", func(ctx context.Context, c *Client) error {
			_, err := c.PropertyRevokeUserAccess(ctx, models.PropertyRevokeUserAccessInput{PropertyID: "prop-1", UserID: []string{"u-1"}})
			return err
		}, true},
		{"PropertySetAccountWideAccess", func(ctx context.Context, c *Client) error {
			_, err := c.PropertySetAccountWideAccess(ctx, models.PropertySetAccountWideAccessInput{PropertyID: "prop-1", AccountWideAccess: true})
			return err
		}, true},
		{"UserGrantPropertyAccess", func(ctx context.Context, c *Client) error {
			_, err := c.UserGrantPropertyAccess(ctx, models.UserGrantPropertyAccessInput{UserID: "user-1", PropertyID: []string{"prop-1"}})
			return err
		}, true},
		{"UserRevokePropertyAccess", func(ctx context.Context, c *Client) error {
			_, err := c.UserRevokePropertyAccess(ctx, models.UserRevokePropertyAccessInput{UserID: "user-1", PropertyID: []string{"prop-1"}})
			return err
		}, true},

		// --- Role domain ---
		{"RoleCreate", func(ctx context.Context, c *Client) error {
			_, err := c.RoleCreate(ctx, models.RoleCreateInput{AccountID: "acct-1", Name: "r", Permissions: models.PermissionsInput{Grant: []string{"p"}}})
			return err
		}, false},
		{"RoleSetName", func(ctx context.Context, c *Client) error {
			_, err := c.RoleSetName(ctx, models.RoleSetNameInput{AccountID: "acct-1", RoleID: "role-1", Name: "n"})
			return err
		}, true},
		{"RoleSetPermissions", func(ctx context.Context, c *Client) error {
			_, err := c.RoleSetPermissions(ctx, models.RoleSetPermissionsInput{AccountID: "acct-1", RoleID: "role-1", Permissions: models.PermissionsInput{Grant: []string{"p"}}})
			return err
		}, true},
		{"RoleSetDescription", func(ctx context.Context, c *Client) error {
			desc := "d"
			_, err := c.RoleSetDescription(ctx, models.RoleSetDescriptionInput{AccountID: "acct-1", RoleID: "role-1", Description: &desc})
			return err
		}, true},

		// --- Webhook domain ---
		{"WebhookCreate", func(ctx context.Context, c *Client) error {
			_, err := c.WebhookCreate(ctx, models.WebhookCreateInput{SubscriberID: "sub-1", SubscriberType: "ACCOUNT", URL: "https://example.com/hook"})
			return err
		}, false},
		{"WebhookUpdate", func(ctx context.Context, c *Client) error {
			_, err := c.WebhookUpdate(ctx, models.WebhookUpdateInput{ID: "wh-1"})
			return err
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount int32
			srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&requestCount, 1)
				w.WriteHeader(http.StatusInternalServerError)
			})
			_ = srv

			err := tt.call(context.Background(), c)
			require.Error(t, err)

			count := atomic.LoadInt32(&requestCount)
			if tt.idempotent {
				assert.Greater(t, count, int32(1), "%s should retry on 500 (idempotent)", tt.name)
			} else {
				assert.Equal(t, int32(1), count, "%s should NOT retry on 500 (non-idempotent)", tt.name)
			}
		})
	}
}

func TestDoMutationIdempotentRetriesOnTransient500(t *testing.T) {
	var requestCount int32
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"testMutation": map[string]interface{}{"id": "456"},
		}))
	})
	_ = srv

	var result json.RawMessage
	err := c.doMutationIdempotent(context.Background(), "mutation { testMutation { id } }", nil, &result)
	require.NoError(t, err)
	assert.Contains(t, string(result), "456")
	// Should have retried: 2 failures + 1 success = 3
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
}

// TestDoMutationDestructiveIdempotent_NotFoundAfterTransientReturnsAlreadyApplied
// verifies that destructive mutations (Archive/Delete/Expire/Revoke) return
// ErrAlreadyApplied (rather than nil) when a "not found" error follows a
// transient failure. The previous attempt likely committed before the
// transient error was reported. Returning a sentinel error (not nil) prevents
// callers from surfacing the zero-value result struct as if it were real data.
func TestDoMutationDestructiveIdempotent_NotFoundAfterTransientReturnsAlreadyApplied(t *testing.T) {
	var requestCount int32
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Second attempt: server says "not found" because the first attempt
		// already committed.
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]interface{}{{"message": "Resource not found"}},
		})
	})
	_ = srv

	var result json.RawMessage
	err := c.doMutationDestructiveIdempotent(context.Background(), "mutation { archive { id } }", nil, &result)
	require.Error(t, err, "not-found after transient must surface as ErrAlreadyApplied so callers don't return zero-value entities")
	require.ErrorIs(t, err, ErrAlreadyApplied)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount))
}

// TestDoMutationDestructiveIdempotent_AlreadyAppliedSurvivesMultipleTransients
// covers the chain 5xx → 5xx → not-found, exercising the multi-transient
// retry path with synthesis at the end.
func TestDoMutationDestructiveIdempotent_AlreadyAppliedSurvivesMultipleTransients(t *testing.T) {
	var requestCount int32
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]interface{}{{"message": "already archived"}},
		})
	})
	_ = srv

	var result json.RawMessage
	err := c.doMutationDestructiveIdempotent(context.Background(), "mutation { archive { id } }", nil, &result)
	require.ErrorIs(t, err, ErrAlreadyApplied)
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
}

// TestDoMutationDestructiveIdempotent_NotFoundOnFirstAttemptIsError verifies
// the synthesised-success path does NOT fire for a true "not found" — the
// transient flag is required.
func TestDoMutationDestructiveIdempotent_NotFoundOnFirstAttemptIsError(t *testing.T) {
	var requestCount int32
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]interface{}{{"message": "Resource not found"}},
		})
	})
	_ = srv

	var result json.RawMessage
	err := c.doMutationDestructiveIdempotent(context.Background(), "mutation { archive { id } }", nil, &result)
	require.Error(t, err, "not-found on first attempt must surface — there was no prior transient failure")
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
}

// TestDoMutationDestructiveIdempotent_HappyPath verifies normal success
// path is unchanged.
func TestDoMutationDestructiveIdempotent_HappyPath(t *testing.T) {
	var requestCount int32
	srv, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		json.NewEncoder(w).Encode(gqlResponse(map[string]interface{}{
			"archive": map[string]interface{}{"id": "wo-1"},
		}))
	})
	_ = srv

	var result json.RawMessage
	err := c.doMutationDestructiveIdempotent(context.Background(), "mutation { archive { id } }", nil, &result)
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
}

func TestIsLikelyAlreadyAppliedError(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"Resource not found", true},
		{"resource NOT FOUND", true},
		{"work order does not exist", true},
		{"no such inspection", true},
		{"already archived", true},
		{"already deleted", true},
		{"already expired", true},
		{"already removed", true},
		{"Validation failed: name is required", false},
		{"Permission denied", false},
		{"unauthorized", false},
		{"", false},
	}
	for _, c := range cases {
		got := isLikelyAlreadyAppliedError(c.msg)
		assert.Equal(t, c.want, got, "msg=%q", c.msg)
	}
}
