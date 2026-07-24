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

		jitter := time.Duration(rand.Float64() * float64(delay) * 0.2)
		wait := delay + jitter
		if wait > cfg.Max {
			wait = cfg.Max
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
