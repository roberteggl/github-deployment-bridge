// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package ocilabels_test

import (
	"testing"

	"github.com/roberteggl/github-deployment-bridge/pkg/ocilabels"
)

func FuzzExtract(f *testing.F) {
	f.Add(
		"https://github.com/example/backend",
		"0123456789abcdef",
		"v1.8.4",
	)
	f.Add("acme/widgets", "abc", "1.0.0")
	f.Add("https://gitlab.com/example/backend", "abc", "v1")
	f.Add("", "", "")
	f.Add("git@github.com:org/repo.git", "deadbeef", "v0.1.0")

	f.Fuzz(func(t *testing.T, source, revision, version string) {
		labels := map[string]string{
			ocilabels.LabelSource:   source,
			ocilabels.LabelRevision: revision,
			ocilabels.LabelVersion:  version,
		}

		meta, err := ocilabels.Extract(labels)
		if err != nil {
			return
		}
		if meta.Source != source || meta.Revision != revision || meta.Version != version {
			t.Fatalf("labels not preserved: got %#v", meta)
		}
		if meta.Repo.Owner == "" || meta.Repo.Name == "" {
			t.Fatalf("Extract succeeded with empty repo: %#v", meta)
		}
	})
}
