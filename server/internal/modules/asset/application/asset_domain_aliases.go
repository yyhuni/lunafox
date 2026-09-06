// Package application orchestrates asset module use cases across asset resource types.
package application

import assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"

type Website = assetdomain.Website
type Subdomain = assetdomain.Subdomain
type Endpoint = assetdomain.Endpoint
type Directory = assetdomain.Directory
type HostPort = assetdomain.HostPort
type Screenshot = assetdomain.Screenshot
