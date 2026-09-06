import type { DatabaseHealthSnapshot } from '@/types/database-health.types'

export const mockDatabaseHealth: DatabaseHealthSnapshot = {
  status: 'degraded',
  observedAt: '2026-02-23T08:15:21Z',
  role: 'primary',
  version: 'PostgreSQL 15.4',
  readOnly: false,
  uptimeSeconds: 1051200,
  databaseSizeBytes: 42 * 1024 ** 3,
  coreSignals: {
    probeLatencyMs: 28,
    connectionsUsed: 42,
    connectionsMax: 120,
    connectionUsagePercent: 35,
    lockWaitCount: 0,
    deadlocks1h: 0,
    longTransactionCount: 0,
    oldestPendingTaskAgeSec: 900,
  },
  optionalSignals: {
    qps: 512,
    walGeneratedMb24h: 820,
    cacheHitRate: 99.3,
  },
  unavailableSignals: [],
  findings: [
    {
      severity: 'critical',
      signal: 'oldestPendingTaskAgeSec',
      title: 'Task backlog age critical',
      description: 'The oldest pending or blocked scan task has waited longer than the 2-minute target window.',
      evidence: ['oldestPendingTaskAgeSec=900', 'target<2m', 'criticalThreshold=10m'],
      recommendation: 'Check Agent availability and execution capacity, task leasing, and scan task status writes before new scan results are delayed further.',
    },
  ],
  alerts: [
    {
      id: 'long-tx-1',
      severity: 'critical',
      title: 'Task backlog age critical',
      description: 'Oldest pending scan task has waited at least 10 minutes',
      occurredAt: '2026-02-23T08:08:00Z',
    },
  ],
}
