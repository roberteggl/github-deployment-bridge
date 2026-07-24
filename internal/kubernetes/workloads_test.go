// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import "testing"

func TestParseInventoryID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id        string
		namespace string
		name      string
		group     string
		kind      string
		wantErr   bool
	}{
		{
			id:        "default_podinfo_apps_Deployment",
			namespace: "default",
			name:      "podinfo",
			group:     "apps",
			kind:      "Deployment",
		},
		{
			id:        "default_podinfo__Service",
			namespace: "default",
			name:      "podinfo",
			group:     "",
			kind:      "Service",
		},
		{
			id:      "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()
			ns, name, group, kind, err := parseInventoryID(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ns != tt.namespace || name != tt.name || group != tt.group || kind != tt.kind {
				t.Fatalf("got %q %q %q %q", ns, name, group, kind)
			}
		})
	}
}
