package lyrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type serviceError struct {
	Status     int
	RetryAfter time.Duration
}

func (e *serviceError) Error() string {
	if e.Status == 429 {
		return "Lyrics service request limit reached. Wait a little, then press F5 to retry."
	}
	if e.Status >= 500 {
		return fmt.Sprintf("Lyrics service is temporarily unavailable (HTTP %d). Press F5 to retry.", e.Status)
	}
	return fmt.Sprintf("Lyrics request failed (HTTP %d). Press F5 to retry.", e.Status)
}
func statusIs(err error, n int) bool { var e *serviceError; return errors.As(err, &e) && e.Status == n }
func retryAfter(raw string) time.Duration {
	if n, e := strconv.Atoi(strings.TrimSpace(raw)); e == nil && n >= 0 {
		return time.Duration(min(n, 86400)) * time.Second
	}
	if at, e := http.ParseTime(raw); e == nil {
		return max(time.Duration(0), time.Until(at))
	}
	return 0
}
func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func (c *Client) get(ctx context.Context, path string, out any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for attempt := 0; attempt < 3; attempt++ {
		if delay := time.Until(c.notBefore); delay > 0 {
			if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= delay && c.blockedStatus != 0 {
				return &serviceError{Status: c.blockedStatus, RetryAfter: delay}
			}
			if err := waitContext(ctx, delay); err != nil {
				return err
			}
		}
		if attempt > 0 {
			if err := waitContext(ctx, time.Duration(attempt)*500*time.Millisecond); err != nil {
				return err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "wisp/0.3.3 (terminal music player)")
		req.Header.Set("Accept", "application/json")
		res, err := c.HTTP.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if attempt < 2 {
				continue
			}
			return errors.New("Could not reach the lyrics service. Check your connection, then press F5 to retry.")
		}
		c.notBefore = time.Now().Add(250 * time.Millisecond)
		c.blockedStatus = 0
		if res.StatusCode == http.StatusOK {
			raw, e := io.ReadAll(io.LimitReader(res.Body, (2<<20)+1))
			res.Body.Close()
			if e != nil {
				return e
			}
			if len(raw) > 2<<20 {
				return errors.New("Lyrics response is too large.")
			}
			if e = json.Unmarshal(raw, out); e != nil {
				return errors.New("Lyrics service returned an invalid response. Press F5 to retry.")
			}
			return nil
		}
		failure := &serviceError{Status: res.StatusCode, RetryAfter: retryAfter(res.Header.Get("Retry-After"))}
		res.Body.Close()
		if failure.RetryAfter > 0 {
			c.notBefore = time.Now().Add(failure.RetryAfter)
			c.blockedStatus = failure.Status
		}
		temporary := failure.Status == 500 || failure.Status == 502 || failure.Status == 503 || failure.Status == 504
		if !temporary || attempt == 2 {
			return failure
		}
		if failure.RetryAfter > 0 {
			if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= failure.RetryAfter {
				return failure
			}
			if e := waitContext(ctx, failure.RetryAfter); e != nil {
				return e
			}
		}
	}
	return errors.New("Lyrics service unavailable. Press F5 to retry.")
}
func Message(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "Lyrics service took too long to respond. Press F5 to retry."
	}
	if errors.Is(err, context.Canceled) {
		return "Lyrics lookup cancelled. Press F5 to retry."
	}
	return err.Error()
}
