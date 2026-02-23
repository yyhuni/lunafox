package handler

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	dbHealthStatusOnline      = "online"
	dbHealthStatusDegraded    = "degraded"
	dbHealthStatusOffline     = "offline"
	dbHealthStatusMaintenance = "maintenance"
)

const (
	signalScopeCore     = "core"
	signalScopeOptional = "optional"
)

const (
	reasonPermissionDenied = "permission_denied"
	reasonTimeout          = "timeout"
	reasonUnsupported      = "unsupported"
	reasonQueryFailed      = "query_failed"
	reasonUnknown          = "unknown"
)

type databaseHealthSnapshotResponse struct {
	Status             string                              `json:"status"`
	ObservedAt         string                              `json:"observedAt"`
	Role               string                              `json:"role"`
	Region             *string                             `json:"region"`
	Version            string                              `json:"version"`
	ReadOnly           bool                                `json:"readOnly"`
	UptimeSeconds      int64                               `json:"uptimeSeconds"`
	CoreSignals        databaseCoreSignalsResponse         `json:"coreSignals"`
	OptionalSignals    databaseOptionalSignalsResponse     `json:"optionalSignals"`
	UnavailableSignals []databaseUnavailableSignalResponse `json:"unavailableSignals"`
	Alerts             []databaseHealthAlertResponse       `json:"alerts"`
}

type databaseCoreSignalsResponse struct {
	ProbeLatencyMs          float64 `json:"probeLatencyMs"`
	ConnectionsUsed         int     `json:"connectionsUsed"`
	ConnectionsMax          int     `json:"connectionsMax"`
	ConnectionUsagePercent  float64 `json:"connectionUsagePercent"`
	LockWaitCount           int     `json:"lockWaitCount"`
	Deadlocks1h             float64 `json:"deadlocks1h"`
	LongTransactionCount    int     `json:"longTransactionCount"`
	OldestPendingTaskAgeSec int64   `json:"oldestPendingTaskAgeSec"`
}

type databaseOptionalSignalsResponse struct {
	QPS               *float64 `json:"qps"`
	WalGeneratedMb24h *float64 `json:"walGeneratedMb24h"`
	CacheHitRate      *float64 `json:"cacheHitRate"`
}

type databaseUnavailableSignalResponse struct {
	Name       string  `json:"name"`
	Scope      string  `json:"scope"`
	ReasonCode string  `json:"reasonCode"`
	Message    *string `json:"message"`
}

type databaseHealthAlertResponse struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	OccurredAt  string `json:"occurredAt"`
}

func (h *HealthHandler) DatabaseHealth(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    "database_unavailable",
			"message": "database is not configured",
		})
		return
	}

	now := time.Now().UTC()
	region := optionalString(os.Getenv("DB_REGION"))

	snapshot := databaseHealthSnapshotResponse{
		Status:        dbHealthStatusOnline,
		ObservedAt:    now.Format(time.RFC3339),
		Role:          "primary",
		Region:        region,
		Version:       "unknown",
		ReadOnly:      false,
		UptimeSeconds: 0,
		CoreSignals: databaseCoreSignalsResponse{
			ProbeLatencyMs:          0,
			ConnectionsUsed:         0,
			ConnectionsMax:          0,
			ConnectionUsagePercent:  0,
			LockWaitCount:           0,
			Deadlocks1h:             0,
			LongTransactionCount:    0,
			OldestPendingTaskAgeSec: 0,
		},
		OptionalSignals: databaseOptionalSignalsResponse{
			QPS:               nil,
			WalGeneratedMb24h: nil,
			CacheHitRate:      nil,
		},
		UnavailableSignals: make([]databaseUnavailableSignalResponse, 0),
		Alerts:             make([]databaseHealthAlertResponse, 0),
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		snapshot.Status = dbHealthStatusOffline
		h.addUnavailable(&snapshot, "database", signalScopeCore, reasonUnknown, err.Error())
		h.addAlert(&snapshot, "critical", "Database unavailable", "Cannot acquire SQL handle")
		c.JSON(http.StatusOK, snapshot)
		return
	}

	stats := sqlDB.Stats()
	snapshot.CoreSignals.ConnectionsUsed = stats.OpenConnections
	snapshot.CoreSignals.ConnectionsMax = stats.MaxOpenConnections
	if snapshot.CoreSignals.ConnectionsMax <= 0 {
		maxConnections, maxErr := h.queryMaxConnections(c.Request.Context())
		if maxErr == nil && maxConnections > 0 {
			snapshot.CoreSignals.ConnectionsMax = maxConnections
		}
	}
	if snapshot.CoreSignals.ConnectionsMax > 0 {
		snapshot.CoreSignals.ConnectionUsagePercent = clampFloat(
			float64(snapshot.CoreSignals.ConnectionsUsed)*100/float64(snapshot.CoreSignals.ConnectionsMax), 0, 100,
		)
	} else {
		h.addUnavailable(&snapshot, "connectionsMax", signalScopeCore, reasonUnknown, "max connections is unavailable")
	}

	probeStart := time.Now()
	probeErr := h.runProbe(c.Request.Context())
	snapshot.CoreSignals.ProbeLatencyMs = math.Max(0, float64(time.Since(probeStart).Milliseconds()))
	if probeErr != nil {
		snapshot.Status = dbHealthStatusOffline
		h.addUnavailable(&snapshot, "probeLatencyMs", signalScopeCore, classifyReason(probeErr), probeErr.Error())
		h.addAlert(&snapshot, "critical", "Probe failed", "Database probe query failed")
		c.JSON(http.StatusOK, snapshot)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 1500*time.Millisecond)
	defer cancel()

	if role, roleErr := h.queryRole(ctx); roleErr != nil {
		h.addUnavailable(&snapshot, "role", signalScopeCore, classifyReason(roleErr), roleErr.Error())
	} else {
		snapshot.Role = role
	}

	if version, versionErr := h.queryVersion(ctx); versionErr != nil {
		h.addUnavailable(&snapshot, "version", signalScopeCore, classifyReason(versionErr), versionErr.Error())
	} else {
		snapshot.Version = version
	}

	if readOnly, readOnlyErr := h.queryReadOnly(ctx); readOnlyErr != nil {
		h.addUnavailable(&snapshot, "readOnly", signalScopeCore, classifyReason(readOnlyErr), readOnlyErr.Error())
	} else {
		snapshot.ReadOnly = readOnly
	}

	if uptime, uptimeErr := h.queryUptimeSeconds(ctx); uptimeErr != nil {
		h.addUnavailable(&snapshot, "uptimeSeconds", signalScopeCore, classifyReason(uptimeErr), uptimeErr.Error())
	} else {
		snapshot.UptimeSeconds = maxInt64(0, uptime)
	}

	if lockWaitCount, lockWaitErr := h.queryLockWaitCount(ctx); lockWaitErr != nil {
		h.addUnavailable(&snapshot, "lockWaitCount", signalScopeCore, classifyReason(lockWaitErr), lockWaitErr.Error())
	} else {
		snapshot.CoreSignals.LockWaitCount = maxInt(0, lockWaitCount)
	}

	if deadlocks1h, ok, deadlocksErr := h.queryDeadlocks1h(ctx); deadlocksErr != nil {
		h.addUnavailable(&snapshot, "deadlocks1h", signalScopeCore, classifyReason(deadlocksErr), deadlocksErr.Error())
	} else if ok {
		snapshot.CoreSignals.Deadlocks1h = maxFloat64(0, deadlocks1h)
	}

	if longTxCount, longTxErr := h.queryLongTransactionCount(ctx); longTxErr != nil {
		h.addUnavailable(&snapshot, "longTransactionCount", signalScopeCore, classifyReason(longTxErr), longTxErr.Error())
	} else {
		snapshot.CoreSignals.LongTransactionCount = maxInt(0, longTxCount)
	}

	if oldestPendingTaskAgeSec, pendingAgeErr := h.queryOldestPendingTaskAgeSec(ctx); pendingAgeErr != nil {
		h.addUnavailable(&snapshot, "oldestPendingTaskAgeSec", signalScopeCore, classifyReason(pendingAgeErr), pendingAgeErr.Error())
	} else {
		snapshot.CoreSignals.OldestPendingTaskAgeSec = maxInt64(0, oldestPendingTaskAgeSec)
	}

	if qps, ok, qpsErr := h.queryQPS(ctx); qpsErr != nil {
		h.addUnavailable(&snapshot, "qps", signalScopeOptional, classifyReason(qpsErr), qpsErr.Error())
	} else if ok {
		snapshot.OptionalSignals.QPS = floatPtr(maxFloat64(0, qps))
	}

	if walMB, ok, walErr := h.queryWalGeneratedMb24h(ctx); walErr != nil {
		h.addUnavailable(&snapshot, "walGeneratedMb24h", signalScopeOptional, classifyReason(walErr), walErr.Error())
	} else if ok {
		snapshot.OptionalSignals.WalGeneratedMb24h = floatPtr(maxFloat64(0, walMB))
	}

	if cacheHit, ok, cacheErr := h.queryCacheHitRate(ctx); cacheErr != nil {
		h.addUnavailable(&snapshot, "cacheHitRate", signalScopeOptional, classifyReason(cacheErr), cacheErr.Error())
	} else if ok {
		snapshot.OptionalSignals.CacheHitRate = floatPtr(clampFloat(cacheHit, 0, 100))
	}

	maintenanceMode := strings.EqualFold(os.Getenv("DB_MAINTENANCE_MODE"), "true")
	if maintenanceMode {
		snapshot.Status = dbHealthStatusMaintenance
		h.addAlert(&snapshot, "info", "Maintenance mode", "Database maintenance mode is enabled")
		c.JSON(http.StatusOK, snapshot)
		return
	}

	h.finalizeSnapshotStatus(&snapshot)

	c.JSON(http.StatusOK, snapshot)
}

func (h *HealthHandler) runProbe(ctx context.Context) error {
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.WithContext(probeCtx).Raw("SELECT 1").Error
}

func (h *HealthHandler) queryRole(ctx context.Context) (string, error) {
	var role string
	err := h.db.WithContext(ctx).Raw("SELECT CASE WHEN pg_is_in_recovery() THEN 'replica' ELSE 'primary' END").Scan(&role).Error
	return role, err
}

func (h *HealthHandler) queryVersion(ctx context.Context) (string, error) {
	var version string
	err := h.db.WithContext(ctx).Raw("SHOW server_version").Scan(&version).Error
	return version, err
}

func (h *HealthHandler) queryReadOnly(ctx context.Context) (bool, error) {
	var readOnlyRaw string
	if err := h.db.WithContext(ctx).Raw("SHOW transaction_read_only").Scan(&readOnlyRaw).Error; err != nil {
		return false, err
	}
	readOnlyRaw = strings.TrimSpace(strings.ToLower(readOnlyRaw))
	return readOnlyRaw == "on" || readOnlyRaw == "true", nil
}

func (h *HealthHandler) queryUptimeSeconds(ctx context.Context) (int64, error) {
	var uptime sql.NullInt64
	err := h.db.WithContext(ctx).Raw("SELECT EXTRACT(EPOCH FROM now() - pg_postmaster_start_time())::bigint").Scan(&uptime).Error
	if err != nil {
		return 0, err
	}
	if !uptime.Valid {
		return 0, nil
	}
	return uptime.Int64, nil
}

func (h *HealthHandler) queryMaxConnections(ctx context.Context) (int, error) {
	var maxConn sql.NullInt64
	err := h.db.WithContext(ctx).Raw("SHOW max_connections").Scan(&maxConn).Error
	if err != nil {
		return 0, err
	}
	if !maxConn.Valid {
		return 0, nil
	}
	return int(maxConn.Int64), nil
}

func (h *HealthHandler) queryQPS(ctx context.Context) (float64, bool, error) {
	var qps sql.NullFloat64
	err := h.db.WithContext(ctx).Raw(`
		SELECT
		  CASE
		    WHEN EXTRACT(EPOCH FROM now() - COALESCE(stats_reset, pg_postmaster_start_time())) > 0
		    THEN (xact_commit + xact_rollback)::double precision /
		         EXTRACT(EPOCH FROM now() - COALESCE(stats_reset, pg_postmaster_start_time()))
		    ELSE NULL
		  END
		FROM pg_stat_database
		WHERE datname = current_database()
	`).Scan(&qps).Error
	if err != nil {
		return 0, false, err
	}
	if !qps.Valid {
		return 0, false, nil
	}
	return qps.Float64, true, nil
}

func (h *HealthHandler) queryWalGeneratedMb24h(ctx context.Context) (float64, bool, error) {
	var walMB sql.NullFloat64
	err := h.db.WithContext(ctx).Raw(`
		SELECT
		  CASE
		    WHEN EXTRACT(EPOCH FROM now() - COALESCE(stats_reset, now())) > 0
		    THEN (wal_bytes / 1024.0 / 1024.0) *
		         (86400.0 / EXTRACT(EPOCH FROM now() - COALESCE(stats_reset, now())))
		    ELSE NULL
		  END
		FROM pg_stat_wal
	`).Scan(&walMB).Error
	if err != nil {
		return 0, false, err
	}
	if !walMB.Valid {
		return 0, false, nil
	}
	return walMB.Float64, true, nil
}

func (h *HealthHandler) queryCacheHitRate(ctx context.Context) (float64, bool, error) {
	var rate sql.NullFloat64
	err := h.db.WithContext(ctx).Raw(`
		SELECT
		  CASE
		    WHEN (blks_hit + blks_read) > 0
		    THEN (blks_hit::double precision / (blks_hit + blks_read)::double precision) * 100
		    ELSE NULL
		  END
		FROM pg_stat_database
		WHERE datname = current_database()
	`).Scan(&rate).Error
	if err != nil {
		return 0, false, err
	}
	if !rate.Valid {
		return 0, false, nil
	}
	return rate.Float64, true, nil
}

func (h *HealthHandler) queryLockWaitCount(ctx context.Context) (int, error) {
	var count sql.NullInt64
	err := h.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)::bigint
		FROM pg_stat_activity
		WHERE datname = current_database()
		  AND wait_event_type = 'Lock'
		  AND state <> 'idle'
	`).Scan(&count).Error
	if err != nil {
		return 0, err
	}
	if !count.Valid {
		return 0, nil
	}
	return int(count.Int64), nil
}

func (h *HealthHandler) queryDeadlocks1h(ctx context.Context) (float64, bool, error) {
	var deadlocks sql.NullFloat64
	err := h.db.WithContext(ctx).Raw(`
		SELECT
		  CASE
		    WHEN EXTRACT(EPOCH FROM now() - COALESCE(stats_reset, pg_postmaster_start_time())) > 0
		    THEN deadlocks::double precision *
		         (3600.0 / EXTRACT(EPOCH FROM now() - COALESCE(stats_reset, pg_postmaster_start_time())))
		    ELSE NULL
		  END
		FROM pg_stat_database
		WHERE datname = current_database()
	`).Scan(&deadlocks).Error
	if err != nil {
		return 0, false, err
	}
	if !deadlocks.Valid {
		return 0, false, nil
	}
	return deadlocks.Float64, true, nil
}

func (h *HealthHandler) queryLongTransactionCount(ctx context.Context) (int, error) {
	var count sql.NullInt64
	err := h.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)::bigint
		FROM pg_stat_activity
		WHERE datname = current_database()
		  AND xact_start IS NOT NULL
		  AND state <> 'idle'
		  AND now() - xact_start > interval '60 seconds'
	`).Scan(&count).Error
	if err != nil {
		return 0, err
	}
	if !count.Valid {
		return 0, nil
	}
	return int(count.Int64), nil
}

func (h *HealthHandler) queryOldestPendingTaskAgeSec(ctx context.Context) (int64, error) {
	var ageSec sql.NullInt64
	err := h.db.WithContext(ctx).Raw(`
		SELECT COALESCE(EXTRACT(EPOCH FROM now() - MIN(created_at)), 0)::bigint
		FROM scan_task
		WHERE status IN ('pending', 'blocked')
	`).Scan(&ageSec).Error
	if err != nil {
		return 0, err
	}
	if !ageSec.Valid {
		return 0, nil
	}
	return ageSec.Int64, nil
}

func (h *HealthHandler) addUnavailable(snapshot *databaseHealthSnapshotResponse, name, scope, reasonCode, message string) {
	snapshot.UnavailableSignals = append(snapshot.UnavailableSignals, databaseUnavailableSignalResponse{
		Name:       name,
		Scope:      scope,
		ReasonCode: reasonCode,
		Message:    optionalString(message),
	})
}

func (h *HealthHandler) addAlert(snapshot *databaseHealthSnapshotResponse, severity, title, description string) {
	snapshot.Alerts = append(snapshot.Alerts, databaseHealthAlertResponse{
		ID:          fmt.Sprintf("%s-%d", strings.ToLower(strings.ReplaceAll(title, " ", "-")), time.Now().UnixNano()),
		Severity:    severity,
		Title:       title,
		Description: description,
		OccurredAt:  time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *HealthHandler) countUnavailable(signals []databaseUnavailableSignalResponse, scope string) int {
	count := 0
	for _, signal := range signals {
		if signal.Scope == scope {
			count++
		}
	}
	return count
}

func (h *HealthHandler) finalizeSnapshotStatus(snapshot *databaseHealthSnapshotResponse) {
	if snapshot == nil {
		return
	}

	coreUnavailable := h.countUnavailable(snapshot.UnavailableSignals, signalScopeCore)
	degraded := coreUnavailable > 0
	if snapshot.CoreSignals.ConnectionUsagePercent >= 85 {
		degraded = true
		h.addAlert(snapshot, "warning", "High connection usage", "Database connections exceed 85% usage")
	}
	if snapshot.CoreSignals.LockWaitCount >= 5 {
		degraded = true
		h.addAlert(snapshot, "warning", "Lock waits high", "Lock wait sessions exceed threshold")
	}
	if snapshot.CoreSignals.Deadlocks1h >= 1 {
		degraded = true
		h.addAlert(snapshot, "warning", "Deadlocks detected", "Deadlocks per hour is above 1")
	}
	if snapshot.CoreSignals.LongTransactionCount >= 5 {
		degraded = true
		h.addAlert(snapshot, "warning", "Long transactions high", "Long transaction count is above 5")
	}
	if snapshot.CoreSignals.OldestPendingTaskAgeSec >= 600 {
		degraded = true
		h.addAlert(snapshot, "warning", "Task backlog age high", "Oldest pending task age exceeds 600 seconds")
	}
	if snapshot.CoreSignals.ProbeLatencyMs >= 500 {
		degraded = true
		h.addAlert(snapshot, "warning", "Probe latency high", "Probe latency exceeds 500 ms")
	}
	if snapshot.Role == "primary" && snapshot.ReadOnly {
		degraded = true
		h.addAlert(snapshot, "warning", "Primary is read-only", "Primary role reports read-only mode")
	}
	if coreUnavailable > 0 {
		h.addAlert(snapshot, "warning", "Core signals unavailable", fmt.Sprintf("%d core signals are unavailable", coreUnavailable))
	}

	if degraded {
		snapshot.Status = dbHealthStatusDegraded
		return
	}
	snapshot.Status = dbHealthStatusOnline
}

func classifyReason(err error) string {
	if err == nil {
		return reasonUnknown
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "permission"):
		return reasonPermissionDenied
	case strings.Contains(msg, "timeout"):
		return reasonTimeout
	case strings.Contains(msg, "does not exist"), strings.Contains(msg, "undefined"):
		return reasonUnsupported
	case strings.Contains(msg, "query"):
		return reasonQueryFailed
	default:
		return reasonUnknown
	}
}

func optionalString(v string) *string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func floatPtr(v float64) *float64 {
	return &v
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
