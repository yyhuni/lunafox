import type { ScanRecord, GetScansResponse, ScanListRecord, ScanLog, ScanStatus } from '@/types/scan.types'
import type { ScanStatistics } from '@/services/scan.service'
import { getMockCanonicalWorkflowConfiguration } from '@/mock/data/scan-workflows'

const mockContainerExitFailureMessage = 'Engine Container exited with code 1 while running subdomain_discovery: recon is required'

export const mockScans: ScanListRecord[] = [
  {
    id: 1,
    targetId: 1,
    target: { id: 1, name: 'acme.com', displayName: 'acme.com', type: 'domain' },
    cachedStats: {
      subdomainsCount: 156,
      websitesCount: 89,
      directoriesCount: 234,
      endpointsCount: 2341,
      ipsCount: 45,
      screenshotsCount: 50,
      vulnsTotal: 23,
      vulnsCritical: 1,
      vulnsHigh: 4,
      vulnsMedium: 8,
      vulnsLow: 10,
    },
    scanWorkflow: 'default',
    plannedEngineIds: ['engine.lunafox.subdomain_discovery', 'engine.lunafox.port_scan', 'engine.lunafox.web_crawling', 'engine.lunafox.directory_scan', 'engine.lunafox.screenshot', 'engine.lunafox.nuclei_vulnerability'],
    triggerType: 'manual',
    inputSource: 'scanSnapshot',
    createdAt: '2024-12-28T10:00:00Z',
    stoppedAt: '2024-12-28T10:03:42Z',
    status: 'succeeded',
    progress: 100,
    currentStage: 'nuclei_vulnerability',
  },
  {
    id: 2,
    targetId: 2,
    target: { id: 2, name: 'acme.io', displayName: 'acme.io', type: 'domain' },
    cachedStats: {
      subdomainsCount: 78,
      websitesCount: 45,
      directoriesCount: 123,
      endpointsCount: 892,
      ipsCount: 23,
      screenshotsCount: 30,
      vulnsTotal: 12,
      vulnsCritical: 0,
      vulnsHigh: 2,
      vulnsMedium: 5,
      vulnsLow: 5,
    },
    scanWorkflow: 'default',
    plannedEngineIds: ['engine.lunafox.subdomain_discovery', 'engine.lunafox.web_crawling', 'engine.lunafox.nuclei_vulnerability'],
    triggerType: 'scheduled',
    inputSource: 'scanSnapshot',
    createdAt: '2024-12-27T14:30:00Z',
    status: 'running',
    progress: 65,
    currentStage: 'web_crawling',
  },
  {
    id: 3,
    targetId: 3,
    target: { id: 3, name: 'techstart.io', displayName: 'techstart.io', type: 'domain' },
    cachedStats: {
      subdomainsCount: 45,
      websitesCount: 28,
      directoriesCount: 89,
      endpointsCount: 567,
      ipsCount: 12,
      screenshotsCount: 20,
      vulnsTotal: 8,
      vulnsCritical: 0,
      vulnsHigh: 1,
      vulnsMedium: 3,
      vulnsLow: 4,
    },
    scanWorkflow: 'default',
    plannedEngineIds: ['engine.lunafox.subdomain_discovery', 'engine.lunafox.web_crawling', 'engine.lunafox.nuclei_vulnerability'],
    triggerType: 'ai',
    inputSource: 'scanSnapshot',
    createdAt: '2024-12-26T08:45:00Z',
    status: 'succeeded',
    progress: 100,
  },
  {
    id: 4,
    targetId: 4,
    target: {
      id: 4,
      name: 'globalfinance.test.globalfinance.globalfinance.globalfinance.com',
      displayName: 'globalfinance.test.globalfinance.globalfinance.globalfinance.com',
      type: 'domain',
    },
    cachedStats: {
      subdomainsCount: 0,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 0,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    scanWorkflow: 'default',
    plannedEngineIds: ['engine.lunafox.subdomain_discovery', 'engine.lunafox.web_crawling', 'engine.lunafox.nuclei_vulnerability'],
    triggerType: 'manual',
    inputSource: 'scanSnapshot',
    createdAt: '2024-12-25T16:20:00Z',
    status: 'failed',
    progress: 15,
    errorMessage: 'Connection timeout: Unable to reach target',
    failure: {
      kind: 'engine_execution_failed',
      message: 'nuclei_vulnerability failed after repeated connection timeouts',
    },
  },
  {
    id: 5,
    targetId: 6,
    target: { id: 6, name: 'healthcareplus.com', displayName: 'healthcareplus.com', type: 'domain' },
    cachedStats: {
      subdomainsCount: 34,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 8,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    scanWorkflow: 'default',
    plannedEngineIds: ['engine.lunafox.subdomain_discovery'],
    triggerType: 'scheduled',
    inputSource: 'scanSnapshot',
    createdAt: '2024-12-29T09:00:00Z',
    status: 'running',
    progress: 25,
    currentStage: 'subdomain_discovery',
  },
  {
    id: 6,
    targetId: 7,
    target: { id: 7, name: 'edutech.io', displayName: 'edutech.io', type: 'domain' },
    cachedStats: {
      subdomainsCount: 0,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 0,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    scanWorkflow: 'default',
    plannedEngineIds: ['engine.lunafox.subdomain_discovery'],
    triggerType: 'manual',
    inputSource: 'scanSnapshot',
    createdAt: '2024-12-29T10:30:00Z',
    status: 'pending',
    progress: 0,
  },
  {
    id: 7,
    targetId: 8,
    target: { id: 8, name: 'retailmax.com', displayName: 'retailmax.com', type: 'domain' },
    cachedStats: {
      subdomainsCount: 89,
      websitesCount: 56,
      directoriesCount: 178,
      endpointsCount: 1234,
      ipsCount: 28,
      screenshotsCount: 40,
      vulnsTotal: 15,
      vulnsCritical: 0,
      vulnsHigh: 3,
      vulnsMedium: 6,
      vulnsLow: 6,
    },
    scanWorkflow: 'default',
    plannedEngineIds: ['engine.lunafox.subdomain_discovery'],
    triggerType: 'manual',
    inputSource: 'scanSnapshot',
    createdAt: '2024-12-21T10:45:00Z',
    status: 'succeeded',
    progress: 100,
  },
  {
    id: 8,
    targetId: 11,
    target: { id: 11, name: 'mediastream.tv', displayName: 'mediastream.tv', type: 'domain' },
    cachedStats: {
      subdomainsCount: 67,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 15,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    scanWorkflow: 'default',
    plannedEngineIds: ['engine.lunafox.subdomain_discovery'],
    triggerType: 'scheduled',
    inputSource: 'scanSnapshot',
    createdAt: '2024-12-29T08:00:00Z',
    status: 'running',
    progress: 45,
    currentStage: 'web_crawling',
  },
  {
    id: 9,
    targetId: 1,
    target: { id: 1, name: 'test.com', displayName: 'test.com', type: 'domain' },
    cachedStats: {
      subdomainsCount: 0,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 0,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    scanWorkflow: 'default',
    plannedEngineIds: ['engine.lunafox.subdomain_discovery', 'engine.lunafox.port_scan'],
    triggerType: 'manual',
    inputSource: 'scanSnapshot',
    createdAt: '2026-07-03T07:44:42Z',
    stoppedAt: '2026-07-03T07:44:49Z',
    status: 'failed',
    progress: 50,
    currentStage: 'subdomain_discovery',
    errorMessage: mockContainerExitFailureMessage,
    failure: {
      kind: 'container_exit_failed',
      message: mockContainerExitFailureMessage,
    },
  },
  {
    id: 10,
    targetId: 12,
    target: { id: 12, name: 'disabled-step.example.com', displayName: 'disabled-step.example.com', type: 'domain' },
    cachedStats: {
      subdomainsCount: 0,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 0,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    scanWorkflow: 'default',
    plannedEngineIds: [],
    triggerType: 'manual',
    inputSource: 'scanSnapshot',
    createdAt: '2024-12-20T08:00:00Z',
    stoppedAt: '2024-12-20T08:00:01Z',
    status: 'succeeded',
    progress: 100,
  },
  {
    id: 11,
    targetId: 10,
    target: { id: 10, name: 'cloudnine.host', displayName: 'cloudnine.host', type: 'domain' },
    cachedStats: {
      subdomainsCount: 0,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 0,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    scanWorkflow: 'default',
    plannedEngineIds: ['engine.lunafox.subdomain_discovery', 'engine.lunafox.web_crawling', 'engine.lunafox.nuclei_vulnerability'],
    triggerType: 'scheduled',
    inputSource: 'scanSnapshot',
    createdAt: '2024-12-29T10:15:00Z',
    status: 'running',
    progress: 18,
    currentStage: 'subdomain_discovery',
  },
  {
    id: 12,
    targetId: 12,
    target: { id: 12, name: 'api.acme.com', displayName: 'api.acme.com', type: 'domain' },
    cachedStats: {
      subdomainsCount: 0,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 0,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    scanWorkflow: 'default',
    plannedEngineIds: ['engine.lunafox.subdomain_discovery', 'engine.lunafox.web_crawling', 'engine.lunafox.nuclei_vulnerability'],
    triggerType: 'manual',
    inputSource: 'scanSnapshot',
    createdAt: '2024-12-29T07:30:00Z',
    status: 'running',
    progress: 32,
    currentStage: 'web_crawling',
  },
]

const mockScanRuntimeDetailsById: Record<
  number,
  Pick<ScanRecord, 'agentId' | 'agentName' | 'configuration' | 'runtimeTasks'>
> = {
  1: {
    agentId: 101,
    agentName: 'scan-agent-east-01.example.internal',
    configuration: getMockCanonicalWorkflowConfiguration(),
    runtimeTasks: [
      {
        id: 1001,
        name: 'scans/1/tasks/1001',
        stepId: 'subdomain_discovery',
        stageId: 'discovery',
        engineId: 'engine.lunafox.subdomain_discovery',
        status: 'succeeded',
        order: 0,
        startedAt: '2024-12-28T10:00:00Z',
        duration: 72,
        diagnostics: {
          compatibilityRevision: 'engine-execution-diagnostics-r1',
          availability: 'available',
          resultState: 'complete',
          resultTypeWatermarks: [],
        },
      },
      {
        id: 1002,
        name: 'scans/1/tasks/1002',
        stepId: 'port_scan',
        stageId: 'discovery',
        engineId: 'engine.lunafox.port_scan',
        status: 'succeeded',
        order: 1,
        startedAt: '2024-12-28T10:01:12Z',
        duration: 68,
      },
      {
        id: 1003,
        name: 'scans/1/tasks/1003',
        stepId: 'web_crawling',
        stageId: 'crawl',
        engineId: 'engine.lunafox.web_crawling',
        status: 'succeeded',
        order: 2,
        startedAt: '2024-12-28T10:02:20Z',
        duration: 86,
      },
      {
        id: 1004,
        name: 'scans/1/tasks/1004',
        stepId: 'directory_scan',
        stageId: 'crawl',
        engineId: 'engine.lunafox.directory_scan',
        status: 'succeeded',
        order: 3,
        startedAt: '2024-12-28T10:03:46Z',
        duration: 74,
      },
      {
        id: 1005,
        name: 'scans/1/tasks/1005',
        stepId: 'screenshot',
        stageId: 'evidence',
        engineId: 'engine.lunafox.screenshot',
        status: 'succeeded',
        order: 4,
        startedAt: '2024-12-28T10:05:00Z',
        duration: 53,
      },
      {
        id: 1006,
        name: 'scans/1/tasks/1006',
        stepId: 'nuclei_vulnerability',
        stageId: 'risk',
        engineId: 'engine.lunafox.nuclei_vulnerability',
        status: 'succeeded',
        order: 5,
        startedAt: '2024-12-28T10:05:53Z',
        duration: 109,
      },
    ],
  },
  2: {
    agentId: 102,
    agentName: 'scan-agent-02.example.internal',
    runtimeTasks: [
      {
        id: 2001,
        name: 'scans/2/tasks/2001',
        stepId: 'subdomain_discovery',
        stageId: 'discovery',
        engineId: 'engine.lunafox.subdomain_discovery',
        status: 'succeeded',
        order: 0,
        startedAt: '2024-12-27T14:30:00Z',
        duration: 1200,
      },
      {
        id: 2002,
        name: 'scans/2/tasks/2002',
        stepId: 'web_crawling',
        stageId: 'crawl',
        engineId: 'engine.lunafox.web_crawling',
        status: 'running',
        order: 1,
        startedAt: '2024-12-27T14:50:00Z',
      },
    ],
  },
  3: {
    agentId: 103,
    agentName: 'scan-agent-prod-us-east-01.example.internal',
  },
  4: {
    agentId: 104,
    agentName: 'scan-agent-timeout-lab-01',
    runtimeTasks: [
      {
        id: 4001,
        name: 'scans/4/tasks/4001',
        stepId: 'subdomain_discovery',
        stageId: 'discovery',
        engineId: 'engine.lunafox.subdomain_discovery',
        status: 'succeeded',
        order: 0,
        startedAt: '2024-12-25T16:20:00Z',
        duration: 35,
      },
      {
        id: 4002,
        name: 'scans/4/tasks/4002',
        stepId: 'web_crawling',
        stageId: 'crawl',
        engineId: 'engine.lunafox.web_crawling',
        status: 'succeeded',
        order: 1,
        startedAt: '2024-12-25T16:20:35Z',
        duration: 48,
      },
      {
        id: 4003,
        name: 'scans/4/tasks/4003',
        stepId: 'nuclei_vulnerability',
        stageId: 'risk',
        engineId: 'engine.lunafox.nuclei_vulnerability',
        status: 'failed',
        order: 2,
        startedAt: '2024-12-25T16:21:23Z',
        duration: 12,
        error: 'connection timeout while probing https://globalfinance.test.globalfinance.globalfinance.globalfinance.com',
        failureKind: 'engine_execution_failed',
        diagnostics: {
          compatibilityRevision: 'engine-execution-diagnostics-r1',
          availability: 'available',
          resultState: 'partial',
          failedStage: 'result_submit',
          errorType: 'result_submit_failed',
          resultTypeWatermarks: [
            {
              resultType: 'network.port',
              receivedItems: 8,
              encodedItems: 8,
              submittedItems: 8,
              acknowledgedItems: 5,
              submittedBatches: 2,
              acknowledgedBatches: 1,
            },
          ],
        },
      },
    ],
  },
  5: {
    agentId: 105,
    agentName: 'scan-agent-west-02.example.internal',
    runtimeTasks: [
      {
        id: 5001,
        name: 'scans/5/tasks/5001',
        stepId: 'subdomain_discovery',
        stageId: 'discovery',
        engineId: 'engine.lunafox.subdomain_discovery',
        status: 'running',
        order: 0,
        startedAt: '2024-12-29T09:00:00Z',
      },
      {
        id: 5002,
        name: 'scans/5/tasks/5002',
        stepId: 'web_crawling',
        stageId: 'crawl',
        engineId: 'engine.lunafox.web_crawling',
        status: 'pending',
        order: 1,
      },
      {
        id: 5003,
        name: 'scans/5/tasks/5003',
        stepId: 'nuclei_vulnerability',
        stageId: 'risk',
        engineId: 'engine.lunafox.nuclei_vulnerability',
        status: 'pending',
        order: 2,
      },
    ],
  },
  6: {
    agentId: 106,
    agentName: 'local',
  },
  7: {
    agentId: 107,
    agentName: 'retailmax-long-lived-runner-0001.internal.cluster',
  },
  8: {
    agentId: 108,
    agentName: 'scan-agent-central-03.example.internal',
    runtimeTasks: [
      {
        id: 8001,
        name: 'scans/8/tasks/8001',
        stepId: 'subdomain_discovery',
        stageId: 'discovery',
        engineId: 'engine.lunafox.subdomain_discovery',
        status: 'succeeded',
        order: 0,
        startedAt: '2024-12-29T08:00:00Z',
        duration: 900,
      },
      {
        id: 8002,
        name: 'scans/8/tasks/8002',
        stepId: 'web_crawling',
        stageId: 'crawl',
        engineId: 'engine.lunafox.web_crawling',
        status: 'running',
        order: 1,
        startedAt: '2024-12-29T08:15:00Z',
      },
      {
        id: 8003,
        name: 'scans/8/tasks/8003',
        stepId: 'nuclei_vulnerability',
        stageId: 'risk',
        engineId: 'engine.lunafox.nuclei_vulnerability',
        status: 'pending',
        order: 2,
      },
    ],
  },
  9: {
    agentId: 109,
    agentName: 'scan-agent-central-03.example.internal',
    runtimeTasks: [
      {
        id: 9001,
        name: 'scans/9/tasks/9001',
        stepId: 'subdomain_discovery',
        stageId: 'discovery',
        engineId: 'engine.lunafox.subdomain_discovery',
        status: 'failed',
        order: 0,
        startedAt: '2026-07-03T07:44:42Z',
        duration: 7,
        error: mockContainerExitFailureMessage,
        failureKind: 'container_exit_failed',
      },
      {
        id: 9002,
        name: 'scans/9/tasks/9002',
        stepId: 'port_scan',
        stageId: 'discovery',
        engineId: 'engine.lunafox.port_scan',
        status: 'cancelled',
        order: 1,
      },
    ],
  },
  10: {
    agentName: 'planning-only-skip-fixture',
    runtimeTasks: [
      {
        id: 10001,
        name: 'scans/10/tasks/10001',
        stepId: 'port_scan',
        stageId: 'discovery',
        engineId: 'engine.lunafox.port_scan',
        status: 'skipped',
        skipReason: 'user_disabled',
        order: 0,
      },
      {
        id: 10002,
        name: 'scans/10/tasks/10002',
        stepId: 'web_crawling',
        stageId: 'crawl',
        engineId: 'engine.lunafox.web_crawling',
        status: 'skipped',
        skipReason: 'target_not_applicable',
        order: 1,
      },
    ],
  },
}

export const mockScanStatistics: ScanStatistics = {
  total: 158,
  pending: 1,
  running: 5,
  succeeded: 139,
  failed: 11,
  cancelled: 2,
  totalVulns: 89,
  totalSubdomains: 4823,
  totalEndpoints: 12456,
  totalWebsites: 3421,
  totalAssets: 21638,
  retentionPolicy: { minimumRetentionSeconds: 30 * 24 * 60 * 60, automaticCleanupEnabled: true },
}

function buildMockProgressBurst({
  startId,
  taskId,
  step,
  startedAt,
  messages,
}: {
  startId: number
  taskId: number
  step: string
  startedAt: string
  messages: Array<{ offsetSeconds: number; level?: ScanLog['level']; content: string }>
}): ScanLog[] {
  const start = new Date(startedAt).getTime()
  return messages.map((message, index) => ({
    id: startId + index,
    taskId,
    level: message.level ?? 'info',
    content: `[${step}] ${message.content}`,
    createdAt: new Date(start + message.offsetSeconds * 1000).toISOString(),
  }))
}

const mockScanLogsById: Record<number, ScanLog[]> = {
  1: [
    ...buildMockProgressBurst({
      startId: 101,
      taskId: 1001,
      step: 'subdomain_discovery',
      startedAt: '2024-12-28T10:00:00Z',
      messages: [
        { offsetSeconds: 3, content: 'queued passive sources: crtsh, alienvault, threatbook, dnsdumpster' },
        { offsetSeconds: 8, content: 'crtsh returned 413 candidates before de-duplication' },
        { offsetSeconds: 14, content: 'alienvault returned 96 candidates before de-duplication' },
        { offsetSeconds: 21, content: 'start stage 1/3 recon: collect subdomains from passive sources' },
        { offsetSeconds: 29, content: 'complete stage 1/3 recon with 509 candidates' },
        { offsetSeconds: 37, content: 'start stage 2/3 bruteforce: enumerate subdomains from wordlist' },
        { offsetSeconds: 42, content: 'complete stage 2/3 bruteforce with 73 candidates' },
        { offsetSeconds: 49, content: 'merged recon and bruteforce candidates with exact deduplication' },
        { offsetSeconds: 53, content: 'start stage 3/3 resolve: verify discovered subdomains' },
        { offsetSeconds: 58, content: 'PureDNS wildcard filtering retained during DNS validation' },
        { offsetSeconds: 63, content: 'resolver pool warmed with 12 upstream resolvers' },
        { offsetSeconds: 66, content: 'complete stage 3/3 resolve with 156 validated subdomains' },
        { offsetSeconds: 69, content: 'completed result reporting for 156 unique subdomains' },
        { offsetSeconds: 72, content: 'completed with 156 unique subdomains' },
      ],
    }),
    ...buildMockProgressBurst({
      startId: 121,
      taskId: 1002,
      step: 'port_scan',
      startedAt: '2024-12-28T10:01:12Z',
      messages: [
        { offsetSeconds: 4, content: 'loaded 45 active IPs from discovery result set' },
        { offsetSeconds: 9, content: 'probing top 100 TCP ports with concurrency 256' },
        { offsetSeconds: 18, content: 'identified 212 open sockets across 37 hosts' },
        { offsetSeconds: 27, content: 'service fingerprint pass completed for HTTP/TLS candidates' },
        { offsetSeconds: 34, content: 'udp probe skipped by profile because fast mode was enabled' },
        { offsetSeconds: 42, content: 'rate limiter adjusted after 3 transient resets', level: 'warning' },
        { offsetSeconds: 49, content: 'tls handshake classifier labelled 73 services' },
        { offsetSeconds: 55, content: 'banner capture stored 146 service fingerprints' },
        { offsetSeconds: 61, content: 'non-http services queued for evidence-only enrichment' },
        { offsetSeconds: 68, content: 'completed with 89 HTTP services and 24 non-HTTP services' },
      ],
    }),
    ...buildMockProgressBurst({
      startId: 141,
      taskId: 1003,
      step: 'web_crawling',
      startedAt: '2024-12-28T10:02:20Z',
      messages: [
        { offsetSeconds: 2, content: 'seeded 89 websites for crawler queue' },
        { offsetSeconds: 11, content: 'batch 1/6 crawled 380 pages and extracted 219 URLs' },
        { offsetSeconds: 19, content: 'batch 2/6 crawled 424 pages and extracted 231 URLs' },
        { offsetSeconds: 27, content: 'batch 3/6 crawled 396 pages and extracted 211 URLs' },
        { offsetSeconds: 33, content: 'detected duplicate sitemap on docs.acme.com', level: 'warning' },
        { offsetSeconds: 48, content: 'batch 4/6 crawled 411 pages and extracted 227 URLs' },
        { offsetSeconds: 54, content: 'javascript route discovery added 76 SPA endpoints' },
        { offsetSeconds: 61, content: 'batch 5/6 crawled 395 pages and extracted 214 URLs' },
        { offsetSeconds: 68, content: 'form parser identified 44 candidate input surfaces' },
        { offsetSeconds: 73, content: 'api schema hints extracted from 9 swagger documents' },
        { offsetSeconds: 79, content: 'completed with 2,341 pages and 1,284 unique endpoints' },
      ],
    }),
    ...buildMockProgressBurst({
      startId: 161,
      taskId: 1004,
      step: 'directory_scan',
      startedAt: '2024-12-28T10:03:46Z',
      messages: [
        { offsetSeconds: 5, content: 'loaded 89 HTTP services for directory brute force' },
        { offsetSeconds: 16, content: 'wordlist chunk 1/5 produced 41 candidate directories' },
        { offsetSeconds: 28, content: 'wordlist chunk 2/5 produced 63 candidate directories' },
        { offsetSeconds: 35, content: 'wordlist chunk 3/5 produced 58 candidate directories' },
        { offsetSeconds: 41, content: 'filtered 17 soft-404 responses' },
        { offsetSeconds: 46, content: 'wordlist chunk 4/5 produced 44 candidate directories' },
        { offsetSeconds: 51, content: 'wordlist chunk 5/5 produced 45 candidate directories' },
        { offsetSeconds: 54, content: 'merged redirect-equivalent directories across 11 hosts' },
        { offsetSeconds: 56, content: 'completed with 234 directories' },
      ],
    }),
    ...buildMockProgressBurst({
      startId: 181,
      taskId: 1005,
      step: 'screenshot',
      startedAt: '2024-12-28T10:05:00Z',
      messages: [
        { offsetSeconds: 4, content: 'browser pool started with 8 workers' },
        { offsetSeconds: 13, content: 'captured 18 screenshots from primary services' },
        { offsetSeconds: 19, content: 'classified 7 login pages and 5 admin consoles' },
        { offsetSeconds: 28, content: 'captured 37 screenshots; 2 pages timed out', level: 'warning' },
        { offsetSeconds: 36, content: 'retried timed out pages with reduced viewport concurrency' },
        { offsetSeconds: 41, content: 'compressed screenshot artifacts and linked them to website records' },
        { offsetSeconds: 44, content: 'deduplicated 9 visually identical landing pages' },
        { offsetSeconds: 53, content: 'completed with 50 screenshots' },
      ],
    }),
    ...buildMockProgressBurst({
      startId: 201,
      taskId: 1006,
      step: 'nuclei_vulnerability',
      startedAt: '2024-12-28T10:05:53Z',
      messages: [
        { offsetSeconds: 3, content: 'loaded 1,284 endpoints and 89 website roots' },
        { offsetSeconds: 15, content: 'template group exposures matched 4 candidates', level: 'warning' },
        { offsetSeconds: 28, content: 'template group cves matched 3 candidates', level: 'warning' },
        { offsetSeconds: 36, content: 'template group panels matched 6 candidates', level: 'warning' },
        { offsetSeconds: 44, content: 'template group misconfig matched 11 candidates', level: 'warning' },
        { offsetSeconds: 52, content: 'template group takeovers matched 2 candidates', level: 'warning' },
        { offsetSeconds: 63, content: 'evidence collector stored 31 raw matcher payloads' },
        { offsetSeconds: 71, content: 'deduplicated 31 raw matches into 23 findings' },
        { offsetSeconds: 87, content: 'risk scoring completed for critical/high/medium/low findings' },
        { offsetSeconds: 109, content: 'completed with 23 findings across 89 websites' },
      ],
    }),
    {
      id: 220,
      taskId: 1006,
      level: 'info',
      content: '[nuclei_vulnerability] Completed successfully in 00:07:42 under high-volume mock load',
      createdAt: '2024-12-28T10:07:42Z',
    },
  ],
  2: [
    {
      id: 201,
      taskId: 2001,
      level: 'info',
      content: '[subdomain_discovery] Collected 78 subdomains',
      createdAt: '2024-12-27T14:49:58Z',
    },
    {
      id: 202,
      taskId: 2002,
      level: 'info',
      content: '[web_crawling] Crawling seed pages for acme.io',
      createdAt: '2024-12-27T14:50:04Z',
    },
  ],
  5: [
    {
      id: 501,
      taskId: 5001,
      level: 'info',
      content: '[subdomain_discovery] Resolving discovered DNS records',
      createdAt: '2024-12-29T09:00:12Z',
    },
  ],
  4: [
    {
      id: 401,
      taskId: 4001,
      level: 'info',
      content: '[subdomain_discovery] Found 14 candidates for globalfinance test target',
      createdAt: '2024-12-25T16:20:12Z',
    },
    {
      id: 402,
      taskId: 4002,
      level: 'info',
      content: '[web_crawling] Confirmed target landing page and queued risk probes',
      createdAt: '2024-12-25T16:21:02Z',
    },
    {
      id: 403,
      taskId: 4003,
      level: 'info',
      content: '[nuclei_vulnerability] Started timeout-sensitive templates for globalfinance target',
      createdAt: '2024-12-25T16:21:25Z',
    },
    {
      id: 404,
      taskId: 4003,
      level: 'warning',
      content: '[nuclei_vulnerability] retry 1/3 after upstream connection timeout',
      createdAt: '2024-12-25T16:21:29Z',
    },
    {
      id: 405,
      taskId: 4003,
      level: 'warning',
      content: '[nuclei_vulnerability] retry 2/3 after upstream connection timeout',
      createdAt: '2024-12-25T16:21:33Z',
    },
    {
      id: 406,
      taskId: 4003,
      level: 'error',
      content: '[nuclei_vulnerability] failed: connection timeout while probing globalfinance target',
      createdAt: '2024-12-25T16:21:35Z',
    },
  ],
  8: [
    {
      id: 801,
      taskId: 8002,
      level: 'info',
      content: '[web_crawling] Fetching live URLs from mediastream.tv',
      createdAt: '2024-12-29T08:18:00Z',
    },
  ],
  9: [
    {
      id: 901,
      taskId: 9001,
      level: 'info',
      content: '[subdomain_discovery] Engine Container started for test.com',
      createdAt: '2026-07-03T07:44:42Z',
    },
    {
      id: 902,
      taskId: 9001,
      level: 'error',
      content: `[subdomain_discovery] ${mockContainerExitFailureMessage}`,
      createdAt: '2026-07-03T07:44:49Z',
    },
  ],
}

export function getMockScans(params?: {
  page?: number
  pageSize?: number
  pageToken?: string
  target?: number
  status?: ScanStatus
  search?: string
  filter?: string
  orderBy?: string
}): GetScansResponse {
  const page = getMockScanPage(params?.pageToken) || params?.page || 1
  const pageSize = params?.pageSize || 10
  const target = params?.target
  const status = params?.status
  const search = params?.search?.toLowerCase() || ''
  const parsedFilter = parseMockScanFilter(params?.filter)

  let filtered = mockScans

  if (target) {
    filtered = filtered.filter(scan => scan.targetId === target)
  }

  if (status) {
    filtered = filtered.filter(scan => scan.status === status)
  }

  if (parsedFilter.statuses.length > 0) {
    filtered = filtered.filter(scan => parsedFilter.statuses.includes(scan.status))
  }

  const targetNameSearch = parsedFilter.targetName || search
  if (targetNameSearch) {
    filtered = filtered.filter(scan =>
      scan.target?.name?.toLowerCase().includes(targetNameSearch)
    )
  }

  filtered = sortMockScans(filtered, params?.orderBy)

  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const results = filtered.slice(start, start + pageSize)
  const nextPageToken = page * pageSize < total ? String(page + 1) : undefined

  return {
    results,
    total,
    totalSize: total,
    page,
    pageSize,
    totalPages,
    nextPageToken,
  }
}

function getMockScanPage(pageToken?: string) {
  const token = pageToken?.trim()
  if (!token) return undefined
  const page = Number.parseInt(token, 10)
  return Number.isFinite(page) && page > 0 ? page : undefined
}

function parseMockScanFilter(filter?: string) {
  const normalized = filter?.trim() ?? ''
  const statuses = Array.from(normalized.matchAll(/status=="([^"]+)"/g))
    .map(match => match[1])
    .filter((value): value is ScanStatus => isScanStatus(value))
  const targetName = normalized.match(/targetName="([^"]+)"/)?.[1]?.toLowerCase() ?? ''

  if (!statuses.length && !targetName && normalized && !normalized.includes('==') && !normalized.includes('=')) {
    return { statuses, targetName: normalized.toLowerCase() }
  }

  return { statuses, targetName }
}

function isScanStatus(value: string): value is ScanStatus {
  return ['pending', 'running', 'succeeded', 'failed', 'cancelled'].includes(value)
}

function sortMockScans(scans: ScanListRecord[], orderBy?: string) {
  const direction = orderBy?.trim().toLowerCase() === 'createdat' ? 1 : -1
  return [...scans].sort((left, right) => {
    const byCreatedAt = new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime()
    if (byCreatedAt !== 0) return byCreatedAt * direction
    return (left.id - right.id) * direction
  })
}

export function getMockScanById(id: number): ScanRecord | undefined {
  const scan = mockScans.find(scan => scan.id === id)
  if (!scan) {
    return undefined
  }
  const runtime = mockScanRuntimeDetailsById[id]
  if (!runtime?.agentId) {
    return {
      ...scan,
      ...(runtime?.configuration ? { configuration: runtime.configuration } : {}),
      ...(runtime?.runtimeTasks ? { runtimeTasks: runtime.runtimeTasks } : {}),
      assignmentMode: 'automatic',
    }
  }
  return {
    ...scan,
    ...runtime,
    agent: `agents/${runtime.agentId}`,
    agentStatus: id === 4 ? 'offline' : 'online',
    agentHealthState: id === 2 ? 'paused' : 'healthy',
    agentDeleted: false,
    assignmentMode: id % 2 === 0 ? 'pinned' : 'automatic',
  }
}

export function deleteMockScan(id: number): boolean {
  const index = mockScans.findIndex(scan => scan.id === id)
  if (index === -1) {
    return false
  }

  mockScans.splice(index, 1)
  return true
}

export function bulkDeleteMockScans(ids: number[]) {
  let deletedCount = 0

  for (const id of ids) {
    if (deleteMockScan(id)) {
      deletedCount += 1
    }
  }

  return {
    message: "Deleted scans",
    deletedCount,
  }
}

export function stopMockScan(id: number) {
  const scan = mockScans.find((item) => item.id === id)
  const wasActive = scan?.status === "pending" || scan?.status === "running"
  if (scan && wasActive) {
    scan.status = "cancelled"
    scan.stoppedAt = new Date().toISOString()
  }

  return {
    message: "Stopped scan",
    revokedTaskCount: wasActive ? 1 : 0,
  }
}

export function batchStopMockScans(ids: number[]) {
  const stoppedAt = new Date().toISOString()
  let stoppedCount = 0
  let skippedCount = 0
  let revokedTaskCount = 0

  for (const id of ids) {
    const scan = mockScans.find((item) => item.id === id)
    if (!scan) {
      continue
    }
    if (scan.status === "pending" || scan.status === "running") {
      scan.status = "cancelled"
      scan.stoppedAt = stoppedAt
      stoppedCount += 1
      revokedTaskCount += 1
    } else {
      skippedCount += 1
    }
  }

  return { stoppedCount, skippedCount, revokedTaskCount }
}

export function getMockScanLogs(
  scanId: number,
  params?: {
    pageSize?: number
    pageToken?: string
  }
): { results: ScanLog[]; nextPageToken?: string } {
  const source = mockScanLogsById[scanId] || []
  const decodedAfterId = decodeMockScanLogPageToken(params?.pageToken)
  const filtered = typeof decodedAfterId === 'number' ? source.filter((log) => log.id > decodedAfterId) : source
  const limit = params?.pageSize && params.pageSize > 0 ? params.pageSize : filtered.length
  const results = filtered.slice(0, limit)
  const lastLog = results[results.length - 1]

  return {
    results,
    nextPageToken: filtered.length > results.length && lastLog ? encodeMockScanLogPageToken(lastLog.id) : undefined,
  }
}

function encodeMockScanLogPageToken(lastSeenLogId: number): string {
  return `mock-scan-log:${lastSeenLogId}`
}

function decodeMockScanLogPageToken(pageToken: string | undefined): number | undefined {
  if (!pageToken) return undefined
  const match = pageToken.match(/^mock-scan-log:(\d+)$/)
  if (!match) return undefined
  const parsed = Number.parseInt(match[1], 10)
  return Number.isFinite(parsed) ? parsed : undefined
}
