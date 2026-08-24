package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	sdkErrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
)

// Retry policy for transient Aliyun API failures (upstream issue #9: "Will the
// related API be throttled?"). Full spot-price pagination raises per-run API
// call volume, so one 429/503/timeout must not drop an instance type for the
// whole run — the pre-retry behavior.
const maxAPIRetries = 3 // retries after the initial attempt (4 attempts total)

const (
	retryBaseDelay = 1 * time.Second
	maxRetryDelay  = 10 * time.Second
)

// retrySleep is the sleep used between retries; tests replace it to keep the
// suite instant. Package-level on purpose (mirrors the flag globals).
var retrySleep = time.Sleep

// isRetryableError reports whether err is a TRANSIENT Aliyun API failure worth
// retrying: throttling (HTTP 429, or a Throttling*/FlowControl error code —
// which the API can also return with HTTP 400), server-side 5xx, or a request
// timeout — either the SDK-wrapped SDK.TimeoutError ClientError, or (under the
// default AutoRetry=false config) the RAW *url.Error/net.Error the transport
// returns, which is the shape that actually reaches this code in production.
// Non-transient failures (invalid params, auth errors, endpoint
// misconfiguration) return false so they fail fast.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var serverErr *sdkErrors.ServerError
	if errors.As(err, &serverErr) {
		status := serverErr.HttpStatus()
		code := serverErr.ErrorCode()
		return status == 429 || status >= 500 ||
			strings.HasPrefix(code, "Throttling") || code == "FlowControl"
	}
	var clientErr *sdkErrors.ClientError
	if errors.As(err, &clientErr) {
		// Request timeouts are transient; other client errors (endpoint
		// resolution, credentials, marshaling) are not worth retrying.
		return clientErr.ErrorCode() == sdkErrors.TimeoutErrorCode
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

// retryDelay returns the backoff before retry number `attempt` (1-based):
// 1s, 2s, 4s, ... capped at maxRetryDelay.
func retryDelay(attempt int) time.Duration {
	d := retryBaseDelay * time.Duration(1<<(attempt-1))
	if d > maxRetryDelay {
		d = maxRetryDelay
	}
	return d
}

// callWithRetry invokes fn and retries transient failures up to maxAPIRetries
// times with exponential backoff, narrating each retry on stderr (consistent
// with the other fetch warnings; JSON on stdout stays clean). A non-retryable
// error — or exhausted retries — returns the last error immediately.
func callWithRetry[T any](op string, fn func() (T, error)) (T, error) {
	var zero T
	for attempt := 0; ; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		if !isRetryableError(err) || attempt >= maxAPIRetries {
			return zero, err
		}
		delay := retryDelay(attempt + 1)
		fmt.Fprintf(os.Stderr, "Warning: %s hit a transient failure (attempt %d/%d), retrying in %v: %v\n",
			op, attempt+1, maxAPIRetries+1, delay, err)
		retrySleep(delay)
	}
}
