export type DatabaseHealthStatus = 'online' | 'degraded' | 'maintenance' | 'offline'

export type DatabaseRole = 'primary' | 'replica'

export type DatabaseAlertSeverity = 'info' | 'warning' | 'critical'

export interface DatabaseHealthCoreSignals {
  probeLatencyMs: number
  /** Established pool connections, including both in-use and idle sessions. */
  connectionsUsed: number
  connectionsMax: number
  /** The established-connection count as a percentage of the configured limit. */
  connectionUsagePercent: number
  lockWaitCount: number
  deadlocks1h: number
  longTransactionCount: number
  oldestPendingTaskAgeSec: number
}

export interface DatabaseHealthOptionalSignals {
  qps: number | null
  walGeneratedMb24h: number | null
  cacheHitRate: number | null
}

export type DatabaseSignalScope = 'core' | 'optional'

export type DatabaseUnavailableReason =
  | 'permission_denied'
  | 'timeout'
  | 'unsupported'
  | 'query_failed'
  | 'unknown'

export interface DatabaseUnavailableSignal {
  name: string
  scope: DatabaseSignalScope
  reasonCode: DatabaseUnavailableReason
  message: string | null
}

export interface DatabaseHealthAlert {
  id: string
  severity: DatabaseAlertSeverity
  title: string
  description: string
  occurredAt: string
}

export interface DatabaseHealthFinding {
  severity: DatabaseAlertSeverity
  signal: string
  title: string
  description: string
  evidence: string[]
  recommendation: string
}

export interface DatabaseHealthSnapshot {
  status: DatabaseHealthStatus
  observedAt: string
  role: DatabaseRole
  version: string
  readOnly: boolean
  uptimeSeconds: number
  databaseSizeBytes: number | null
  coreSignals: DatabaseHealthCoreSignals
  optionalSignals: DatabaseHealthOptionalSignals
  unavailableSignals: DatabaseUnavailableSignal[]
  findings: DatabaseHealthFinding[]
  alerts: DatabaseHealthAlert[]
}
