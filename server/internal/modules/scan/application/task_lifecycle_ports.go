package application

// TaskCancelNotifier is intentionally best effort. A false result is a
// post-commit delivery observation and must never reverse a committed Stop.
type TaskCancelNotifier interface {
	TrySendTaskCancel(agentID, scanID, taskID int) bool
}
