package runtime

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net"
	neturl "net/url"
	"strings"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/domain"
	grpcauth "github.com/yyhuni/lunafox/agent/internal/grpc/runtime/auth"
	"github.com/yyhuni/lunafox/agent/internal/logger"
	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrRuntimeNotConnected = errors.New("runtime stream is not connected")
	ErrPullAlreadyInFlight = errors.New("runtime pull request already in flight")
)

type pullResult struct {
	assign *runtimev1.TaskAssign
	err    error
}

type Backoff struct {
	base    time.Duration
	max     time.Duration
	current time.Duration
	randSrc *rand.Rand
}

func NewBackoff(base, max time.Duration) *Backoff {
	return &Backoff{
		base:    base,
		max:     max,
		current: base,
		randSrc: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (b *Backoff) Next() time.Duration {
	if b == nil {
		return 0
	}
	delay := b.current
	next := b.current * 2
	if next > b.max {
		next = b.max
	}
	b.current = next
	if delay <= 0 || b.randSrc == nil {
		return delay
	}
	jitter := b.randSrc.Float64() * 0.2
	return delay + time.Duration(float64(delay)*jitter)
}

func (b *Backoff) Reset() {
	if b == nil {
		return
	}
	b.current = b.base
}

type Client struct {
	runtimeURL string
	apiKey     string
	backoff    *Backoff

	streamMu sync.Mutex
	stateMu  sync.RWMutex
	conn     *grpc.ClientConn
	stream   runtimev1.AgentRuntimeService_ConnectClient

	taskSessionMu sync.RWMutex
	taskSessions  map[int]string

	pendingPull chan pullResult

	onTaskCancel   func(int)
	onConfigUpdate func(domain.ConfigUpdate)
	onUpdateReq    func(domain.UpdateRequiredPayload)
	dialTimeout    time.Duration
}

func NewClient(runtimeURL, apiKey string) *Client {
	return &Client{
		runtimeURL:   strings.TrimSpace(runtimeURL),
		apiKey:       strings.TrimSpace(apiKey),
		backoff:      NewBackoff(1*time.Second, 30*time.Second),
		dialTimeout:  10 * time.Second,
		taskSessions: map[int]string{},
	}
}

func (c *Client) OnTaskCancel(fn func(int)) {
	c.stateMu.Lock()
	c.onTaskCancel = fn
	c.stateMu.Unlock()
}

func (c *Client) OnConfigUpdate(fn func(domain.ConfigUpdate)) {
	c.stateMu.Lock()
	c.onConfigUpdate = fn
	c.stateMu.Unlock()
}

func (c *Client) OnUpdateRequired(fn func(domain.UpdateRequiredPayload)) {
	c.stateMu.Lock()
	c.onUpdateReq = fn
	c.stateMu.Unlock()
}

func (c *Client) RegisterTaskSession(taskID int, taskToken string) {
	taskToken = strings.TrimSpace(taskToken)
	if c == nil || taskID <= 0 || taskToken == "" {
		return
	}
	c.taskSessionMu.Lock()
	c.taskSessions[taskID] = taskToken
	c.taskSessionMu.Unlock()
}

func (c *Client) ClearTaskSession(taskID int, taskToken string) {
	taskToken = strings.TrimSpace(taskToken)
	if c == nil || taskID <= 0 || taskToken == "" {
		return
	}
	c.taskSessionMu.Lock()
	if current, ok := c.taskSessions[taskID]; ok && current == taskToken {
		delete(c.taskSessions, taskID)
	}
	c.taskSessionMu.Unlock()
}

func (c *Client) ValidateTaskSession(taskID int, taskToken string) bool {
	taskToken = strings.TrimSpace(taskToken)
	if c == nil || taskID <= 0 || taskToken == "" {
		return false
	}
	c.taskSessionMu.RLock()
	current, ok := c.taskSessions[taskID]
	c.taskSessionMu.RUnlock()
	return ok && current == taskToken
}

func (c *Client) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := c.connectAndPump(ctx)
		if err != nil && ctx.Err() == nil {
			logger.Log.Warn("runtime stream disconnected", zap.Error(err))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		delay := c.backoff.Next()
		if !sleepWithContext(ctx, delay) {
			return ctx.Err()
		}
	}
}

func (c *Client) PullTask(ctx context.Context) (*domain.Task, error) {
	stream := c.currentStream()
	if stream == nil {
		return nil, ErrRuntimeNotConnected
	}

	respCh := make(chan pullResult, 1)
	if err := c.setPendingPull(respCh); err != nil {
		return nil, err
	}
	defer c.clearPendingPull(respCh)

	req := &runtimev1.AgentRuntimeRequest{
		Payload: &runtimev1.AgentRuntimeRequest_RequestTask{
			RequestTask: &runtimev1.RequestTask{},
		},
	}
	if err := c.send(req); err != nil {
		c.failPendingPull(err)
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respCh:
		if resp.err != nil {
			return nil, resp.err
		}
		assign := resp.assign
		if assign == nil || !assign.Found {
			return nil, nil
		}
		return &domain.Task{
			ID:           int(assign.TaskId),
			ScanID:       int(assign.ScanId),
			Stage:        int(assign.Stage),
			WorkflowName: assign.WorkflowName,
			TargetID:     int(assign.TargetId),
			TargetName:   assign.TargetName,
			TargetType:   assign.TargetType,
			WorkspaceDir: assign.WorkspaceDir,
			Config:       assign.Config,
		}, nil
	}
}

func (c *Client) UpdateStatus(ctx context.Context, taskID int, status, errorMessage string) error {
	if taskID <= 0 {
		return errors.New("task ID is required")
	}

	req := &runtimev1.AgentRuntimeRequest{
		Payload: &runtimev1.AgentRuntimeRequest_TaskStatus{
			TaskStatus: &runtimev1.TaskStatus{
				TaskId:  int32(taskID),
				Status:  status,
				Message: errorMessage,
			},
		},
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return c.send(req)
}

func (c *Client) SendHeartbeat(ctx context.Context, payload *runtimev1.Heartbeat) error {
	if payload == nil {
		return errors.New("heartbeat payload is required")
	}
	req := &runtimev1.AgentRuntimeRequest{
		Payload: &runtimev1.AgentRuntimeRequest_Heartbeat{
			Heartbeat: payload,
		},
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return c.send(req)
}

func (c *Client) connectAndPump(ctx context.Context) error {
	target, secure, err := buildRuntimeTarget(c.runtimeURL)
	if err != nil {
		return err
	}

	dialCtx, cancel := context.WithTimeout(ctx, c.dialTimeout)
	defer cancel()

	dialOptions := []grpc.DialOption{}
	if secure {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	} else {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(target, dialOptions...)
	if err != nil {
		return err
	}
	if err := waitForConnectionReady(dialCtx, conn); err != nil {
		_ = conn.Close()
		return err
	}

	client := runtimev1.NewAgentRuntimeServiceClient(conn)
	streamCtx := grpcauth.WithAgentKey(ctx, c.apiKey)
	stream, err := client.Connect(streamCtx)
	if err != nil {
		_ = conn.Close()
		return err
	}

	c.stateMu.Lock()
	c.conn = conn
	c.stream = stream
	c.stateMu.Unlock()
	c.backoff.Reset()
	logger.Log.Info("runtime gRPC connected", zap.String("target", target))

	defer func() {
		c.stateMu.Lock()
		c.stream = nil
		c.conn = nil
		c.stateMu.Unlock()
		c.failPendingPull(ErrRuntimeNotConnected)
		_ = conn.Close()
	}()

	for {
		event, recvErr := stream.Recv()
		if recvErr != nil {
			return recvErr
		}
		c.dispatchEvent(event)
	}
}

func waitForConnectionReady(ctx context.Context, conn *grpc.ClientConn) error {
	conn.Connect()
	for {
		state := conn.GetState()
		switch state {
		case connectivity.Ready:
			return nil
		case connectivity.Shutdown:
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errors.New("runtime gRPC connection shutdown")
		}
		if !conn.WaitForStateChange(ctx, state) {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errors.New("runtime gRPC connection timed out")
		}
	}
}

func (c *Client) dispatchEvent(event *runtimev1.AgentRuntimeEvent) {
	if event == nil {
		return
	}

	switch payload := event.Payload.(type) {
	case *runtimev1.AgentRuntimeEvent_TaskAssign:
		c.resolvePendingPull(payload.TaskAssign, nil)
	case *runtimev1.AgentRuntimeEvent_TaskCancel:
		if payload.TaskCancel == nil || payload.TaskCancel.TaskId <= 0 {
			return
		}
		c.stateMu.RLock()
		handler := c.onTaskCancel
		c.stateMu.RUnlock()
		if handler != nil {
			handler(int(payload.TaskCancel.TaskId))
		}
	case *runtimev1.AgentRuntimeEvent_ConfigUpdate:
		if payload.ConfigUpdate == nil {
			return
		}
		c.stateMu.RLock()
		handler := c.onConfigUpdate
		c.stateMu.RUnlock()
		if handler != nil {
			handler(domain.ConfigUpdate{
				MaxTasks:      int32PtrToIntPtr(payload.ConfigUpdate.MaxTasks),
				CPUThreshold:  int32PtrToIntPtr(payload.ConfigUpdate.CpuThreshold),
				MemThreshold:  int32PtrToIntPtr(payload.ConfigUpdate.MemThreshold),
				DiskThreshold: int32PtrToIntPtr(payload.ConfigUpdate.DiskThreshold),
			})
		}
	case *runtimev1.AgentRuntimeEvent_UpdateRequired:
		if payload.UpdateRequired == nil {
			return
		}
		c.stateMu.RLock()
		handler := c.onUpdateReq
		c.stateMu.RUnlock()
		if handler != nil {
			handler(domain.UpdateRequiredPayload{
				Version:  payload.UpdateRequired.TargetVersion,
				ImageRef: payload.UpdateRequired.ImageRef,
			})
		}
	default:
		// Ignore unknown events for forward compatibility.
	}
}

func (c *Client) currentStream() runtimev1.AgentRuntimeService_ConnectClient {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.stream
}

func (c *Client) send(req *runtimev1.AgentRuntimeRequest) error {
	stream := c.currentStream()
	if stream == nil {
		return ErrRuntimeNotConnected
	}
	c.streamMu.Lock()
	defer c.streamMu.Unlock()
	return stream.Send(req)
}

func (c *Client) setPendingPull(ch chan pullResult) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if c.pendingPull != nil {
		return ErrPullAlreadyInFlight
	}
	c.pendingPull = ch
	return nil
}

func (c *Client) clearPendingPull(ch chan pullResult) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if c.pendingPull == ch {
		c.pendingPull = nil
	}
}

func (c *Client) resolvePendingPull(assign *runtimev1.TaskAssign, err error) {
	c.stateMu.Lock()
	pending := c.pendingPull
	c.pendingPull = nil
	c.stateMu.Unlock()
	if pending == nil {
		return
	}
	select {
	case pending <- pullResult{assign: assign, err: err}:
	default:
	}
}

func (c *Client) failPendingPull(err error) {
	c.resolvePendingPull(nil, err)
}

func buildRuntimeTarget(runtimeURL string) (target string, secure bool, err error) {
	trimmed := strings.TrimSpace(runtimeURL)
	if trimmed == "" {
		return "", false, errors.New("runtime URL is required")
	}
	parsed, err := neturl.Parse(trimmed)
	if err != nil {
		return "", false, err
	}
	if parsed.Scheme == "" {
		return "", false, errors.New("runtime URL scheme is required")
	}

	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "https", "wss":
		secure = true
	case "http", "ws":
		secure = false
	default:
		return "", false, fmt.Errorf("unsupported runtime URL scheme: %s", parsed.Scheme)
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return "", false, errors.New("runtime host is required")
	}
	port := strings.TrimSpace(parsed.Port())
	if port == "" {
		if secure {
			port = "443"
		} else {
			port = "80"
		}
	}
	return net.JoinHostPort(host, port), secure, nil
}

func int32PtrToIntPtr(value *int32) *int {
	if value == nil {
		return nil
	}
	cast := int(*value)
	return &cast
}

func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
