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

const (
	databasePendingTaskBacklogTargetSec   int64 = 2 * 60
	databasePendingTaskBacklogCriticalSec int64 = 10 * 60
)

type databaseHealthSnapshotResponse struct {
	Status             string                              `json:"status"`
	ObservedAt         string                              `json:"observedAt"`
	Role               string                              `json:"role"`
	Version            string                              `json:"version"`
	ReadOnly           bool                                `json:"readOnly"`
	UptimeSeconds      int64                               `json:"uptimeSeconds"`
	DatabaseSizeBytes  *int64                              `json:"databaseSizeBytes"`
	CoreSignals        databaseCoreSignalsResponse         `json:"coreSignals"`
	OptionalSignals    databaseOptionalSignalsResponse     `json:"optionalSignals"`
	UnavailableSignals []databaseUnavailableSignalResponse `json:"unavailableSignals"`
	Findings           []databaseHealthFindingResponse     `json:"findings"`
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

type databaseHealthFindingResponse struct {
	Severity       string   `json:"severity"`
	Signal         string   `json:"signal"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Evidence       []string `json:"evidence"`
	Recommendation string   `json:"recommendation"`
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

	snapshot := databaseHealthSnapshotResponse{
		Status:            dbHealthStatusOnline,
		ObservedAt:        now.Format(time.RFC3339),
		Role:              "primary",
		Version:           "unknown",
		ReadOnly:          false,
		UptimeSeconds:     0,
		DatabaseSizeBytes: nil,
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
		Findings:           make([]databaseHealthFindingResponse, 0),
		Alerts:             make([]databaseHealthAlertResponse, 0),
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		snapshot.Status = dbHealthStatusOffline
		h.addUnavailable(&snapshot, "database", signalScopeCore, reasonUnknown, err.Error())
		h.addAlert(&snapshot, "critical", "Database unavailable", "Cannot acquire SQL handle")
		snapshot.Findings = h.buildDatabaseHealthFindings(&snapshot)
		c.JSON(http.StatusOK, snapshot)
		return
	}

	stats := sqlDB.Stats()
	// OpenConnections includes in-use and idle pool sessions. Keep the stable
	// API field name for compatibility; presentation must label this as established connections.
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
		snapshot.Findings = h.buildDatabaseHealthFindings(&snapshot)
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

	if databaseSizeBytes, ok, sizeErr := h.queryDatabaseSizeBytes(ctx); sizeErr != nil {
		h.addUnavailable(&snapshot, "databaseSizeBytes", signalScopeOptional, classifyReason(sizeErr), sizeErr.Error())
	} else if ok {
		snapshot.DatabaseSizeBytes = int64Ptr(maxInt64(0, databaseSizeBytes))
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
		snapshot.Findings = h.buildDatabaseHealthFindings(&snapshot)
		c.JSON(http.StatusOK, snapshot)
		return
	}

	h.finalizeSnapshotStatus(&snapshot)
	snapshot.Findings = h.buildDatabaseHealthFindings(&snapshot)

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

func (h *HealthHandler) queryDatabaseSizeBytes(ctx context.Context) (int64, bool, error) {
	var sizeBytes sql.NullInt64
	err := h.db.WithContext(ctx).Raw("SELECT pg_database_size(current_database())::bigint").Scan(&sizeBytes).Error
	if err != nil {
		return 0, false, err
	}
	if !sizeBytes.Valid {
		return 0, false, nil
	}
	return sizeBytes.Int64, true, nil
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
		SELECT COALESCE(EXTRACT(EPOCH FROM now() - MIN(st.created_at)), 0)::bigint
		FROM scan_task st
		JOIN scan s ON s.id = st.scan_id
		WHERE st.status IN ('pending', 'blocked')
		  AND s.status IN ('pending', 'running')
		  AND s.deleted_at IS NULL
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
		h.addAlert(snapshot, "warning", "Established connection count is high", "Established database connections exceed 85% of the configured pool limit.")
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
	if snapshot.CoreSignals.OldestPendingTaskAgeSec >= databasePendingTaskBacklogCriticalSec {
		degraded = true
		h.addAlert(snapshot, "critical", "Task backlog age critical", "Oldest pending scan task has waited at least 10 minutes")
	} else if snapshot.CoreSignals.OldestPendingTaskAgeSec >= databasePendingTaskBacklogTargetSec {
		degraded = true
		h.addAlert(snapshot, "warning", "Task backlog age high", "Oldest pending scan task has waited at least 2 minutes")
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

func (h *HealthHandler) buildDatabaseHealthFindings(snapshot *databaseHealthSnapshotResponse) []databaseHealthFindingResponse {
	if snapshot == nil {
		return []databaseHealthFindingResponse{}
	}

	findings := make([]databaseHealthFindingResponse, 0)
	core := snapshot.CoreSignals

	if core.ConnectionUsagePercent >= 85 {
		findings = append(findings, databaseHealthFindingResponse{
			Severity:       "warning",
			Signal:         "connectionUsagePercent",
			Title:          "Established connection count is high",
			Description:    "Established database connections are close to the configured pool limit.",
			Evidence:       []string{fmt.Sprintf("connectionUsagePercent=%.1f", core.ConnectionUsagePercent), fmt.Sprintf("connections=%d/%d", core.ConnectionsUsed, core.ConnectionsMax)},
			Recommendation: "Check the configured pool limit, sustained in-use demand, and long-running sessions or workers.",
		})
	}
	if core.LockWaitCount >= 5 {
		findings = append(findings, databaseHealthFindingResponse{
			Severity:       "warning",
			Signal:         "lockWaitCount",
			Title:          "Lock waits are elevated",
			Description:    "Active sessions are waiting on database locks.",
			Evidence:       []string{fmt.Sprintf("lockWaitCount=%d", core.LockWaitCount)},
			Recommendation: "Inspect long transactions and recent writes that may be blocking task updates.",
		})
	}
	if core.Deadlocks1h >= 1 {
		findings = append(findings, databaseHealthFindingResponse{
			Severity:       "critical",
			Signal:         "deadlocks1h",
			Title:          "Deadlocks were detected",
			Description:    "Recent deadlocks can roll back task or scan status writes.",
			Evidence:       []string{fmt.Sprintf("deadlocks1h=%.2f", core.Deadlocks1h)},
			Recommendation: "Review concurrent update paths and retry logs around the affected time window.",
		})
	}
	if core.LongTransactionCount >= 5 {
		findings = append(findings, databaseHealthFindingResponse{
			Severity:       "warning",
			Signal:         "longTransactionCount",
			Title:          "Long transactions are elevated",
			Description:    "Several active transactions have been open longer than expected.",
			Evidence:       []string{fmt.Sprintf("longTransactionCount=%d", core.LongTransactionCount)},
			Recommendation: "Find sessions with old transaction start times and confirm they are not blocking scan task writes.",
		})
	}
	if core.OldestPendingTaskAgeSec >= databasePendingTaskBacklogTargetSec {
		severity := "warning"
		if core.OldestPendingTaskAgeSec >= databasePendingTaskBacklogCriticalSec {
			severity = "critical"
		}
		findings = append(findings, databaseHealthFindingResponse{
			Severity:    severity,
			Signal:      "oldestPendingTaskAgeSec",
			Title:       "Task backlog age high",
			Description: "The oldest pending or blocked scan task has waited longer than the 2-minute target window.",
			Evidence: []string{
				fmt.Sprintf("oldestPendingTaskAgeSec=%d", core.OldestPendingTaskAgeSec),
				"target<2m",
				"criticalThreshold=10m",
			},
			Recommendation: "Check active Agent capacity, task leasing, and scan task status writes before new scan results are delayed further.",
		})
	}
	if core.ProbeLatencyMs >= 500 {
		findings = append(findings, databaseHealthFindingResponse{
			Severity:       "critical",
			Signal:         "probeLatencyMs",
			Title:          "Database probe latency is high",
			Description:    "The basic database probe is slower than the critical threshold.",
			Evidence:       []string{fmt.Sprintf("probeLatencyMs=%.0f", core.ProbeLatencyMs)},
			Recommendation: "Check database load, network path, and connection saturation before retrying scan operations.",
		})
	}
	if snapshot.Role == "primary" && snapshot.ReadOnly {
		findings = append(findings, databaseHealthFindingResponse{
			Severity:       "critical",
			Signal:         "readOnly",
			Title:          "Primary database is read-only",
			Description:    "The primary role reports read-only mode, so writes may fail.",
			Evidence:       []string{"role=primary", "readOnly=true"},
			Recommendation: "Confirm maintenance mode, failover state, and transaction_read_only before starting write-heavy work.",
		})
	}

	for _, signal := range snapshot.UnavailableSignals {
		if signal.Scope != signalScopeCore {
			continue
		}
		evidence := []string{fmt.Sprintf("reasonCode=%s", signal.ReasonCode)}
		if signal.Message != nil {
			evidence = append(evidence, fmt.Sprintf("message=%s", *signal.Message))
		}
		findings = append(findings, databaseHealthFindingResponse{
			Severity:       "warning",
			Signal:         signal.Name,
			Title:          "Core database signal is unavailable",
			Description:    "A required health signal could not be collected, so the report may be incomplete.",
			Evidence:       evidence,
			Recommendation: "Check database permissions, PostgreSQL compatibility, and query errors for this signal.",
		})
	}

	return findings
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

func int64Ptr(v int64) *int64 {
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
