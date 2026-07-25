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
		"backend",
		"2026-01-01T00:00:00Z",
	)
	f.Add("acme/widgets", "abc", "1.0.0", "", "")
	f.Add("", "", "", "", "")

	f.Fuzz(func(t *testing.T, source, revision, version, title, created string) {
		labels := map[string]string{
			ocilabels.LabelSource:   source,
			ocilabels.LabelRevision: revision,
			ocilabels.LabelVersion:  version,
			ocilabels.LabelTitle:    title,
			ocilabels.LabelCreated:  created,
		}

		meta := ocilabels.Extract(labels)
		if meta.Source != source || meta.Revision != revision || meta.Version != version {
			t.Fatalf("required fields not preserved: got %#v", meta)
		}
		if meta.Title != title || meta.Created != created {
			t.Fatalf("optional fields not preserved: got %#v", meta)
		}
	})
}
