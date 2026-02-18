package install

import _ "embed"

// AgentInstallScript is the embedded agent install script template.
//
//go:embed templates/agent_install.sh
var AgentInstallScript string
