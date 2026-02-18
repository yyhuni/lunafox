export type DatabaseHealthStatus = 'online' | 'degraded' | 'maintenance' | 'offline'

export type DatabaseRole = 'primary' | 'replica'

export type DatabaseAlertSeverity = 'info' | 'warning' | 'critical'

export interface DatabaseHealthSignals {
  latencyP95Ms: number
  qps: number
  connectionsUsed: number
  connectionsMax: number
  replicationLagMs: number
  cacheHitRate: number
  diskUsagePercent: number
  walGeneratedMb24h: number
  deadlocks1h: number
  longTransactions: number
  backupFreshnessMinutes: number
}

export interface DatabaseHealthAlert {
  id: string
  severity: DatabaseAlertSeverity
  title: string
  description: string
  time: string
}

export interface DatabaseHealthSnapshot {
  status: DatabaseHealthStatus
  role: DatabaseRole
  region: string
  version: string
  readOnly: boolean
  lastCheckAt: string
  uptime: string
  signals: DatabaseHealthSignals
  alerts: DatabaseHealthAlert[]
}
