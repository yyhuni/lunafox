package agentdata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"time"

	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	executionArtifactChunkMaxBytes = 65_536
	executionArtifactFrameMaxBytes = 131_072
)

func executionArtifactRoleForRegistryRole(role executionartifact.Role) (executionArtifactRole, error) {
	switch role {
	case executionartifact.RoleSubdomainsInput:
		return artifactRoleSubdomains, nil
	case executionartifact.RoleHostPortsInput:
		return artifactRoleHostPorts, nil
	case executionartifact.RoleWebsiteURLsInput:
		return artifactRoleWebsiteURLs, nil
	case executionartifact.RoleEndpointURLsInput:
		return artifactRoleEndpointURLs, nil
	case executionartifact.RoleWordlistConfigResource:
		return artifactRoleWordlist, nil
	case executionartifact.RoleSubfinderProviderConfigPlatformResource:
		return artifactRoleProviderConfig, nil
	case executionartifact.RoleFingerprintLibraryFingerPrintHub:
		return artifactRoleFingerprintLibrary, nil
	case executionartifact.RoleRuntimeArtifact:
		return artifactRoleRuntimeArtifact, nil
	default:
		return 0, fmt.Errorf("execution artifact role is unsupported")
	}
}

var (
	executionArtifactMeter     = otel.Meter("lunafox.execution.artifacts")
	executionArtifactTransfers = mustExecutionArtifactCounter("execution_artifact_grpc_transfers_total")
	executionArtifactBytes     = mustExecutionArtifactCounter("execution_artifact_grpc_bytes_total")
	executionArtifactDuration  = mustExecutionArtifactHistogram("execution_artifact_grpc_transfer_duration_seconds")
)

func mustExecutionArtifactCounter(name string) metric.Int64Counter {
	counter, err := executionArtifactMeter.Int64Counter(name)
	if err != nil {
		return metricnoop.Int64Counter{}
	}
	return counter
}

func mustExecutionArtifactHistogram(name string) metric.Float64Histogram {
	histogram, err := executionArtifactMeter.Float64Histogram(name)
	if err != nil {
		return metricnoop.Float64Histogram{}
	}
	return histogram
}

type executionArtifactRole uint8

const (
	artifactRoleSubdomains executionArtifactRole = iota + 1
	artifactRoleHostPorts
	artifactRoleWebsiteURLs
	artifactRoleEndpointURLs
	artifactRoleWordlist
	artifactRoleProviderConfig
	artifactRoleFingerprintLibrary
	artifactRoleRuntimeArtifact
)

func streamAuthorizedExecutionArtifact(
	ctx context.Context,
	send func(*agentdatav1.ExecutionArtifactFrame) error,
	role executionArtifactRole,
	header *agentdatav1.ExecutionArtifactHeader,
	source AuthorizedExecutionArtifact,
) error {
	startedAt := time.Now()
	outcome := "failed"
	var transferredBytes uint64
	defer func() {
		attributes := metric.WithAttributes(attribute.String("execution.artifact.role", executionArtifactMetricRole(role)), attribute.String("execution.artifact.transport", "grpc"), attribute.String("execution.artifact.outcome", outcome))
		executionArtifactTransfers.Add(ctx, 1, attributes)
		executionArtifactDuration.Record(ctx, time.Since(startedAt).Seconds(), attributes)
		if outcome == "completed" {
			executionArtifactBytes.Add(ctx, int64(transferredBytes), attributes)
		}
		pkg.Info("execution artifact gRPC transfer", zap.String("execution.artifact.role", executionArtifactMetricRole(role)), zap.String("execution.artifact.transport", "grpc"), zap.String("execution.artifact.outcome", outcome), zap.Uint64("execution.artifact.bytes", transferredBytes))
	}()
	if send == nil || source.Produce == nil {
		return status.Error(codes.Internal, "execution artifact stream is not configured")
	}
	if header == nil {
		return status.Error(codes.Internal, "execution artifact header is required")
	}
	descriptor, err := descriptorForExecutionArtifactRole(role)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	if role != artifactRoleFingerprintLibrary && header.GetContentType() != descriptor.ContentType {
		return status.Error(codes.Internal, "execution artifact content type does not match its typed role")
	}
	if err := validateExpectedIntegrity(role, header.GetExpectedIntegrity()); err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	if source.Preflight != nil {
		if err := source.Preflight(ctx); err != nil {
			return mapExecutionArtifactError(err)
		}
	}
	if err := sendExecutionArtifactFrameWithTransportObservation(send, &agentdatav1.ExecutionArtifactFrame{Payload: &agentdatav1.ExecutionArtifactFrame_Header{Header: header}}, source.executionInputBlacklistTransportInterrupted); err != nil {
		return err
	}

	writer := newExecutionArtifactChunkWriter(ctx, send, source.executionInputBlacklistTransportInterrupted)
	recordCount, produceErr := source.Produce(ctx, writer)
	if produceErr != nil {
		return mapExecutionArtifactError(produceErr)
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if role == artifactRoleProviderConfig && writer.sizeBytes == 0 {
		return status.Error(codes.Internal, "provider config producer returned empty content")
	}

	actual := &agentdatav1.ExecutionArtifactIntegrity{
		SizeBytes:    writer.sizeBytes,
		Sha256Digest: "sha256:" + hex.EncodeToString(writer.digest.Sum(nil)),
	}
	if descriptor.RecordCountPresence == executionartifact.RecordCountRequired {
		actual.RecordCount = &recordCount
	}
	if expected := header.GetExpectedIntegrity(); expected != nil && !equalExecutionArtifactIntegrity(expected, actual) {
		return status.Error(codes.DataLoss, "execution artifact bytes do not match plan-pinned integrity")
	}
	trailer := &agentdatav1.ExecutionArtifactTrailer{Integrity: actual}
	if err := sendExecutionArtifactFrameWithTransportObservation(send, &agentdatav1.ExecutionArtifactFrame{Payload: &agentdatav1.ExecutionArtifactFrame_Trailer{Trailer: trailer}}, source.executionInputBlacklistTransportInterrupted); err != nil {
		return err
	}
	transferredBytes = writer.sizeBytes
	outcome = "completed"
	return nil
}

func executionArtifactMetricRole(role executionArtifactRole) string {
	switch role {
	case artifactRoleSubdomains:
		return "subdomains"
	case artifactRoleHostPorts:
		return "hostPorts"
	case artifactRoleWebsiteURLs:
		return "websiteURLs"
	case artifactRoleEndpointURLs:
		return "endpointURLs"
	case artifactRoleWordlist:
		return "wordlist"
	case artifactRoleProviderConfig:
		return "provider_config"
	case artifactRoleFingerprintLibrary:
		return "fingerprint_library"
	case artifactRoleRuntimeArtifact:
		return "runtime_artifact"
	default:
		return "unknown"
	}
}

func descriptorForExecutionArtifactRole(role executionArtifactRole) (executionartifact.Descriptor, error) {
	var registryRole executionartifact.Role
	switch role {
	case artifactRoleSubdomains:
		registryRole = executionartifact.RoleSubdomainsInput
	case artifactRoleHostPorts:
		registryRole = executionartifact.RoleHostPortsInput
	case artifactRoleWebsiteURLs:
		registryRole = executionartifact.RoleWebsiteURLsInput
	case artifactRoleEndpointURLs:
		registryRole = executionartifact.RoleEndpointURLsInput
	case artifactRoleWordlist:
		registryRole = executionartifact.RoleWordlistConfigResource
	case artifactRoleProviderConfig:
		registryRole = executionartifact.RoleSubfinderProviderConfigPlatformResource
	case artifactRoleFingerprintLibrary:
		// The platform-resource request has already been checked against the
		// closed fingerprint registry before streaming. This synthetic stream
		// role only selects the required-integrity transport contract.
		return executionartifact.Descriptor{Role: executionartifact.RoleFingerprintLibraryFingerPrintHub, RecordCountPresence: executionartifact.RecordCountRequired}, nil
	case artifactRoleRuntimeArtifact:
		registryRole = executionartifact.RoleRuntimeArtifact
	default:
		return executionartifact.Descriptor{}, fmt.Errorf("execution artifact role is unsupported")
	}
	descriptor, ok := executionartifact.Lookup(registryRole)
	if !ok {
		return executionartifact.Descriptor{}, fmt.Errorf("execution artifact registry is incomplete")
	}
	return descriptor, nil
}

func validateExpectedIntegrity(role executionArtifactRole, integrity *agentdatav1.ExecutionArtifactIntegrity) error {
	if role != artifactRoleWordlist && role != artifactRoleFingerprintLibrary && role != artifactRoleRuntimeArtifact {
		if integrity != nil {
			return fmt.Errorf("derived or late-bound artifact must not declare expected integrity")
		}
		return nil
	}
	if integrity == nil || integrity.RecordCount == nil || !isCanonicalSHA256Digest(integrity.GetSha256Digest()) {
		return fmt.Errorf("integrity-pinned artifact expected integrity is incomplete")
	}
	return nil
}

func isCanonicalSHA256Digest(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || value[:len("sha256:")] != "sha256:" {
		return false
	}
	decoded, err := hex.DecodeString(value[len("sha256:"):])
	return err == nil && len(decoded) == sha256.Size
}

func equalExecutionArtifactIntegrity(left, right *agentdatav1.ExecutionArtifactIntegrity) bool {
	if left == nil || right == nil || left.GetSizeBytes() != right.GetSizeBytes() || left.GetSha256Digest() != right.GetSha256Digest() {
		return false
	}
	if (left.RecordCount == nil) != (right.RecordCount == nil) {
		return false
	}
	return left.RecordCount == nil || left.GetRecordCount() == right.GetRecordCount()
}

func sendExecutionArtifactFrame(send func(*agentdatav1.ExecutionArtifactFrame) error, frame *agentdatav1.ExecutionArtifactFrame) error {
	return sendExecutionArtifactFrameWithTransportObservation(send, frame, nil)
}

func sendExecutionArtifactFrameWithTransportObservation(send func(*agentdatav1.ExecutionArtifactFrame) error, frame *agentdatav1.ExecutionArtifactFrame, observeTransportInterruption func()) error {
	if frame == nil || frame.GetPayload() == nil {
		return status.Error(codes.Internal, "execution artifact frame payload is required")
	}
	if chunk := frame.GetChunk(); chunk != nil && (len(chunk.GetData()) == 0 || len(chunk.GetData()) > executionArtifactChunkMaxBytes) {
		return status.Error(codes.Internal, "execution artifact chunk is outside protocol bounds")
	}
	if proto.Size(frame) > executionArtifactFrameMaxBytes {
		return status.Error(codes.Internal, "execution artifact frame exceeds protocol bounds")
	}
	if err := send(frame); err != nil {
		if observeTransportInterruption != nil {
			observeTransportInterruption()
		}
		return err
	}
	return nil
}

type executionArtifactChunkWriter struct {
	ctx                          context.Context
	send                         func(*agentdatav1.ExecutionArtifactFrame) error
	observeTransportInterruption func()
	buffer                       []byte
	digest                       hash.Hash
	sizeBytes                    uint64
	closed                       bool
}

func newExecutionArtifactChunkWriter(ctx context.Context, send func(*agentdatav1.ExecutionArtifactFrame) error, observeTransportInterruption func()) *executionArtifactChunkWriter {
	return &executionArtifactChunkWriter{
		ctx:                          ctx,
		send:                         send,
		observeTransportInterruption: observeTransportInterruption,
		buffer:                       make([]byte, 0, executionArtifactChunkMaxBytes),
		digest:                       sha256.New(),
	}
}

func (writer *executionArtifactChunkWriter) Write(payload []byte) (int, error) {
	if writer.closed {
		return 0, fmt.Errorf("execution artifact writer is closed")
	}
	written := 0
	for len(payload) > 0 {
		if err := writer.ctx.Err(); err != nil {
			return written, err
		}
		remaining := executionArtifactChunkMaxBytes - len(writer.buffer)
		count := len(payload)
		if count > remaining {
			count = remaining
		}
		writer.buffer = append(writer.buffer, payload[:count]...)
		payload = payload[count:]
		written += count
		if len(writer.buffer) == executionArtifactChunkMaxBytes {
			if err := writer.flush(); err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

func (writer *executionArtifactChunkWriter) Close() error {
	if writer.closed {
		return nil
	}
	writer.closed = true
	return writer.flush()
}

func (writer *executionArtifactChunkWriter) flush() error {
	if len(writer.buffer) == 0 {
		return nil
	}
	data := bytes.Clone(writer.buffer)
	frame := &agentdatav1.ExecutionArtifactFrame{Payload: &agentdatav1.ExecutionArtifactFrame_Chunk{Chunk: &agentdatav1.ExecutionArtifactChunk{Data: data}}}
	if err := sendExecutionArtifactFrameWithTransportObservation(writer.send, frame, writer.observeTransportInterruption); err != nil {
		return err
	}
	if _, err := writer.digest.Write(data); err != nil {
		return err
	}
	if ^uint64(0)-writer.sizeBytes < uint64(len(data)) {
		return status.Error(codes.Internal, "execution artifact size counter overflow")
	}
	writer.sizeBytes += uint64(len(data))
	writer.buffer = writer.buffer[:0]
	return nil
}

var _ io.WriteCloser = (*executionArtifactChunkWriter)(nil)
