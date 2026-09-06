package repository

import (
	"errors"
	"strings"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type persistedAgentExecutionSession struct {
	SessionID    string `gorm:"column:session_id"`
	SessionEpoch int64  `gorm:"column:session_epoch"`
}

// requirePersistedAgentExecutionSession locks the persisted fencing row so a
// claim is ordered against concurrent process takeover and same-request replay.
func requirePersistedAgentExecutionSession(tx *gorm.DB, agentID int, sessionID string, sessionEpoch int64) error {
	sessionID = strings.TrimSpace(sessionID)
	if tx == nil || agentID <= 0 || sessionID == "" || sessionEpoch <= 0 {
		return scandomain.ErrAgentExecutionSessionFenced
	}
	query := tx.Table("agent_runtime_status").
		Select("session_id, session_epoch").
		Where("agent_id = ?", agentID)
	if tx.Dialector.Name() == "postgres" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var current persistedAgentExecutionSession
	if err := query.Take(&current).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return scandomain.ErrAgentExecutionSessionFenced
		}
		return err
	}
	if strings.TrimSpace(current.SessionID) != sessionID || current.SessionEpoch != sessionEpoch {
		return scandomain.ErrAgentExecutionSessionFenced
	}
	return nil
}
