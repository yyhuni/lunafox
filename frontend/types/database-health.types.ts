export type DatabaseHealthStatus = 'online' | 'degraded' | 'maintenance' | 'offline'

export type DatabaseRole = 'primary' | 'replica'

export type DatabaseAlertSeverity = 'info' | 'warning' | 'critical'

export interface DatabaseHealthCoreSignals {
  probeLatencyMs: number
  connectionsUsed: number
  connectionsMax: number
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

export interface DatabaseHealthSnapshot {
  status: DatabaseHealthStatus
  observedAt: string
  role: DatabaseRole
  region: string | null
  version: string
  readOnly: boolean
  uptimeSeconds: number
  coreSignals: DatabaseHealthCoreSignals
  optionalSignals: DatabaseHealthOptionalSignals
  unavailableSignals: DatabaseUnavailableSignal[]
  alerts: DatabaseHealthAlert[]
}
