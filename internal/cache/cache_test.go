// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/roberteggl/github-deployment-bridge/internal/cache"
)

func TestDuplicateDetection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := cache.Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	key := cache.Key{
		Owner:       "acme",
		Repo:        "api",
		Environment: "production",
		CommitSHA:   "abc123",
	}

	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("get empty: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil entry, got %#v", got)
	}
	if cache.AlreadyReported(got) {
		t.Fatal("empty cache should not be already reported")
	}

	if err := store.Put(ctx, cache.Entry{
		Key:          key,
		DeploymentID: 42,
		Status:       cache.StatusSuccess,
		ReportedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, err = store.Get(ctx, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !cache.AlreadyReported(got) {
		t.Fatal("expected already reported")
	}
	if got.DeploymentID != 42 {
		t.Fatalf("deployment id = %d, want 42", got.DeploymentID)
	}
}
