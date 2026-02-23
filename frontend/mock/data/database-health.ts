import type { DatabaseHealthSnapshot } from '@/types/database-health.types'

export const mockDatabaseHealth: DatabaseHealthSnapshot = {
  status: 'online',
  observedAt: '2026-02-23T08:15:00Z',
  role: 'primary',
  region: 'ap-southeast-1',
  version: 'PostgreSQL 15.4',
  readOnly: false,
  uptimeSeconds: 1051200,
  coreSignals: {
    probeLatencyMs: 28,
    connectionsUsed: 42,
    connectionsMax: 120,
    connectionUsagePercent: 35,
    lockWaitCount: 0,
    deadlocks1h: 0,
    longTransactionCount: 0,
    oldestPendingTaskAgeSec: 180,
  },
  optionalSignals: {
    qps: 512,
    walGeneratedMb24h: 820,
    cacheHitRate: 99.3,
  },
  unavailableSignals: [],
  alerts: [
    {
      id: 'autovacuum-1',
      severity: 'info',
      title: 'Autovacuum running',
      description: '2 tables in progress',
      occurredAt: '2026-02-23T08:12:00Z',
    },
    {
      id: 'long-tx-1',
      severity: 'warning',
      title: 'Long transaction',
      description: '1 session > 5m',
      occurredAt: '2026-02-23T08:08:00Z',
    },
  ],
}
