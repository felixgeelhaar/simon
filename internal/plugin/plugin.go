package plugin

import (
	"context"

	"github.com/felixgeelhaar/simon/internal/coach"
	"github.com/felixgeelhaar/simon/internal/guard"
	"github.com/felixgeelhaar/simon/internal/provider"
	"github.com/hashicorp/go-plugin"
)

// HandshakeConfig is used to handshake between host and plugin.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "SIMON_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "simon-runtime",
}

// Plugin defines the handshake and capabilities.
type Plugin interface {
	Name() string
	Version() string
	Type() PluginType
}

type PluginType string

const (
	PluginTypeCoach    PluginType = "coach"
	PluginTypeGuard    PluginType = "guard"
	PluginTypeProvider PluginType = "provider"
	PluginTypeReducer  PluginType = "reducer"
)

// CoachPlugin allows external validation logic.
type CoachPlugin interface {
	Plugin
	Validate(ctx context.Context, spec coach.TaskSpec) (coach.ValidationResult, error)
}

// GuardPlugin allows external policy enforcement.
type GuardPlugin interface {
	Plugin
	Check(ctx context.Context, action string, context map[string]interface{}) (*guard.Violation, error)
}

// ProviderPlugin allows integrating external AI models/agents.
// This matches the internal provider.Provider interface but lifted to a plugin.
type ProviderPlugin interface {
	Plugin
	Chat(ctx context.Context, messages []provider.Message) (*provider.Response, error)
}

// ReducerPlugin allows summarizing/digesting tool outputs (Artifacts).
type ReducerPlugin interface {
	Plugin
	Reduce(ctx context.Context, content []byte) (string, error)
}
