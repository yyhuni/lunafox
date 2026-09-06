package repository

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// GetIPAggregation returns scan-scoped IP aggregates with their earliest created_at after filtering.
func (r *HostPortSnapshotRepository) GetIPAggregation(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.HostPortIPAggregationRow, int64, error) {
	baseQuery := r.db.Model(&model.HostPortSnapshot{}).Where("scan_id = ?", scanID)
	baseQuery = baseQuery.Scopes(applyHostPortSnapshotListFilter(filter))

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
		Scopes(func(db *gorm.DB) *gorm.DB { return applyHostPortSnapshotAggregateOrder(db, orderBy) }).
		Scopes(scope.WithPagination(page, pageSize)).
		Scan(&queryRows).Error
	if err != nil {
		return nil, 0, err
	}

	results := make([]snapshotdomain.HostPortIPAggregationRow, 0, len(queryRows))
	for index := range queryRows {
		results = append(results, snapshotdomain.HostPortIPAggregationRow{IP: queryRows[index].IP, CreatedAt: queryRows[index].CreatedAt.UTC()})
	}
	return results, total, nil
}

// ListPortOptionsByScanID returns scan-scoped port filter options.
// Performance is backed by idx_hpm_snap_scan_port_ip on (scan_id, port, ip).
func (r *HostPortSnapshotRepository) ListPortOptionsByScanID(scanID int) ([]snapshotdomain.FilterOption, error) {
	type optionRow struct {
		Value string
		Count int64
	}
	var rows []optionRow
	err := r.db.Model(&model.HostPortSnapshot{}).
		Select("port::text AS value, COUNT(DISTINCT ip) AS count").
		Where("scan_id = ?", scanID).
		Group("port").
		Order("port ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	options := make([]snapshotdomain.FilterOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, snapshotdomain.FilterOption{Value: row.Value, Label: row.Value, Count: row.Count})
	}
	return options, nil
}

func applyHostPortSnapshotAggregateOrder(db *gorm.DB, orderBy string) *gorm.DB {
	switch strings.TrimSpace(orderBy) {
	case "ip", "ip asc":
		return db.Order("ip ASC")
	case "ip desc":
		return db.Order("ip DESC")
	case "createdAt", "createdAt asc":
		return db.Order("MIN(created_at) ASC").Order("ip ASC")
	case "createdAt desc", "":
		return db.Order("MIN(created_at) DESC").Order("ip DESC")
	default:
		return db.Where("1 = 0")
	}
}

// GetHostsAndPortsByIP returns hosts and ports for a scan IP aggregate.
func (r *HostPortSnapshotRepository) GetHostsAndPortsByIP(scanID int, ip string, filter string) ([]string, []int, error) {
	baseQuery := r.db.Model(&model.HostPortSnapshot{}).
		Where("scan_id = ? AND ip = ?", scanID, ip)
	baseQuery = baseQuery.Scopes(applyHostPortSnapshotListFilter(filter))

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

func applyHostPortSnapshotListFilter(filter string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		trimmed := strings.TrimSpace(filter)
		if trimmed == "" {
			return db
		}

		groups := scope.ParseFilter(trimmed)
		if len(groups) == 0 {
			return applyHostPortSnapshotDefaultSearch(db, trimmed)
		}

		segments := make([]hostPortSnapshotFilterSegment, 0, len(groups))
		for _, group := range groups {
			condition, args := hostPortSnapshotFilterCondition(group.Filter)
			if condition == "" {
				return db.Where("1 = 0")
			}
			if group.LogicalOp == scope.LogicalOr && len(segments) > 0 {
				last := &segments[len(segments)-1]
				last.conditions = append(last.conditions, condition)
				last.args = append(last.args, args...)
				continue
			}
			segments = append(segments, hostPortSnapshotFilterSegment{conditions: []string{condition}, args: args})
		}

		for _, segment := range segments {
			if len(segment.conditions) == 1 {
				db = db.Where(segment.conditions[0], segment.args...)
				continue
			}
			db = db.Where("("+strings.Join(segment.conditions, " OR ")+")", segment.args...)
		}
		return db
	}
}

type hostPortSnapshotFilterSegment struct {
	conditions []string
	args       []any
}

func applyHostPortSnapshotDefaultSearch(db *gorm.DB, value string) *gorm.DB {
	ipCondition, ipArgs := hostPortSnapshotIPSearchCondition(value)
	return db.Where("("+ipCondition+" OR host ILIKE ?)", append(ipArgs, "%"+value+"%")...)
}

func hostPortSnapshotFilterCondition(filter scope.ParsedFilter) (string, []any) {
	switch strings.ToLower(filter.Field) {
	case "ip":
		return hostPortSnapshotIPSearchCondition(filter.Value)
	case "host":
		return "host ILIKE ?", []any{"%" + filter.Value + "%"}
	case "port":
		return "port = ?", []any{filter.Value}
	default:
		return "", nil
	}
}

func hostPortSnapshotIPSearchCondition(value string) (string, []any) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "1 = 0", nil
	}
	if strings.Contains(trimmed, "/") {
		if _, _, err := net.ParseCIDR(trimmed); err == nil {
			return "ip <<= ?::cidr", []any{trimmed}
		}
		return "1 = 0", nil
	}
	if net.ParseIP(trimmed) != nil {
		return "ip = ?::inet", []any{trimmed}
	}
	if cidr, ok := snapshotIPv4PrefixToCIDR(trimmed); ok {
		return "ip <<= ?::cidr", []any{cidr}
	}
	return "1 = 0", nil
}

func snapshotIPv4PrefixToCIDR(value string) (string, bool) {
	parts := strings.Split(strings.Trim(value, "."), ".")
	if len(parts) == 0 || len(parts) > 3 {
		return "", false
	}
	octets := []int{0, 0, 0, 0}
	for index, part := range parts {
		if part == "" {
			return "", false
		}
		octet, err := strconv.Atoi(part)
		if err != nil || octet < 0 || octet > 255 {
			return "", false
		}
		octets[index] = octet
	}
	prefixLength := len(parts) * 8
	return fmt.Sprintf("%d.%d.%d.%d/%d", octets[0], octets[1], octets[2], octets[3], prefixLength), true
}

// ForEachByScanID streams host-port snapshots for a scan without exposing SQL cursor lifecycle to callers.
func (r *HostPortSnapshotRepository) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.HostPortSnapshot) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.HostPortSnapshot{}).
		Where("scan_id = ?", scanID).
		Order("created_at DESC").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var snapshot model.HostPortSnapshot
		if err := r.db.ScanRows(rows, &snapshot); err != nil {
			return err
		}
		if err := visit(*hostPortSnapshotModelToDomain(&snapshot)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ForEachHostPortByScanID streams the complete finalized host/ip/port
// projection in an explicit stable order. It never folds records by effective
// host and does not add a DISTINCT or Target-derived filter.
func (r *HostPortSnapshotRepository) ForEachHostPortByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.HostPortInputEvidence) error) error {
	if visit == nil {
		return fmt.Errorf("HostPort visitor is required")
	}
	rows, err := r.db.WithContext(ctx).Model(&model.HostPortSnapshot{}).
		Select("host, ip, port").
		Where("scan_id = ?", scanID).
		Order("host ASC").
		Order("ip ASC").
		Order("port ASC").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var evidence snapshotdomain.HostPortInputEvidence
		if err := rows.Scan(&evidence.Host, &evidence.IP, &evidence.Port); err != nil {
			return err
		}
		if err := visit(evidence); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CountByScanID returns the count of host-port snapshots for a scan
func (r *HostPortSnapshotRepository) CountByScanID(scanID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.HostPortSnapshot{}).Where("scan_id = ?", scanID).Count(&count).Error
	return count, err
}
