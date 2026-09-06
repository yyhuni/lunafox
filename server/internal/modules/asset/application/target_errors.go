package application

import "errors"

var ErrTargetNotFound = errors.New("target not found")

var (
	ErrUnsupportedSubdomainFilter   = errors.New("unsupported subdomain filter")
	ErrUnsupportedSubdomainOrderBy  = errors.New("unsupported subdomain orderBy")
	ErrInvalidSubdomainPageToken    = errors.New("invalid subdomain pageToken")
	ErrUnsupportedWebsiteFilter     = errors.New("unsupported website filter")
	ErrUnsupportedWebsiteOrderBy    = errors.New("unsupported website orderBy")
	ErrInvalidWebsitePageToken      = errors.New("invalid website pageToken")
	ErrUnsupportedDirectoryFilter   = errors.New("unsupported directory filter")
	ErrUnsupportedDirectoryOrderBy  = errors.New("unsupported directory orderBy")
	ErrInvalidDirectoryPageToken    = errors.New("invalid directory pageToken")
	ErrUnsupportedScreenshotFilter  = errors.New("unsupported screenshot filter")
	ErrUnsupportedScreenshotOrderBy = errors.New("unsupported screenshot orderBy")
	ErrInvalidScreenshotPageToken   = errors.New("invalid screenshot pageToken")
	ErrUnsupportedEndpointFilter    = errors.New("unsupported endpoint filter")
	ErrUnsupportedEndpointOrderBy   = errors.New("unsupported endpoint orderBy")
	ErrInvalidEndpointPageToken     = errors.New("invalid endpoint pageToken")
	ErrUnsupportedHostPortFilter    = errors.New("unsupported hostPort filter")
	ErrUnsupportedHostPortOrderBy   = errors.New("unsupported hostPort orderBy")
	ErrInvalidHostPortPageToken     = errors.New("invalid hostPort pageToken")
)
