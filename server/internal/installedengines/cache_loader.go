package installedengines

import (
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/server/internal/engineinstall"
)

// CacheInstallerExactPackageLoader connects the installed-engine query port
// to the Server-owned verified archive cache without exposing cache paths.
type CacheInstallerExactPackageLoader struct {
	cache engineinstall.CacheInstaller
}

var _ ExactPackageCacheLoader = (*CacheInstallerExactPackageLoader)(nil)

func NewCacheInstallerExactPackageLoader(
	cache engineinstall.CacheInstaller,
) (*CacheInstallerExactPackageLoader, error) {
	if strings.TrimSpace(cache.Root) == "" || cache.MaxArchiveBytes <= 0 {
		return nil, fmt.Errorf("Engine Package v2 cache root and size limit are required")
	}
	return &CacheInstallerExactPackageLoader{cache: cache}, nil
}

func (loader *CacheInstallerExactPackageLoader) LoadExactPackage(
	expectedDigest ociartifact.PackageDigest,
) (ExactPackageCacheEntry, error) {
	if loader == nil {
		return ExactPackageCacheEntry{}, fmt.Errorf("Engine Package v2 cache loader is required")
	}
	cached, err := loader.cache.LoadOrRebuildPackage(expectedDigest)
	if err != nil {
		return ExactPackageCacheEntry{}, err
	}
	return ExactPackageCacheEntry{
		PackageDigest: cached.PackageDigest,
		Layout:        cached.Layout,
	}, nil
}
