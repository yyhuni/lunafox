package infrastructure

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type planTaskExactPackageReader struct {
	query installedengines.Query
}

func NewPlanTaskExactPackageReader(query installedengines.Query) (scanapp.PlanTaskPackageReader, error) {
	if query == nil {
		return nil, fmt.Errorf("installed Engine Package v2 query is required")
	}
	return &planTaskExactPackageReader{query: query}, nil
}

func (reader *planTaskExactPackageReader) LoadExactPackage(ctx context.Context, identity scanapp.PlanTaskPackageIdentity) (scanapp.PlanTaskPackage, error) {
	if reader == nil || reader.query == nil {
		return scanapp.PlanTaskPackage{}, fmt.Errorf("installed Engine Package v2 query is not configured")
	}
	if identity.EngineID == "" || identity.EngineID != strings.TrimSpace(identity.EngineID) {
		return scanapp.PlanTaskPackage{}, fmt.Errorf("exact package engine identity is required")
	}
	if err := ctx.Err(); err != nil {
		return scanapp.PlanTaskPackage{}, err
	}
	digest, err := ociartifact.ParsePackageDigest(identity.PackageDigest)
	if err != nil {
		return scanapp.PlanTaskPackage{}, err
	}
	resolved, err := reader.query.GetInstalledEnginePackage(identity.EngineID)
	if err != nil {
		return scanapp.PlanTaskPackage{}, err
	}
	if err := ctx.Err(); err != nil {
		return scanapp.PlanTaskPackage{}, err
	}
	if resolved.Registration.EngineID != identity.EngineID || resolved.Registration.PackageDigest != string(digest) {
		return scanapp.PlanTaskPackage{}, fmt.Errorf("current installed package does not match pinned package identity")
	}
	packageManifest := resolved.Layout.Definition.PackageManifest
	definition := resolved.Layout.Definition.EngineDefinition
	if packageManifest.EngineID != identity.EngineID || definition.EngineID != identity.EngineID {
		return scanapp.PlanTaskPackage{}, fmt.Errorf("exact package manifests do not match pinned engine identity")
	}
	return scanapp.PlanTaskPackage{
		Identity:         identity,
		PackageVersion:   packageManifest.EngineVersion,
		Definition:       definition,
		RuntimeImageRefs: append([]string(nil), packageManifest.RuntimeImage.Refs...),
	}, nil
}

type PlanTaskWordlistCatalog interface {
	GetByID(id int) (*catalogdomain.Wordlist, error)
}

type configResourceResolver struct {
	catalog PlanTaskWordlistCatalog
}

func NewConfigResourceResolver(catalog PlanTaskWordlistCatalog) (scanapp.ConfigResourceResolver, error) {
	if catalog == nil {
		return nil, fmt.Errorf("wordlist catalog is required")
	}
	return &configResourceResolver{catalog: catalog}, nil
}

func (resolver *configResourceResolver) ResolveConfigResource(ctx context.Context, request scanapp.ConfigResourceResolveRequest) (scanapp.PlanTaskWordlist, error) {
	if resolver == nil || resolver.catalog == nil {
		return scanapp.PlanTaskWordlist{}, scanapp.NewConfigResourceInternalError(
			request.Field, request.ResourceKind, request.ResourceName, errors.New("wordlist catalog is not configured"),
		)
	}
	if err := ctx.Err(); err != nil {
		return scanapp.PlanTaskWordlist{}, err
	}
	if request.ResourceKind != "wordlist" {
		return scanapp.PlanTaskWordlist{}, scanapp.NewConfigResourceInternalError(
			request.Field, request.ResourceKind, request.ResourceName, fmt.Errorf("unsupported config resource kind %q", request.ResourceKind),
		)
	}
	id, err := resourcenames.ParseWordlist(request.ResourceName)
	canonical := strings.TrimSpace(request.ResourceName)
	if err != nil || resourcenames.Wordlist(id) != canonical {
		return scanapp.PlanTaskWordlist{}, &scanapp.ConfigResourceInputError{
			Field: request.Field, ResourceKind: request.ResourceKind, ResourceName: request.ResourceName,
			Err: fmt.Errorf("wordlist name must be canonical: %w", err),
		}
	}
	wordlist, err := resolver.catalog.GetByID(id)
	if err != nil {
		if errors.Is(err, catalogdomain.ErrWordlistNotFound) || dberrors.IsRecordNotFound(err) {
			return scanapp.PlanTaskWordlist{}, scanapp.NewConfigResourceUnavailableError(request.Field, request.ResourceKind, canonical, err)
		}
		return scanapp.PlanTaskWordlist{}, scanapp.NewConfigResourceValidationUnavailableError(request.Field, request.ResourceKind, canonical, err)
	}
	if wordlist == nil || wordlist.ID != id {
		return scanapp.PlanTaskWordlist{}, scanapp.NewConfigResourceUnavailableError(
			request.Field, request.ResourceKind, canonical, errors.New("wordlist catalog identity mismatch"),
		)
	}
	metadata, err := readExactWordlistMetadata(ctx, wordlist.FilePath)
	if err != nil {
		return scanapp.PlanTaskWordlist{}, scanapp.NewConfigResourceUnavailableError(request.Field, request.ResourceKind, canonical, err)
	}
	if wordlist.FileName == "" || filepath.Base(wordlist.FilePath) != wordlist.FileName || wordlist.FileSize != metadata.SizeBytes || int64(wordlist.LineCount) != metadata.LineCount || wordlist.FileHash != strings.TrimPrefix(metadata.SHA256, "sha256:") {
		return scanapp.PlanTaskWordlist{}, scanapp.NewConfigResourceUnavailableError(
			request.Field, request.ResourceKind, canonical, errors.New("wordlist catalog metadata does not match exact file bytes"),
		)
	}
	metadata.Resource = canonical
	return metadata, nil
}

func readExactWordlistMetadata(ctx context.Context, path string) (scanapp.PlanTaskWordlist, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return scanapp.PlanTaskWordlist{}, fmt.Errorf("wordlist file path is required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return scanapp.PlanTaskWordlist{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return scanapp.PlanTaskWordlist{}, fmt.Errorf("wordlist content must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return scanapp.PlanTaskWordlist{}, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return scanapp.PlanTaskWordlist{}, err
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		return scanapp.PlanTaskWordlist{}, fmt.Errorf("wordlist file changed before reading")
	}

	hasher := sha256.New()
	counter := &wordlistLineCounter{}
	size, err := io.Copy(io.MultiWriter(hasher, counter), contextReader{ctx: ctx, reader: file})
	if err != nil {
		return scanapp.PlanTaskWordlist{}, err
	}
	if size != info.Size() || size != openedInfo.Size() {
		return scanapp.PlanTaskWordlist{}, fmt.Errorf("wordlist file changed while reading")
	}
	return scanapp.PlanTaskWordlist{
		Basename:  filepath.Base(path),
		SizeBytes: size,
		LineCount: counter.Count(),
		SHA256:    "sha256:" + hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}

type wordlistLineCounter struct {
	newlines int64
	size     int64
	lastByte byte
}

func (counter *wordlistLineCounter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	counter.newlines += int64(bytes.Count(data, []byte{'\n'}))
	counter.size += int64(len(data))
	counter.lastByte = data[len(data)-1]
	return len(data), nil
}

func (counter *wordlistLineCounter) Count() int64 {
	if counter.size == 0 {
		return 0
	}
	if counter.lastByte == '\n' {
		return counter.newlines
	}
	return counter.newlines + 1
}
