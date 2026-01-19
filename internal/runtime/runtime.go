package runtime

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/felixgeelhaar/simon/internal/coach"
	"github.com/felixgeelhaar/simon/internal/guard"
	"github.com/felixgeelhaar/simon/internal/mcp"
	"github.com/felixgeelhaar/simon/internal/observe"
	"github.com/felixgeelhaar/simon/internal/provider"
	"github.com/felixgeelhaar/simon/internal/store"
	"github.com/felixgeelhaar/simon/internal/ui"
)

// Runtime orchestrates the execution loop.
type Runtime struct {
	store    store.Storage
	guard    *guard.Guard
	coach    *coach.Coach
	observe  *observe.Observer
	provider provider.Provider
	mcpProxy *mcp.Proxy
	ui       ui.UI
}

func New(s store.Storage, g *guard.Guard, c *coach.Coach, o *observe.Observer, p provider.Provider, mp *mcp.Proxy) *Runtime {
	return &Runtime{
		store:    s,
		guard:    g,
		coach:    c,
		observe:  o,
		provider: p,
		mcpProxy: mp,
		ui:       ui.SilentUI{},
	}
}

func (r *Runtime) SetUI(u ui.UI) {
	if u != nil {
		r.ui = u
	}
}

// ExecuteSession runs the main loop for a session.
func (r *Runtime) ExecuteSession(ctx context.Context, sessionID string) error {
	ctx, span := r.observe.StartSpan(ctx, "ExecuteSession")
	defer span.End()

	session, err := r.store.GetSession(sessionID)
	if err != nil {
		r.observe.Log().Error().Str("sessionID", sessionID).Err(err).Msg("failed to load session")
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Load Spec from metadata
	specPath, ok := session.Metadata["spec"]
	if !ok {
		return fmt.Errorf("session %s has no spec in metadata", sessionID)
	}
	spec, err := r.coach.LoadSpec(specPath)
	if err != nil {
		return fmt.Errorf("failed to load spec from %s: %w", specPath, err)
	}

	r.observe.Log().Info().
		Str("sessionID", session.ID).
		Str("goal", spec.Goal).
		Msg("starting session execution")

	// State tracking for this run
	currentIteration := 0
	totalPromptTokens := 0
	totalOutputTokens := 0
	
	// 0. Retrieve Context (Advanced Context Management)
	var contextContext string
	if vec, err := r.provider.Embed(ctx, spec.Goal); err == nil {
		memories, err := r.store.SearchMemory(vec, 3)
		if err == nil && len(memories) > 0 {
			var sb strings.Builder
			sb.WriteString("Relevant past experiences:\n")
			for _, m := range memories {
				sb.WriteString(fmt.Sprintf("- %s\n", m.Content))
			}
			contextContext = sb.String()
			r.observe.Log().Info().Int("count", len(memories)).Msg("retrieved relevant memories")
		}
	} else {
		// Log warning but continue if embedding fails (e.g. CLI provider)
		r.observe.Log().Warn().Err(err).Msg("failed to embed goal for context retrieval")
	}

	history := []provider.Message{
		{Role: "user", Content: fmt.Sprintf("Goal: %s\nDoD: %s\nConstraints: %v\n\n%s\nPlease execute.", spec.Goal, spec.DefinitionOfDone, spec.Constraints, contextContext)},
	}

	for {
		currentIteration++
		r.ui.UpdateIteration(currentIteration)
		iterLog := r.observe.Log().With().Int("iteration", currentIteration).Logger()

		// 1. Guard Check (Pre-Flight)
		if v := r.guard.CheckBudget(currentIteration, totalPromptTokens, totalOutputTokens); v != nil {
			iterLog.Warn().Str("violation", v.Rule).Msg("guard violation, stopping")
			session.Status = "halted"
			_ = r.store.UpdateSession(session)
			return fmt.Errorf("guard violation: %s", v.Message)
		}

		// 1.5 Context Management (Summarization)
		if len(history) > 20 || totalPromptTokens > 3000 {
			iterLog.Info().Msg("context limit approaching, summarizing history")
			summary, err := r.summarizeHistory(ctx, history)
			if err != nil {
				iterLog.Error().Err(err).Msg("failed to summarize, continuing without pruning")
			} else {
				newHistory := []provider.Message{
					{
						Role: "user", 
						Content: fmt.Sprintf("Goal: %s\nDoD: %s\nConstraints: %v\n\nProgress Summary: %s\n\nPlease continue execution.", 
							spec.Goal, spec.DefinitionOfDone, spec.Constraints, summary),
					},
				}
				history = newHistory
			}
		}

		// 2. Execute
		r.ui.Log(fmt.Sprintf("Iteration %d: Calling Provider...", currentIteration))
		resp, err := r.provider.Chat(ctx, history)
		if err != nil {
			iterLog.Error().Err(err).Msg("provider call failed")
			return err
		}

		// 3. Update Usage
		totalPromptTokens += resp.Usage.PromptTokens
		totalOutputTokens += resp.Usage.CompletionTokens
		r.ui.Log(fmt.Sprintf("Provider responded (%d tokens)", resp.Usage.TotalTokens))
		
		history = append(history, provider.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// 4. Process Tools
		if len(resp.ToolCalls) > 0 {
			r.ui.Log(fmt.Sprintf("Executing %d tool calls...", len(resp.ToolCalls)))
			results, err := r.mcpProxy.HandleToolCalls(ctx, sessionID, resp.ToolCalls)
			if err != nil {
				iterLog.Error().Err(err).Msg("mcp proxy failed")
				return err
			}

			for _, res := range results {
				history = append(history, provider.Message{
					Role:       "tool",
					Content:    res.Digest,
					ToolCallID: res.ToolCallID,
				})
			}
		}

		// 5. Verification & Completion Check
		session.UpdatedAt = time.Now()
		
		isDoneHint := resp.Content != "" && (
			strings.Contains(strings.ToLower(resp.Content), "task complete") ||
			strings.Contains(strings.ToLower(resp.Content), "i have finished") ||
			strings.Contains(strings.ToLower(resp.Content), "done"))

		if isDoneHint {
			iterLog.Info().Msg("completion suggested, verifying evidence")
			if err := r.verifyEvidence(ctx, spec); err != nil {
				iterLog.Warn().Err(err).Msg("verification failed")
				history = append(history, provider.Message{
					Role:    "user",
					Content: fmt.Sprintf("Verification failed: %v. Please correct and ensure the Evidence is present.", err),
				})
				session.Status = "running"
			} else {
				iterLog.Info().Msg("verification successful")
				session.Status = "completed"
				_ = r.store.UpdateSession(session)
				
				// 6. Archive Memory
			summaryReq := append(history, provider.Message{
					Role: "user",
					Content: "The task is complete. Provide a 1-sentence summary of what was built and key lessons learned for future reference.",
				})
			if summaryResp, err := r.provider.Chat(ctx, summaryReq); err == nil {
				if vec, err := r.provider.Embed(ctx, spec.Goal); err == nil {
					meta := map[string]string{"session_id": sessionID, "goal": spec.Goal}
					if err := r.store.AddMemory(summaryResp.Content, vec, meta); err != nil {
						r.observe.Log().Warn().Err(err).Msg("failed to archive memory")
					} else {
						r.observe.Log().Info().Msg("archived session memory")
					}
				}
			}
			break
			}
		} else {
			session.Status = "running"
		}
		
		if err := r.store.UpdateSession(session); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runtime) summarizeHistory(ctx context.Context, history []provider.Message) (string, error) {
	summaryReq := []provider.Message{}
	summaryReq = append(summaryReq, history...)
	summaryReq = append(summaryReq, provider.Message{
		Role: "user",
		Content: "Summarize the actions taken so far, the current state of the system, and what remains to be done. Be concise.",
	})

	resp, err := r.provider.Chat(ctx, summaryReq)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

func (r *Runtime) verifyEvidence(ctx context.Context, spec *coach.TaskSpec) error {
	for _, e := range spec.Evidence {
		if _, err := os.Stat(e); os.IsNotExist(err) {
			return fmt.Errorf("missing evidence: %s", e)
		}
	}
	return nil
}