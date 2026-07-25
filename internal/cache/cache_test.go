// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

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
		Owner:          "acme",
		Repo:           "api",
		Environment:    "production",
		CommitSHA:      "abc123",
		DeploymentName: "api",
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

	// Distinct deployment-name is an independent cache entry.
	other := key
	other.DeploymentName = "worker"
	got, err = store.Get(ctx, other)
	if err != nil {
		t.Fatalf("get other: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for different deployment-name, got %#v", got)
	}
}

func TestLegacySchemaMigration(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, err = db.Exec(`
CREATE TABLE deployments (
	owner TEXT NOT NULL,
	repo TEXT NOT NULL,
	environment TEXT NOT NULL,
	commit_sha TEXT NOT NULL,
	deployment_id INTEGER NOT NULL,
	status TEXT NOT NULL,
	reported_at TEXT NOT NULL,
	PRIMARY KEY (owner, repo, environment, commit_sha)
);
INSERT INTO deployments VALUES ('acme','api','production','abc123',7,'success','2026-01-01T00:00:00Z');
`)
	if err != nil {
		t.Fatalf("seed legacy: %v", err)
	}
	_ = db.Close()

	store, err := cache.Open(path)
	if err != nil {
		t.Fatalf("open migrated: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	got, err := store.Get(context.Background(), cache.Key{
		Owner:          "acme",
		Repo:           "api",
		Environment:    "production",
		CommitSHA:      "abc123",
		DeploymentName: "",
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !cache.AlreadyReported(got) || got.DeploymentID != 7 {
		t.Fatalf("migrated entry = %#v", got)
	}
}
