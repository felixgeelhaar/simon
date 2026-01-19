package coach

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCoach_LoadSpec(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "coach-test-*")
	defer os.RemoveAll(tmpDir)

	yamlPath := filepath.Join(tmpDir, "spec.yaml")
	os.WriteFile(yamlPath, []byte("goal: test goal\ndefinition_of_done: done\nconstraints: [c1]\nevidence: [e1]"), 0600)

	jsonPath := filepath.Join(tmpDir, "spec.json")
	os.WriteFile(jsonPath, []byte(`{"goal": "test json", "definition_of_done": "done json", "constraints": ["c2"], "evidence": ["e2"]}`), 0600)

	c := New()

	t.Run("YAML", func(t *testing.T) {
		spec, err := c.LoadSpec(yamlPath)
		if err != nil {
			t.Fatalf("Failed to load YAML: %v", err)
		}
		if spec.Goal != "test goal" {
			t.Errorf("Expected 'test goal', got '%s'", spec.Goal)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		spec, err := c.LoadSpec(jsonPath)
		if err != nil {
			t.Fatalf("Failed to load JSON: %v", err)
		}
		if spec.Goal != "test json" {
			t.Errorf("Expected 'test json', got '%s'", spec.Goal)
		}
	})

	t.Run("Invalid Extension", func(t *testing.T) {
		_, err := c.LoadSpec(filepath.Join(tmpDir, "spec.txt"))
		if err == nil {
			t.Error("Expected error for .txt extension")
		}
	})
}

func TestCoach_Validate(t *testing.T) {
	c := New()

	t.Run("Valid", func(t *testing.T) {
		spec := TaskSpec{
			Goal:             "Execute a complex refactor",
			DefinitionOfDone: "Code compiles",
			Constraints:      []string{"No breaking changes"},
			Evidence:         []string{"tests"},
		}
		res := c.Validate(spec)
		if !res.Valid {
			t.Errorf("Expected valid, got invalid: %v", res.Errors)
		}
	})

	t.Run("Short Goal", func(t *testing.T) {
		spec := TaskSpec{
			Goal:             "Short",
			DefinitionOfDone: "Done",
			Constraints:      []string{"None"},
			Evidence:         []string{"Evidence"},
		}
		res := c.Validate(spec)
		if len(res.Warnings) == 0 {
			t.Error("Expected warning for short goal")
		}
	})

	t.Run("Missing Fields", func(t *testing.T) {
		spec := TaskSpec{}
		res := c.Validate(spec)
		if res.Valid {
			t.Error("Expected invalid for empty spec")
		}
		if len(res.Errors) < 3 { // Goal, DoD, Evidence
			t.Errorf("Expected at least 3 errors, got %d", len(res.Errors))
		}
	})
}

func TestCoach_LintPrompt(t *testing.T) {
	c := New()
	if err := c.LintPrompt(""); err == nil {
		t.Error("Expected error for empty prompt")
	}
	if err := c.LintPrompt("Valid"); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}