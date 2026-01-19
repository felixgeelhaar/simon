package store

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

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
	_, err = s.db.Exec(query, content, vecBuf.Bytes(), string(metaJSON))
	return err
}

func (s *SQLiteStore) SearchMemory(queryVector []float32, limit int) ([]MemoryItem, error) {
	// Naive implementation: Load all, compute cosine, sort.
	// OK for local V1 (<10k memories).

	rows, err := s.db.Query(`SELECT content, vector, metadata FROM memories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type scoredItem struct {
		item  MemoryItem
		score float32
	}
	var scored []scoredItem

	for rows.Next() {
		var content string
		var vecBlob []byte
		var metaJSON string

		if err := rows.Scan(&content, &vecBlob, &metaJSON); err != nil {
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

		score := cosineSimilarity(queryVector, vector)
		
		scored = append(scored, scoredItem{
			item: MemoryItem{
				Content:    content,
				Metadata:   meta,
				Similarity: score,
			},
			score: score,
		})
	}

	// Sort desc
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Limit
	if len(scored) > limit {
		scored = scored[:limit]
	}

	result := make([]MemoryItem, len(scored))
	for i, s := range scored {
		result[i] = s.item
	}

	return result, nil
}

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
