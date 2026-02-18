package repository

import (
	"database/sql"
	"sort"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// GetIPAggregation returns paginated IPs with their earliest created_at, ordered by created_at DESC.
func (r *HostPortRepository) GetIPAggregation(targetID int, page, pageSize int, filter string) ([]assetdomain.IPAggregationRow, int64, error) {
	baseQuery := r.db.Model(&model.HostPort{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, hostPortFilterMappingNormalized, "ip"))

	countQuery := baseQuery.Select("ip").Group("ip")
	var total int64
	if err := r.db.Table("(?) as ip_groups", countQuery).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	type row struct {
		IP        string
		CreatedAt time.Time
	}
	var queryRows []row
	err := baseQuery.
		Select("ip, MIN(created_at) as created_at").
		Group("ip").
		Order("MIN(created_at) DESC").
		Scopes(scope.WithPagination(page, pageSize)).
		Scan(&queryRows).Error
	if err != nil {
		return nil, 0, err
	}

	results := make([]assetdomain.IPAggregationRow, 0, len(queryRows))
	for index := range queryRows {
		results = append(results, assetdomain.IPAggregationRow{IP: queryRows[index].IP, CreatedAt: queryRows[index].CreatedAt.UTC()})
	}
	return results, total, nil
}

// GetHostsAndPortsByIP returns hosts and ports for a specific IP
func (r *HostPortRepository) GetHostsAndPortsByIP(targetID int, ip string, filter string) ([]string, []int, error) {
	baseQuery := r.db.Model(&model.HostPort{}).
		Where("target_id = ? AND ip = ?", targetID, ip)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, hostPortFilterMappingNormalized, "ip"))

	var mappings []struct {
		Host string
		Port int
	}
	err := baseQuery.
		Select("DISTINCT host, port").
		Scan(&mappings).Error
	if err != nil {
		return nil, nil, err
	}

	hostSet := make(map[string]struct{})
	portSet := make(map[int]struct{})
	for _, mapping := range mappings {
		hostSet[mapping.Host] = struct{}{}
		portSet[mapping.Port] = struct{}{}
	}

	hosts := make([]string, 0, len(hostSet))
	for host := range hostSet {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)

	ports := make([]int, 0, len(portSet))
	for port := range portSet {
		ports = append(ports, port)
	}
	sort.Ints(ports)

	return hosts, ports, nil
}

// StreamByTargetID returns a sql.Rows cursor for streaming export (raw format)
func (r *HostPortRepository) StreamByTargetID(targetID int) (*sql.Rows, error) {
	return r.db.Model(&model.HostPort{}).
		Where("target_id = ?", targetID).
		Order("ip, host, port").
		Rows()
}

// StreamByTargetIDAndIPs returns a sql.Rows cursor for streaming export filtered by IPs
func (r *HostPortRepository) StreamByTargetIDAndIPs(targetID int, ips []string) (*sql.Rows, error) {
	return r.db.Model(&model.HostPort{}).
		Where("target_id = ? AND ip IN ?", targetID, ips).
		Order("ip, host, port").
		Rows()
}

// CountByTargetID returns the count of unique IPs for a target
func (r *HostPortRepository) CountByTargetID(targetID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.HostPort{}).
		Where("target_id = ?", targetID).
		Distinct("ip").
		Count(&count).Error
	return count, err
}

// ScanRow scans a single row into HostPort domain object
func (r *HostPortRepository) ScanRow(rows *sql.Rows) (*assetdomain.HostPort, error) {
	var mapping model.HostPort
	if err := r.db.ScanRows(rows, &mapping); err != nil {
		return nil, err
	}
	return hostPortModelToDomain(&mapping), nil
}
