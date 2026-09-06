package agent

import agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"

var _ agentapp.ServerLocationReader = (*agentServerLocationReaderAdapter)(nil)
