package memory

import "context"

// Memory defines the interface for long-term storage and retrieval.
type Memory interface {
	// Add stores a memory item (e.g. a summary of a successful session).
	Add(ctx context.Context, item Item) error

	// Retrieve finds relevant memories based on a query (text).
	Retrieve(ctx context.Context, query string, limit int) ([]Item, error)
}

// Item represents a unit of memory.
type Item struct {
	ID        string
	Content   string
	Metadata  map[string]string
	Vector    []float32 // Embedding vector
	Similarity float32   // Search score
}
