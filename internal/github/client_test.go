// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package github_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	ghclient "github.com/roberteggl/github-deployment-bridge/internal/github"
	"github.com/roberteggl/github-deployment-bridge/pkg/retry"
)

func TestAppClientCreateDeploymentAndStatus(t *testing.T) {
	t.Parallel()

	keyPath := generateRSAKey(t)

	var gotDeployment map[string]any
	var gotStatus map[string]any

	mux := http.NewServeMux()
	tokenHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      "v1.install-token",
			"expires_at": "2099-01-01T00:00:00Z",
		})
	}
	mux.HandleFunc("/app/installations/99/access_tokens", tokenHandler)
	mux.HandleFunc("/api/v3/app/installations/99/access_tokens", tokenHandler)

	deployHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&gotDeployment); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 12345})
	}
	mux.HandleFunc("/repos/example/backend/deployments", deployHandler)
	mux.HandleFunc("/api/v3/repos/example/backend/deployments", deployHandler)

	statusHandler := func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotStatus); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "state": "success"})
	}
	mux.HandleFunc("/repos/example/backend/deployments/12345/statuses", statusHandler)
	mux.HandleFunc("/api/v3/repos/example/backend/deployments/12345/statuses", statusHandler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client, err := ghclient.NewAppClient(ghclient.Options{
		AppID:          1,
		InstallationID: 99,
		PrivateKeyPath: keyPath,
		BaseURL:        srv.URL,
		Retry:          retry.Config{MaxAttempts: 1},
		Transport:      http.DefaultTransport,
	})
	if err != nil {
		t.Fatalf("NewAppClient: %v", err)
	}

	ctx := context.Background()
	dep, err := client.CreateDeployment(ctx, ghclient.DeploymentRequest{
		Owner:                 "example",
		Repo:                  "backend",
		Ref:                   "abc123",
		Environment:           "production",
		ProductionEnvironment: true,
		Description:           "Deployed by FluxCD",
		Payload: map[string]any{
			"cluster":        "prod",
			"kustomization":  "backend",
			"deploymentName": "backend",
		},
	})
	if err != nil {
		t.Fatalf("CreateDeployment: %v", err)
	}
	if dep.ID != 12345 {
		t.Fatalf("deployment id = %d, want 12345", dep.ID)
	}

	if gotDeployment["ref"] != "abc123" {
		t.Fatalf("ref = %#v", gotDeployment["ref"])
	}
	if gotDeployment["auto_merge"] != false {
		t.Fatalf("auto_merge = %#v", gotDeployment["auto_merge"])
	}
	contexts, ok := gotDeployment["required_contexts"].([]any)
	if !ok || len(contexts) != 0 {
		t.Fatalf("required_contexts = %#v, want []", gotDeployment["required_contexts"])
	}
	if gotDeployment["production_environment"] != true {
		t.Fatalf("production_environment = %#v", gotDeployment["production_environment"])
	}
	payload, ok := gotDeployment["payload"].(map[string]any)
	if !ok || payload["cluster"] != "prod" {
		t.Fatalf("payload = %#v", gotDeployment["payload"])
	}

	if err := client.CreateDeploymentStatus(ctx, ghclient.DeploymentStatusRequest{
		Owner:        "example",
		Repo:         "backend",
		DeploymentID: dep.ID,
		State:        "success",
		Description:  "Deployment completed successfully.",
		AutoInactive: true,
	}); err != nil {
		t.Fatalf("CreateDeploymentStatus: %v", err)
	}
	if gotStatus["state"] != "success" {
		t.Fatalf("status state = %#v", gotStatus["state"])
	}
	if gotStatus["auto_inactive"] != true {
		t.Fatalf("auto_inactive = %#v", gotStatus["auto_inactive"])
	}
}


func TestAppClientFindDeployment(t *testing.T) {
	t.Parallel()

	keyPath := generateRSAKey(t)

	mux := http.NewServeMux()
	tokenHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      "v1.install-token",
			"expires_at": "2099-01-01T00:00:00Z",
		})
	}
	mux.HandleFunc("/app/installations/99/access_tokens", tokenHandler)
	mux.HandleFunc("/api/v3/app/installations/99/access_tokens", tokenHandler)

	listHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Query().Get("ref") != "abc123" || r.URL.Query().Get("environment") != "production" {
			http.Error(w, "query", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"id":          777,
			"ref":         "abc123",
			"environment": "production",
			"payload": map[string]any{
				"cluster":        "prod",
				"kustomization":  "backend",
				"deploymentName": "backend",
			},
		}})
	}
	mux.HandleFunc("/repos/example/backend/deployments", listHandler)
	mux.HandleFunc("/api/v3/repos/example/backend/deployments", listHandler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client, err := ghclient.NewAppClient(ghclient.Options{
		AppID:          1,
		InstallationID: 99,
		PrivateKeyPath: keyPath,
		BaseURL:        srv.URL,
		Retry:          retry.Config{MaxAttempts: 1},
		Transport:      http.DefaultTransport,
	})
	if err != nil {
		t.Fatalf("NewAppClient: %v", err)
	}

	found, err := client.FindDeployment(context.Background(), ghclient.FindDeploymentRequest{
		Owner:       "example",
		Repo:        "backend",
		Ref:         "abc123",
		Environment: "production",
		Payload: map[string]any{
			"cluster":        "prod",
			"kustomization":  "backend",
			"deploymentName": "backend",
		},
	})
	if err != nil {
		t.Fatalf("FindDeployment: %v", err)
	}
	if found == nil || found.ID != 777 {
		t.Fatalf("found = %#v, want id 777", found)
	}
}

func generateRSAKey(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "key.pem")
	cmd := exec.Command("openssl", "genrsa", "-out", path, "2048")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("openssl genrsa: %v\n%s", err, out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("key missing: %v", err)
	}
	return path
}
