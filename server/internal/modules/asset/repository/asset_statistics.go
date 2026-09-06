package repository

import (
	"context"
	"strings"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

// AssetStatisticsRepository reads current-state aggregates without exposing other modules' persistence models.
type AssetStatisticsRepository struct {
	db *gorm.DB
}

func NewAssetStatisticsRepository(db *gorm.DB) *AssetStatisticsRepository {
	return &AssetStatisticsRepository{db: db}
}

type assetStatisticsRow struct {
	TotalTargets    int64
	TotalSubdomains int64
	TotalIPs        int64
	TotalEndpoints  int64
	TotalWebsites   int64
	TotalVulns      int64
	RunningScans    int64

	ChangeTargets    int64
	ChangeSubdomains int64
	ChangeIPs        int64
	ChangeEndpoints  int64
	ChangeWebsites   int64
	ChangeVulns      int64
}

type vulnerabilitySeverityRow struct {
	Severity string
	Count    int64
}

func (r *AssetStatisticsRepository) GetCurrentAssetStatistics(ctx context.Context, changeSince time.Time) (assetdomain.AssetStatistics, error) {
	var statistics assetdomain.AssetStatistics
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := queryCurrentAssetStatistics(tx, changeSince.UTC())
		if err != nil {
			return err
		}
		statistics = assetStatisticsFromRow(row)

		var severityRows []vulnerabilitySeverityRow
		if err := tx.Raw(`SELECT severity, COUNT(*) AS count FROM vulnerability GROUP BY severity`).Scan(&severityRows).Error; err != nil {
			return err
		}
		statistics.VulnsBySeverity = mapVulnerabilitySeverityCounts(severityRows)
		return nil
	})
	return statistics, err
}

func (r *AssetStatisticsRepository) ListAssetStatisticsHistory(ctx context.Context, start time.Time, days int) ([]assetdomain.AssetStatisticsHistoryItem, error) {
	history := make([]assetdomain.AssetStatisticsHistoryItem, 0, days)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for offset := 0; offset < days; offset++ {
			day := start.UTC().AddDate(0, 0, offset)
			row, err := queryHistoricalAssetStatistics(tx, day.AddDate(0, 0, 1))
			if err != nil {
				return err
			}
			history = append(history, assetdomain.AssetStatisticsHistoryItem{
				Date:            day,
				TotalTargets:    row.TotalTargets,
				TotalSubdomains: row.TotalSubdomains,
				TotalIPs:        row.TotalIPs,
				TotalEndpoints:  row.TotalEndpoints,
				TotalWebsites:   row.TotalWebsites,
				TotalVulns:      row.TotalVulns,
				TotalAssets:     row.TotalSubdomains + row.TotalIPs + row.TotalEndpoints + row.TotalWebsites,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return history, nil
}

func queryCurrentAssetStatistics(tx *gorm.DB, changeSince time.Time) (assetStatisticsRow, error) {
	var row assetStatisticsRow
	err := tx.Raw(`
		SELECT
			(SELECT COUNT(*) FROM target WHERE deleted_at IS NULL) AS total_targets,
			(SELECT COUNT(*) FROM subdomain) AS total_subdomains,
			(SELECT COUNT(DISTINCT ip) FROM host_port_mapping) AS total_ips,
			(SELECT COUNT(*) FROM endpoint) AS total_endpoints,
			(SELECT COUNT(*) FROM website) AS total_websites,
			(SELECT COUNT(*) FROM vulnerability) AS total_vulns,
			(SELECT COUNT(*) FROM scan WHERE deleted_at IS NULL AND status = 'running') AS running_scans,
			(SELECT COUNT(*) FROM target WHERE deleted_at IS NULL AND created_at >= ?) AS change_targets,
			(SELECT COUNT(*) FROM subdomain WHERE created_at >= ?) AS change_subdomains,
			(SELECT COUNT(*) FROM (
				SELECT ip FROM host_port_mapping GROUP BY ip HAVING MIN(created_at) >= ?
			) AS recently_observed_ips) AS change_ips,
			(SELECT COUNT(*) FROM endpoint WHERE created_at >= ?) AS change_endpoints,
			(SELECT COUNT(*) FROM website WHERE created_at >= ?) AS change_websites,
			(SELECT COUNT(*) FROM vulnerability WHERE created_at >= ?) AS change_vulns
	`, changeSince, changeSince, changeSince, changeSince, changeSince, changeSince).Scan(&row).Error
	return row, err
}

func queryHistoricalAssetStatistics(tx *gorm.DB, boundary time.Time) (assetStatisticsRow, error) {
	var row assetStatisticsRow
	err := tx.Raw(`
		SELECT
			(SELECT COUNT(*) FROM target WHERE deleted_at IS NULL AND created_at < ?) AS total_targets,
			(SELECT COUNT(*) FROM subdomain WHERE created_at < ?) AS total_subdomains,
			(SELECT COUNT(*) FROM (
				SELECT ip FROM host_port_mapping WHERE created_at < ? GROUP BY ip
			) AS observed_ips) AS total_ips,
			(SELECT COUNT(*) FROM endpoint WHERE created_at < ?) AS total_endpoints,
			(SELECT COUNT(*) FROM website WHERE created_at < ?) AS total_websites,
			(SELECT COUNT(*) FROM vulnerability WHERE created_at < ?) AS total_vulns
	`, boundary, boundary, boundary, boundary, boundary, boundary).Scan(&row).Error
	return row, err
}

func assetStatisticsFromRow(row assetStatisticsRow) assetdomain.AssetStatistics {
	return assetdomain.AssetStatistics{
		TotalTargets:     row.TotalTargets,
		TotalSubdomains:  row.TotalSubdomains,
		TotalIPs:         row.TotalIPs,
		TotalEndpoints:   row.TotalEndpoints,
		TotalWebsites:    row.TotalWebsites,
		TotalVulns:       row.TotalVulns,
		TotalAssets:      row.TotalSubdomains + row.TotalIPs + row.TotalEndpoints + row.TotalWebsites,
		RunningScans:     row.RunningScans,
		ChangeTargets:    row.ChangeTargets,
		ChangeSubdomains: row.ChangeSubdomains,
		ChangeIPs:        row.ChangeIPs,
		ChangeEndpoints:  row.ChangeEndpoints,
		ChangeWebsites:   row.ChangeWebsites,
		ChangeVulns:      row.ChangeVulns,
		ChangeAssets:     row.ChangeSubdomains + row.ChangeIPs + row.ChangeEndpoints + row.ChangeWebsites,
	}
}

func mapVulnerabilitySeverityCounts(rows []vulnerabilitySeverityRow) assetdomain.VulnerabilitySeverityCounts {
	var counts assetdomain.VulnerabilitySeverityCounts
	for _, row := range rows {
		switch strings.ToLower(strings.TrimSpace(row.Severity)) {
		case "critical":
			counts.Critical = row.Count
		case "high":
			counts.High = row.Count
		case "medium":
			counts.Medium = row.Count
		case "low":
			counts.Low = row.Count
		case "info":
			counts.Info = row.Count
		}
	}
	return counts
}
