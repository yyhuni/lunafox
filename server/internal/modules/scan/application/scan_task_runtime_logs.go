package application

import (
	"context"
	"fmt"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

func (service *ScanTaskBridgeService) appendTaskRuntimeLog(ctx context.Context, task *ScanTaskRecord, eventID, level, content string) {
	if service.runtimeLogs == nil || task == nil {
		return
	}
	service.runtimeLogs.AppendTaskRuntimeLog(ctx, TaskRuntimeLogEvent{
		ScanID: task.ScanID, TaskID: task.ID, EventID: eventID, Level: level, Content: content,
	})
}

func (service *ScanTaskBridgeService) appendTerminalRuntimeLog(ctx context.Context, task *ScanTaskRecord, nextStatus scandomain.TaskStatus, failure *FailureDetail) {
	level := "info"
	content := fmt.Sprintf("[server] task state committed: %s", nextStatus)
	if nextStatus == scandomain.TaskStatusFailed {
		level = "error"
		content += ": " + failure.Message
		if failure.DisplayMessage != "" {
			content += "\n" + failure.DisplayMessage
		}
	}
	epoch := int64(0)
	if task.AssignedSessionEpoch != nil {
		epoch = *task.AssignedSessionEpoch
	}
	service.appendTaskRuntimeLog(ctx, task, fmt.Sprintf("terminal:%d:%s", epoch, nextStatus), level, content)
}
