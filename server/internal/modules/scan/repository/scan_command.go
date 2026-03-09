package repository

import (
	"fmt"
	"strings"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"gorm.io/gorm"
)

// CreateWithTasks creates a scan and tasks in a single transaction.
func (r *ScanRepository) CreateWithTasks(scan *ScanCreateRecord, tasks []ScanTaskCreateRecord) error {
	modelScan := scanCreateRecordToModel(scan)
	modelTasks := scanTaskCreateRecordsToModel(tasks)

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(modelScan).Error; err != nil {
			return err
		}
		if len(modelTasks) > 0 {
			for index := range modelTasks {
				modelTasks[index].ScanID = modelScan.ID
			}
			if err := tx.Create(&modelTasks).Error; err != nil {
				return err
			}
		}
		if scan != nil {
			scan.ID = modelScan.ID
			scan.CreatedAt = modelScan.CreatedAt
		}
		return nil
	})
}

// SoftDelete soft deletes a scan.
func (r *ScanRepository) SoftDelete(id int) error {
	now := time.Now().UTC()
	return r.db.Model(&model.Scan{}).Where("id = ?", id).Update("deleted_at", now).Error
}

// BulkSoftDelete soft deletes multiple scans by IDs.
func (r *ScanRepository) BulkSoftDelete(ids []int) (int64, []string, error) {
	if len(ids) == 0 {
		return 0, nil, nil
	}

	var scans []model.Scan
	if err := r.db.Select("id, target_id").
		Where("id IN ? AND deleted_at IS NULL", ids).
		Preload("Target", "deleted_at IS NULL").
		Find(&scans).Error; err != nil {
		return 0, nil, err
	}

	names := make([]string, 0, len(scans))
	for _, scan := range scans {
		if scan.Target != nil {
			names = append(names, scan.Target.Name)
		}
	}

	now := time.Now().UTC()
	result := r.db.Model(&model.Scan{}).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Update("deleted_at", now)

	return result.RowsAffected, names, result.Error
}

// UpdateStatus updates scan status with structured failure detail.
func (r *ScanRepository) UpdateStatus(id int, status string, failure *scandomain.FailureDetail) error {
	failure, err := normalizeScanFailureDetail(status, failure)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{"status": status}
	if failure != nil {
		updates["error_message"] = failure.Message
		updates["failure_kind"] = failure.Kind
	} else {
		updates["error_message"] = ""
		updates["failure_kind"] = ""
	}
	if status == scanStatusCompleted || status == scanStatusFailed || status == scanStatusCancelled {
		now := time.Now().UTC()
		updates["stopped_at"] = &now
	}
	return r.db.Model(&model.Scan{}).Where("id = ?", id).Updates(updates).Error
}

func normalizeScanFailureDetail(status string, failure *scandomain.FailureDetail) (*scandomain.FailureDetail, error) {
	if strings.TrimSpace(status) != scanStatusFailed {
		return nil, nil
	}
	if failure == nil {
		return nil, fmt.Errorf("failed scan requires failure detail")
	}
	message := strings.TrimSpace(failure.Message)
	if message == "" {
		return nil, fmt.Errorf("failed scan requires non-empty failure message")
	}
	kind := strings.TrimSpace(failure.Kind)
	if kind == "" {
		kind = "unknown"
	}
	return &scandomain.FailureDetail{Kind: kind, Message: message}, nil
}
