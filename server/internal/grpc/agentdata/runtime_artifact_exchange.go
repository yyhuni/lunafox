package agentdata

// This file contains the bounded, task-scoped Nuclei template exchange.  It
// intentionally has no persistence hook: the returned object is an
// exchange-local read boundary and is released when the bidi RPC ends.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
	"golang.org/x/text/unicode/norm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

const (
	nucleiTemplateManifestMaxEntries = 100_000
	nucleiTemplateManifestMaxBytes   = uint64(512 << 20)
	nucleiTemplateMaxYAMLBytes       = uint64(16 << 20)
	runtimeExchangeFrameMaxBytes     = 128 << 10
	// Leave enough room for protobuf field overhead while keeping every
	// serialized frame below the protocol ceiling.
	runtimeExchangeBlobChunkMaxBytes = 64 << 10
	runtimeExchangeManifestBatchMax  = 256
)

// RuntimeArtifactManifestEntry is the Agent-facing canonical identity of one
// enabled YAML file.  Content itself is deliberately kept private to the
// exchange boundary.
type RuntimeArtifactManifestEntry struct {
	RelativePath string
	Digest       string
	SizeBytes    uint64
}

type runtimeArtifactExchangeBoundary struct {
	entries        []RuntimeArtifactManifestEntry
	byDigest       map[string][]byte
	snapshotDigest string
}

// AuthorizedRuntimeArtifactExchange is returned only after the normal task,
// session, lease and saved-plan checks.  Its blob method reads the same
// in-memory boundary used to build the manifest, so a catalog mutation cannot
// produce a manifest/blob split.
type AuthorizedRuntimeArtifactExchange struct {
	Entries        []RuntimeArtifactManifestEntry
	SnapshotDigest string
	boundary       *runtimeArtifactExchangeBoundary
}

func (exchange *AuthorizedRuntimeArtifactExchange) blob(digest string) ([]byte, bool) {
	if exchange == nil || exchange.boundary == nil {
		return nil, false
	}
	data, ok := exchange.boundary.byDigest[digest]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), data...), true
}

func (exchange *AuthorizedRuntimeArtifactExchange) close() {
	if exchange == nil || exchange.boundary == nil {
		return
	}
	exchange.boundary.byDigest = nil
	exchange.boundary.entries = nil
	exchange.boundary = nil
	exchange.Entries = nil
}

// RuntimeArtifactExchangeResolver is intentionally optional on the generic
// resolver interface.  This lets non-template artifact fixtures continue to
// use the typed one-way streams while making the new exchange the only route
// for the nucleiTemplates capability.
type RuntimeArtifactExchangeResolver interface {
	AuthorizeRuntimeArtifactExchange(context.Context, AgentExecutionLease, *agentdatav1.RuntimeArtifactExchangeBegin) (*AuthorizedRuntimeArtifactExchange, error)
}

func validateRuntimeArtifactExchangeBegin(begin *agentdatav1.RuntimeArtifactExchangeBegin) error {
	if begin == nil {
		return errors.New("runtime artifact exchange begin is required")
	}
	if err := validateArtifactScope(begin.GetTask(), begin.GetExecution()); err != nil {
		return err
	}
	if begin.GetArtifactId() != "nucleiTemplates" {
		return errors.New("runtime artifact exchange artifact_id is unsupported")
	}
	if !runtimeExchangeRevisionAccepted(begin.GetCompatibilityRevision()) {
		return errors.New("runtime artifact exchange compatibility revision is invalid")
	}
	return nil
}

func buildNucleiTemplateExchange(templates []domain.POC) (*runtimeArtifactExchangeBoundary, error) {
	if len(templates) == 0 {
		return nil, ErrNoEnabledNucleiTemplates
	}
	if len(templates) > nucleiTemplateManifestMaxEntries {
		return nil, fmt.Errorf("Nuclei template catalog exceeds the 100000-entry limit")
	}
	ordered := append([]domain.POC(nil), templates...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].RelativePath != ordered[j].RelativePath {
			return ordered[i].RelativePath < ordered[j].RelativePath
		}
		return ordered[i].TemplateID < ordered[j].TemplateID
	})

	entries := make([]RuntimeArtifactManifestEntry, 0, len(ordered))
	byDigest := make(map[string][]byte, len(ordered))
	seenPaths := make(map[string]string, len(ordered))
	var total uint64
	for _, template := range ordered {
		content := []byte(template.Content)
		if err := validateNucleiTemplateEntryForExchange(template, content); err != nil {
			return nil, err
		}
		if uint64(len(content)) > nucleiTemplateMaxYAMLBytes {
			return nil, fmt.Errorf("Nuclei template exceeds the 16 MiB per-YAML limit")
		}
		if uint64(len(content)) > math.MaxUint64-total {
			return nil, errors.New("Nuclei template catalog size overflows")
		}
		total += uint64(len(content))
		if total > nucleiTemplateManifestMaxBytes {
			return nil, fmt.Errorf("Nuclei template catalog exceeds the 512 MiB limit")
		}
		key := norm.NFC.String(strings.ToLower(template.RelativePath))
		if previous, exists := seenPaths[key]; exists {
			return nil, fmt.Errorf("Nuclei template path collides with %q", previous)
		}
		seenPaths[key] = template.RelativePath
		digestBytes := sha256.Sum256(content)
		digest := "sha256:" + hex.EncodeToString(digestBytes[:])
		if strings.ToLower(template.ContentSHA256) != hex.EncodeToString(digestBytes[:]) {
			return nil, errors.New("enabled Nuclei template content digest mismatch")
		}
		entries = append(entries, RuntimeArtifactManifestEntry{RelativePath: template.RelativePath, Digest: digest, SizeBytes: uint64(len(content))})
		if _, exists := byDigest[digest]; !exists {
			byDigest[digest] = append([]byte(nil), content...)
		}
	}
	if len(entries) == 0 {
		return nil, ErrNoEnabledNucleiTemplates
	}
	canonical := make([]canonicalManifestEntry, len(entries))
	for index, entry := range entries {
		canonical[index] = canonicalManifestEntry{RelativePath: entry.RelativePath, SHA256Digest: entry.Digest, SizeBytes: entry.SizeBytes}
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("encode canonical Nuclei template manifest: %w", err)
	}
	snapshot := sha256.Sum256(payload)
	return &runtimeArtifactExchangeBoundary{entries: entries, byDigest: byDigest, snapshotDigest: "sha256:" + hex.EncodeToString(snapshot[:])}, nil
}

type canonicalManifestEntry struct {
	RelativePath string `json:"relativePath"`
	SHA256Digest string `json:"sha256Digest"`
	SizeBytes    uint64 `json:"sizeBytes"`
}

func validateNucleiTemplateEntryForExchange(template domain.POC, content []byte) error {
	if !template.IsEnabled {
		return errors.New("enabled Nuclei template metadata is disabled")
	}
	if template.TemplateID == "" || template.TemplateID != strings.TrimSpace(template.TemplateID) || strings.IndexFunc(template.TemplateID, unicode.IsControl) >= 0 {
		return errors.New("enabled Nuclei template metadata has an invalid template ID")
	}
	if !isSafeNucleiTemplateRelativePathExchange(template.RelativePath) {
		return errors.New("enabled Nuclei template metadata has an unsafe relative path")
	}
	if len(content) == 0 || !utf8.Valid(content) || len(template.ContentSHA256) != 64 || !isLowerHexDigest(template.ContentSHA256) {
		return errors.New("enabled Nuclei template metadata is invalid")
	}
	if err := validateSingleNucleiYAML(content, template.TemplateID); err != nil {
		return fmt.Errorf("enabled Nuclei template YAML is invalid: %w", err)
	}
	return nil
}

func isLowerHexDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

// validateSingleNucleiYAML performs the exchange-time syntax and identity
// check.  The catalog repository already verifies the stored byte digest, but
// the exchange boundary must also reject malformed or concatenated documents
// before exposing any manifest frame to an Agent.
func validateSingleNucleiYAML(content []byte, expectedID string) error {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		return err
	}
	if len(document) == 0 {
		return errors.New("template document is empty")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("template contains multiple YAML documents")
		}
		return err
	}
	value, ok := document["id"].(string)
	if !ok || strings.TrimSpace(value) == "" || value != expectedID {
		return errors.New("template id does not match catalog identity")
	}
	return nil
}

func isSafeNucleiTemplateRelativePathExchange(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || strings.IndexFunc(value, func(r rune) bool { return unicode.IsSpace(r) }) >= 0 && (unicode.IsSpace([]rune(value)[0]) || unicode.IsSpace([]rune(value)[len([]rune(value))-1])) {
		return false
	}
	if !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') || strings.ContainsRune(value, '\\') || strings.HasPrefix(value, "/") {
		return false
	}
	cleaned := path.Clean(value)
	if cleaned != value || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return false
	}
	ext := path.Ext(cleaned)
	return ext == ".yaml" || ext == ".yml"
}

// ValidateRuntimeArtifactExchangeFrame is shared by the service and tests so
// a peer cannot accidentally bypass the serialized frame ceiling.
func ValidateRuntimeArtifactExchangeFrame(message proto.Message) error {
	if message == nil {
		return status.Error(codes.InvalidArgument, "runtime artifact exchange frame is required")
	}
	if size := proto.Size(message); size > runtimeExchangeFrameMaxBytes {
		return status.Error(codes.ResourceExhausted, "runtime artifact exchange frame exceeds 128 KiB")
	}
	return nil
}
