// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package github_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/roberteggl/github-deployment-bridge/internal/cache"
	ghclient "github.com/roberteggl/github-deployment-bridge/internal/github"
	"github.com/roberteggl/github-deployment-bridge/pkg/retry"
)

func TestAutomaticInstallationResolutionAndSQLiteCache(t *testing.T) {
	t.Parallel()
	keyPath := generateRSAKey(t)
	store, err := cache.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	var lists atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/app/installations", func(w http.ResponseWriter, _ *http.Request) {
		lists.Add(1)
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 99, "account": map[string]any{"login": "Acme"}}})
	})
	mux.HandleFunc("/api/v3/app/installations/99/access_tokens", installationToken)
	mux.HandleFunc("/api/v3/repos/acme/api/deployments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 7})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client, err := ghclient.NewAppClient(ghclient.Options{AppID: 1, PrivateKeyPath: keyPath, BaseURL: srv.URL,
		InstallationCache: store, InstallationCacheTTL: time.Hour, Retry: retry.Config{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := client.CreateDeployment(ctx, ghclient.DeploymentRequest{Owner: "acme", Repo: "api", Ref: "abc", Environment: "prod"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.FindDeployment(ctx, ghclient.FindDeploymentRequest{Owner: "acme", Repo: "api", Ref: "abc", Environment: "prod"}); err != nil {
		t.Fatal(err)
	}
	if got := lists.Load(); got != 1 {
		t.Fatalf("installation lists = %d, want 1", got)
	}
	entry, err := store.GetInstallation(ctx, "acme")
	if err != nil || entry == nil || entry.InstallationID != 99 {
		t.Fatalf("cached entry = %#v, err = %v", entry, err)
	}
}

func TestInstallationFailureInvalidatesAndResolvesOnce(t *testing.T) {
	t.Parallel()
	keyPath := generateRSAKey(t)
	store, err := cache.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	_ = store.PutInstallation(context.Background(), cache.InstallationEntry{Owner: "acme", InstallationID: 99, ResolvedAt: time.Now()})

	var lists atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/app/installations", func(w http.ResponseWriter, _ *http.Request) {
		lists.Add(1)
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 100, "account": map[string]any{"login": "acme"}}})
	})
	mux.HandleFunc("/api/v3/app/installations/99/access_tokens", installationToken)
	mux.HandleFunc("/api/v3/app/installations/100/access_tokens", installationToken)
	var calls atomic.Int32
	mux.HandleFunc("/api/v3/repos/acme/api/deployments", func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "stale installation", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 8})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client, err := ghclient.NewAppClient(ghclient.Options{AppID: 1, PrivateKeyPath: keyPath, BaseURL: srv.URL,
		InstallationCache: store, InstallationCacheTTL: time.Hour, Retry: retry.Config{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.CreateDeployment(context.Background(), ghclient.DeploymentRequest{Owner: "acme", Repo: "api", Ref: "abc", Environment: "prod"})
	if err != nil || result.ID != 8 {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	if calls.Load() != 2 || lists.Load() != 1 {
		t.Fatalf("API calls = %d, lists = %d", calls.Load(), lists.Load())
	}
}

func installationToken(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{"token": "token", "expires_at": "2099-01-01T00:00:00Z"})
}
