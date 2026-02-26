package service

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	runtimev1 "github.com/yyhuni/lunafox/server/internal/grpc/runtime/v1/gen"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestDataProxyGetProviderConfig(t *testing.T) {
	provider := &providerConfigStub{content: "sources:\n- foo"}
	svc := NewAgentDataProxyServiceWithDeps(provider, &wordlistRuntimeStub{})

	resp, err := svc.GetProviderConfig(context.Background(), &runtimev1.GetProviderConfigRequest{
		ScanId:   3,
		ToolName: "subfinder",
	})
	if err != nil {
		t.Fatalf("get provider config failed: %v", err)
	}
	if provider.lastScanID != 3 || provider.lastTool != "subfinder" {
		t.Fatalf("unexpected provider call args: scan=%d tool=%q", provider.lastScanID, provider.lastTool)
	}
	if resp.Content != "sources:\n- foo" {
		t.Fatalf("unexpected content: %q", resp.Content)
	}
}

func TestDataProxyGetWordlistMeta(t *testing.T) {
	wordlists := &wordlistRuntimeStub{
		wordlist: &catalogapp.Wordlist{
			ID:          10,
			Name:        "subs.txt",
			FilePath:    "/opt/lunafox/wordlists/subs.txt",
			FileHash:    "abc",
			FileSize:    1234,
			LineCount:   99,
			Description: "test",
		},
	}
	svc := NewAgentDataProxyServiceWithDeps(&providerConfigStub{}, wordlists)

	resp, err := svc.GetWordlistMeta(context.Background(), &runtimev1.GetWordlistMetaRequest{
		WordlistName: "subs.txt",
	})
	if err != nil {
		t.Fatalf("get wordlist meta failed: %v", err)
	}
	if resp.Name != "subs.txt" || resp.FileHash != "abc" || resp.LineCount != 99 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestDataProxyDownloadWordlist(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wl.txt")
	if err := os.WriteFile(filePath, []byte("a\nb\nc\n"), 0600); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}

	wordlists := &wordlistRuntimeStub{filePath: filePath}
	svc := NewAgentDataProxyServiceWithDeps(&providerConfigStub{}, wordlists)

	stream := &fakeDownloadStream{ctx: context.Background()}
	if err := svc.DownloadWordlist(&runtimev1.DownloadWordlistRequest{WordlistName: "wl.txt"}, stream); err != nil {
		t.Fatalf("download failed: %v", err)
	}

	if len(stream.chunks) == 0 {
		t.Fatalf("expected chunks")
	}
	all := make([]byte, 0)
	for _, chunk := range stream.chunks {
		all = append(all, chunk.Data...)
	}
	if string(all) != "a\nb\nc\n" {
		t.Fatalf("unexpected downloaded data: %q", string(all))
	}
}

func TestDataProxyMissingDepsReturnsUnimplemented(t *testing.T) {
	svc := NewAgentDataProxyService()
	_, err := svc.GetProviderConfig(context.Background(), &runtimev1.GetProviderConfigRequest{})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("expected unimplemented, got=%v", err)
	}
}

func TestDataProxyBatchUpsertAssetsRoutesByType(t *testing.T) {
	subdomains := &subdomainSnapshotRuntimeStub{}
	websites := &websiteSnapshotRuntimeStub{}
	endpoints := &endpointSnapshotRuntimeStub{}
	hostPorts := &hostPortSnapshotRuntimeStub{}
	svc := NewAgentDataProxyServiceWithDeps(
		&providerConfigStub{},
		&wordlistRuntimeStub{},
		AssetBatchUpsertRuntimes{
			Subdomains: subdomains,
			Websites:   websites,
			Endpoints:  endpoints,
			HostPorts:  hostPorts,
		},
	)

	cases := []struct {
		name     string
		dataType string
		items    []string
		assert   func(t *testing.T)
	}{
		{
			name:     "subdomain",
			dataType: "subdomain",
			items:    []string{`{"name":"api.example.com"}`},
			assert: func(t *testing.T) {
				if len(subdomains.lastItems) != 1 || subdomains.lastItems[0].Name != "api.example.com" {
					t.Fatalf("unexpected subdomain payload: %+v", subdomains.lastItems)
				}
			},
		},
		{
			name:     "website",
			dataType: "website",
			items:    []string{`{"url":"https://example.com","host":"example.com","title":"Example"}`},
			assert: func(t *testing.T) {
				if len(websites.lastItems) != 1 || websites.lastItems[0].URL != "https://example.com" {
					t.Fatalf("unexpected website payload: %+v", websites.lastItems)
				}
			},
		},
		{
			name:     "endpoint",
			dataType: "endpoint",
			items:    []string{`{"url":"https://example.com/login","host":"example.com","title":"Login"}`},
			assert: func(t *testing.T) {
				if len(endpoints.lastItems) != 1 || endpoints.lastItems[0].URL != "https://example.com/login" {
					t.Fatalf("unexpected endpoint payload: %+v", endpoints.lastItems)
				}
			},
		},
		{
			name:     "port",
			dataType: "port",
			items:    []string{`{"host":"example.com","ip":"1.2.3.4","port":443}`},
			assert: func(t *testing.T) {
				if len(hostPorts.lastItems) != 1 || hostPorts.lastItems[0].Port != 443 {
					t.Fatalf("unexpected host port payload: %+v", hostPorts.lastItems)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := svc.BatchUpsertAssets(context.Background(), &runtimev1.BatchUpsertAssetsRequest{
				ScanId:    12,
				TargetId:  34,
				DataType:  tc.dataType,
				ItemsJson: tc.items,
			})
			if err != nil {
				t.Fatalf("batch upsert failed: %v", err)
			}
			if resp.GetAccepted() != int32(len(tc.items)) {
				t.Fatalf("unexpected accepted count: %d", resp.GetAccepted())
			}
			tc.assert(t)
		})
	}
}

func TestDataProxyBatchUpsertAssetsErrorMapping(t *testing.T) {
	t.Run("unsupported data type", func(t *testing.T) {
		svc := NewAgentDataProxyServiceWithDeps(
			&providerConfigStub{},
			&wordlistRuntimeStub{},
			AssetBatchUpsertRuntimes{Subdomains: &subdomainSnapshotRuntimeStub{}},
		)
		_, err := svc.BatchUpsertAssets(context.Background(), &runtimev1.BatchUpsertAssetsRequest{
			ScanId:    1,
			TargetId:  2,
			DataType:  "directory",
			ItemsJson: []string{`{"url":"https://example.com"}`},
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		svc := NewAgentDataProxyServiceWithDeps(
			&providerConfigStub{},
			&wordlistRuntimeStub{},
			AssetBatchUpsertRuntimes{Subdomains: &subdomainSnapshotRuntimeStub{}},
		)
		_, err := svc.BatchUpsertAssets(context.Background(), &runtimev1.BatchUpsertAssetsRequest{
			ScanId:    1,
			TargetId:  2,
			DataType:  "subdomain",
			ItemsJson: []string{`{"name":`},
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
	})

	t.Run("scan not found", func(t *testing.T) {
		svc := NewAgentDataProxyServiceWithDeps(
			&providerConfigStub{},
			&wordlistRuntimeStub{},
			AssetBatchUpsertRuntimes{
				Subdomains: &subdomainSnapshotRuntimeStub{err: snapshotapp.ErrScanNotFoundForSnapshot},
			},
		)
		_, err := svc.BatchUpsertAssets(context.Background(), &runtimev1.BatchUpsertAssetsRequest{
			ScanId:    1,
			TargetId:  2,
			DataType:  "subdomain",
			ItemsJson: []string{`{"name":"a.example.com"}`},
		})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("expected not found, got=%v", err)
		}
	})

	t.Run("target mismatch", func(t *testing.T) {
		svc := NewAgentDataProxyServiceWithDeps(
			&providerConfigStub{},
			&wordlistRuntimeStub{},
			AssetBatchUpsertRuntimes{
				Subdomains: &subdomainSnapshotRuntimeStub{err: snapshotapp.ErrTargetMismatch},
			},
		)
		_, err := svc.BatchUpsertAssets(context.Background(), &runtimev1.BatchUpsertAssetsRequest{
			ScanId:    1,
			TargetId:  2,
			DataType:  "subdomain",
			ItemsJson: []string{`{"name":"a.example.com"}`},
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got=%v", err)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		svc := NewAgentDataProxyServiceWithDeps(
			&providerConfigStub{},
			&wordlistRuntimeStub{},
			AssetBatchUpsertRuntimes{
				Subdomains: &subdomainSnapshotRuntimeStub{err: errors.New("boom")},
			},
		)
		_, err := svc.BatchUpsertAssets(context.Background(), &runtimev1.BatchUpsertAssetsRequest{
			ScanId:    1,
			TargetId:  2,
			DataType:  "subdomain",
			ItemsJson: []string{`{"name":"a.example.com"}`},
		})
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected internal, got=%v", err)
		}
	})

	t.Run("missing batch deps", func(t *testing.T) {
		svc := NewAgentDataProxyServiceWithDeps(&providerConfigStub{}, &wordlistRuntimeStub{})
		_, err := svc.BatchUpsertAssets(context.Background(), &runtimev1.BatchUpsertAssetsRequest{
			ScanId:    1,
			TargetId:  2,
			DataType:  "subdomain",
			ItemsJson: []string{`{"name":"a.example.com"}`},
		})
		if status.Code(err) != codes.Unimplemented {
			t.Fatalf("expected unimplemented, got=%v", err)
		}
	})
}

type providerConfigStub struct {
	content    string
	err        error
	lastScanID int
	lastTool   string
}

func (s *providerConfigStub) GetProviderConfig(scanID int, toolName string) (string, error) {
	s.lastScanID = scanID
	s.lastTool = toolName
	if s.err != nil {
		return "", s.err
	}
	return s.content, nil
}

type wordlistRuntimeStub struct {
	wordlist *catalogapp.Wordlist
	filePath string
	err      error
}

func (s *wordlistRuntimeStub) GetByName(_ string) (*catalogapp.Wordlist, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.wordlist == nil {
		return nil, catalogapp.ErrWordlistNotFound
	}
	return s.wordlist, nil
}

func (s *wordlistRuntimeStub) GetFilePath(_ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.filePath == "" {
		return "", catalogapp.ErrWordlistNotFound
	}
	return s.filePath, nil
}

type subdomainSnapshotRuntimeStub struct {
	lastScanID int
	lastTarget int
	lastItems  []snapshotapp.SubdomainSnapshotItem
	savedCount int64
	assetCount int64
	err        error
}

func (s *subdomainSnapshotRuntimeStub) SaveAndSync(scanID int, targetID int, items []snapshotapp.SubdomainSnapshotItem) (int64, int64, error) {
	s.lastScanID = scanID
	s.lastTarget = targetID
	s.lastItems = append([]snapshotapp.SubdomainSnapshotItem(nil), items...)
	if s.err != nil {
		return 0, 0, s.err
	}
	if s.savedCount == 0 {
		s.savedCount = int64(len(items))
	}
	if s.assetCount == 0 {
		s.assetCount = int64(len(items))
	}
	return s.savedCount, s.assetCount, nil
}

type websiteSnapshotRuntimeStub struct {
	lastScanID int
	lastTarget int
	lastItems  []snapshotapp.WebsiteSnapshotItem
	err        error
}

func (s *websiteSnapshotRuntimeStub) SaveAndSync(scanID int, targetID int, items []snapshotapp.WebsiteSnapshotItem) (int64, int64, error) {
	s.lastScanID = scanID
	s.lastTarget = targetID
	s.lastItems = append([]snapshotapp.WebsiteSnapshotItem(nil), items...)
	if s.err != nil {
		return 0, 0, s.err
	}
	count := int64(len(items))
	return count, count, nil
}

type endpointSnapshotRuntimeStub struct {
	lastScanID int
	lastTarget int
	lastItems  []snapshotapp.EndpointSnapshotItem
	err        error
}

func (s *endpointSnapshotRuntimeStub) SaveAndSync(scanID int, targetID int, items []snapshotapp.EndpointSnapshotItem) (int64, int64, error) {
	s.lastScanID = scanID
	s.lastTarget = targetID
	s.lastItems = append([]snapshotapp.EndpointSnapshotItem(nil), items...)
	if s.err != nil {
		return 0, 0, s.err
	}
	count := int64(len(items))
	return count, count, nil
}

type hostPortSnapshotRuntimeStub struct {
	lastScanID int
	lastTarget int
	lastItems  []snapshotapp.HostPortSnapshotItem
	err        error
}

func (s *hostPortSnapshotRuntimeStub) SaveAndSync(scanID int, targetID int, items []snapshotapp.HostPortSnapshotItem) (int64, int64, error) {
	s.lastScanID = scanID
	s.lastTarget = targetID
	s.lastItems = append([]snapshotapp.HostPortSnapshotItem(nil), items...)
	if s.err != nil {
		return 0, 0, s.err
	}
	count := int64(len(items))
	return count, count, nil
}

type fakeDownloadStream struct {
	ctx    context.Context
	chunks []*runtimev1.DownloadWordlistChunk
}

func (s *fakeDownloadStream) Context() context.Context { return s.ctx }
func (s *fakeDownloadStream) Send(chunk *runtimev1.DownloadWordlistChunk) error {
	s.chunks = append(s.chunks, chunk)
	return nil
}
func (s *fakeDownloadStream) SetHeader(metadata.MD) error  { return nil }
func (s *fakeDownloadStream) SendHeader(metadata.MD) error { return nil }
func (s *fakeDownloadStream) SetTrailer(metadata.MD)       {}
func (s *fakeDownloadStream) SendMsg(any) error            { return nil }
func (s *fakeDownloadStream) RecvMsg(any) error            { return io.EOF }
