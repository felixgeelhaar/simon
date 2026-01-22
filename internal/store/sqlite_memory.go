package store

import (
	"bytes"
	"container/heap"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sync"
)

// vectorIndex provides in-memory caching for vector search.
// It uses a min-heap for efficient top-k selection.
type vectorIndex struct {
	mu      sync.RWMutex
	entries []indexEntry
	dirty   bool
}

type indexEntry struct {
	id       int64
	content  string
	vector   []float32
	metadata map[string]string
}

// scoredEntry is used for heap-based top-k selection
type scoredEntry struct {
	entry indexEntry
	score float32
}

// minHeap implements a min-heap for efficient top-k selection
type minHeap []scoredEntry

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].score < h[j].score }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *minHeap) Push(x interface{}) {
	*h = append(*h, x.(scoredEntry))
}

func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// newVectorIndex creates a new vector index
func newVectorIndex() *vectorIndex {
	return &vectorIndex{
		entries: make([]indexEntry, 0),
		dirty:   true,
	}
}

// add adds an entry to the index
func (idx *vectorIndex) add(id int64, content string, vector []float32, metadata map[string]string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.entries = append(idx.entries, indexEntry{
		id:       id,
		content:  content,
		vector:   vector,
		metadata: metadata,
	})
}

// search performs a top-k similarity search using a min-heap
// This is O(n * log(k)) which is more efficient than O(n * log(n)) for k << n
func (idx *vectorIndex) search(queryVector []float32, limit int) []MemoryItem {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if limit <= 0 || len(idx.entries) == 0 {
		return nil
	}

	// Pre-compute query magnitude for efficiency
	queryMag := magnitude(queryVector)
	if queryMag == 0 {
		return nil
	}

	// Use a min-heap to keep track of top-k results
	h := &minHeap{}
	heap.Init(h)

	for _, entry := range idx.entries {
		score := cosineSimilarityOptimized(queryVector, entry.vector, queryMag)

		if h.Len() < limit {
			heap.Push(h, scoredEntry{entry: entry, score: score})
		} else if score > (*h)[0].score {
			// Replace the minimum if current score is higher
			heap.Pop(h)
			heap.Push(h, scoredEntry{entry: entry, score: score})
		}
	}

	// Extract results in descending order
	results := make([]MemoryItem, h.Len())
	for i := h.Len() - 1; i >= 0; i-- {
		se := heap.Pop(h).(scoredEntry)
		results[i] = MemoryItem{
			Content:    se.entry.content,
			Metadata:   se.entry.metadata,
			Similarity: se.score,
		}
	}

	return results
}

// clear removes all entries from the index
func (idx *vectorIndex) clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.entries = make([]indexEntry, 0)
	idx.dirty = true
}

// size returns the number of entries in the index
func (idx *vectorIndex) size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.entries)
}

func (s *SQLiteStore) AddMemory(content string, vector []float32, meta map[string]string) error {
	// Serialize vector
	vecBuf := new(bytes.Buffer)
	if err := binary.Write(vecBuf, binary.LittleEndian, vector); err != nil {
		return fmt.Errorf("failed to encode vector: %w", err)
	}

	// Serialize meta
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `INSERT INTO memories (content, vector, metadata) VALUES (?, ?, ?)`
	result, err := s.db.Exec(query, content, vecBuf.Bytes(), string(metaJSON))
	if err != nil {
		return err
	}

	// Add to in-memory index for fast search
	if s.memoryIndex == nil {
		s.memoryIndex = newVectorIndex()
	}
	id, _ := result.LastInsertId()
	s.memoryIndex.add(id, content, vector, meta)

	return nil
}

func (s *SQLiteStore) SearchMemory(queryVector []float32, limit int) ([]MemoryItem, error) {
	// Try to use the in-memory index first
	if s.memoryIndex != nil && s.memoryIndex.size() > 0 {
		return s.memoryIndex.search(queryVector, limit), nil
	}

	// Fall back to database scan if index not available
	// This also rebuilds the index for future queries
	return s.searchMemoryFromDB(queryVector, limit)
}

// searchMemoryFromDB performs a database scan and rebuilds the in-memory index
func (s *SQLiteStore) searchMemoryFromDB(queryVector []float32, limit int) ([]MemoryItem, error) {
	rows, err := s.db.Query(`SELECT id, content, vector, metadata FROM memories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialize or reset the index
	if s.memoryIndex == nil {
		s.memoryIndex = newVectorIndex()
	} else {
		s.memoryIndex.clear()
	}

	// Pre-compute query magnitude
	queryMag := magnitude(queryVector)
	if queryMag == 0 {
		return nil, nil
	}

	// Use a min-heap for efficient top-k selection
	h := &minHeap{}
	heap.Init(h)

	for rows.Next() {
		var id int64
		var content string
		var vecBlob []byte
		var metaJSON string

		if err := rows.Scan(&id, &content, &vecBlob, &metaJSON); err != nil {
			continue
		}

		// Decode vector
		vector := make([]float32, len(vecBlob)/4)
		if err := binary.Read(bytes.NewReader(vecBlob), binary.LittleEndian, &vector); err != nil {
			continue
		}

		// Decode meta
		var meta map[string]string
		json.Unmarshal([]byte(metaJSON), &meta)

		// Add to index for future queries
		s.memoryIndex.add(id, content, vector, meta)

		// Calculate similarity
		score := cosineSimilarityOptimized(queryVector, vector, queryMag)

		// Maintain top-k using min-heap
		if h.Len() < limit {
			heap.Push(h, scoredEntry{
				entry: indexEntry{id: id, content: content, vector: vector, metadata: meta},
				score: score,
			})
		} else if score > (*h)[0].score {
			heap.Pop(h)
			heap.Push(h, scoredEntry{
				entry: indexEntry{id: id, content: content, vector: vector, metadata: meta},
				score: score,
			})
		}
	}

	// Extract results in descending order
	results := make([]MemoryItem, h.Len())
	for i := h.Len() - 1; i >= 0; i-- {
		se := heap.Pop(h).(scoredEntry)
		results[i] = MemoryItem{
			Content:    se.entry.content,
			Metadata:   se.entry.metadata,
			Similarity: se.score,
		}
	}

	return results, nil
}

// magnitude calculates the magnitude of a vector
func magnitude(v []float32) float32 {
	var sum float32
	for _, x := range v {
		sum += x * x
	}
	return float32(math.Sqrt(float64(sum)))
}

// cosineSimilarityOptimized computes cosine similarity with pre-computed query magnitude
func cosineSimilarityOptimized(a, b []float32, aMag float32) float32 {
	if len(a) != len(b) || len(a) == 0 || aMag == 0 {
		return 0.0
	}

	var dot, bMag float32
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		bMag += b[i] * b[i]
	}

	if bMag == 0 {
		return 0.0
	}

	return dot / (aMag * float32(math.Sqrt(float64(bMag))))
}

// cosineSimilarity computes cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}
	var dot, magA, magB float32
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0.0
	}
	return dot / (float32(math.Sqrt(float64(magA))) * float32(math.Sqrt(float64(magB))))
}
