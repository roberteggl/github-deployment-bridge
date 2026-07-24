// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package integration_test

import (
	"testing"
)

// Integration tests are intentionally gated behind the integration build tag.
// They exercise a kind cluster with Flux, a local registry, and a fake GitHub API.
//
// Run with:
//
//	go test ./test/integration -tags=integration -count=1
func TestIntegrationHarnessPlaceholder(t *testing.T) {
	t.Skip("integration harness not configured in this environment")
}
