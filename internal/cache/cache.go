// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package cache provides duplicate-prevention storage for reported deployments.
package cache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Status values stored for reported deployments (GitHub Deployment states).
const (
	StatusQueued     = "queued"
	StatusInProgress = "in_progress"
	StatusSuccess    = "success"
	StatusFailure    = "failure"
	StatusError      = "error"
	StatusInactive   = "inactive"
)

// Key uniquely identifies a reported deployment.
type Key struct {
	Owner          string
	Repo           string
	Environment    string
	CommitSHA      string
	DeploymentName string
}

// Identity uniquely identifies a deployment lineage across commits.
type Identity struct {
	Owner          string
	Repo           string
	Environment    string
	DeploymentName string
}

// Entry is a cached deployment report.
type Entry struct {
	Key          Key
	DeploymentID int64
	Status       string
	ReportedAt   time.Time
}

// Store persists deployment reports.
type Store interface {
	Get(ctx context.Context, key Key) (*Entry, error)
	Put(ctx context.Context, entry Entry) error
	ListByIdentity(ctx context.Context, id Identity) ([]Entry, error)
	Close() error
}

// InstallationEntry maps a repository owner to a GitHub App installation.
type InstallationEntry struct {
	Owner          string
	InstallationID int64
	ResolvedAt     time.Time
}

// InstallationStore persists automatically resolved GitHub App installations.
type InstallationStore interface {
	GetInstallation(ctx context.Context, owner string) (*InstallationEntry, error)
	PutInstallation(ctx context.Context, entry InstallationEntry) error
	DeleteInstallation(ctx context.Context, owner string) error
}

// SQLiteStore is a SQLite-backed Store.
type SQLiteStore struct {
	db *sql.DB
}

// Open opens (or creates) a SQLite database at path.
func Open(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS deployments (
	owner TEXT NOT NULL,
	repo TEXT NOT NULL,
	environment TEXT NOT NULL,
	commit_sha TEXT NOT NULL,
	deployment_name TEXT NOT NULL DEFAULT '',
	deployment_id INTEGER NOT NULL,
	status TEXT NOT NULL,
	reported_at TEXT NOT NULL,
	PRIMARY KEY (owner, repo, environment, commit_sha, deployment_name)
);
CREATE TABLE IF NOT EXISTS github_installations (
	owner TEXT NOT NULL PRIMARY KEY COLLATE NOCASE,
	installation_id INTEGER NOT NULL,
	resolved_at TEXT NOT NULL
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate sqlite: %w", err)
	}

	// Upgrade legacy schemas that lack deployment_name in the primary key.
	if err := s.ensureDeploymentNameColumn(); err != nil {
		return err
	}

	// Improve concurrent safety for a single-writer controller.
	if _, err := s.db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return fmt.Errorf("enable wal: %w", err)
	}
	if _, err := s.db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		return fmt.Errorf("set busy_timeout: %w", err)
	}
	return nil
}

// GetInstallation returns the cached installation for owner, or nil when absent.
func (s *SQLiteStore) GetInstallation(ctx context.Context, owner string) (*InstallationEntry, error) {
	var entry InstallationEntry
	var resolvedAt string
	err := s.db.QueryRowContext(ctx, `SELECT owner, installation_id, resolved_at FROM github_installations WHERE owner = ?`, owner).
		Scan(&entry.Owner, &entry.InstallationID, &resolvedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("installation cache get: %w", err)
	}
	entry.ResolvedAt, err = time.Parse(time.RFC3339Nano, resolvedAt)
	if err != nil {
		return nil, fmt.Errorf("parse installation resolved_at: %w", err)
	}
	return &entry, nil
}

// PutInstallation upserts the installation for owner.
func (s *SQLiteStore) PutInstallation(ctx context.Context, entry InstallationEntry) error {
	if entry.ResolvedAt.IsZero() {
		entry.ResolvedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO github_installations (owner, installation_id, resolved_at) VALUES (?, ?, ?)
ON CONFLICT(owner) DO UPDATE SET installation_id = excluded.installation_id, resolved_at = excluded.resolved_at`,
		entry.Owner, entry.InstallationID, entry.ResolvedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("installation cache put: %w", err)
	}
	return nil
}

// DeleteInstallation invalidates the cached installation for owner.
func (s *SQLiteStore) DeleteInstallation(ctx context.Context, owner string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM github_installations WHERE owner = ?`, owner); err != nil {
		return fmt.Errorf("installation cache delete: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ensureDeploymentNameColumn() error {
	rows, err := s.db.Query(`PRAGMA table_info(deployments)`)
	if err != nil {
		return fmt.Errorf("pragma table_info: %w", err)
	}
	defer func() { _ = rows.Close() }()

	hasDeploymentName := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("scan table_info: %w", err)
		}
		if name == "deployment_name" {
			hasDeploymentName = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("table_info rows: %w", err)
	}
	if hasDeploymentName {
		return nil
	}

	// Rebuild table so deployment_name is part of the primary key.
	const rebuild = `
CREATE TABLE deployments_new (
	owner TEXT NOT NULL,
	repo TEXT NOT NULL,
	environment TEXT NOT NULL,
	commit_sha TEXT NOT NULL,
	deployment_name TEXT NOT NULL DEFAULT '',
	deployment_id INTEGER NOT NULL,
	status TEXT NOT NULL,
	reported_at TEXT NOT NULL,
	PRIMARY KEY (owner, repo, environment, commit_sha, deployment_name)
);
INSERT INTO deployments_new (owner, repo, environment, commit_sha, deployment_name, deployment_id, status, reported_at)
SELECT owner, repo, environment, commit_sha, '', deployment_id, status, reported_at FROM deployments;
DROP TABLE deployments;
ALTER TABLE deployments_new RENAME TO deployments;
`
	if _, err := s.db.Exec(rebuild); err != nil {
		return fmt.Errorf("migrate deployment_name: %w", err)
	}
	return nil
}

// Get returns a cached entry, or nil if none exists.
func (s *SQLiteStore) Get(ctx context.Context, key Key) (*Entry, error) {
	const q = `
SELECT owner, repo, environment, commit_sha, deployment_name, deployment_id, status, reported_at
FROM deployments
WHERE owner = ? AND repo = ? AND environment = ? AND commit_sha = ? AND deployment_name = ?
`
	row := s.db.QueryRowContext(ctx, q, key.Owner, key.Repo, key.Environment, key.CommitSHA, key.DeploymentName)
	e, err := scanEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache get: %w", err)
	}
	return e, nil
}

// Put upserts a deployment report.
func (s *SQLiteStore) Put(ctx context.Context, entry Entry) error {
	if entry.ReportedAt.IsZero() {
		entry.ReportedAt = time.Now().UTC()
	}
	const q = `
INSERT INTO deployments (owner, repo, environment, commit_sha, deployment_name, deployment_id, status, reported_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(owner, repo, environment, commit_sha, deployment_name) DO UPDATE SET
	deployment_id = excluded.deployment_id,
	status = excluded.status,
	reported_at = excluded.reported_at
`
	_, err := s.db.ExecContext(ctx, q,
		entry.Key.Owner, entry.Key.Repo, entry.Key.Environment, entry.Key.CommitSHA, entry.Key.DeploymentName,
		entry.DeploymentID, entry.Status, entry.ReportedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("cache put: %w", err)
	}
	return nil
}

// ListByIdentity returns all cached entries for a deployment identity (any commit).
func (s *SQLiteStore) ListByIdentity(ctx context.Context, id Identity) ([]Entry, error) {
	const q = `
SELECT owner, repo, environment, commit_sha, deployment_name, deployment_id, status, reported_at
FROM deployments
WHERE owner = ? AND repo = ? AND environment = ? AND deployment_name = ?
ORDER BY reported_at ASC
`
	rows, err := s.db.QueryContext(ctx, q, id.Owner, id.Repo, id.Environment, id.DeploymentName)
	if err != nil {
		return nil, fmt.Errorf("cache list by identity: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("cache list scan: %w", err)
		}
		out = append(out, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cache list rows: %w", err)
	}
	return out, nil
}

// Close closes the underlying database.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanEntry(row scannable) (*Entry, error) {
	var e Entry
	var reportedAt string
	err := row.Scan(
		&e.Key.Owner, &e.Key.Repo, &e.Key.Environment, &e.Key.CommitSHA, &e.Key.DeploymentName,
		&e.DeploymentID, &e.Status, &reportedAt,
	)
	if err != nil {
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, reportedAt)
	if err != nil {
		t, err = time.Parse(time.RFC3339, reportedAt)
		if err != nil {
			return nil, fmt.Errorf("parse reported_at: %w", err)
		}
	}
	e.ReportedAt = t
	return &e, nil
}

// LatestStatus returns the cached status string, or empty when missing.
func LatestStatus(entry *Entry) string {
	if entry == nil {
		return ""
	}
	return entry.Status
}
