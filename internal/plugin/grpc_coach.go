package plugin

import (
	"context"

	"github.com/felixgeelhaar/simon/internal/coach"
	"github.com/felixgeelhaar/simon/internal/plugin/proto"
	hcplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]hcplugin.Plugin{
	"coach": &CoachGRPCPlugin{},
}

// CoachGRPCPlugin is the implementation of hcplugin.GRPCPlugin so we can serve/consume this.
type CoachGRPCPlugin struct {
	hcplugin.Plugin
	Impl CoachPlugin
}

func (p *CoachGRPCPlugin) GRPCServer(broker *hcplugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterCoachServer(s, &CoachGRPCServer{Impl: p.Impl})
	return nil
}

func (p *CoachGRPCPlugin) GRPCClient(ctx context.Context, broker *hcplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &CoachGRPCClient{client: proto.NewCoachClient(c)}, nil
}

// CoachGRPCClient is an implementation of CoachPlugin that talks over RPC.
type CoachGRPCClient struct {
	client proto.CoachClient
}

func (m *CoachGRPCClient) Name() string { return "grpc-coach" }
func (m *CoachGRPCClient) Version() string { return "1.0" }
func (m *CoachGRPCClient) Type() PluginType { return PluginTypeCoach }

func (m *CoachGRPCClient) Validate(ctx context.Context, spec coach.TaskSpec) (coach.ValidationResult, error) {
	resp, err := m.client.Validate(ctx, &proto.ValidateRequest{
		Goal:             spec.Goal,
		DefinitionOfDone: spec.DefinitionOfDone,
		Constraints:      spec.Constraints,
		Evidence:         spec.Evidence,
	})
	if err != nil {
		return coach.ValidationResult{}, err
	}
	return coach.ValidationResult{
		Valid:    resp.Valid,
		Warnings: resp.Warnings,
		Errors:   resp.Errors,
	}, nil
}

// CoachGRPCServer is the gRPC server that calls the local implementation.
type CoachGRPCServer struct {
	proto.UnimplementedCoachServer
	Impl CoachPlugin
}

func (m *CoachGRPCServer) Validate(ctx context.Context, req *proto.ValidateRequest) (*proto.ValidateResponse, error) {
	spec := coach.TaskSpec{
		Goal:             req.Goal,
		DefinitionOfDone: req.DefinitionOfDone,
		Constraints:      req.Constraints,
		Evidence:         req.Evidence,
	}
	res, err := m.Impl.Validate(ctx, spec)
	if err != nil {
		return nil, err
	}
	return &proto.ValidateResponse{
		Valid:    res.Valid,
		Warnings: res.Warnings,
		Errors:   res.Errors,
	}, nil
}

// Interfaces (re-stating for context, assume they exist in plugin.go)
