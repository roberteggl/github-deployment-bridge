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
		name   string
		labels map[string]string
		want   ocilabels.Metadata
	}{
		{
			name: "all labels",
			labels: map[string]string{
				ocilabels.LabelSource:   "https://github.com/example/backend",
				ocilabels.LabelRevision: "0123456789abcdef",
				ocilabels.LabelVersion:  "v1.8.4",
				ocilabels.LabelTitle:    "backend",
				ocilabels.LabelCreated:  "2026-07-25T12:00:00Z",
			},
			want: ocilabels.Metadata{
				Source:   "https://github.com/example/backend",
				Revision: "0123456789abcdef",
				Version:  "v1.8.4",
				Title:    "backend",
				Created:  "2026-07-25T12:00:00Z",
			},
		},
		{
			name: "optional fields omitted",
			labels: map[string]string{
				ocilabels.LabelSource:   "https://github.com/example/backend",
				ocilabels.LabelRevision: "0123456789abcdef",
			},
			want: ocilabels.Metadata{
				Source:   "https://github.com/example/backend",
				Revision: "0123456789abcdef",
			},
		},
		{
			name:   "nil labels",
			labels: nil,
			want:   ocilabels.Metadata{},
		},
		{
			name:   "empty labels",
			labels: map[string]string{},
			want:   ocilabels.Metadata{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ocilabels.Extract(tt.labels)
			if got != tt.want {
				t.Fatalf("Extract() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
