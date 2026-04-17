package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/findingsimple/hppy-connect/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConfigPathDefault(t *testing.T) {
	got := resolveConfigPathFrom("")
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".hppycli.yaml"), got)
}

func TestResolveConfigPathFlag(t *testing.T) {
	got := resolveConfigPathFrom("/custom/path/config.yaml")
	assert.Equal(t, "/custom/path/config.yaml", got)
}

// --- authenticateWithRetry ---

// stubInputs supplies a sequence of stdin lines (email, "y/n", etc.) plus a
// fixed password.
func stubInputs(stdinLines []string, password string) (*bufio.Reader, passwordReader) {
	reader := bufio.NewReader(strings.NewReader(strings.Join(stdinLines, "\n") + "\n"))
	pwFn := func() (string, error) { return password, nil }
	return reader, pwFn
}

// loginAlwaysFails returns a loginFn that errors on every call and counts attempts.
func loginAlwaysFails(counter *atomic.Int32) loginFn {
	return func(ctx context.Context, email, password, endpoint string) (*api.LoginResult, error) {
		counter.Add(1)
		return nil, fmt.Errorf("invalid credentials")
	}
}

// loginSucceedsOnAttempt returns a loginFn that fails the first N attempts then succeeds.
func loginSucceedsOnAttempt(failFirst int, counter *atomic.Int32) loginFn {
	return func(ctx context.Context, email, password, endpoint string) (*api.LoginResult, error) {
		n := counter.Add(1)
		if int(n) <= failFirst {
			return nil, fmt.Errorf("invalid credentials")
		}
		return &api.LoginResult{
			AccountIDs: []string{"acct-1"},
			Token:      "tok",
			ExpiresAt:  time.Now().Add(time.Hour),
		}, nil
	}
}

func TestAuthenticateWithRetry_HappyPathFirstAttempt(t *testing.T) {
	reader, pw := stubInputs([]string{"user@example.com"}, "secret")
	var stderr bytes.Buffer
	var calls atomic.Int32

	email, password, result, err := authenticateWithRetry(
		context.Background(), reader, &stderr,
		loginSucceedsOnAttempt(0, &calls), pw, "https://api.example.com",
	)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", email)
	assert.Equal(t, "secret", password)
	require.NotNil(t, result)
	assert.Equal(t, []string{"acct-1"}, result.AccountIDs)
	assert.Equal(t, int32(1), calls.Load())
}

func TestAuthenticateWithRetry_RetriesAndSucceeds(t *testing.T) {
	// User fat-fingers password twice, gets it right the third time.
	// Stdin sequence: email, y, y (after first failure), then success on 3rd login.
	// (No "Try again?" prompt after the successful login.)
	reader, pw := stubInputs([]string{"user@example.com", "y", "y"}, "secret")
	var stderr bytes.Buffer
	var calls atomic.Int32

	_, _, result, err := authenticateWithRetry(
		context.Background(), reader, &stderr,
		loginSucceedsOnAttempt(2, &calls), pw, "https://api.example.com",
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int32(3), calls.Load(), "must call login 3 times (2 failures + 1 success)")
}

func TestAuthenticateWithRetry_ExhaustionReturnsDocumentedError(t *testing.T) {
	// All 3 attempts fail. Stdin: email, y (after attempt 1), y (after attempt 2).
	// Attempt 3 fails too — no further prompt because we hit max attempts.
	reader, pw := stubInputs([]string{"user@example.com", "y", "y"}, "wrong")
	var stderr bytes.Buffer
	var calls atomic.Int32

	_, _, _, err := authenticateWithRetry(
		context.Background(), reader, &stderr,
		loginAlwaysFails(&calls), pw, "https://api.example.com",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed after 3 attempts",
		"exhaustion error message must mention the attempt count")
	assert.Equal(t, int32(configInitMaxAttempts), calls.Load(),
		"must attempt exactly configInitMaxAttempts times before giving up")
}

func TestAuthenticateWithRetry_ExplicitAbort(t *testing.T) {
	// User fails first attempt then says "n" to retry.
	reader, pw := stubInputs([]string{"user@example.com", "n"}, "wrong")
	var stderr bytes.Buffer
	var calls atomic.Int32

	_, _, _, err := authenticateWithRetry(
		context.Background(), reader, &stderr,
		loginAlwaysFails(&calls), pw, "https://api.example.com",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aborted")
	assert.Equal(t, int32(1), calls.Load(), "abort must NOT trigger another login attempt")
}

func TestAuthenticateWithRetry_EmptyEmailRejected(t *testing.T) {
	reader, pw := stubInputs([]string{""}, "secret")
	var stderr bytes.Buffer
	var calls atomic.Int32

	_, _, _, err := authenticateWithRetry(
		context.Background(), reader, &stderr,
		loginAlwaysFails(&calls), pw, "https://api.example.com",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email is required")
	assert.Equal(t, int32(0), calls.Load())
}

func TestAuthenticateWithRetry_EmptyPasswordRejected(t *testing.T) {
	reader, pw := stubInputs([]string{"user@example.com"}, "")
	var stderr bytes.Buffer
	var calls atomic.Int32

	_, _, _, err := authenticateWithRetry(
		context.Background(), reader, &stderr,
		loginAlwaysFails(&calls), pw, "https://api.example.com",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
	assert.Equal(t, int32(0), calls.Load())
}
