package targetcleanupwiring

import targetcleanupapp "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/application"

var _ targetcleanupapp.TargetScheduleCleaner = (*targetCleanupScheduleCleanerAdapter)(nil)
var _ targetcleanupapp.TargetScanCanceller = (*targetCleanupScanCancellerAdapter)(nil)
var _ targetcleanupapp.TaskCancelPublisher = (*targetCleanupTaskCancelPublisherAdapter)(nil)
