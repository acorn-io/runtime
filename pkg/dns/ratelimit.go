package dns

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type rl struct {
	mu      sync.RWMutex
	limited bool
	error   error
}

var (
	authedRateLimit   rl
	unauthedRateLimit rl
)

func setRateLimited(resp *http.Response, limit *rl) (string, error) {
	limit.mu.Lock()
	defer limit.mu.Unlock()

	// The retry-after header gives us a date in UTC that we could use directly, but we don't know if the local time is accurate.
	// So, calculate the duration between the time the server made the response (the Date header) and its retry-after header and
	// use that duration to determine when we can start retrying
	respDate := resp.Header.Get("Date")
	respRetryAfter := resp.Header.Get("Retry-After")
	if respDate == "" || respRetryAfter == "" {
		return "", fmt.Errorf("acornDNS is currently rate limted, but cannot calculate retry time. date header: %v retry-after header: %v",
			respDate, respRetryAfter)
	}
	date, err := time.Parse(time.RFC1123, respDate)
	if err != nil {
		return "", fmt.Errorf("can't parse rate limit date: %w", err)
	}
	retryAfter, err := time.Parse(time.RFC1123, respRetryAfter)
	if err != nil {
		return "", fmt.Errorf("can't parse rate limit retry-after: %w", err)
	}
	duration := retryAfter.Sub(date)

	limit.limited = true
	limit.error = fmt.Errorf("cannot perform DNS request. Rate limited until %v", time.Now().Add(duration).Format(time.UnixDate))

	go time.AfterFunc(duration, func() {
		limit.mu.Lock()
		defer limit.mu.Unlock()
		limit.limited = false
		limit.error = nil
	})

	return limit.error.Error(), nil
}

func checkRateLimit(limit *rl) error {
	limit.mu.RLock()
	defer limit.mu.RUnlock()
	if limit.limited {
		return limit.error
	}
	return nil
}
