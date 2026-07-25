// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package health serves Kubernetes probe endpoints.
package health

import (
	"encoding/json"
	"net/http"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
)

// Checker tracks process readiness.
type Checker struct {
	ready atomic.Bool
}

// New returns a Checker that starts as not ready.
func New() *Checker {
	return &Checker{}
}

// SetReady marks the process as ready or not ready.
func (c *Checker) SetReady(v bool) {
	c.ready.Store(v)
}

// Ready reports current readiness.
func (c *Checker) Ready() bool {
	return c.ready.Load()
}

// Handler returns an HTTP handler exposing /healthz and /readyz.
func (c *Checker) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", c.healthz)
	r.Get("/readyz", c.readyz)
	return r
}

func (c *Checker) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (c *Checker) readyz(w http.ResponseWriter, _ *http.Request) {
	if !c.Ready() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
