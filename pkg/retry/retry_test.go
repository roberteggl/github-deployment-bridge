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
