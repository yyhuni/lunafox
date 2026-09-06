package enginemanifest

import (
	"regexp"
)

const (
	TargetTypeDomain = "domain"
	TargetTypeIP     = "ip"
	TargetTypeCIDR   = "cidr"
)

var engineIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)
