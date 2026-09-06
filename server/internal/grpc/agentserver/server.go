package agentserver

import (
	"context"
	"errors"
	"fmt"
	"net"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	"google.golang.org/grpc"
)

type Server struct {
	addr       string
	listener   net.Listener
	grpcServer *grpc.Server
}

func New(
	addr string,
	agentControlPlane agentcontrolv1.ControlPlaneServiceServer,
	agentDataPlane agentdatav1.DataPlaneServiceServer,
	executionArtifacts agentdatav1.ExecutionArtifactServiceServer,
	opts ...grpc.ServerOption,
) (*Server, error) {
	if agentControlPlane == nil {
		return nil, errors.New("agent control-plane service is nil")
	}
	if agentDataPlane == nil {
		return nil, errors.New("agent data-plane service is nil")
	}
	if executionArtifacts == nil {
		return nil, errors.New("execution artifact service is nil")
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen agent plane grpc: %w", err)
	}

	gs := grpc.NewServer(opts...)
	agentcontrolv1.RegisterControlPlaneServiceServer(gs, agentControlPlane)
	agentdatav1.RegisterDataPlaneServiceServer(gs, agentDataPlane)
	agentdatav1.RegisterExecutionArtifactServiceServer(gs, executionArtifacts)

	return &Server{
		addr:       lis.Addr().String(),
		listener:   lis,
		grpcServer: gs,
	}, nil
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) Serve() error {
	if s == nil || s.grpcServer == nil || s.listener == nil {
		return errors.New("agent plane grpc server not initialized")
	}
	return s.grpcServer.Serve(s.listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.grpcServer == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.grpcServer.Stop()
		<-done
		return ctx.Err()
	}
}
