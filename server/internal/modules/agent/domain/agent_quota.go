package domain

import "errors"

// DeploymentAgentLimit counts registered identities, including offline and bootstrap Agents.
const DeploymentAgentLimit = 3

// ErrAgentQuotaExceeded means a new identity would exceed the deployment quota.
var ErrAgentQuotaExceeded = errors.New("agent deployment quota exceeded")
