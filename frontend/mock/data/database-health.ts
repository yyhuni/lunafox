import type { DatabaseHealthSnapshot } from '@/types/database-health.types'

export const mockDatabaseHealth: DatabaseHealthSnapshot = {
  status: 'online',
  role: 'primary',
  region: 'ap-southeast-1',
  version: 'PostgreSQL 15.4',
  readOnly: false,
  lastCheckAt: '2026-01-26 10:12',
  uptime: '12d 4h',
  signals: {
    latencyP95Ms: 28,
    qps: 512,
    connectionsUsed: 42,
    connectionsMax: 120,
    replicationLagMs: 120,
    cacheHitRate: 99.3,
    diskUsagePercent: 68,
    walGeneratedMb24h: 820,
    deadlocks1h: 0,
    longTransactions: 1,
    backupFreshnessMinutes: 42,
  },
  alerts: [
    {
      id: 'autovacuum-1',
      severity: 'info',
      title: 'Autovacuum running',
      description: '2 tables in progress',
      time: '3m ago',
    },
    {
      id: 'long-tx-1',
      severity: 'warning',
      title: 'Long transaction',
      description: '1 session > 5m',
      time: '7m ago',
    },
  ],
}
