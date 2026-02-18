package repository

// TargetAssetCounts holds asset count statistics for a target.
type TargetAssetCounts struct {
	Subdomains  int64
	Websites    int64
	Endpoints   int64
	IPs         int64
	Directories int64
	Screenshots int64
}

// VulnerabilityCounts holds vulnerability count statistics by severity.
type VulnerabilityCounts struct {
	Total    int64
	Critical int64
	High     int64
	Medium   int64
	Low      int64
}

// GetAssetCounts returns asset counts for a target.
func (r *TargetRepository) GetAssetCounts(targetID int) (*TargetAssetCounts, error) {
	counts := &TargetAssetCounts{}

	if err := r.db.Table("subdomain").Where("target_id = ?", targetID).Count(&counts.Subdomains).Error; err != nil {
		return nil, err
	}
	if err := r.db.Table("website").Where("target_id = ?", targetID).Count(&counts.Websites).Error; err != nil {
		return nil, err
	}
	if err := r.db.Table("endpoint").Where("target_id = ?", targetID).Count(&counts.Endpoints).Error; err != nil {
		return nil, err
	}
	if err := r.db.Table("host_port_mapping").Where("target_id = ?", targetID).Select("COUNT(DISTINCT ip)").Scan(&counts.IPs).Error; err != nil {
		return nil, err
	}
	if err := r.db.Table("directory").Where("target_id = ?", targetID).Count(&counts.Directories).Error; err != nil {
		return nil, err
	}
	if err := r.db.Table("screenshot").Where("target_id = ?", targetID).Count(&counts.Screenshots).Error; err != nil {
		return nil, err
	}

	return counts, nil
}

// GetVulnerabilityCounts returns vulnerability counts by severity for a target.
func (r *TargetRepository) GetVulnerabilityCounts(targetID int) (*VulnerabilityCounts, error) {
	counts := &VulnerabilityCounts{}

	if err := r.db.Table("vulnerability").Where("target_id = ?", targetID).Count(&counts.Total).Error; err != nil {
		return nil, err
	}

	type severityCount struct {
		Severity string
		Count    int64
	}
	var severityCounts []severityCount

	if err := r.db.Table("vulnerability").
		Select("severity, COUNT(*) as count").
		Where("target_id = ?", targetID).
		Group("severity").
		Scan(&severityCounts).Error; err != nil {
		return nil, err
	}

	for _, item := range severityCounts {
		switch item.Severity {
		case "critical":
			counts.Critical = item.Count
		case "high":
			counts.High = item.Count
		case "medium":
			counts.Medium = item.Count
		case "low":
			counts.Low = item.Count
		}
	}

	return counts, nil
}
