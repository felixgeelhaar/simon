package memory

import (
	"testing"
)

func TestItem_Fields(t *testing.T) {
	item := Item{
		ID:         "test-id",
		Content:    "test content",
		Metadata:   map[string]string{"key": "value"},
		Vector:     []float32{0.1, 0.2, 0.3},
		Similarity: 0.95,
	}

	if item.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", item.ID)
	}
	if item.Content != "test content" {
		t.Errorf("expected Content 'test content', got %q", item.Content)
	}
	if item.Metadata["key"] != "value" {
		t.Errorf("expected Metadata['key'] = 'value', got %q", item.Metadata["key"])
	}
	if len(item.Vector) != 3 {
		t.Errorf("expected Vector length 3, got %d", len(item.Vector))
	}
	if item.Similarity != 0.95 {
		t.Errorf("expected Similarity 0.95, got %f", item.Similarity)
	}
}

func TestItem_EmptyMetadata(t *testing.T) {
	item := Item{
		ID:      "test-id",
		Content: "test",
	}

	// Metadata should be nil by default
	if item.Metadata != nil {
		t.Errorf("expected nil Metadata, got %v", item.Metadata)
	}
}

func TestItem_ZeroVector(t *testing.T) {
	item := Item{
		ID:      "test-id",
		Content: "test",
	}

	// Vector should be nil by default
	if item.Vector != nil {
		t.Errorf("expected nil Vector, got %v", item.Vector)
	}
}
