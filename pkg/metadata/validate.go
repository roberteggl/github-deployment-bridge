// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidGitSHA reports whether s looks like a Git commit SHA (7–40 hex chars).
func ValidGitSHA(s string) bool {
	n := len(s)
	if n < 7 || n > 40 {
		return false
	}
	for i := 0; i < n; i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// ValidHTTPSURL reports whether s is an absolute https:// URL with a host.
func ValidHTTPSURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return false
	}
	return u.Host != ""
}

// ParseBoolAnnotation parses "true"/"false" (case-insensitive). Empty is ok=false.
func ParseBoolAnnotation(raw string) (value bool, ok bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, false, nil
	}
	switch strings.ToLower(raw) {
	case "true":
		return true, true, nil
	case "false":
		return false, true, nil
	default:
		return false, false, fmt.Errorf("must be true or false, got %q", raw)
	}
}
