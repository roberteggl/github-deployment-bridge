// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/roberteggl/github-deployment-bridge/pkg/retry"
)

func TestDoSucceedsAfterRetries(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := retry.Do(context.Background(), retry.Config{
		MaxAttempts: 3,
		Initial:     time.Millisecond,
		Max:         5 * time.Millisecond,
		Multiplier:  2,
	}, func(context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestDoPermanentStops(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := retry.Do(context.Background(), retry.Config{
		MaxAttempts: 5,
		Initial:     time.Millisecond,
		Max:         time.Millisecond,
		Multiplier:  2,
	}, func(context.Context) error {
		attempts++
		return retry.Permanent(errors.New("nope"))
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
	if !retry.IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestDoHonorsSuggestedDelay(t *testing.T) {
	t.Parallel()

	attempts := 0
	start := time.Now()
	err := retry.Do(context.Background(), retry.Config{
		MaxAttempts: 2,
		Initial:     time.Millisecond,
		Max:         time.Millisecond,
		Multiplier:  2,
	}, func(context.Context) error {
		attempts++
		if attempts == 1 {
			return retry.After(errors.New("rate limited"), 80*time.Millisecond)
		}
		return nil
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	// Exponential backoff would be ~1ms; Retry-After should dominate.
	if elapsed < 70*time.Millisecond {
		t.Fatalf("elapsed = %v, want >= 70ms (Retry-After honored)", elapsed)
	}
}

func TestSuggestedDelay(t *testing.T) {
	t.Parallel()

	err := retry.After(errors.New("wait"), 2*time.Second)
	d, ok := retry.SuggestedDelay(err)
	if !ok || d != 2*time.Second {
		t.Fatalf("SuggestedDelay = (%v, %v), want (2s, true)", d, ok)
	}
	if _, ok := retry.SuggestedDelay(errors.New("plain")); ok {
		t.Fatal("expected no suggested delay for plain error")
	}
}
