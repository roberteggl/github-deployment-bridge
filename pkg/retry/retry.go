// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package retry provides context-aware exponential backoff helpers.
package retry

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

// Hard cap so a malicious or mistaken Retry-After cannot block a reconcile for hours.
const retryAfterCap = 5 * time.Minute

// Config controls retry behaviour.
type Config struct {
	MaxAttempts int
	Initial     time.Duration
	Max         time.Duration
	Multiplier  float64
}

// Default returns a sensible production retry configuration.
func Default() Config {
	return Config{
		MaxAttempts: 5,
		Initial:     500 * time.Millisecond,
		Max:         30 * time.Second,
		Multiplier:  2.0,
	}
}

// Do executes fn until it succeeds or attempts are exhausted.
// Permanent errors should be returned wrapped with Permanent so retries stop.
// Transient errors may be wrapped with After to honor server-provided wait times
// (for example GitHub Retry-After); that delay overrides exponential backoff up
// to retryAfterCap and is not limited by Config.Max.
func Do(ctx context.Context, cfg Config, fn func(context.Context) error) error {
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	if cfg.Initial <= 0 {
		cfg.Initial = 200 * time.Millisecond
	}
	if cfg.Max <= 0 {
		cfg.Max = 30 * time.Second
	}
	if cfg.Multiplier < 1 {
		cfg.Multiplier = 2
	}

	var lastErr error
	delay := cfg.Initial
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}
		if IsPermanent(lastErr) {
			return lastErr
		}
		if attempt == cfg.MaxAttempts {
			break
		}

		wait := backoffWait(delay, cfg.Max)
		if suggested, ok := SuggestedDelay(lastErr); ok {
			wait = suggestedWait(suggested)
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		next := time.Duration(float64(delay) * cfg.Multiplier)
		if next > cfg.Max {
			delay = cfg.Max
		} else {
			delay = next
		}
	}
	return fmt.Errorf("retry exhausted after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

func backoffWait(delay, max time.Duration) time.Duration {
	jitter := time.Duration(rand.Float64() * float64(delay) * 0.2)
	wait := delay + jitter
	if wait > max {
		wait = max
	}
	return wait
}

func suggestedWait(suggested time.Duration) time.Duration {
	if suggested < 0 {
		suggested = 0
	}
	if suggested > retryAfterCap {
		suggested = retryAfterCap
	}
	// Small jitter so concurrent reconcilers do not stampede after a shared reset.
	jitter := time.Duration(rand.Float64() * float64(suggested) * 0.05)
	return suggested + jitter
}

// permanentError marks an error as non-retryable.
type permanentError struct {
	err error
}

func (e *permanentError) Error() string { return e.err.Error() }
func (e *permanentError) Unwrap() error { return e.err }

// Permanent wraps err so Do will not retry it.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

// IsPermanent reports whether err (or any wrapped error) is permanent.
func IsPermanent(err error) bool {
	var p *permanentError
	return errors.As(err, &p)
}

// delayError carries a server-suggested wait before the next attempt.
type delayError struct {
	err   error
	after time.Duration
}

func (e *delayError) Error() string { return e.err.Error() }
func (e *delayError) Unwrap() error { return e.err }

// After wraps err with a suggested delay before the next retry (for example
// from an HTTP Retry-After header). Config.Max does not cap this delay; a
// package safety cap of 5 minutes applies instead.
func After(err error, d time.Duration) error {
	if err == nil {
		return nil
	}
	if d < 0 {
		d = 0
	}
	return &delayError{err: err, after: d}
}

// SuggestedDelay reports a server-suggested wait wrapped with After.
func SuggestedDelay(err error) (time.Duration, bool) {
	var d *delayError
	if errors.As(err, &d) {
		return d.after, true
	}
	return 0, false
}
