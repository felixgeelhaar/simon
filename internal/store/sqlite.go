package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // Pure-Go SQLite driver (no CGO required)
)

type SQLiteStore struct {
	db          *sql.DB
	artifactDir string
	memoryIndex *vectorIndex // In-memory index for fast vector search
}

func NewSQLiteStore(dbPath, artifactDir string) (*SQLiteStore, error) {
	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0750); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}
	if err := os.MkdirAll(artifactDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Configure SQLite with connection string options for better performance and safety
	// - _journal_mode=WAL: Write-Ahead Logging for better concurrency
	// - _busy_timeout=5000: Wait up to 5 seconds when database is locked
	// - _synchronous=NORMAL: Good balance between safety and performance
	// - _cache_size=-64000: Use 64MB of cache
	// - _foreign_keys=ON: Enforce foreign key constraints
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=-64000&_foreign_keys=ON", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool settings
	// SQLite works best with a single writer, but can handle multiple readers
	db.SetMaxOpenConns(1)                  // Single connection for writes (SQLite limitation)
	db.SetMaxIdleConns(1)                  // Keep connection alive
	db.SetConnMaxLifetime(time.Hour)       // Recycle connections hourly
	db.SetConnMaxIdleTime(30 * time.Minute) // Close idle connections after 30 minutes

	// Verify connection is working
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
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

// sanitizeArtifactPath validates and sanitizes an artifact path to prevent path traversal attacks.
// It ensures the resulting path is within the artifact directory.
func (s *SQLiteStore) sanitizeArtifactPath(path string) (string, error) {
	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(path)

	// Reject paths that start with / (absolute paths)
	if filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("absolute paths not allowed: %s", path)
	}

	// Reject paths that try to escape the artifact directory
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "/../") {
		return "", fmt.Errorf("path traversal not allowed: %s", path)
	}

	// Construct the full path and verify it's within the artifact directory
	fullPath := filepath.Join(s.artifactDir, cleanPath)
	absArtifactDir, err := filepath.Abs(s.artifactDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve artifact directory: %w", err)
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve artifact path: %w", err)
	}

	// Ensure the resolved path is within the artifact directory
	if !strings.HasPrefix(absFullPath, absArtifactDir+string(filepath.Separator)) && absFullPath != absArtifactDir {
		return "", fmt.Errorf("path escapes artifact directory: %s", path)
	}

	return fullPath, nil
}

func (s *SQLiteStore) SaveArtifact(artifact *Artifact, content []byte) error {
	// 1. Sanitize and validate the artifact path
	fullPath, err := s.sanitizeArtifactPath(artifact.Path)
	if err != nil {
		return fmt.Errorf("invalid artifact path: %w", err)
	}

	// 2. Save content to filesystem
	if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
		return fmt.Errorf("failed to create artifact dir: %w", err)
	}
	if err := os.WriteFile(fullPath, content, 0600); err != nil {
		return fmt.Errorf("failed to write artifact content: %w", err)
	}

	// 3. Save metadata to DB
	query := `INSERT INTO artifacts (id, session_id, path, type, created_at, digest) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = s.db.Exec(query, artifact.ID, artifact.SessionID, artifact.Path, artifact.Type, artifact.CreatedAt, artifact.Digest)
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

	// 2. Sanitize and validate the artifact path from database
	fullPath, err := s.sanitizeArtifactPath(artifact.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid artifact path in database: %w", err)
	}

	// 3. Get content
	content, err := os.ReadFile(fullPath)
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