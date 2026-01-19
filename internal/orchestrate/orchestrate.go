package orchestrate

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/simon/internal/provider"
)

type AgentType string

const (
	AgentTypePlanner  AgentType = "planner"
	AgentTypeExecutor AgentType = "executor"
	AgentTypeReviewer AgentType = "reviewer"
)

type MultiAgentOrchestrator struct {
	Planner  provider.Provider
	Executor provider.Provider
	Reviewer provider.Provider
}

func New(p, e, r provider.Provider) *MultiAgentOrchestrator {
	return &MultiAgentOrchestrator{
		Planner:  p,
		Executor: e,
		Reviewer: r,
	}
}

func (o *MultiAgentOrchestrator) Execute(ctx context.Context, goal string) error {
	// 1. Plan
	fmt.Println("Agent [Planner]: Creating strategy...")
	// ... logic to call Planner ...

	// 2. Execute
	fmt.Println("Agent [Executor]: Implementing...")
	// ... logic to call Executor ...

	// 3. Review
	fmt.Println("Agent [Reviewer]: Verifying...")
	// ... logic to call Reviewer ...

	return nil
}
