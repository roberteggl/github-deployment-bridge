// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"strings"
	"testing"
)

func FuzzParseInventoryID(f *testing.F) {
	seeds := []string{
		"default_podinfo_apps_Deployment",
		"default_podinfo__Service",
		"flux-system_bridge_apps_Deployment",
		"invalid",
		"",
		"a_b_c",
		"a_b_c_d_e",
		"ns_name_group_Kind",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, id string) {
		ns, name, group, kind, err := parseInventoryID(id)
		if err != nil {
			return
		}
		parts := strings.Split(id, "_")
		if len(parts) != 4 {
			t.Fatalf("ParseInventoryID(%q) succeeded but Split has %d parts", id, len(parts))
		}
		if ns != parts[0] || name != parts[1] || group != parts[2] || kind != parts[3] {
			t.Fatalf("got %q/%q/%q/%q, want %v", ns, name, group, kind, parts)
		}
	})
}
