package agentserver

import (
	"context"
	"testing"
	"time"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	agentdata "github.com/yyhuni/lunafox/server/internal/grpc/agentdata"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// TestServerKeepaliveConnectionSurvivesIdlePeriod verifies that a bidirectional
// gRPC stream stays alive through an idle period when both sides have keepalive
// configured. Without transport-layer keepalive, intermediate network devices
// (Docker bridge, iptables/conntrack, proxies) would close the idle TCP
// connection, causing stream.Recv() to return EOF.
func TestServerKeepaliveConnectionSurvivesIdlePeriod(t *testing.T) {
	runtimeSvc := agentcontrol.NewControlPlaneService(
		agentFinderStub{agent: &agentdomain.Agent{ID: 1, InstanceID: "agt-1"}},
		&runtimeLifecycleStub{},
		&taskRuntimeStub{},
	)

	srv, err := New(
		"127.0.0.1:0",
		runtimeSvc,
		&agentdata.DataPlaneService{},
		&agentdata.ExecutionArtifactService{},
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    10 * time.Second,
			Timeout: 20 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	go func() { _ = srv.Serve() }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	conn, err := grpc.NewClient(
		srv.Addr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             20 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer conn.Close()

	client := agentcontrolv1.NewControlPlaneServiceClient(conn)
	streamCtx := metadata.AppendToOutgoingContext(
		context.Background(),
		grpcauth.AgentAuthenticationTokenMetadataKey, "agent-token",
	)
	stream, err := client.Connect(streamCtx)
	if err != nil {
		t.Fatalf("connect stream failed: %v", err)
	}

	// Register session so the server accepts the stream.
	if err := stream.Send(&agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_RegisterSession{
			RegisterSession: &agentcontrolv1.RegisterSession{
				Agent:                 resourcenames.Agent("agt-1"),
				Session:               resourcenames.AgentSession("agt-1", "session-1"),
				ObservedHostname:      "node-1",
				AgentVersion:          "v1.0.0",
				CompatibilityRevision: "engine-execution-diagnostics-r1",
			},
		},
	}); err != nil {
		t.Fatalf("register session failed: %v", err)
	}

	// Send a heartbeat to confirm the stream is functional after registration.
	if err := stream.Send(&agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_Heartbeat{
			Heartbeat: &agentcontrolv1.Heartbeat{
				Agent:                 resourcenames.Agent("agt-1"),
				Session:               resourcenames.AgentSession("agt-1", "session-1"),
				ObservedHostname:      "node-1",
				Health:                &agentcontrolv1.HealthStatus{State: agentcontrolv1.HealthState_HEALTH_STATE_HEALTHY},
				CompatibilityRevision: "engine-execution-diagnostics-r1",
			},
		},
	}); err != nil {
		t.Fatalf("heartbeat send failed: %v", err)
	}

	// Keep the stream idle for 2 seconds. On localhost this doesn't strictly
	// prove keepalive works (no intermediate network device to drop the
	// connection), but it validates that the keepalive parameters are accepted
	// and the stream remains functional after an idle gap. In production with
	// Docker bridge or proxy layers, this idle period is where EOF would occur
	// without keepalive.
	time.Sleep(2 * time.Second)

	// Verify the stream is still alive by sending another message.
	if err := stream.Send(&agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_Heartbeat{
			Heartbeat: &agentcontrolv1.Heartbeat{
				Agent:                 resourcenames.Agent("agt-1"),
				Session:               resourcenames.AgentSession("agt-1", "session-1"),
				ObservedHostname:      "node-1",
				Health:                &agentcontrolv1.HealthStatus{State: agentcontrolv1.HealthState_HEALTH_STATE_HEALTHY},
				CompatibilityRevision: "engine-execution-diagnostics-r1",
			},
		},
	}); err != nil {
		t.Fatalf("stream should survive idle period with keepalive, got send error: %v", err)
	}
}
