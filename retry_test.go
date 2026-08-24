package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
	"time"

	sdkErrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
)

// urlTimeoutErr builds a raw transport timeout in the shape the SDK actually
// returns under its default AutoRetry=false config (a *url.Error wrapping a
// net.Error with Timeout()==true) — NOT the SDK.TimeoutError ClientError,
// which the SDK only constructs when AutoRetry is enabled.
func urlTimeoutErr() error {
	return &url.Error{Op: "Post", URL: "https://ecs.example.com", Err: &net.DNSError{Err: "i/o timeout", IsTimeout: true}}
}

// serverErr builds a real *sdkErrors.ServerError the way the SDK does from an
// API response, so isRetryableError is tested against the genuine type.
func serverErr(status int, code string) error {
	return sdkErrors.NewServerError(status, fmt.Sprintf(`{"Code":%q,"Message":"m"}`, code), "")
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"throttling 429", serverErr(429, "Throttling"), true},
		{"throttling user 429", serverErr(429, "Throttling.User"), true},
		{"throttling code on http 400", serverErr(400, "Throttling"), true},
		{"flow control", serverErr(400, "FlowControl"), true},
		{"server 500", serverErr(500, "InternalError"), true},
		{"server 503", serverErr(503, "ServiceUnavailable"), true},
		{"invalid params (real-world case)", serverErr(400, "InvalidParams.EndTime"), false},
		{"auth error", serverErr(403, "InvalidAccessKeyId.NotFound"), false},
		{"not found", serverErr(404, "InvalidInstanceId.NotFound"), false},
		{"sdk timeout", sdkErrors.NewClientError(sdkErrors.TimeoutErrorCode, "timed out", nil), true},
		{"sdk endpoint error", sdkErrors.NewClientError(sdkErrors.CanNotResolveEndpointErrorCode, "no endpoint", nil), false},
		{"raw url.Error timeout (default-config production shape)", urlTimeoutErr(), true},
		{"wrapped url.Error timeout", fmt.Errorf("call ecs: %w", urlTimeoutErr()), true},
		{"context deadline exceeded", context.DeadlineExceeded, true},
		{"url.Error connection refused (not a timeout)", &url.Error{Op: "Post", URL: "https://ecs.example.com", Err: errors.New("connection refused")}, false},
		{"plain error", errors.New("boom"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableError(tt.err); got != tt.want {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestRetryDelay(t *testing.T) {
	for attempt, want := range map[int]time.Duration{
		1: 1 * time.Second,
		2: 2 * time.Second,
		3: 4 * time.Second,
		9: maxRetryDelay, // capped
	} {
		if got := retryDelay(attempt); got != want {
			t.Errorf("retryDelay(%d) = %v, want %v", attempt, got, want)
		}
	}
}

func TestCallWithRetry(t *testing.T) {
	sleeps := make([]time.Duration, 0)
	orig := retrySleep
	retrySleep = func(d time.Duration) { sleeps = append(sleeps, d) }
	defer func() { retrySleep = orig }()

	t.Run("succeeds after transient failures", func(t *testing.T) {
		sleeps = nil
		calls := 0
		v, err := callWithRetry("op", func() (int, error) {
			calls++
			if calls < 3 {
				return 0, serverErr(429, "Throttling")
			}
			return 42, nil
		})
		if err != nil || v != 42 {
			t.Fatalf("got (%v, %v), want (42, nil)", v, err)
		}
		if calls != 3 {
			t.Errorf("calls = %d, want 3 (initial + 2 retries)", calls)
		}
		if len(sleeps) != 2 || sleeps[0] != 1*time.Second || sleeps[1] != 2*time.Second {
			t.Errorf("backoff sleeps = %v, want [1s 2s]", sleeps)
		}
	})

	t.Run("non-retryable fails fast", func(t *testing.T) {
		sleeps = nil
		calls := 0
		_, err := callWithRetry("op", func() (int, error) {
			calls++
			return 0, serverErr(400, "InvalidParams.EndTime")
		})
		if err == nil {
			t.Fatal("expected the error to propagate")
		}
		if calls != 1 || len(sleeps) != 0 {
			t.Errorf("calls = %d, sleeps = %v; want fail-fast (1 call, no sleep)", calls, sleeps)
		}
	})

	t.Run("retries exhausted returns the last error", func(t *testing.T) {
		sleeps = nil
		calls := 0
		wantErr := serverErr(429, "Throttling")
		_, err := callWithRetry("op", func() (int, error) {
			calls++
			return 0, wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected the last error back, got %v", err)
		}
		if calls != maxAPIRetries+1 {
			t.Errorf("calls = %d, want %d (initial + all retries)", calls, maxAPIRetries+1)
		}
	})
}
