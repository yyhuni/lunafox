package agentdata

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	defaultRuntimeExchangeAdmission = 16
	hardRuntimeExchangeAdmission    = 64
)

var defaultRuntimeExchangeSlots = make(chan struct{}, defaultRuntimeExchangeAdmission)

// ExchangeRuntimeArtifact implements the bounded Nuclei manifest/blob
// exchange.  It is defined in a separate file so the older typed one-way
// streams remain easy to audit and cannot accidentally share their framing
// state with this protocol.
func (service *ExecutionArtifactService) ExchangeRuntimeArtifact(stream grpc.BidiStreamingServer[agentdatav1.ExchangeRuntimeArtifactRequest, agentdatav1.ExchangeRuntimeArtifactResponse]) error {
	if service == nil || stream == nil {
		return status.Error(codes.Internal, "runtime artifact exchange is not configured")
	}
	release, err := service.acquireRuntimeExchange()
	if err != nil {
		return err
	}
	defer release()

	first, err := stream.Recv()
	if err != nil {
		return mapExchangeTransportError(err)
	}
	if err := ValidateRuntimeArtifactExchangeFrame(first); err != nil {
		return err
	}
	beginPayload, ok := first.GetPayload().(*agentdatav1.ExchangeRuntimeArtifactRequest_Begin)
	if !ok || beginPayload == nil || beginPayload.Begin == nil {
		return status.Error(codes.InvalidArgument, "runtime artifact exchange must begin with Begin")
	}
	begin := beginPayload.Begin
	if err := validateRuntimeArtifactExchangeBegin(begin); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	lease, err := service.authenticate(stream.Context())
	if err != nil {
		return err
	}
	resolver, ok := service.resolver.(RuntimeArtifactExchangeResolver)
	if !ok || resolver == nil {
		return status.Error(codes.Unimplemented, "runtime artifact exchange resolver is not configured")
	}
	exchange, err := resolver.AuthorizeRuntimeArtifactExchange(stream.Context(), lease, begin)
	if err != nil {
		return mapExecutionArtifactError(err)
	}
	if exchange == nil {
		return status.Error(codes.Internal, "runtime artifact exchange boundary is missing")
	}
	defer exchange.close()

	if err := service.sendRuntimeManifest(stream, exchange); err != nil {
		return err
	}

	missing, err := receiveMissingBlobDigests(stream, exchange)
	if err != nil {
		return err
	}
	for _, digest := range missing {
		data, ok := exchange.blob(digest)
		if !ok {
			return status.Error(codes.DataLoss, "requested blob is outside the exchange read boundary")
		}
		if err := service.sendRuntimeBlob(stream, digest, data); err != nil {
			return err
		}
	}
	end := &agentdatav1.ExchangeRuntimeArtifactResponse{Payload: &agentdatav1.ExchangeRuntimeArtifactResponse_ExchangeEnd{ExchangeEnd: &agentdatav1.RuntimeArtifactExchangeEnd{Sequence: uint64(len(missing) + 1)}}}
	if err := sendRuntimeExchangeResponse(stream, end); err != nil {
		return err
	}

	ack, err := receivePublicationAck(stream)
	if err != nil {
		return err
	}
	if ack.GetSnapshotDigest() != exchange.SnapshotDigest || ack.GetEntryCount() != uint64(len(exchange.Entries)) {
		return status.Error(codes.DataLoss, "runtime artifact publication acknowledgement does not match manifest")
	}
	complete := &agentdatav1.ExchangeRuntimeArtifactResponse{Payload: &agentdatav1.ExchangeRuntimeArtifactResponse_ExchangeComplete{ExchangeComplete: &agentdatav1.RuntimeArtifactExchangeComplete{SnapshotDigest: exchange.SnapshotDigest, EntryCount: uint64(len(exchange.Entries))}}}
	return sendRuntimeExchangeResponse(stream, complete)
}

func (service *ExecutionArtifactService) sendRuntimeManifest(stream grpc.BidiStreamingServer[agentdatav1.ExchangeRuntimeArtifactRequest, agentdatav1.ExchangeRuntimeArtifactResponse], exchange *AuthorizedRuntimeArtifactExchange) error {
	sequence := uint64(1)
	batch := make([]*agentdatav1.RuntimeArtifactManifestEntry, 0, runtimeExchangeManifestBatchMax)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		entries := append([]*agentdatav1.RuntimeArtifactManifestEntry(nil), batch...)
		response := &agentdatav1.ExchangeRuntimeArtifactResponse{Payload: &agentdatav1.ExchangeRuntimeArtifactResponse_ManifestFrame{ManifestFrame: &agentdatav1.RuntimeArtifactManifestFrame{Sequence: sequence, Entries: entries}}}
		if err := sendRuntimeExchangeResponse(stream, response); err != nil {
			return err
		}
		sequence++
		batch = batch[:0]
		return nil
	}
	for _, entry := range exchange.Entries {
		candidate := append(batch, &agentdatav1.RuntimeArtifactManifestEntry{RelativePath: entry.RelativePath, Sha256Digest: entry.Digest, SizeBytes: entry.SizeBytes})
		if len(batch) > 0 {
			probe := &agentdatav1.ExchangeRuntimeArtifactResponse{Payload: &agentdatav1.ExchangeRuntimeArtifactResponse_ManifestFrame{ManifestFrame: &agentdatav1.RuntimeArtifactManifestFrame{Sequence: sequence, Entries: candidate}}}
			if proto.Size(probe) > runtimeExchangeFrameMaxBytes {
				if err := flush(); err != nil {
					return err
				}
				candidate = []*agentdatav1.RuntimeArtifactManifestEntry{{RelativePath: entry.RelativePath, Sha256Digest: entry.Digest, SizeBytes: entry.SizeBytes}}
			}
		}
		batch = candidate
	}
	if err := flush(); err != nil {
		return err
	}
	end := &agentdatav1.ExchangeRuntimeArtifactResponse{Payload: &agentdatav1.ExchangeRuntimeArtifactResponse_ManifestEnd{ManifestEnd: &agentdatav1.RuntimeArtifactManifestEnd{Sequence: sequence, EntryCount: uint64(len(exchange.Entries)), TotalSizeBytes: manifestTotalBytes(exchange.Entries), SnapshotDigest: exchange.SnapshotDigest}}}
	return sendRuntimeExchangeResponse(stream, end)
}

func manifestTotalBytes(entries []RuntimeArtifactManifestEntry) uint64 {
	var total uint64
	for _, entry := range entries {
		total += entry.SizeBytes
	}
	return total
}

func receiveMissingBlobDigests(stream grpc.BidiStreamingServer[agentdatav1.ExchangeRuntimeArtifactRequest, agentdatav1.ExchangeRuntimeArtifactResponse], exchange *AuthorizedRuntimeArtifactExchange) ([]string, error) {
	seen := make(map[string]struct{})
	missing := make([]string, 0)
	sequence := uint64(1)
	maxDigests := len(exchange.boundary.byDigest)
	if maxDigests <= 0 {
		return nil, status.Error(codes.DataLoss, "runtime artifact exchange manifest has no blobs")
	}
	for {
		request, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, status.Error(codes.DataLoss, "runtime artifact exchange ended before MissingEnd")
			}
			return nil, mapExchangeTransportError(err)
		}
		if err := ValidateRuntimeArtifactExchangeFrame(request); err != nil {
			return nil, err
		}
		switch payload := request.GetPayload().(type) {
		case *agentdatav1.ExchangeRuntimeArtifactRequest_MissingBlobDigests:
			if payload.MissingBlobDigests == nil || payload.MissingBlobDigests.GetSequence() != sequence || len(payload.MissingBlobDigests.GetDigests()) == 0 {
				return nil, status.Error(codes.InvalidArgument, "runtime artifact missing digest sequence is invalid")
			}
			for _, digest := range payload.MissingBlobDigests.GetDigests() {
				if !isCanonicalSHA256Digest(digest) {
					return nil, status.Error(codes.InvalidArgument, "runtime artifact missing digest is invalid")
				}
				if _, duplicate := seen[digest]; duplicate {
					return nil, status.Error(codes.InvalidArgument, "runtime artifact missing digest is duplicated")
				}
				if !exchangeHasDigest(exchange, digest) {
					return nil, status.Error(codes.DataLoss, "runtime artifact missing digest is absent from manifest")
				}
				seen[digest] = struct{}{}
				missing = append(missing, digest)
			}
			if len(missing) > maxDigests {
				return nil, status.Error(codes.ResourceExhausted, "runtime artifact missing digest set exceeds manifest")
			}
			sequence++
		case *agentdatav1.ExchangeRuntimeArtifactRequest_MissingEnd:
			if payload.MissingEnd == nil || payload.MissingEnd.GetSequence() != sequence || payload.MissingEnd.GetDigestCount() != uint64(len(missing)) {
				return nil, status.Error(codes.InvalidArgument, "runtime artifact MissingEnd is invalid")
			}
			return missing, nil
		default:
			return nil, status.Error(codes.InvalidArgument, "runtime artifact exchange expected missing digests")
		}
	}
}

func exchangeHasDigest(exchange *AuthorizedRuntimeArtifactExchange, digest string) bool {
	if exchange == nil || exchange.boundary == nil {
		return false
	}
	_, ok := exchange.boundary.byDigest[digest]
	return ok
}

func (service *ExecutionArtifactService) sendRuntimeBlob(stream grpc.BidiStreamingServer[agentdatav1.ExchangeRuntimeArtifactRequest, agentdatav1.ExchangeRuntimeArtifactResponse], digest string, data []byte) error {
	if !isCanonicalSHA256Digest(digest) || len(data) == 0 {
		return status.Error(codes.DataLoss, "runtime artifact blob is invalid")
	}
	sequence := uint64(1)
	for offset := 0; offset < len(data); {
		end := offset + runtimeExchangeBlobChunkMaxBytes
		if end > len(data) {
			end = len(data)
		}
		response := &agentdatav1.ExchangeRuntimeArtifactResponse{Payload: &agentdatav1.ExchangeRuntimeArtifactResponse_BlobGroup{BlobGroup: &agentdatav1.RuntimeArtifactBlobGroup{Digest: digest, Sequence: sequence, Data: append([]byte(nil), data[offset:end]...)}}}
		if err := sendRuntimeExchangeResponse(stream, response); err != nil {
			return err
		}
		sequence++
		offset = end
	}
	end := &agentdatav1.ExchangeRuntimeArtifactResponse{Payload: &agentdatav1.ExchangeRuntimeArtifactResponse_BlobEnd{BlobEnd: &agentdatav1.RuntimeArtifactBlobEnd{Digest: digest, Sequence: sequence, SizeBytes: uint64(len(data)), Sha256Digest: digest}}}
	return sendRuntimeExchangeResponse(stream, end)
}

func receivePublicationAck(stream grpc.BidiStreamingServer[agentdatav1.ExchangeRuntimeArtifactRequest, agentdatav1.ExchangeRuntimeArtifactResponse]) (*agentdatav1.RuntimeArtifactPublicationAck, error) {
	request, err := stream.Recv()
	if err != nil {
		return nil, mapExchangeTransportError(err)
	}
	if err := ValidateRuntimeArtifactExchangeFrame(request); err != nil {
		return nil, err
	}
	payload, ok := request.GetPayload().(*agentdatav1.ExchangeRuntimeArtifactRequest_PublicationAck)
	if !ok || payload == nil || payload.PublicationAck == nil {
		return nil, status.Error(codes.InvalidArgument, "runtime artifact exchange expected PublicationAck")
	}
	if !isCanonicalSHA256Digest(payload.PublicationAck.GetSnapshotDigest()) || payload.PublicationAck.GetEntryCount() == 0 {
		return nil, status.Error(codes.InvalidArgument, "runtime artifact publication acknowledgement is invalid")
	}
	return payload.PublicationAck, nil
}

func sendRuntimeExchangeResponse(stream grpc.BidiStreamingServer[agentdatav1.ExchangeRuntimeArtifactRequest, agentdatav1.ExchangeRuntimeArtifactResponse], response *agentdatav1.ExchangeRuntimeArtifactResponse) error {
	if err := ValidateRuntimeArtifactExchangeFrame(response); err != nil {
		return err
	}
	if response == nil || response.GetPayload() == nil {
		return status.Error(codes.Internal, "runtime artifact exchange response payload is required")
	}
	if err := stream.Send(response); err != nil {
		return mapExchangeTransportError(err)
	}
	return nil
}

func mapExchangeTransportError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "runtime artifact exchange canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "runtime artifact exchange deadline exceeded")
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Error(codes.Unavailable, "runtime artifact exchange transport failed")
}

func (service *ExecutionArtifactService) acquireRuntimeExchange() (func(), error) {
	if service == nil {
		return nil, status.Error(codes.Internal, "runtime artifact exchange service is required")
	}
	// A nil channel occurs only for package-local test literals. Use a shared
	// bounded fallback rather than accidentally creating an unbounded queue.
	slots := service.runtimeExchangeSlots()
	select {
	case slots <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-slots }) }, nil
	default:
		return nil, status.Error(codes.ResourceExhausted, "runtime artifact exchange admission is exhausted")
	}
}

func (service *ExecutionArtifactService) runtimeExchangeSlots() chan struct{} {
	if service.runtimeExchangeAdmission != nil {
		return service.runtimeExchangeAdmission
	}
	return defaultRuntimeExchangeSlots
}

// SetRuntimeExchangeAdmissionLimit configures a bounded per-service exchange
// admission gate. Values above the hard safety ceiling are rejected.
func (service *ExecutionArtifactService) SetRuntimeExchangeAdmissionLimit(limit int) error {
	if service == nil {
		return errors.New("runtime artifact exchange service is required")
	}
	if limit <= 0 || limit > hardRuntimeExchangeAdmission {
		return errors.New("runtime artifact exchange admission limit is outside the safety ceiling")
	}
	service.runtimeExchangeAdmission = make(chan struct{}, limit)
	return nil
}

// Keep the protocol revision check close to the exchange boundary. The
// revision is intentionally compared to the Engine API hard-cut revision used
// by the rest of the generated contracts.
func runtimeExchangeRevisionAccepted(value string) bool {
	return strings.TrimSpace(value) == engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision
}
