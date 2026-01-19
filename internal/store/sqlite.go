package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db          *sql.DB
	artifactDir string
}

func NewSQLiteStore(dbPath, artifactDir string) (*SQLiteStore, error) {
	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0750); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}
	if err := os.MkdirAll(artifactDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create artifact directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{
		db:          db,
		artifactDir: artifactDir,
	}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			status TEXT,
			metadata TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS artifacts (
			id TEXT PRIMARY KEY,
			session_id TEXT,
			path TEXT,
			type TEXT,
			created_at DATETIME,
			digest TEXT,
			FOREIGN KEY(session_id) REFERENCES sessions(id)
		);`,
		`CREATE TABLE IF NOT EXISTS configuration (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS memories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT,
			vector BLOB,
			metadata TEXT
		);`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to init schema: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Configuration Implementation

func (s *SQLiteStore) SetConfig(key, value string) error {
	query := `INSERT INTO configuration (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	_, err := s.db.Exec(query, key, value)
	return err
}

func (s *SQLiteStore) GetConfig(key string) (string, error) {
	query := `SELECT value FROM configuration WHERE key = ?`
	row := s.db.QueryRow(query, key)
	var value string
	if err := row.Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return "", nil // Return empty string if not found, or error? Let's return empty for now.
		}
		return "", err
	}
	return value, nil
}

// Session Implementation

func (s *SQLiteStore) CreateSession(session *Session) error {
	metaJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `INSERT INTO sessions (id, created_at, updated_at, status, metadata) VALUES (?, ?, ?, ?, ?)`
	_, err = s.db.Exec(query, session.ID, session.CreatedAt, session.UpdatedAt, session.Status, string(metaJSON))
	return err
}

func (s *SQLiteStore) GetSession(id string) (*Session, error) {
	query := `SELECT id, created_at, updated_at, status, metadata FROM sessions WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var session Session
	var metaJSON string
	if err := row.Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt, &session.Status, &metaJSON); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(metaJSON), &session.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &session, nil
}

func (s *SQLiteStore) UpdateSession(session *Session) error {
	metaJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `UPDATE sessions SET updated_at = ?, status = ?, metadata = ? WHERE id = ?`
	_, err = s.db.Exec(query, time.Now(), session.Status, string(metaJSON), session.ID)
	return err
}

// Artifact Implementation

func (s *SQLiteStore) SaveArtifact(artifact *Artifact, content []byte) error {
	// 1. Save content to filesystem
	fullPath := filepath.Join(s.artifactDir, artifact.Path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
		return fmt.Errorf("failed to create artifact dir: %w", err)
	}
	if err := os.WriteFile(fullPath, content, 0600); err != nil {
		return fmt.Errorf("failed to write artifact content: %w", err)
	}

	// 2. Save metadata to DB
	query := `INSERT INTO artifacts (id, session_id, path, type, created_at, digest) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, artifact.ID, artifact.SessionID, artifact.Path, artifact.Type, artifact.CreatedAt, artifact.Digest)
	return err
}

func (s *SQLiteStore) GetArtifact(id string) (*Artifact, []byte, error) {
	// 1. Get metadata
	query := `SELECT id, session_id, path, type, created_at, digest FROM artifacts WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var artifact Artifact
	if err := row.Scan(&artifact.ID, &artifact.SessionID, &artifact.Path, &artifact.Type, &artifact.CreatedAt, &artifact.Digest); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("artifact not found: %s", id)
		}
		return nil, nil, err
	}

	// 2. Get content
	fullPath := filepath.Join(s.artifactDir, artifact.Path)
	content, err := os.ReadFile(fullPath) // #nosec G304
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read artifact content: %w", err)
	}

	return &artifact, content, nil
}

func (s *SQLiteStore) ListArtifacts(sessionID string) ([]*Artifact, error) {
	query := `SELECT id, session_id, path, type, created_at, digest FROM artifacts WHERE session_id = ?`
	rows, err := s.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []*Artifact
	for rows.Next() {
		var a Artifact
		if err := rows.Scan(&a.ID, &a.SessionID, &a.Path, &a.Type, &a.CreatedAt, &a.Digest); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, &a)
	}
	return artifacts, nil
}