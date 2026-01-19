package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStore(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "store-test-*")
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "meta.db")
	artDir := filepath.Join(tmpDir, "artifacts")

	s, err := NewSQLiteStore(dbPath, artDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	t.Run("Sessions", func(t *testing.T) {
		sess := &Session{
			ID:        "s1",
			CreatedAt: time.Now(),
			Status:    "active",
			Metadata:  map[string]string{"key": "val"},
		}

		if err := s.CreateSession(sess); err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}

		got, err := s.GetSession("s1")
		if err != nil {
			t.Fatalf("GetSession failed: %v", err)
		}
		if got.Metadata["key"] != "val" {
			t.Errorf("Expected metadata 'val', got '%s'", got.Metadata["key"])
		}

		got.Status = "completed"
		if err := s.UpdateSession(got); err != nil {
			t.Fatalf("UpdateSession failed: %v", err)
		}

		updated, _ := s.GetSession("s1")
		if updated.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", updated.Status)
		}

		if _, err := s.GetSession("non-existent"); err == nil {
			t.Error("Expected error for non-existent session")
		}
	})

	t.Run("Artifacts", func(t *testing.T) {
		art := &Artifact{
			ID:        "a1",
			SessionID: "s1",
			Path:      "test.txt",
			Type:      "log",
			CreatedAt: time.Now(),
			Digest:    "d1",
		}
		content := []byte("hello artifact")

		if err := s.SaveArtifact(art, content); err != nil {
			t.Fatalf("SaveArtifact failed: %v", err)
		}

		gotArt, gotContent, err := s.GetArtifact("a1")
		if err != nil {
			t.Fatalf("GetArtifact failed: %v", err)
		}
		if string(gotContent) != "hello artifact" {
			t.Errorf("Expected 'hello artifact', got '%s'", string(gotContent))
		}
		if gotArt.Digest != "d1" {
			t.Errorf("Expected digest 'd1', got '%s'", gotArt.Digest)
		}

		list, _ := s.ListArtifacts("s1")
		if len(list) != 1 {
			t.Errorf("Expected 1 artifact in list, got %d", len(list))
		}

		if _, _, err := s.GetArtifact("non-existent"); err == nil {
			t.Error("Expected error for non-existent artifact")
		}

		// Test missing file but entry exists
		missingArt := &Artifact{ID: "missing", SessionID: "s1", Path: "missing.txt"}
		s.db.Exec("INSERT INTO artifacts (id, session_id, path) VALUES (?, ?, ?)", missingArt.ID, missingArt.SessionID, missingArt.Path)
		if _, _, err := s.GetArtifact("missing"); err == nil {
			t.Error("Expected error for missing artifact file")
		}
	})

	t.Run("Config", func(t *testing.T) {
		if err := s.SetConfig("k1", "v1"); err != nil {
			t.Fatalf("SetConfig failed: %v", err)
		}

		val, err := s.GetConfig("k1")
		if err != nil {
			t.Fatalf("GetConfig failed: %v", err)
		}
		if val != "v1" {
			t.Errorf("Expected 'v1', got '%s'", val)
		}

		val2, _ := s.GetConfig("unknown")
		if val2 != "" {
			t.Errorf("Expected empty string for unknown config, got '%s'", val2)
		}
	})
}