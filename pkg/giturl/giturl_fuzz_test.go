// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package giturl_test

import (
	"testing"

	"github.com/roberteggl/github-deployment-bridge/pkg/giturl"
)

func FuzzParse(f *testing.F) {
	seeds := []string{
		"https://github.com/example/backend",
		"https://github.com/example/backend.git",
		"git@github.com:org/repo.git",
		"ssh://git@github.com/org/repo.git",
		"git://github.com/org/repo.git",
		"acme/widgets",
		"",
		"https://gitlab.com/org/repo",
		"https://github.com/org",
		"git@github.com:org",
		"  https://www.github.com/a/b  ",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		repo, err := giturl.Parse(raw)
		if err != nil {
			return
		}
		if repo.Owner == "" || repo.Name == "" {
			t.Fatalf("Parse(%q) succeeded with empty owner/name: %#v", raw, repo)
		}
		if got := repo.String(); got != repo.Owner+"/"+repo.Name {
			t.Fatalf("String() = %q, want %s/%s", got, repo.Owner, repo.Name)
		}
	})
}
