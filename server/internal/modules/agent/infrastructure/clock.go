package infrastructure

import (
	"time"

	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
)

type systemClock struct{}

func (systemClock) NowUTC() time.Time {
	return time.Now().UTC()
}

func NewSystemClock() agentapp.Clock {
	return systemClock{}
}
