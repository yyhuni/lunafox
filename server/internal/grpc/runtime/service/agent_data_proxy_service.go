package service

import (
	"context"
	"errors"
	"io"
	"os"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	grpcauth "github.com/yyhuni/lunafox/server/internal/grpc/runtime/auth"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentDataProxyService struct {
	runtimev1.UnimplementedAgentDataProxyServiceServer

	providerConfig ProviderConfigRuntime
	wordlists      WordlistRuntime
	batchUpsert    AssetBatchUpsertRuntimes
	agentFinder    AgentFinder
}

func NewAgentDataProxyService() *AgentDataProxyService {
	return &AgentDataProxyService{}
}

type ProviderConfigRuntime interface {
	GetProviderConfig(scanID int, toolName string) (string, error)
}

type WordlistRuntime interface {
	GetByName(name string) (*catalogapp.Wordlist, error)
	GetFilePath(name string) (string, error)
}

type SubdomainSnapshotRuntime interface {
	SaveAndSync(scanID int, targetID int, items []snapshotapp.SubdomainSnapshotItem) (snapshotCount int64, assetCount int64, err error)
}

type WebsiteSnapshotRuntime interface {
	SaveAndSync(scanID int, targetID int, items []snapshotapp.WebsiteSnapshotItem) (snapshotCount int64, assetCount int64, err error)
}

type EndpointSnapshotRuntime interface {
	SaveAndSync(scanID int, targetID int, items []snapshotapp.EndpointSnapshotItem) (snapshotCount int64, assetCount int64, err error)
}

type HostPortSnapshotRuntime interface {
	SaveAndSync(scanID int, targetID int, items []snapshotapp.HostPortSnapshotItem) (snapshotCount int64, assetCount int64, err error)
}

type AssetBatchUpsertRuntimes struct {
	Subdomains SubdomainSnapshotRuntime
	Websites   WebsiteSnapshotRuntime
	Endpoints  EndpointSnapshotRuntime
	HostPorts  HostPortSnapshotRuntime
}

func NewAgentDataProxyServiceWithDeps(providerConfig ProviderConfigRuntime, wordlists WordlistRuntime, batchUpsert ...AssetBatchUpsertRuntimes) *AgentDataProxyService {
	service := &AgentDataProxyService{
		providerConfig: providerConfig,
		wordlists:      wordlists,
	}
	if len(batchUpsert) > 0 {
		service.batchUpsert = batchUpsert[0]
	}
	return service
}

func (s *AgentDataProxyService) WithAgentFinder(agentFinder AgentFinder) *AgentDataProxyService {
	if s == nil {
		return nil
	}
	s.agentFinder = agentFinder
	return s
}

func (s *AgentDataProxyService) GetProviderConfig(ctx context.Context, req *runtimev1.GetProviderConfigRequest) (*runtimev1.GetProviderConfigResponse, error) {
	if err := s.requireAgentKey(ctx); err != nil {
		return nil, err
	}
	if s.providerConfig == nil {
		return nil, status.Error(codes.Unimplemented, errDataProxyUnimplemented)
	}

	content, err := s.providerConfig.GetProviderConfig(int(req.ScanId), req.ToolName)
	if err != nil {
		switch {
		case errors.Is(err, catalogapp.ErrWorkerScanNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, catalogapp.ErrWorkerToolRequired):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	return &runtimev1.GetProviderConfigResponse{Content: content}, nil
}

func (s *AgentDataProxyService) GetWordlistMeta(ctx context.Context, req *runtimev1.GetWordlistMetaRequest) (*runtimev1.GetWordlistMetaResponse, error) {
	if err := s.requireAgentKey(ctx); err != nil {
		return nil, err
	}
	if s.wordlists == nil {
		return nil, status.Error(codes.Unimplemented, errDataProxyUnimplemented)
	}

	wordlist, err := s.wordlists.GetByName(req.WordlistName)
	if err != nil {
		if errors.Is(err, catalogapp.ErrWordlistNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &runtimev1.GetWordlistMetaResponse{
		Id:          int32(wordlist.ID),
		Name:        wordlist.Name,
		FilePath:    wordlist.FilePath,
		FileHash:    wordlist.FileHash,
		FileSize:    wordlist.FileSize,
		LineCount:   int32(wordlist.LineCount),
		Description: wordlist.Description,
	}, nil
}

func (s *AgentDataProxyService) DownloadWordlist(req *runtimev1.DownloadWordlistRequest, stream grpc.ServerStreamingServer[runtimev1.DownloadWordlistChunk]) error {
	if err := s.requireAgentKey(stream.Context()); err != nil {
		return err
	}
	if s.wordlists == nil {
		return status.Error(codes.Unimplemented, errDataProxyUnimplemented)
	}

	filePath, err := s.wordlists.GetFilePath(req.WordlistName)
	if err != nil {
		if errors.Is(err, catalogapp.ErrWordlistNotFound) {
			return status.Error(codes.NotFound, err.Error())
		}
		return status.Error(codes.Internal, err.Error())
	}

	f, err := os.Open(filePath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	defer func() {
		_ = f.Close()
	}()

	buf := make([]byte, 64*1024)
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			chunk := &runtimev1.DownloadWordlistChunk{Data: append([]byte(nil), buf[:n]...)}
			if sendErr := stream.Send(chunk); sendErr != nil {
				return sendErr
			}
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			return status.Error(codes.Internal, readErr.Error())
		}
	}
}

func (s *AgentDataProxyService) BatchUpsertAssets(ctx context.Context, req *runtimev1.BatchUpsertAssetsRequest) (*runtimev1.BatchUpsertAssetsResponse, error) {
	if err := s.requireAgentKey(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.ScanId <= 0 || req.TargetId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "scanId and targetId are required")
	}
	if len(req.ItemsJson) == 0 {
		return nil, status.Error(codes.InvalidArgument, "itemsJson must not be empty")
	}

	switch req.DataType {
	case "subdomain":
		if s.batchUpsert.Subdomains == nil {
			return nil, status.Error(codes.Unimplemented, errDataProxyUnimplemented)
		}
		items, err := parseSubdomainItems(req.ItemsJson)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if _, _, err := s.batchUpsert.Subdomains.SaveAndSync(int(req.ScanId), int(req.TargetId), items); err != nil {
			return nil, mapBatchUpsertError(err)
		}
	case "website":
		if s.batchUpsert.Websites == nil {
			return nil, status.Error(codes.Unimplemented, errDataProxyUnimplemented)
		}
		items, err := parseWebsiteItems(req.ItemsJson)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if _, _, err := s.batchUpsert.Websites.SaveAndSync(int(req.ScanId), int(req.TargetId), items); err != nil {
			return nil, mapBatchUpsertError(err)
		}
	case "endpoint":
		if s.batchUpsert.Endpoints == nil {
			return nil, status.Error(codes.Unimplemented, errDataProxyUnimplemented)
		}
		items, err := parseEndpointItems(req.ItemsJson)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if _, _, err := s.batchUpsert.Endpoints.SaveAndSync(int(req.ScanId), int(req.TargetId), items); err != nil {
			return nil, mapBatchUpsertError(err)
		}
	case "port", "host_port":
		if s.batchUpsert.HostPorts == nil {
			return nil, status.Error(codes.Unimplemented, errDataProxyUnimplemented)
		}
		items, err := parseHostPortItems(req.ItemsJson)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if _, _, err := s.batchUpsert.HostPorts.SaveAndSync(int(req.ScanId), int(req.TargetId), items); err != nil {
			return nil, mapBatchUpsertError(err)
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported dataType: %s", req.DataType)
	}

	return &runtimev1.BatchUpsertAssetsResponse{Accepted: int32(len(req.ItemsJson))}, nil
}

func mapBatchUpsertError(err error) error {
	switch {
	case errors.Is(err, snapshotapp.ErrScanNotFoundForSnapshot):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, snapshotapp.ErrTargetMismatch):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, snapshotapp.ErrInvalidTargetType):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func (s *AgentDataProxyService) requireAgentKey(ctx context.Context) error {
	if s == nil || s.agentFinder == nil {
		return nil
	}
	agentKey, ok := grpcauth.ReadIncomingToken(ctx, grpcauth.AgentKeyHeader)
	if !ok {
		return grpcauth.MapError(grpcauth.ErrInvalidAgentKey)
	}
	agent, err := s.agentFinder.FindByAPIKey(ctx, agentKey)
	if err != nil || agent == nil {
		return grpcauth.MapError(grpcauth.ErrInvalidAgentKey)
	}
	return nil
}
