package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/simon/internal/coach"
	"github.com/felixgeelhaar/simon/internal/guard"
	"github.com/felixgeelhaar/simon/internal/mcp"
	"github.com/felixgeelhaar/simon/internal/observe"
	"github.com/felixgeelhaar/simon/internal/provider"
	"github.com/felixgeelhaar/simon/internal/runtime"
	"github.com/felixgeelhaar/simon/internal/store"
	"github.com/felixgeelhaar/simon/internal/ui"
)

type Runner struct {
	Observer *observe.Observer
	Store    store.Storage
	Provider provider.Provider
	SpecPath string
	UI       ui.UI
}

func (r *Runner) Run(ctx context.Context) error {
	r.UI.UpdateStatus("Starting Simon...")
	r.Observer.Log().Info().Msg("Simon: AI Agent Governance Runtime (Initialized)")

	g := guard.New(guard.DefaultPolicy)
	c := coach.New()
	mp := mcp.NewProxy(r.Store, g)
	rt := runtime.New(r.Store, g, c, r.Observer, r.Provider, mp)
	rt.SetUI(r.UI)

	// Create session
	sessID := fmt.Sprintf("session-%d", time.Now().Unix())
	session := &store.Session{
		ID:        sessID,
		CreatedAt: time.Now(),
		Status:    "initialized",
		Metadata:  map[string]string{"env": "dev", "spec": r.SpecPath},
	}

	if err := r.Store.CreateSession(session); err != nil {
		r.Observer.Log().Error().Err(err).Msg("Failed to create session")
		return err
	}

	// Validate spec
	r.UI.UpdateStatus("Loading Spec...")
	r.Observer.Log().Info().Str("path", r.SpecPath).Msg("loading spec")
	spec, err := c.LoadSpec(r.SpecPath)
	if err != nil {
		r.Observer.Log().Error().Err(err).Msg("Failed to load spec")
		return err
	}

	validation := c.Validate(*spec)
	if !validation.Valid {
		r.Observer.Log().Error().Str("errors", strings.Join(validation.Errors, ", ")).Msg("Invalid spec")
		return fmt.Errorf("invalid spec")
	}

	r.UI.UpdateStatus("Executing Session...")
	// Run
	if err := rt.ExecuteSession(ctx, sessID); err != nil {
		r.UI.UpdateStatus("Execution Failed")
		r.Observer.Log().Error().Err(err).Msg("Execution failed")
		return err
	}

	r.UI.UpdateStatus("Completed")
	fmt.Println("Session execution cycle complete.")
	return nil
}

func NewRunner(obs *observe.Observer, s store.Storage, p provider.Provider, specPath string, u ui.UI) *Runner {
	if u == nil {
		u = ui.SilentUI{}
	}
	return &Runner{
		Observer: obs,
		Store:    s,
		Provider: p,
		SpecPath: specPath,
		UI:       u,
	}
}