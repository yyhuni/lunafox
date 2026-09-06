package repository

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// GetIPAggregation returns paginated IPs with their earliest created_at, ordered by created_at DESC.
func (r *HostPortRepository) GetIPAggregation(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.IPAggregationRow, int64, error) {
	return r.GetIPAggregationContext(context.Background(), targetID, page, pageSize, filter, orderBy)
}

// GetIPAggregationContext preserves a caller-owned cancellation/deadline through host-port list queries.
func (r *HostPortRepository) GetIPAggregationContext(ctx context.Context, targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.IPAggregationRow, int64, error) {
	db := dbtx.Resolve(ctx, r.db).WithContext(ctx)
	baseQuery := db.Model(&model.HostPort{}).Where("target_id = ?", targetID)
	baseQuery = baseQuery.Scopes(applyHostPortListFilter(filter))

	countQuery := baseQuery.Select("ip").Group("ip")
	var total int64
	if err := db.Table("(?) as ip_groups", countQuery).Count(&total).Error; err != nil {
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
		Scopes(func(db *gorm.DB) *gorm.DB { return applyHostPortAggregateOrder(db, orderBy) }).
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

// ListPortOptionsByTargetID returns target-scoped port filter options.
// Performance is backed by idx_hpm_target_port_ip on (target_id, port, ip).
func (r *HostPortRepository) ListPortOptionsByTargetID(targetID int) ([]assetdomain.FilterOption, error) {
	type optionRow struct {
		Value string
		Count int64
	}
	var rows []optionRow
	err := r.db.Model(&model.HostPort{}).
		Select("port::text AS value, COUNT(DISTINCT ip) AS count").
		Where("target_id = ?", targetID).
		Group("port").
		Order("port ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	options := make([]assetdomain.FilterOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, assetdomain.FilterOption{Value: row.Value, Label: row.Value, Count: row.Count})
	}
	return options, nil
}

func applyHostPortAggregateOrder(db *gorm.DB, orderBy string) *gorm.DB {
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

// GetHostsAndPortsByIP returns hosts and ports for a specific IP
func (r *HostPortRepository) GetHostsAndPortsByIP(targetID int, ip string, filter string) ([]string, []int, error) {
	return r.GetHostsAndPortsByIPContext(context.Background(), targetID, ip, filter)
}

// GetHostsAndPortsByIPContext preserves a caller-owned cancellation/deadline.
func (r *HostPortRepository) GetHostsAndPortsByIPContext(ctx context.Context, targetID int, ip string, filter string) ([]string, []int, error) {
	baseQuery := dbtx.Resolve(ctx, r.db).WithContext(ctx).Model(&model.HostPort{}).
		Where("target_id = ? AND ip = ?", targetID, ip)
	baseQuery = baseQuery.Scopes(applyHostPortListFilter(filter))

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

func applyHostPortListFilter(filter string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		trimmed := strings.TrimSpace(filter)
		if trimmed == "" {
			return db
		}

		groups := scope.ParseFilter(trimmed)
		if len(groups) == 0 {
			return applyHostPortDefaultSearch(db, trimmed)
		}

		segments := make([]hostPortFilterSegment, 0, len(groups))
		for _, group := range groups {
			condition, args := hostPortFilterCondition(group.Filter)
			if condition == "" {
				return db.Where("1 = 0")
			}
			if group.LogicalOp == scope.LogicalOr && len(segments) > 0 {
				last := &segments[len(segments)-1]
				last.conditions = append(last.conditions, condition)
				last.args = append(last.args, args...)
				continue
			}
			segments = append(segments, hostPortFilterSegment{conditions: []string{condition}, args: args})
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

type hostPortFilterSegment struct {
	conditions []string
	args       []any
}

func applyHostPortDefaultSearch(db *gorm.DB, value string) *gorm.DB {
	ipCondition, ipArgs := hostPortIPSearchCondition(value)
	return db.Where("("+ipCondition+" OR host ILIKE ?)", append(ipArgs, "%"+value+"%")...)
}

func hostPortFilterCondition(filter scope.ParsedFilter) (string, []any) {
	switch strings.ToLower(filter.Field) {
	case "ip":
		return hostPortIPSearchCondition(filter.Value)
	case "host":
		if filter.Operator == "==" {
			return "LOWER(host) = ?", []any{strings.ToLower(strings.TrimSpace(filter.Value))}
		}
		return "host ILIKE ?", []any{"%" + filter.Value + "%"}
	case "port":
		return "port = ?", []any{filter.Value}
	default:
		return "", nil
	}
}

func hostPortIPSearchCondition(value string) (string, []any) {
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
	if cidr, ok := ipv4PrefixToCIDR(trimmed); ok {
		return "ip <<= ?::cidr", []any{cidr}
	}
	return "1 = 0", nil
}

func ipv4PrefixToCIDR(value string) (string, bool) {
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

// ForEachByTargetID streams host-port mappings for a target without exposing SQL cursor lifecycle to callers.
func (r *HostPortRepository) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.HostPort) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.HostPort{}).
		Where("target_id = ?", targetID).
		Order("ip, host, port").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var mapping model.HostPort
		if err := r.db.ScanRows(rows, &mapping); err != nil {
			return err
		}
		if err := visit(*hostPortModelToDomain(&mapping)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ForEachByTargetIDAndIPs streams host-port mappings filtered by IPs without exposing SQL cursor lifecycle to callers.
func (r *HostPortRepository) ForEachByTargetIDAndIPs(ctx context.Context, targetID int, ips []string, visit func(assetdomain.HostPort) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.HostPort{}).
		Where("target_id = ? AND ip IN ?", targetID, ips).
		Order("ip, host, port").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var mapping model.HostPort
		if err := r.db.ScanRows(rows, &mapping); err != nil {
			return err
		}
		if err := visit(*hostPortModelToDomain(&mapping)); err != nil {
			return err
		}
	}
	return rows.Err()
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
