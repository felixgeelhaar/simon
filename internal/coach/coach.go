package coach

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TaskSpec represents the structured input required to start a Simon session.
type TaskSpec struct {
	Goal             string   `json:"goal" yaml:"goal"`
	DefinitionOfDone string   `json:"definition_of_done" yaml:"definition_of_done"`
	Constraints      []string `json:"constraints" yaml:"constraints"`
	Evidence         []string `json:"evidence" yaml:"evidence"` // Paths or commands to verify completion
}

// ValidationResult represents the outcome of a linting pass.
type ValidationResult struct {
	Valid    bool
	Warnings []string
	Errors   []string
}

// Coach provides the logic to validate and refine task specifications.
type Coach struct{}

func New() *Coach {
	return &Coach{}
}

// LoadSpec reads a task specification from a file (JSON or YAML).
func (c *Coach) LoadSpec(path string) (*TaskSpec, error) {
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	var spec TaskSpec
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON spec: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML spec: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported spec format: %s (use .json or .yaml)", ext)
	}

	return &spec, nil
}

// Validate checks the TaskSpec for completeness and quality.
func (c *Coach) Validate(spec TaskSpec) ValidationResult {
	res := ValidationResult{
		Valid:    true,
		Warnings: []string{},
		Errors:   []string{},
	}

	if spec.Goal == "" {
		res.Valid = false
		res.Errors = append(res.Errors, "Goal is required")
	} else if len(spec.Goal) < 10 {
		res.Warnings = append(res.Warnings, "Goal is very short; consider adding more detail")
	}

	if spec.DefinitionOfDone == "" {
		res.Valid = false
		res.Errors = append(res.Errors, "Definition of Done (DoD) is required")
	}

	if len(spec.Constraints) == 0 {
		res.Warnings = append(res.Warnings, "No constraints specified. Are there really no limits?")
	}

	if len(spec.Evidence) == 0 {
		res.Valid = false
		res.Errors = append(res.Errors, "Evidence (verification steps) is required")
	}

	return res
}

// LintPrompt provides a simple check for raw text prompts (future use).
func (c *Coach) LintPrompt(prompt string) error {
	if prompt == "" {
		return errors.New("prompt cannot be empty")
	}
	return nil
}
