package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/agentexecution"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errSavedPlanClaimRace = errors.New("saved execution plan claim lost compare-and-swap")

var errAgentExecutionRestricted = errors.New("agent execution is license restricted")

type savedPlanClaimCandidate struct {
	ID                    int
	ScanID                int `gorm:"column:scan_id"`
	Status                string
	ScanAgentID           *int   `gorm:"column:scan_agent_id"`
	ResolvedExecutionPlan []byte `gorm:"column:resolved_execution_plan"`
}

// ClaimNextCompatibleSavedExecutionPlan is the additive Engine Container claim
// transaction. It reads only immutable plan bytes and generic Agent capability;
// package, config, Target applicability, and workflow facts are never rebuilt.
func (r *scanTaskRepository) ClaimNextCompatibleSavedExecutionPlan(
	ctx context.Context,
	agentID int,
	sessionID string,
	sessionEpoch int64,
	requestID string,
	supportedEngineAPIMajors []uint32,
) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	if r == nil || r.db == nil || r.agentExecutionGuard == nil {
		return nil, fmt.Errorf("scan task repository is not initialized")
	}
	if ctx == nil || agentID <= 0 || sessionEpoch <= 0 {
		return nil, fmt.Errorf("claim scope is invalid")
	}
	sessionID = strings.TrimSpace(sessionID)
	requestID = strings.TrimSpace(requestID)
	parsedRequestID, err := uuid.Parse(requestID)
	if sessionID == "" || err != nil || parsedRequestID.String() != requestID {
		return nil, fmt.Errorf("canonical claim session and request ID are required")
	}
	if len(supportedEngineAPIMajors) == 0 {
		return nil, fmt.Errorf("supported Engine API majors are required")
	}

	var claimed *agentexecutionv1.ResolvedEngineExecutionPlan
	err = r.agentExecutionGuard.WithinClaim(ctx, func(txContext context.Context) error {
		tx := dbtx.Resolve(txContext, r.db).WithContext(txContext)
		// Lock order is quota, identity, persisted session, then Scan/Task. A
		// KEY SHARE identity lock avoids deadlocking session takeover updates.
		if err := requireClaimingAgentExists(tx, agentID); err != nil {
			return err
		}
		if err := requirePersistedAgentExecutionSession(tx, agentID, sessionID, sessionEpoch); err != nil {
			return err
		}
		replayed, err := findSavedPlanClaimReplay(tx, agentID, sessionID, sessionEpoch, requestID)
		if err != nil {
			return err
		}
		if replayed != nil {
			claimed = replayed
			return nil
		}

		// An existing correlated assignment remains replayable after expiry;
		// only a new assignment consumes the current execution entitlement.
		eligibility, err := r.agentExecutionGuard.ExecutionEligibilityAt(txContext, agentID, time.Now().UTC())
		if err != nil {
			return err
		}
		if eligibility == "license_restricted" {
			return nil
		}
		if eligibility != "eligible" {
			return fmt.Errorf("unknown Agent execution eligibility %q", eligibility)
		}

		candidate, plan, err := selectCompatibleSavedPlanCandidate(tx, agentID, supportedEngineAPIMajors)
		if err != nil || candidate == nil {
			return err
		}
		if err := bindScanToClaimingAgent(tx, candidate.ScanID, agentID); err != nil {
			return err
		}
		now := time.Now().UTC()
		// Row-lock waits may cross the signed expiry even while the shared
		// entitlement lock prevents replacement. Recheck immediately before CAS.
		eligibility, err = r.agentExecutionGuard.ExecutionEligibilityAt(txContext, agentID, now)
		if err != nil {
			return err
		}
		if eligibility == "license_restricted" {
			return errAgentExecutionRestricted
		}
		if eligibility != "eligible" {
			return fmt.Errorf("unknown Agent execution eligibility %q", eligibility)
		}
		result := tx.Model(&model.ScanTask{}).
			Where("id = ? AND status = ?", candidate.ID, taskStatusPending).
			Updates(map[string]interface{}{
				"status":                 taskStatusRunning,
				"assigned_agent_id":      agentID,
				"assigned_session_id":    sessionID,
				"assigned_session_epoch": sessionEpoch,
				"assigned_request_id":    requestID,
				"started_at":             now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errSavedPlanClaimRace
		}
		if err := tx.Model(&model.Scan{}).
			Where("id = ? AND status = ?", candidate.ScanID, scanStatusPending).
			Update("status", scanStatusRunning).Error; err != nil {
			return err
		}
		claimed = proto.Clone(plan).(*agentexecutionv1.ResolvedEngineExecutionPlan)
		return nil
	})
	if errors.Is(err, errAgentExecutionRestricted) {
		return nil, nil
	}
	if errors.Is(err, errSavedPlanClaimRace) {
		replayed, replayErr := findSavedPlanClaimReplay(r.db.WithContext(ctx), agentID, sessionID, sessionEpoch, requestID)
		if replayErr != nil {
			return nil, replayErr
		}
		return replayed, nil
	}
	if err != nil {
		// A replay must never convert a session fence or entitlement/storage
		// failure into a successful response. The runtime row lock already
		// serializes concurrent requests from this Agent before replay lookup.
		return nil, err
	}
	return claimed, nil
}

func findSavedPlanClaimReplay(db *gorm.DB, agentID int, sessionID string, sessionEpoch int64, requestID string) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	var row savedPlanClaimCandidate
	result := db.Table("scan_task AS st").
		Select("st.id, st.scan_id, st.status, st.resolved_execution_plan").
		Where("st.assigned_agent_id = ? AND st.assigned_session_id = ? AND st.assigned_session_epoch = ? AND st.assigned_request_id = ?", agentID, sessionID, sessionEpoch, requestID).
		Limit(1).
		Find(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	if err := validateSavedExecutionPlanLeaseRow(savedExecutionPlanRow{
		ID: row.ID, ScanID: row.ScanID, Status: row.Status, ResolvedExecutionPlan: row.ResolvedExecutionPlan,
	}); err != nil {
		return nil, err
	}
	return agentexecution.UnmarshalResolvedEngineExecutionPlan(row.ResolvedExecutionPlan)
}

func selectCompatibleSavedPlanCandidate(db *gorm.DB, agentID int, supported []uint32) (*savedPlanClaimCandidate, *agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	query := db.Table("scan_task AS st").
		Joins("JOIN scan AS s ON s.id = st.scan_id").
		Select("st.id, st.scan_id, st.status, s.agent_id AS scan_agent_id, st.resolved_execution_plan").
		Where("st.status = ? AND LENGTH(st.resolved_execution_plan) > 0 AND s.status IN (?, ?) AND s.deleted_at IS NULL AND (s.agent_id IS NULL OR s.agent_id = ?)", taskStatusPending, scanStatusPending, scanStatusRunning, agentID).
		Order(clause.Expr{SQL: "CASE WHEN s.agent_id = ? THEN 0 ELSE 1 END", Vars: []interface{}{agentID}}).
		Order("st.stage_order DESC, st.created_at ASC, st.id ASC")
	if db.Dialector.Name() == "postgres" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
	}
	rows, err := query.Rows()
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var candidate savedPlanClaimCandidate
		if err := db.ScanRows(rows, &candidate); err != nil {
			return nil, nil, err
		}
		if err := validateSavedExecutionPlanLeaseRow(savedExecutionPlanRow{
			ID: candidate.ID, ScanID: candidate.ScanID, Status: candidate.Status, ResolvedExecutionPlan: candidate.ResolvedExecutionPlan,
		}); err != nil {
			return nil, nil, err
		}
		plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(candidate.ResolvedExecutionPlan)
		if err != nil {
			return nil, nil, err
		}
		if supportsEngineAPIMajor(supported, plan.GetEngineRelease().GetEngineApiMajor()) {
			copy := candidate
			return &copy, plan, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return nil, nil, nil
}

func bindScanToClaimingAgent(db *gorm.DB, scanID, agentID int) error {
	var row struct {
		AgentID *int `gorm:"column:agent_id"`
	}
	query := db.Table("scan").Select("agent_id").Where("id = ?", scanID)
	if db.Dialector.Name() == "postgres" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.Take(&row).Error; err != nil {
		return err
	}
	if row.AgentID != nil {
		if *row.AgentID != agentID {
			return errSavedPlanClaimRace
		}
		return nil
	}
	result := db.Model(&model.Scan{}).Where("id = ? AND agent_id IS NULL", scanID).Update("agent_id", agentID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errSavedPlanClaimRace
	}
	return nil
}

func supportsEngineAPIMajor(supported []uint32, required uint32) bool {
	for _, major := range supported {
		if major == required {
			return true
		}
	}
	return false
}
