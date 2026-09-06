package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	controlservice "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/planeauth"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const e2eAgentInstanceID = "agent-ingress-e2e"

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: agent-ingress-dns-e2e <server|client>")
	}

	var err error
	switch os.Args[1] {
	case "server":
		err = runServer(os.Args[2:])
	case "client":
		err = runClient(os.Args[2:])
	default:
		err = fmt.Errorf("unknown mode %q", os.Args[1])
	}
	if err != nil {
		fatalf("%v", err)
	}
}

func runServer(args []string) error {
	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	listenAddress := flags.String("listen", ":9090", "gRPC listen address")
	authenticationToken := flags.String("token", "", "required Agent authentication token")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*authenticationToken) == "" {
		return errors.New("server token is required")
	}

	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		return err
	}
	defer listener.Close()

	service := controlservice.NewControlPlaneService(
		&e2eAgentFinder{authenticationToken: *authenticationToken},
		&e2eLifecycle{},
		&e2eTaskBridge{},
	)
	grpcServer := grpc.NewServer()
	agentcontrolv1.RegisterControlPlaneServiceServer(grpcServer, service)
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- grpcServer.Serve(listener)
	}()
	fmt.Printf("LISTENING %s\n", listener.Addr())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case <-signals:
		grpcServer.Stop()
		return nil
	case err := <-serveErrors:
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	}
}

func runClient(args []string) error {
	flags := flag.NewFlagSet("client", flag.ContinueOnError)
	address := flags.String("address", "", "Nginx gRPC address")
	authenticationToken := flags.String("token", "", "Agent authentication token")
	requiredReady := flags.Int("required-ready", 1, "number of successful ready sessions before exit")
	timeout := flags.Duration("timeout", 45*time.Second, "overall connection deadline")
	expectUnauthenticated := flags.Bool("expect-unauthenticated", false, "succeed only when token authentication is rejected")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*address) == "" || strings.TrimSpace(*authenticationToken) == "" {
		return errors.New("client address and token are required")
	}
	if *requiredReady <= 0 {
		return errors.New("required-ready must be positive")
	}
	if *timeout <= 0 {
		return errors.New("timeout must be positive")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	readyCount := 0
	attempt := 0
	for readyCount < *requiredReady {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("timed out after %d ready sessions: %w", readyCount, err)
		}
		attempt++
		session, err := openControlSession(ctx, *address, *authenticationToken, attempt)
		if err != nil {
			if *expectUnauthenticated && status.Code(err) == codes.Unauthenticated {
				fmt.Println("AUTH_REJECTED")
				return nil
			}
			if err := waitForRetry(ctx); err != nil {
				return fmt.Errorf("open control session: %w", err)
			}
			continue
		}
		if *expectUnauthenticated {
			session.close()
			return errors.New("invalid token unexpectedly reached SessionReady")
		}

		readyCount++
		fmt.Printf("READY %d\n", readyCount)
		if readyCount >= *requiredReady {
			session.close()
			return nil
		}

		err = session.keepAlive(ctx)
		session.close()
		if err == nil {
			return errors.New("control stream ended without a transport error")
		}
		fmt.Printf("DISCONNECTED %d\n", readyCount)
		if err := waitForRetry(ctx); err != nil {
			return fmt.Errorf("reconnect after stream disruption: %w", err)
		}
	}
	return nil
}

type e2eControlSession struct {
	connection *grpc.ClientConn
	stream     agentcontrolv1.ControlPlaneService_ConnectClient
	sessionID  string
}

func openControlSession(ctx context.Context, address, authenticationToken string, attempt int) (*e2eControlSession, error) {
	connection, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, // The isolated test generates a transient Nginx certificate.
		})),
	)
	if err != nil {
		return nil, err
	}
	closeOnError := func(err error) (*e2eControlSession, error) {
		_ = connection.Close()
		return nil, err
	}

	sessionID := fmt.Sprintf("agent-ingress-%d", attempt)
	streamContext := metadata.AppendToOutgoingContext(ctx, grpcauth.AgentAuthenticationTokenMetadataKey, authenticationToken)
	stream, err := agentcontrolv1.NewControlPlaneServiceClient(connection).Connect(streamContext)
	if err != nil {
		return closeOnError(err)
	}
	if err := stream.Send(registerSessionRequest(sessionID)); err != nil {
		return closeOnError(err)
	}
	response, err := stream.Recv()
	if err != nil {
		return closeOnError(err)
	}
	if response.GetSessionReady() == nil {
		return closeOnError(errors.New("control stream did not return SessionReady"))
	}
	return &e2eControlSession{connection: connection, stream: stream, sessionID: sessionID}, nil
}

func (session *e2eControlSession) keepAlive(ctx context.Context) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := session.stream.Send(heartbeatRequest(session.sessionID)); err != nil {
				return err
			}
		}
	}
}

func (session *e2eControlSession) close() {
	if session == nil || session.connection == nil {
		return
	}
	_ = session.connection.Close()
}

func registerSessionRequest(sessionID string) *agentcontrolv1.ConnectRequest {
	return &agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_RegisterSession{
			RegisterSession: &agentcontrolv1.RegisterSession{
				Agent:                    resourcenames.Agent(e2eAgentInstanceID),
				Session:                  resourcenames.AgentSession(e2eAgentInstanceID, sessionID),
				ObservedHostname:         "agent-ingress-e2e",
				AgentVersion:             "e2e",
				OperatingSystem:          "linux",
				Architecture:             runtime.GOARCH,
				ContainerRuntimeReady:    true,
				SupportedEngineApiMajors: []uint32{1},
				CompatibilityRevision:    "engine-execution-diagnostics-r1",
			},
		},
	}
}

func heartbeatRequest(sessionID string) *agentcontrolv1.ConnectRequest {
	return &agentcontrolv1.ConnectRequest{
		Payload: &agentcontrolv1.ConnectRequest_Heartbeat{
			Heartbeat: &agentcontrolv1.Heartbeat{
				Agent:                    resourcenames.Agent(e2eAgentInstanceID),
				Session:                  resourcenames.AgentSession(e2eAgentInstanceID, sessionID),
				ObservedHostname:         "agent-ingress-e2e",
				AgentVersion:             "e2e",
				OperatingSystem:          "linux",
				Architecture:             runtime.GOARCH,
				ContainerRuntimeReady:    true,
				SupportedEngineApiMajors: []uint32{1},
				CompatibilityRevision:    "engine-execution-diagnostics-r1",
			},
		},
	}
}

func waitForRetry(ctx context.Context) error {
	timer := time.NewTimer(300 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type e2eAgentFinder struct {
	authenticationToken string
}

func (finder *e2eAgentFinder) FindByAuthenticationToken(_ context.Context, authenticationToken string) (*agentdomain.Agent, error) {
	if finder == nil || authenticationToken != finder.authenticationToken {
		return nil, nil
	}
	return &agentdomain.Agent{
		ID:            1,
		InstanceID:    e2eAgentInstanceID,
		DisplayName:   "Agent ingress DNS E2E",
		MaxTasks:      1,
		CPUThreshold:  80,
		MemThreshold:  80,
		DiskThreshold: 80,
	}, nil
}

type e2eLifecycle struct{}

func (*e2eLifecycle) OnConnected(context.Context, *agentdomain.Agent, string) error { return nil }
func (*e2eLifecycle) OnControlConnectionDetached(context.Context, int) error        { return nil }
func (*e2eLifecycle) OnDisconnected(context.Context, int) error                     { return nil }
func (*e2eLifecycle) RecordHeartbeat(context.Context, int, agentdomain.AgentHeartbeatEvent) error {
	return nil
}

type e2eTaskBridge struct{}

func (*e2eTaskBridge) ReportTerminalTaskResult(context.Context, int, string, int64, int, string, *scanapp.FailureDetail) error {
	return nil
}

func (*e2eTaskBridge) FenceSupersededAgentSession(context.Context, int, int64) error {
	return nil
}

func fatalf(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", arguments...)
	os.Exit(1)
}
