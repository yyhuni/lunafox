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

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/upgrader"
)

const (
	maxChannelRecordBytes   = 16 * 1024
	maxReleaseManifestBytes = 4 * 1024 * 1024
	manifestFetchTimeout    = 15 * time.Second
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
	baseURL  *url.URL
	channel  string
	cacheDir string
	client   *http.Client
	mu       sync.Mutex
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
	if err := ensureManifestCacheDirectory(root, cacheDir); err != nil {
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
	return &ChannelManifestSource{baseURL: baseURL, channel: channel, cacheDir: cacheDir, client: client}, nil
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
	if err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	if "v"+manifest.ReleaseVersion != record.version {
		return nil, domain.NewManifestIdentityMismatch("lunafox-"+strings.TrimPrefix(record.version, "v"), manifest.Upgrade.ManifestID)
	}
	if _, err := domain.TargetFromManifest(manifest); err != nil {
		return nil, err
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
		return nil, domain.WrapManifestInvalid(err)
	}
	if manifest.Digest() != digest {
		return nil, domain.NewManifestDigestMismatch(digest, manifest.Digest())
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

func (source *ChannelManifestSource) cachePath(digest string) (string, error) {
	if !strings.HasPrefix(digest, "sha256:") || !channelDigestPattern.MatchString(strings.TrimPrefix(digest, "sha256:")) {
		return "", fmt.Errorf("release manifest digest is invalid")
	}
	return filepath.Join(source.cacheDir, strings.TrimPrefix(digest, "sha256:")+".yaml"), nil
}

func ensureManifestCacheDirectory(root, cacheDir string) error {
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
