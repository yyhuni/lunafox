package agent

import (
	"errors"

	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	systemapp "github.com/yyhuni/lunafox/server/internal/modules/system/application"
)

func NewAgentServerLocationReaderAdapter(reader *systemapp.ServerLocationReader) (agentapp.ServerLocationReader, error) {
	if reader == nil {
		return nil, errors.New("system Server location reader is required")
	}
	return &agentServerLocationReaderAdapter{reader: reader}, nil
}
