package plugin

import (
	"context"
	"net"
	"testing"

	"github.com/felixgeelhaar/simon/internal/coach"
	"github.com/felixgeelhaar/simon/internal/plugin/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type MockCoach struct{}

func (m *MockCoach) Name() string { return "mock" }
func (m *MockCoach) Version() string { return "0.1" }
func (m *MockCoach) Type() PluginType { return PluginTypeCoach }
func (m *MockCoach) Validate(ctx context.Context, spec coach.TaskSpec) (coach.ValidationResult, error) {
	if spec.Goal == "fail" {
		return coach.ValidationResult{Valid: false, Errors: []string{"failed"}}, nil
	}
	return coach.ValidationResult{Valid: true}, nil
}

func TestCoachGRPC(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	proto.RegisterCoachServer(s, &CoachGRPCServer{Impl: &MockCoach{}})
	
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := &CoachGRPCClient{client: proto.NewCoachClient(conn)}

	// Test Valid
	res, err := client.Validate(context.Background(), coach.TaskSpec{Goal: "ok"})
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !res.Valid {
		t.Error("Expected valid result")
	}

	// Test Invalid
	res, err = client.Validate(context.Background(), coach.TaskSpec{Goal: "fail"})
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if res.Valid {
		t.Error("Expected invalid result")
	}
	if len(res.Errors) != 1 || res.Errors[0] != "failed" {
		t.Errorf("Expected 'failed' error, got %v", res.Errors)
	}
}
