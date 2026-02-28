package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	"google.golang.org/grpc"
)

type Server struct {
	addr       string
	listener   net.Listener
	grpcServer *grpc.Server
}

func New(
	addr string,
	agentRuntime runtimev1.AgentRuntimeServiceServer,
	dataProxy runtimev1.AgentDataProxyServiceServer,
	opts ...grpc.ServerOption,
) (*Server, error) {
	if agentRuntime == nil {
		return nil, errors.New("agent runtime service is nil")
	}
	if dataProxy == nil {
		return nil, errors.New("agent data proxy service is nil")
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen runtime grpc: %w", err)
	}

	gs := grpc.NewServer(opts...)
	runtimev1.RegisterAgentRuntimeServiceServer(gs, agentRuntime)
	runtimev1.RegisterAgentDataProxyServiceServer(gs, dataProxy)

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
		return errors.New("runtime grpc server not initialized")
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
