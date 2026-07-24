// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package ocilabels_test

import (
	"testing"

	"github.com/roberteggl/github-deployment-bridge/pkg/ocilabels"
)

func TestExtract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		labels  map[string]string
		owner   string
		repo    string
		rev     string
		version string
		wantErr bool
	}{
		{
			name: "valid labels",
			labels: map[string]string{
				ocilabels.LabelSource:   "https://github.com/example/backend",
				ocilabels.LabelRevision: "0123456789abcdef",
				ocilabels.LabelVersion:  "v1.8.4",
			},
			owner:   "example",
			repo:    "backend",
			rev:     "0123456789abcdef",
			version: "v1.8.4",
		},
		{
			name: "missing revision",
			labels: map[string]string{
				ocilabels.LabelSource:  "https://github.com/example/backend",
				ocilabels.LabelVersion: "v1.8.4",
			},
			wantErr: true,
		},
		{
			name:    "nil labels",
			labels:  nil,
			wantErr: true,
		},
		{
			name: "bad source url",
			labels: map[string]string{
				ocilabels.LabelSource:   "https://gitlab.com/example/backend",
				ocilabels.LabelRevision: "abc",
				ocilabels.LabelVersion:  "v1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ocilabels.Extract(tt.labels)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Repo.Owner != tt.owner || got.Repo.Name != tt.repo {
				t.Fatalf("repo = %s, want %s/%s", got.Repo, tt.owner, tt.repo)
			}
			if got.Revision != tt.rev || got.Version != tt.version {
				t.Fatalf("revision/version = %s/%s, want %s/%s", got.Revision, got.Version, tt.rev, tt.version)
			}
		})
	}
}
