// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package giturl_test

import (
	"testing"

	"github.com/roberteggl/github-deployment-bridge/pkg/giturl"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		owner   string
		repo    string
		wantErr bool
	}{
		{
			name:  "https",
			in:    "https://github.com/example/backend",
			owner: "example",
			repo:  "backend",
		},
		{
			name:  "https with git suffix",
			in:    "https://github.com/example/backend.git",
			owner: "example",
			repo:  "backend",
		},
		{
			name:  "scp style",
			in:    "git@github.com:org/repo.git",
			owner: "org",
			repo:  "repo",
		},
		{
			name:  "ssh url",
			in:    "ssh://git@github.com/org/repo.git",
			owner: "org",
			repo:  "repo",
		},
		{
			name:  "bare owner/repo",
			in:    "acme/widgets",
			owner: "acme",
			repo:  "widgets",
		},
		{
			name:    "empty",
			in:      "",
			wantErr: true,
		},
		{
			name:    "unsupported host",
			in:      "https://gitlab.com/org/repo",
			wantErr: true,
		},
		{
			name:    "missing repo",
			in:      "https://github.com/org",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := giturl.Parse(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Owner != tt.owner || got.Name != tt.repo {
				t.Fatalf("got %s/%s, want %s/%s", got.Owner, got.Name, tt.owner, tt.repo)
			}
		})
	}
}
