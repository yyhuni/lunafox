package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	"gorm.io/gorm"
)

// ProducerWriter resolves producer-owned canonical facts and appends validated
// occurrences through the OutboxRepository in the same database transaction.
type ProducerWriter struct {
	outbox                 *OutboxRepository
	vulnerabilityThreshold domain.VulnerabilitySeverity
}

// NewProducerWriter creates the bridge used only by business transaction owners.
func NewProducerWriter(outbox *OutboxRepository, thresholds ...domain.VulnerabilitySeverity) *ProducerWriter {
	if outbox == nil {
		panic("notification outbox repository is required")
	}
	if len(thresholds) > 1 {
		panic("only one vulnerability notification threshold is supported")
	}
	threshold := domain.VulnerabilitySeverityHigh
	if len(thresholds) == 1 {
		threshold = thresholds[0]
	}
	if err := domain.ValidateVulnerabilityThreshold(threshold); err != nil {
		panic(err)
	}
	return &ProducerWriter{outbox: outbox, vulnerabilityThreshold: threshold}
}

// WriteScanTerminal appends exactly one terminal Scan candidate after the Scan
// state update won its conditional write in the containing transaction.
func (writer *ProducerWriter) WriteScanTerminal(tx *gorm.DB, scanID int, status, failureKind, failureMessage string, occurredAt time.Time) error {
	if writer == nil || writer.outbox == nil {
		return fmt.Errorf("notification scan producer is unavailable")
	}
	var scan struct {
		TargetID   int    `gorm:"column:target_id"`
		TargetName string `gorm:"column:target_name"`
	}
	if err := tx.Table("scan").
		Select("scan.target_id, target.name AS target_name").
		Joins("JOIN target ON target.id = scan.target_id").
		Where("scan.id = ? AND scan.deleted_at IS NULL", scanID).
		Take(&scan).Error; err != nil {
		return err
	}

	var occurrence domain.Occurrence
	var err error
	switch strings.TrimSpace(status) {
	case "succeeded":
		occurrence, err = domain.NewScanSucceededOccurrence(scanID, scan.TargetID, scan.TargetName, occurredAt)
	case "failed":
		occurrence, err = domain.NewScanFailedOccurrence(scanID, scan.TargetID, scan.TargetName, failureKind, failureMessage, occurredAt)
	default:
		return fmt.Errorf("scan terminal notification does not support status %q", status)
	}
	if err != nil {
		return err
	}
	return writer.outbox.CreateOccurrence(tx, occurrence)
}

// WriteVulnerabilityObservations appends all qualifying Scan-local facts. The
// event ID is Scan-scoped, so a repeated batch converges through the outbox
// uniqueness constraint without giving a finding cross-Scan identity.
func (writer *ProducerWriter) WriteVulnerabilityObservations(ctxTx *gorm.DB, scanID int, observations []domain.VulnerabilityObservedPayload, occurredAt time.Time) error {
	if writer == nil || writer.outbox == nil {
		return fmt.Errorf("notification vulnerability producer is unavailable")
	}
	for _, observation := range observations {
		if observation.ScanID != scanID {
			return fmt.Errorf("notification vulnerability scan scope mismatch")
		}
		eligible, err := domain.MeetsVulnerabilityThreshold(domain.VulnerabilitySeverity(observation.Severity), writer.vulnerabilityThreshold)
		if err != nil {
			return err
		}
		if !eligible {
			continue
		}
		occurrence, err := domain.NewVulnerabilityObservedOccurrence(observation, occurredAt)
		if err != nil {
			return err
		}
		if err := writer.outbox.CreateOccurrence(ctxTx, occurrence); err != nil {
			return err
		}
	}
	return nil
}

// WriteAgentOffline resolves the persisted Agent monitor transition and appends
// its candidate before the containing status transaction commits.
func (writer *ProducerWriter) WriteAgentOffline(tx *gorm.DB, agentID int, occurredAt time.Time) error {
	if writer == nil || writer.outbox == nil {
		return fmt.Errorf("notification agent producer is unavailable")
	}
	var agent struct {
		DisplayName   string     `gorm:"column:display_name"`
		LastHeartbeat *time.Time `gorm:"column:last_heartbeat"`
	}
	if err := tx.Table("agent").
		Select("agent.display_name, agent_runtime_status.last_heartbeat").
		Joins("JOIN agent_runtime_status ON agent_runtime_status.agent_id = agent.id").
		Where("agent.id = ?", agentID).
		Take(&agent).Error; err != nil {
		return err
	}
	if agent.LastHeartbeat == nil {
		return fmt.Errorf("offline Agent %d has no persisted heartbeat", agentID)
	}
	occurrence, err := domain.NewAgentOfflineOccurrence(agentID, agent.DisplayName, *agent.LastHeartbeat, occurredAt)
	if err != nil {
		return err
	}
	return writer.outbox.CreateOccurrence(tx, occurrence)
}

// WriteNucleiPOCSyncSucceeded resolves the persisted terminal task facts and
// appends its immutable success occurrence in the caller-owned transaction.
func (writer *ProducerWriter) WriteNucleiPOCSyncSucceeded(tx *gorm.DB, taskID uuid.UUID, occurredAt time.Time) error {
	if writer == nil || writer.outbox == nil {
		return fmt.Errorf("notification nuclei POC sync producer is unavailable")
	}
	var task struct {
		SourceType        string `gorm:"column:source_type"`
		State             string `gorm:"column:state"`
		CommitSHA         string `gorm:"column:commit_sha"`
		CommittedPOCCount int64  `gorm:"column:committed_poc_count"`
	}
	if err := tx.Table("nuclei_poc_sync_task").
		Select("source_type, state, commit_sha, committed_poc_count").
		Where("id = ?", taskID).
		Take(&task).Error; err != nil {
		return err
	}
	if task.State != "SUCCEEDED" {
		return fmt.Errorf("nuclei POC sync task %s is not succeeded", taskID)
	}
	occurrence, err := domain.NewNucleiPOCSyncSucceededOccurrence(taskID, task.SourceType, task.CommitSHA, task.CommittedPOCCount, occurredAt)
	if err != nil {
		return err
	}
	return writer.outbox.CreateOccurrence(tx, occurrence)
}

// WriteNucleiPOCSyncFailed resolves the persisted, redacted terminal task
// facts and appends its immutable failure occurrence in the caller-owned
// transaction.
func (writer *ProducerWriter) WriteNucleiPOCSyncFailed(tx *gorm.DB, taskID uuid.UUID, occurredAt time.Time) error {
	if writer == nil || writer.outbox == nil {
		return fmt.Errorf("notification nuclei POC sync producer is unavailable")
	}
	var task struct {
		SourceType     string `gorm:"column:source_type"`
		State          string `gorm:"column:state"`
		FailureCode    string `gorm:"column:failure_code"`
		FailureSummary string `gorm:"column:failure_summary"`
	}
	if err := tx.Table("nuclei_poc_sync_task").
		Select("source_type, state, failure_code, failure_summary").
		Where("id = ?", taskID).
		Take(&task).Error; err != nil {
		return err
	}
	if task.State != "FAILED" {
		return fmt.Errorf("nuclei POC sync task %s is not failed", taskID)
	}
	occurrence, err := domain.NewNucleiPOCSyncFailedOccurrence(taskID, task.SourceType, task.FailureCode, task.FailureSummary, occurredAt)
	if err != nil {
		return err
	}
	return writer.outbox.CreateOccurrence(tx, occurrence)
}
