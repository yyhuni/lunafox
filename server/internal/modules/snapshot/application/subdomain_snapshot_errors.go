package application

import "errors"

var (
	ErrSubdomainSnapshotInvalidTargetType      = errors.New("target type must be domain for subdomains")
	ErrUnsupportedWebsiteSnapshotFilter        = errors.New("unsupported website snapshot filter")
	ErrUnsupportedWebsiteSnapshotOrderBy       = errors.New("unsupported website snapshot orderBy")
	ErrInvalidWebsiteSnapshotPageToken         = errors.New("invalid website snapshot pageToken")
	ErrUnsupportedEndpointSnapshotFilter       = errors.New("unsupported endpoint snapshot filter")
	ErrUnsupportedEndpointSnapshotOrderBy      = errors.New("unsupported endpoint snapshot orderBy")
	ErrInvalidEndpointSnapshotPageToken        = errors.New("invalid endpoint snapshot pageToken")
	ErrUnsupportedDirectorySnapshotFilter      = errors.New("unsupported directory snapshot filter")
	ErrUnsupportedDirectorySnapshotOrderBy     = errors.New("unsupported directory snapshot orderBy")
	ErrInvalidDirectorySnapshotPageToken       = errors.New("invalid directory snapshot pageToken")
	ErrUnsupportedSubdomainSnapshotFilter      = errors.New("unsupported subdomain snapshot filter")
	ErrUnsupportedSubdomainSnapshotOrderBy     = errors.New("unsupported subdomain snapshot orderBy")
	ErrInvalidSubdomainSnapshotPageToken       = errors.New("invalid subdomain snapshot pageToken")
	ErrUnsupportedHostPortSnapshotFilter       = errors.New("unsupported hostPort snapshot filter")
	ErrUnsupportedHostPortSnapshotOrderBy      = errors.New("unsupported hostPort snapshot orderBy")
	ErrInvalidHostPortSnapshotPageToken        = errors.New("invalid hostPort snapshot pageToken")
	ErrUnsupportedScreenshotSnapshotFilter     = errors.New("unsupported screenshot snapshot filter")
	ErrUnsupportedScreenshotSnapshotOrderBy    = errors.New("unsupported screenshot snapshot orderBy")
	ErrInvalidScreenshotSnapshotPageToken      = errors.New("invalid screenshot snapshot pageToken")
	ErrUnsupportedVulnerabilitySnapshotFilter  = errors.New("unsupported vulnerability snapshot filter")
	ErrUnsupportedVulnerabilitySnapshotOrderBy = errors.New("unsupported vulnerability snapshot orderBy")
	ErrInvalidVulnerabilitySnapshotPageToken   = errors.New("invalid vulnerability snapshot pageToken")
)
