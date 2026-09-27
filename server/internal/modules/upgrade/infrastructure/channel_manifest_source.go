package infrastructure

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/contracts/preheatmanifest"
	"github.com/yyhuni/lunafox/contracts/releasemanifest"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/upgrader"
)

const (
	maxChannelRecordBytes      = 16 * 1024
	maxReleaseManifestBytes    = 4 * 1024 * 1024
	maxRuntimeCompositionBytes = 8 * 1024 * 1024
	maxPreheatManifestBytes    = 8 * 1024 * 1024
	maxPublicComposeBytes      = 16 * 1024 * 1024
	maxPublicPolicyBytes       = 2 * 1024 * 1024
	manifestFetchTimeout       = 15 * time.Second
)

var (
	channelVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-(?:alpha|beta|rc)\.[0-9]+)?$`)
	channelDigestPattern  = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type channelRecord struct {
	version      string
	manifestPath string
	manifestSHA  string
}

// ChannelManifestSource loads the candidate selected by one deployment-owned
// release channel. Browser input cannot choose the source, path, or Registry.
type ChannelManifestSource struct {
	baseURL          *url.URL
	channel          string
	cacheDir         string
	compositionCache string
	preheatCache     string
	deploymentCache  string
	client           *http.Client
	mu               sync.Mutex
}

type ChannelManifestSourceConfig struct {
	MetadataBaseURL string
	Channel         string
	DeploymentRoot  string
	HTTPClient      *http.Client
}

func NewChannelManifestSource(config ChannelManifestSourceConfig) (*ChannelManifestSource, error) {
	channel := strings.TrimSpace(config.Channel)
	if channel != "stable" && channel != "canary" {
		return nil, fmt.Errorf("release channel must be stable or canary")
	}
	baseURL, err := url.Parse(strings.TrimSpace(config.MetadataBaseURL))
	if err != nil || !baseURL.IsAbs() || baseURL.Host == "" || baseURL.RawPath != "" || baseURL.RawQuery != "" || baseURL.Fragment != "" || baseURL.User != nil {
		return nil, fmt.Errorf("release metadata base URL is invalid")
	}
	if baseURL.Scheme != "https" && (baseURL.Scheme != "http" || !isLoopbackHost(baseURL.Hostname())) {
		return nil, fmt.Errorf("release metadata base URL must use HTTPS")
	}
	root := filepath.Clean(strings.TrimSpace(config.DeploymentRoot))
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("upgrade deployment root must be absolute")
	}
	cacheDir := filepath.Join(root, upgrader.JournalDirectory, upgrader.ManifestDirectory)
	compositionCache := filepath.Join(root, upgrader.JournalDirectory, upgrader.CompositionDirectory)
	preheatCache := filepath.Join(root, upgrader.JournalDirectory, upgrader.PreheatManifestDirectory)
	deploymentCache := filepath.Join(root, upgrader.JournalDirectory, upgrader.DeploymentAssetDirectory)
	if err := ensureManifestCacheDirectory(root, cacheDir, compositionCache, preheatCache, deploymentCache); err != nil {
		return nil, err
	}
	client := &http.Client{}
	if config.HTTPClient != nil {
		*client = *config.HTTPClient
	}
	client.Timeout = manifestFetchTimeout
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return fmt.Errorf("release metadata redirect limit exceeded")
		}
		if request.URL.Scheme != baseURL.Scheme || !strings.EqualFold(request.URL.Host, baseURL.Host) {
			return fmt.Errorf("release metadata redirect changed origin")
		}
		return nil
	}
	return &ChannelManifestSource{baseURL: baseURL, channel: channel, cacheDir: cacheDir, compositionCache: compositionCache, preheatCache: preheatCache, deploymentCache: deploymentCache, client: client}, nil
}

// Load fetches and validates the current channel alias. A failed refresh never
// falls back to cached bytes because that would make the UI report stale data
// as the current release candidate.
func (source *ChannelManifestSource) Load() (*releasemanifest.Manifest, error) {
	if source == nil || source.baseURL == nil || source.client == nil {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("channel manifest source is not configured"))
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	channelBytes, err := source.fetch("channels/"+source.channel+".env", maxChannelRecordBytes)
	if err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	record, err := parseChannelRecord(channelBytes)
	if err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	manifestBytes, err := source.fetch(record.manifestPath, maxReleaseManifestBytes)
	if err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	hash := sha256.Sum256(manifestBytes)
	actualSHA := fmt.Sprintf("%x", hash[:])
	if actualSHA != record.manifestSHA {
		return nil, domain.NewManifestDigestMismatch("sha256:"+record.manifestSHA, "sha256:"+actualSHA)
	}
	manifest, err := releasemanifest.Parse(manifestBytes)
	legacyAlpha114 := false
	if err != nil {
		// Deployment channel parsing may only admit the policy-pinned alpha.114
		// bytes or the exact registered alpha.164 bridge profile. The ordinary
		// parser remains strict for every other manifest shape.
		manifest, err = releasemanifest.ParseLegacyCompatible(manifestBytes)
		if err != nil {
			return nil, domain.WrapManifestInvalid(err)
		}
		legacyAlpha114 = !manifest.HasRuntimeComposition()
	}
	if "v"+manifest.ReleaseVersion != record.version {
		return nil, domain.NewManifestIdentityMismatch("lunafox-"+strings.TrimPrefix(record.version, "v"), manifest.Upgrade.ManifestID)
	}
	if _, err := domain.TargetFromManifest(manifest); err != nil {
		return nil, err
	}
	if legacyAlpha114 {
		if err := source.persist(manifest.Digest(), manifestBytes); err != nil {
			return nil, domain.WrapManifestInvalid(err)
		}
		return manifest, nil
	}
	// Release assets are version-isolated: a channel record points at
	// manifests/<tag>.yaml while its composition lives at
	// manifests/<tag>/runtime-composition.json. The asset basename is already
	// strict-validated by releasemanifest.Parse; the version directory comes
	// from the canonical channel record rather than from a client-provided path.
	compositionPath := path.Join("manifests", record.version, manifest.RuntimeComposition.Asset)
	compositionBytes, err := source.fetch(compositionPath, maxRuntimeCompositionBytes)
	if err != nil {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("fetch runtime composition: %w", err))
	}
	if err := upgrader.ValidateRuntimeCompositionAsset(compositionBytes, manifest, manifest.Digest(), manifest.RuntimeComposition.SHA256); err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	if err := source.persistComposition(manifest.RuntimeComposition.SHA256, compositionBytes); err != nil {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("cache runtime composition: %w", err))
	}
	preheatPath := path.Join("manifests", record.version, "preheat-manifest.json")
	preheatBytes, err := source.fetch(preheatPath, maxPreheatManifestBytes)
	if err != nil {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("fetch preheat manifest: %w", err))
	}
	if err := validatePreheatReleaseBinding(preheatBytes, manifest, compositionBytes); err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	composeBytes, err := source.fetch(path.Join("manifests", record.version, "compose.yaml"), maxPublicComposeBytes)
	if err != nil {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("fetch deployment Compose: %w", err))
	}
	policyBytes, err := source.fetch(path.Join("manifests", record.version, "third-party-image-policy.json"), maxPublicPolicyBytes)
	if err != nil {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("fetch third-party image policy: %w", err))
	}
	if err := validateDeploymentAssetBinding(composeBytes, policyBytes, preheatBytes); err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	if err := source.persistPreheat(manifest.Digest(), preheatBytes); err != nil {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("cache preheat manifest: %w", err))
	}
	if err := source.persistDeploymentAssets(manifest.Digest(), composeBytes, policyBytes); err != nil {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("cache deployment assets: %w", err))
	}
	if err := source.persist(manifest.Digest(), manifestBytes); err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	return manifest, nil
}

// LoadTarget reloads the immutable bytes bound to an existing Operation.
func (source *ChannelManifestSource) LoadTarget(digest string) (*releasemanifest.Manifest, error) {
	if source == nil {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("channel manifest source is not configured"))
	}
	filePath, err := source.cachePath(digest)
	if err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	info, err := os.Lstat(filePath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("cached release manifest is unavailable"))
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	defer func() { _ = file.Close() }()
	raw, err := readBounded(file, maxReleaseManifestBytes)
	if err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	manifest, err := releasemanifest.Parse(raw)
	if err != nil {
		// Keep retries compatible with the same bounded legacy parser used by the
		// channel refresh path. It rejects every unregistered partial manifest.
		manifest, err = releasemanifest.ParseLegacyCompatible(raw)
		if err != nil {
			return nil, domain.WrapManifestInvalid(err)
		}
	}
	if manifest.Digest() != digest {
		return nil, domain.NewManifestDigestMismatch(digest, manifest.Digest())
	}
	if manifest.HasRuntimeComposition() {
		preheatPath, pathErr := source.preheatPath(digest)
		if pathErr != nil {
			return nil, domain.WrapManifestInvalid(pathErr)
		}
		preheatBytes, readErr := readRegularCachedFile(preheatPath, maxPreheatManifestBytes, 0o600)
		if readErr != nil {
			return nil, domain.WrapManifestInvalid(fmt.Errorf("cached preheat manifest is unavailable: %w", readErr))
		}
		if err := validatePreheatReleaseBinding(preheatBytes, manifest, nil); err != nil {
			return nil, domain.WrapManifestInvalid(err)
		}
		compositionPath, compositionErr := source.compositionPath(manifest.RuntimeComposition.SHA256)
		if compositionErr != nil {
			return nil, domain.WrapManifestInvalid(compositionErr)
		}
		compositionBytes, compositionReadErr := readRegularCachedFile(compositionPath, maxRuntimeCompositionBytes, 0o600)
		if compositionReadErr != nil {
			return nil, domain.WrapManifestInvalid(fmt.Errorf("cached runtime composition is unavailable: %w", compositionReadErr))
		}
		if err := upgrader.ValidateRuntimeCompositionAsset(compositionBytes, manifest, manifest.Digest(), manifest.RuntimeComposition.SHA256); err != nil {
			return nil, domain.WrapManifestInvalid(fmt.Errorf("validate cached runtime composition: %w", err))
		}
		composeBytes, policyBytes, assetErr := source.readDeploymentAssets(digest)
		if assetErr != nil {
			return nil, domain.WrapManifestInvalid(assetErr)
		}
		if err := validateDeploymentAssetBinding(composeBytes, policyBytes, preheatBytes); err != nil {
			return nil, domain.WrapManifestInvalid(err)
		}
	}
	return manifest, nil
}

func (source *ChannelManifestSource) fetch(relative string, limit int64) ([]byte, error) {
	if relative == "" || strings.HasPrefix(relative, "/") || strings.Contains(relative, "..") {
		return nil, fmt.Errorf("release metadata path is unsafe")
	}
	target := *source.baseURL
	target.Path = path.Join(strings.TrimSuffix(source.baseURL.Path, "/"), relative)
	request, err := http.NewRequest(http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/octet-stream")
	request.Header.Set("User-Agent", "lunafox-server-upgrade")
	response, err := source.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch release metadata: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch release metadata: HTTP %d", response.StatusCode)
	}
	if response.ContentLength > limit {
		return nil, fmt.Errorf("release metadata exceeds %d bytes", limit)
	}
	return readBounded(response.Body, limit)
}

func parseChannelRecord(raw []byte) (channelRecord, error) {
	if bytes.Contains(raw, []byte{'\r'}) {
		return channelRecord{}, fmt.Errorf("release channel contains CRLF")
	}
	values := make(map[string]string, 4)
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || key == "" || value == "" || strings.TrimSpace(key) != key || strings.TrimSpace(value) != value {
			return channelRecord{}, fmt.Errorf("release channel contains a malformed entry")
		}
		if _, duplicate := values[key]; duplicate {
			return channelRecord{}, fmt.Errorf("release channel contains duplicate key %s", key)
		}
		values[key] = value
	}
	allowed := map[string]struct{}{"SCHEMA_VERSION": {}, "VERSION": {}, "RELEASE_MANIFEST": {}, "RELEASE_MANIFEST_SHA256": {}}
	if len(values) != len(allowed) {
		return channelRecord{}, fmt.Errorf("release channel must contain exactly four fields")
	}
	for key := range values {
		if _, ok := allowed[key]; !ok {
			return channelRecord{}, fmt.Errorf("release channel contains unsupported key %s", key)
		}
	}
	if values["SCHEMA_VERSION"] != "3" {
		return channelRecord{}, fmt.Errorf("release channel must use schema v3")
	}
	version := values["VERSION"]
	if !channelVersionPattern.MatchString(version) {
		return channelRecord{}, fmt.Errorf("release channel version is invalid")
	}
	manifestPath := values["RELEASE_MANIFEST"]
	if manifestPath != "manifests/"+version+".yaml" {
		return channelRecord{}, fmt.Errorf("release channel manifest path is invalid")
	}
	manifestSHA := values["RELEASE_MANIFEST_SHA256"]
	if !channelDigestPattern.MatchString(manifestSHA) {
		return channelRecord{}, fmt.Errorf("release channel manifest digest is invalid")
	}
	return channelRecord{version: version, manifestPath: manifestPath, manifestSHA: manifestSHA}, nil
}

func (source *ChannelManifestSource) persist(digest string, raw []byte) error {
	target, err := source.cachePath(digest)
	if err != nil {
		return err
	}
	info, statErr := os.Lstat(target)
	if statErr == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("cached manifest digest path must be a regular file")
		}
		file, openErr := os.Open(target)
		if openErr != nil {
			return openErr
		}
		defer func() { _ = file.Close() }()
		openedInfo, statErr := file.Stat()
		if statErr != nil || !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
			return fmt.Errorf("cached manifest digest path changed during validation")
		}
		existing, readErr := readBounded(file, maxReleaseManifestBytes)
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(existing, raw) {
			return fmt.Errorf("cached manifest digest path contains different bytes")
		}
		return nil
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	temporary, err := os.CreateTemp(source.cacheDir, ".manifest-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := temporary.Write(raw); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return err
	}
	directory, err := os.Open(source.cacheDir)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}

func (source *ChannelManifestSource) persistComposition(digest string, raw []byte) error {
	if source == nil || source.compositionCache == "" {
		return fmt.Errorf("runtime composition cache is not configured")
	}
	if !strings.HasPrefix(digest, "sha256:") || !channelDigestPattern.MatchString(strings.TrimPrefix(digest, "sha256:")) {
		return fmt.Errorf("runtime composition digest is invalid")
	}
	target := filepath.Join(source.compositionCache, strings.TrimPrefix(digest, "sha256:")+".json")
	info, statErr := os.Lstat(target)
	if statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
			return fmt.Errorf("cached composition digest path must be a regular 0600 file")
		}
		file, openErr := os.Open(target)
		if openErr != nil {
			return openErr
		}
		defer func() { _ = file.Close() }()
		openedInfo, statErr := file.Stat()
		if statErr != nil || !openedInfo.Mode().IsRegular() || openedInfo.Mode().Perm() != 0o600 || !os.SameFile(info, openedInfo) {
			return fmt.Errorf("cached composition digest path changed during validation")
		}
		existing, readErr := readBounded(file, maxRuntimeCompositionBytes)
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(existing, raw) {
			return fmt.Errorf("cached composition digest path contains different bytes")
		}
		return nil
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	temporary, err := os.CreateTemp(source.compositionCache, ".composition-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := temporary.Write(raw); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return err
	}
	directory, err := os.Open(source.compositionCache)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}

func (source *ChannelManifestSource) cachePath(digest string) (string, error) {
	if !strings.HasPrefix(digest, "sha256:") || !channelDigestPattern.MatchString(strings.TrimPrefix(digest, "sha256:")) {
		return "", fmt.Errorf("release manifest digest is invalid")
	}
	return filepath.Join(source.cacheDir, strings.TrimPrefix(digest, "sha256:")+".yaml"), nil
}

func (source *ChannelManifestSource) preheatPath(releaseDigest string) (string, error) {
	if source == nil || source.preheatCache == "" {
		return "", fmt.Errorf("preheat manifest cache is not configured")
	}
	if !strings.HasPrefix(releaseDigest, "sha256:") || !channelDigestPattern.MatchString(strings.TrimPrefix(releaseDigest, "sha256:")) {
		return "", fmt.Errorf("release manifest digest is invalid")
	}
	return filepath.Join(source.preheatCache, strings.TrimPrefix(releaseDigest, "sha256:")+".json"), nil
}

func (source *ChannelManifestSource) compositionPath(digest string) (string, error) {
	if source == nil || source.compositionCache == "" {
		return "", fmt.Errorf("runtime composition cache is not configured")
	}
	if !strings.HasPrefix(digest, "sha256:") || !channelDigestPattern.MatchString(strings.TrimPrefix(digest, "sha256:")) {
		return "", fmt.Errorf("runtime composition digest is invalid")
	}
	return filepath.Join(source.compositionCache, strings.TrimPrefix(digest, "sha256:")+".json"), nil
}

func (source *ChannelManifestSource) deploymentAssetsPath(releaseDigest string) (string, error) {
	if source == nil || source.deploymentCache == "" {
		return "", fmt.Errorf("deployment asset cache is not configured")
	}
	if !strings.HasPrefix(releaseDigest, "sha256:") || !channelDigestPattern.MatchString(strings.TrimPrefix(releaseDigest, "sha256:")) {
		return "", fmt.Errorf("release manifest digest is invalid")
	}
	return filepath.Join(source.deploymentCache, strings.TrimPrefix(releaseDigest, "sha256:")), nil
}

func (source *ChannelManifestSource) persistPreheat(releaseDigest string, raw []byte) error {
	target, err := source.preheatPath(releaseDigest)
	if err != nil {
		return err
	}
	if info, statErr := os.Lstat(target); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
			return fmt.Errorf("cached preheat manifest path must be a regular 0600 file")
		}
		file, openErr := os.Open(target)
		if openErr != nil {
			return openErr
		}
		defer func() { _ = file.Close() }()
		openedInfo, statErr := file.Stat()
		if statErr != nil || !openedInfo.Mode().IsRegular() || openedInfo.Mode().Perm() != 0o600 || !os.SameFile(info, openedInfo) {
			return fmt.Errorf("cached preheat manifest path changed during validation")
		}
		existing, readErr := readBounded(file, maxPreheatManifestBytes)
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(existing, raw) {
			return fmt.Errorf("cached preheat manifest path contains different bytes")
		}
		return nil
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	temporary, err := os.CreateTemp(source.preheatCache, ".preheat-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := temporary.Write(raw); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return err
	}
	directory, err := os.Open(source.preheatCache)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}

// persistDeploymentAssets stores the target Compose and policy as one
// digest-addressed directory. The directory rename makes the pair visible
// together to the host upgrader; an existing pair is immutable and must match
// byte-for-byte.
func (source *ChannelManifestSource) persistDeploymentAssets(releaseDigest string, compose, policy []byte) error {
	target, err := source.deploymentAssetsPath(releaseDigest)
	if err != nil {
		return err
	}
	if info, statErr := os.Lstat(target); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
			return fmt.Errorf("cached deployment asset path must be a regular 0700 directory")
		}
		cachedCompose, readErr := readRegularCachedFile(filepath.Join(target, "compose.yaml"), maxPublicComposeBytes, 0o600)
		if readErr != nil {
			return readErr
		}
		cachedPolicy, readErr := readRegularCachedFile(filepath.Join(target, "third-party-image-policy.json"), maxPublicPolicyBytes, 0o600)
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(cachedCompose, compose) || !bytes.Equal(cachedPolicy, policy) {
			return fmt.Errorf("cached deployment assets contain different bytes")
		}
		return nil
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	parent := filepath.Dir(target)
	staging, err := os.MkdirTemp(parent, ".deployment-assets-*")
	if err != nil {
		return err
	}
	removeStaging := true
	defer func() {
		if removeStaging {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := os.Chmod(staging, 0o700); err != nil {
		return err
	}
	if err := writePrivateAsset(filepath.Join(staging, "compose.yaml"), compose, maxPublicComposeBytes); err != nil {
		return err
	}
	if err := writePrivateAsset(filepath.Join(staging, "third-party-image-policy.json"), policy, maxPublicPolicyBytes); err != nil {
		return err
	}
	if err := os.Rename(staging, target); err != nil {
		if !os.IsExist(err) {
			return err
		}
		cachedCompose, readErr := readRegularCachedFile(filepath.Join(target, "compose.yaml"), maxPublicComposeBytes, 0o600)
		if readErr != nil {
			return readErr
		}
		cachedPolicy, readErr := readRegularCachedFile(filepath.Join(target, "third-party-image-policy.json"), maxPublicPolicyBytes, 0o600)
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(cachedCompose, compose) || !bytes.Equal(cachedPolicy, policy) {
			return fmt.Errorf("cached deployment assets contain different bytes")
		}
	} else {
		removeStaging = false
		directory, syncErr := os.Open(parent)
		if syncErr != nil {
			return syncErr
		}
		return errors.Join(directory.Sync(), directory.Close())
	}
	return nil
}

func (source *ChannelManifestSource) readDeploymentAssets(releaseDigest string) ([]byte, []byte, error) {
	root, err := source.deploymentAssetsPath(releaseDigest)
	if err != nil {
		return nil, nil, err
	}
	info, err := os.Lstat(root)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return nil, nil, fmt.Errorf("cached deployment assets are unavailable")
	}
	compose, err := readRegularCachedFile(filepath.Join(root, "compose.yaml"), maxPublicComposeBytes, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("read cached deployment Compose: %w", err)
	}
	policy, err := readRegularCachedFile(filepath.Join(root, "third-party-image-policy.json"), maxPublicPolicyBytes, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("read cached third-party policy: %w", err)
	}
	return compose, policy, nil
}

func writePrivateAsset(target string, raw []byte, limit int64) error {
	if len(raw) == 0 || int64(len(raw)) > limit {
		return fmt.Errorf("deployment asset exceeds %d bytes", limit)
	}
	temporary, err := os.OpenFile(target+".tmp", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(temporaryPath)
		}
	}()
	if _, err := temporary.Write(raw); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temporaryPath, 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func validateDeploymentAssetBinding(compose, policy, preheat []byte) error {
	manifest, err := preheatmanifest.Parse(preheat)
	if err != nil {
		return fmt.Errorf("validate preheat manifest for deployment assets: %w", err)
	}
	digest := func(raw []byte) string {
		sum := sha256.Sum256(raw)
		return "sha256:" + fmt.Sprintf("%x", sum[:])
	}
	if digest(compose) != manifest.Release.ComposeDigest {
		return fmt.Errorf("deployment Compose bytes do not match preheat manifest binding")
	}
	if digest(policy) != manifest.Release.ThirdPartyPolicyDigest {
		return fmt.Errorf("third-party policy bytes do not match preheat manifest binding")
	}
	return nil
}

func validatePreheatReleaseBinding(raw []byte, manifest *releasemanifest.Manifest, compositionBytes []byte) error {
	preheat, err := preheatmanifest.Parse(raw)
	if err != nil {
		return fmt.Errorf("validate preheat manifest: %w", err)
	}
	if manifest == nil || !manifest.HasRuntimeComposition() {
		return fmt.Errorf("preheat manifest requires a modern release manifest")
	}
	if preheat.Release.ManifestDigest != manifest.Digest() || preheat.Release.Tag != "v"+manifest.ReleaseVersion {
		return fmt.Errorf("preheat manifest release identity does not match release manifest")
	}
	if preheat.Release.CompositionDigest != manifest.RuntimeComposition.SHA256 {
		return fmt.Errorf("preheat manifest composition binding does not match release manifest")
	}
	if len(compositionBytes) > 0 {
		if err := upgrader.ValidateRuntimeCompositionAsset(compositionBytes, manifest, manifest.Digest(), manifest.RuntimeComposition.SHA256); err != nil {
			return fmt.Errorf("validate preheat runtime composition binding: %w", err)
		}
	}
	return nil
}

func readRegularCachedFile(filePath string, limit int64, mode os.FileMode) ([]byte, error) {
	info, err := os.Lstat(filePath)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != mode.Perm() {
		return nil, fmt.Errorf("cached evidence file must be a regular %04o file", mode.Perm())
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || openedInfo.Mode().Perm() != mode.Perm() || !os.SameFile(info, openedInfo) {
		return nil, fmt.Errorf("cached evidence file changed during validation")
	}
	return readBounded(file, limit)
}

func ensureManifestCacheDirectory(root, cacheDir, compositionCache, preheatCache, deploymentCache string) error {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("upgrade deployment root must be a regular directory")
	}
	current := root
	for _, part := range []string{".lunafox", "upgrade", upgrader.ManifestDirectory} {
		current = filepath.Join(current, part)
		if err := os.Mkdir(current, 0o700); err != nil && !os.IsExist(err) {
			return err
		}
		entry, err := os.Lstat(current)
		if err != nil || !entry.IsDir() || entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("upgrade manifest cache must use regular directories")
		}
		if err := os.Chmod(current, 0o700); err != nil {
			return err
		}
	}
	if filepath.Clean(current) != filepath.Clean(cacheDir) {
		return fmt.Errorf("upgrade manifest cache path is invalid")
	}
	if err := ensurePrivateCacheDirectory(root, compositionCache, []string{".lunafox", "upgrade", upgrader.CompositionDirectory}); err != nil {
		return err
	}
	if err := ensurePrivateCacheDirectory(root, preheatCache, []string{".lunafox", "upgrade", upgrader.PreheatManifestDirectory}); err != nil {
		return err
	}
	if err := ensurePrivateCacheDirectory(root, deploymentCache, []string{".lunafox", "upgrade", upgrader.DeploymentAssetDirectory}); err != nil {
		return err
	}
	return nil
}

func ensurePrivateCacheDirectory(root, cacheDir string, parts []string) error {
	current := root
	for _, part := range parts {
		current = filepath.Join(current, part)
		if err := os.Mkdir(current, 0o700); err != nil && !os.IsExist(err) {
			return err
		}
		entry, err := os.Lstat(current)
		if err != nil || !entry.IsDir() || entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("upgrade cache must use regular directories")
		}
		if err := os.Chmod(current, 0o700); err != nil {
			return err
		}
	}
	if filepath.Clean(current) != filepath.Clean(cacheDir) {
		return fmt.Errorf("upgrade cache path is invalid")
	}
	return nil
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("release metadata exceeds %d bytes", limit)
	}
	return raw, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
