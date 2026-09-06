package infrastructure

import (
	"time"

	systemapp "github.com/yyhuni/lunafox/server/internal/modules/system/application"
)

type serverLocationRuntime struct{}

func NewServerLocationRuntime() systemapp.ServerLocationSchedulerRuntime {
	return serverLocationRuntime{}
}

func (serverLocationRuntime) NowUTC() time.Time { return time.Now().UTC() }

func (serverLocationRuntime) NewTimer(delay time.Duration) systemapp.ServerLocationTimer {
	return serverLocationTimer{timer: time.NewTimer(delay)}
}

type serverLocationTimer struct {
	timer *time.Timer
}

func (timer serverLocationTimer) Channel() <-chan time.Time { return timer.timer.C }
func (timer serverLocationTimer) Stop() bool                { return timer.timer.Stop() }
