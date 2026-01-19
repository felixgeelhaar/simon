package store

import "time"

// Session represents a unique execution run
type Session struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Status    string
	Metadata  map[string]string // simplified for now
}

// Artifact represents a file or data blob generated during execution
type Artifact struct {
	ID        string
	SessionID string
	Path      string    // Relative path in the artifact store
	Type      string    // e.g., "tool_output", "log", "summary"
	CreatedAt time.Time
	Digest    string    // Content hash
}

// Storage defines the interface for persistence
type Storage interface {
	// Session Management
	CreateSession(session *Session) error
	GetSession(id string) (*Session, error)
	UpdateSession(session *Session) error

	// Artifact Management
	// SaveArtifact persists the metadata and the content
	SaveArtifact(artifact *Artifact, content []byte) error
	GetArtifact(id string) (*Artifact, []byte, error)
	ListArtifacts(sessionID string) ([]*Artifact, error)

	// Configuration Management
	SetConfig(key, value string) error
	GetConfig(key string) (string, error)

	// Memory Management
	AddMemory(content string, vector []float32, meta map[string]string) error
	SearchMemory(vector []float32, limit int) ([]MemoryItem, error)

	Close() error
}

type MemoryItem struct {
	Content    string
	Metadata   map[string]string
	Similarity float32
}
