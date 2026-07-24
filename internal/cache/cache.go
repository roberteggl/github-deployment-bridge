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

// Status values stored for reported deployments.
const (
	StatusSuccess = "success"
)

// Key uniquely identifies a reported deployment.
type Key struct {
	Owner       string
	Repo        string
	Environment string
	CommitSHA   string
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
	Close() error
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
	deployment_id INTEGER NOT NULL,
	status TEXT NOT NULL,
	reported_at TEXT NOT NULL,
	PRIMARY KEY (owner, repo, environment, commit_sha)
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate sqlite: %w", err)
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

// Get returns a cached entry, or nil if none exists.
func (s *SQLiteStore) Get(ctx context.Context, key Key) (*Entry, error) {
	const q = `
SELECT owner, repo, environment, commit_sha, deployment_id, status, reported_at
FROM deployments
WHERE owner = ? AND repo = ? AND environment = ? AND commit_sha = ?
`
	row := s.db.QueryRowContext(ctx, q, key.Owner, key.Repo, key.Environment, key.CommitSHA)
	var e Entry
	var reportedAt string
	err := row.Scan(
		&e.Key.Owner, &e.Key.Repo, &e.Key.Environment, &e.Key.CommitSHA,
		&e.DeploymentID, &e.Status, &reportedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache get: %w", err)
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

// Put upserts a deployment report.
func (s *SQLiteStore) Put(ctx context.Context, entry Entry) error {
	if entry.ReportedAt.IsZero() {
		entry.ReportedAt = time.Now().UTC()
	}
	const q = `
INSERT INTO deployments (owner, repo, environment, commit_sha, deployment_id, status, reported_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(owner, repo, environment, commit_sha) DO UPDATE SET
	deployment_id = excluded.deployment_id,
	status = excluded.status,
	reported_at = excluded.reported_at
`
	_, err := s.db.ExecContext(ctx, q,
		entry.Key.Owner, entry.Key.Repo, entry.Key.Environment, entry.Key.CommitSHA,
		entry.DeploymentID, entry.Status, entry.ReportedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("cache put: %w", err)
	}
	return nil
}

// Close closes the underlying database.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// AlreadyReported returns true when a successful report exists for key.
func AlreadyReported(entry *Entry) bool {
	return entry != nil && entry.Status == StatusSuccess
}
