// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package giturl parses GitHub repository URLs into owner/repo pairs.
package giturl

import (
	"fmt"
	"net/url"
	"strings"
)

// Repository identifies a GitHub repository.
type Repository struct {
	Owner string
	Name  string
}

// String returns "owner/name".
func (r Repository) String() string {
	return r.Owner + "/" + r.Name
}

// Parse extracts owner and repository name from common GitHub URL forms:
//
//	https://github.com/org/repo
//	https://github.com/org/repo.git
//	git@github.com:org/repo.git
//	git://github.com/org/repo.git
//	ssh://git@github.com/org/repo.git
func Parse(raw string) (Repository, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Repository{}, fmt.Errorf("empty repository URL")
	}

	if strings.HasPrefix(raw, "git@") {
		return parseSCP(raw)
	}

	if !strings.Contains(raw, "://") {
		// Allow bare "owner/repo".
		parts := strings.Split(strings.TrimSuffix(raw, ".git"), "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return Repository{Owner: parts[0], Name: parts[1]}, nil
		}
		return Repository{}, fmt.Errorf("unsupported repository URL: %q", raw)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return Repository{}, fmt.Errorf("parse repository URL: %w", err)
	}

	host := strings.ToLower(u.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return Repository{}, fmt.Errorf("unsupported git host %q (only github.com is supported)", host)
	}

	path := strings.Trim(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return Repository{}, fmt.Errorf("invalid GitHub repository path in %q", raw)
	}

	return Repository{Owner: parts[0], Name: parts[1]}, nil
}

func parseSCP(raw string) (Repository, error) {
	// git@github.com:org/repo.git
	withoutUser := strings.TrimPrefix(raw, "git@")
	host, path, ok := strings.Cut(withoutUser, ":")
	if !ok {
		return Repository{}, fmt.Errorf("invalid SCP-style git URL: %q", raw)
	}
	if strings.ToLower(host) != "github.com" {
		return Repository{}, fmt.Errorf("unsupported git host %q (only github.com is supported)", host)
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return Repository{}, fmt.Errorf("invalid GitHub repository path in %q", raw)
	}
	return Repository{Owner: parts[0], Name: parts[1]}, nil
}
