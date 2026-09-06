/**
 * Unified export of Mock data
 * 
 * How to use:
 * import { USE_MOCK, mockData } from '@/mock'
 * 
 * if (USE_MOCK) {
 *   return mockData.overview.assetStatistics
 * }
 */

export { USE_MOCK, MOCK_DELAY, MOCK_SCENARIO, mockDelay, resolveMockScenarioFromEnv } from './config'
export {
  mockScenarioIds,
  getMockScenario,
  setMockScenario,
  resetMockScenario,
  resolveScenario,
} from "./scenarios"
export {
  buildOverviewStats,
  buildAssetStatistics,
  buildStatisticsHistory,
  buildOrganizations,
  buildTargets,
  buildTargetDetail,
  buildScans,
  buildScanById,
  buildScanStatistics,
  buildVulnerabilities,
  buildVulnerabilityStats,
  buildVulnerabilityById,
} from "./builders/core"
export { buildTools } from "./builders/tools"
export {
  buildEndpoints,
  buildTargetEndpoints,
  buildEndpointResults,
  buildEndpointFilterOptions,
} from "./builders/endpoints"
export {
  buildScreenshotsByTarget,
  buildScreenshotsByScan,
  buildTargetScreenshotFilterOptions,
  buildScanScreenshotFilterOptions,
} from "./builders/screenshots"
export {
  buildAllSubdomains,
  buildOrganizationSubdomains,
  buildSubdomainResults,
} from "./builders/subdomains"
export { buildFingerprintList } from "./builders/fingerprints"

// Overview
export {
  mockOverviewStats,
  mockAssetStatistics,
  mockStatisticsHistory7Days,
  mockStatisticsHistory30Days,
  getMockStatisticsHistory,
} from './data/overview'

// Database Health
export {
  mockDatabaseHealth,
} from './data/database-health'

// Organizations
export {
  bulkDeleteMockOrganizations,
  deleteMockOrganization,
  mockOrganizations,
  getMockOrganizations,
} from './data/organizations'

// Targets
export {
  bulkDeleteMockTargets,
  deleteMockTarget,
  mockTargets,
  mockTargetDetails,
  getMockTargets,
  getMockTargetById,
} from './data/targets'

// Scans
export {
  bulkDeleteMockScans,
  deleteMockScan,
  mockScans,
  mockScanStatistics,
  getMockScans,
  getMockScanById,
  getMockScanLogs,
  stopMockScan,
  batchStopMockScans,
} from './data/scans'

// Vulnerabilities
export {
  batchUpdateMockVulnerabilityReviewStates,
  bulkDeleteMockVulnerabilities,
  mockVulnerabilities,
  getMockVulnerabilities,
  getMockVulnerabilityStats,
  getMockVulnerabilityById,
  updateMockVulnerabilityReviewState,
} from './data/vulnerabilities'

// Endpoints
export {
  batchDeleteMockEndpoints,
  deleteMockEndpoint,
  mockEndpoints,
  getMockEndpoints,
  getMockEndpointById,
} from './data/endpoints'

// Websites
export {
  bulkDeleteMockWebsites,
  deleteMockWebsite,
  mockWebsites,
  getMockWebsites,
  getMockWebsiteFilterOptions,
  getMockWebsiteById,
} from './data/websites'

// Subdomains
export {
  bulkDeleteMockSubdomains,
  bulkRemoveMockOrganizationSubdomains,
  deleteMockSubdomain,
  mockSubdomains,
  getMockSubdomains,
  getMockSubdomainById,
} from './data/subdomains'

// Auth
export {
  mockUser,
  mockMeResponse,
  mockLoginResponse,
  mockLogoutResponse,
} from './data/auth'

// Workflows
export {
  mockScanWorkflows,
  getMockScanWorkflows,
  getMockScanWorkflowByName,
  getMockScanWorkflowProfile,
  getMockCanonicalWorkflowConfiguration,
} from './data/scan-workflows'

// Distributed
export {
  mockAgents,
  deleteMockAgent,
  getMockAgentClusterSummary,
  getMockAgentFilterOptions,
  getMockAgentLocationMap,
  getMockAgents,
  getMockAgentById,
  getMockRegistrationToken,
  getMockRegistrationTokenById,
} from './data/agents'

// Notifications
export {
  mockNotifications,
  getMockNotification,
  getMockNotifications,
  getMockUnreadCount,
  markAllMockNotificationsRead,
  markMockNotificationRead,
  resetMockNotifications,
} from './data/notifications'

// Scheduled Scans
export {
  mockScheduledScans,
  deleteMockScheduledScan,
  getMockScheduledScanOverviewSummary,
  getMockScheduledScans,
  getMockScheduledScanById,
  resetMockScheduledScans,
} from './data/scheduled-scans'

// Directories
export {
  mockDirectories,
  bulkDeleteMockDirectories,
  deleteMockDirectory,
  getMockDirectories,
  getMockDirectoryById,
  getMockDirectoryFilterOptions,
} from './data/directories'

// Fingerprints
export {
  mockFingerprintName,
  mockFingerPrintHubFingerprints,
  getMockFingerprintList,
  getMockFingerprintFilterOptions,
  parseMockFingerprintImport,
  importMockFingerprints,
  batchDeleteMockFingerprints,
  clearMockFingerprints,
  exportMockFingerprints,
  getMockFingerprintByName,
  getMockFingerprintStats,
} from './data/fingerprints'

// IP Addresses
export {
  bulkDeleteMockIPAddresses,
  mockIPAddresses,
  mockIPAddressMatchesFilter,
  getMockIPAddresses,
  getMockPortOptions,
  getMockIPAddressByIP,
} from './data/ip-addresses'

// Search
export {
  getMockSearchResults,
} from './data/search'

// Tools
export {
  mockTools,
  getMockTools,
  getMockToolById,
  createMockTool,
  updateMockTool,
  deleteMockTool,
} from './data/tools'

// Commands
export {
  mockCommands,
  getMockCommands,
  getMockCommandById,
  getMockCommandDeleteCount,
  createMockCommand,
  updateMockCommand,
  deleteMockCommand,
  batchDeleteMockCommands,
} from './data/commands'

// Wordlists
export {
  mockWordlists,
  mockWordlistContent,
  getMockWordlists,
  getMockWordlistById,
  getMockWordlistContent,
  getMockWordlistTags,
  isMockWordlistEditable,
  createMockWordlist,
  updateMockWordlistMetadata,
} from './data/wordlists'

// Nuclei POC source sync
export {
  createMockNucleiPocSync,
  getMockNucleiPoc,
  getMockNucleiPocFilterOptions,
  getMockNucleiPocs,
  getMockNucleiPocSource,
  getMockNucleiPocSyncTask,
  resetMockNucleiPocs,
  setMockNucleiPocActivation,
  updateMockNucleiPocEnabled,
} from "./data/nuclei-pocs"

// System Logs
export {
  mockSystemLogItems,
  getMockSystemLogs,
} from './data/system-logs'

// Notification Settings
export {
  getMockNotificationDestination,
  getMockNotificationDestinations,
  getMockNotificationTestDeliveryUnavailableResult,
  getMockNotificationLocale,
  mockNotificationDestinations,
  resetMockNotificationSettings,
  updateMockNotificationDestination,
  updateMockNotificationLocale,
} from './data/notification-settings'

// Disk
export {
  mockDiskStats,
  getMockDiskStats,
} from './data/disk'

// Login visual settings
export {
  getMockLoginVisualSettings,
  getMockLoginVisualDiscoverability,
  getMockPublicLoginVisual,
  publishMockLoginVisual,
  restoreMockLoginVisual,
  unlockMockLoginVisualDiscoverability,
  uploadMockLoginVisual,
} from './data/login-visual'

// Version
export {
  mockVersionInfo,
  mockUpdateCheckResult,
  getMockVersionInfo,
  getMockUpdateCheckResult,
} from './data/version'

// Screenshots
export {
  mockScreenshots,
  mockScreenshotSnapshots,
  getMockScreenshotsByTarget,
  getMockScreenshotsByScan,
  getMockScreenshotImageSvg,
  getMockScreenshotImageUrl,
  bulkDeleteMockScreenshots,
} from './data/screenshots'

// API Key Settings
export {
  initialMockApiKeySettings,
  getMockApiKeySettings,
  updateMockApiKeySettings,
} from './data/api-key-settings'

// MCP key lifecycle
export {
  generateMockMcpKey,
  getMockMcpKeyStatus,
  resetMockMcpKey,
} from "./data/mcp-key"

// Blacklist policies
export {
  getMockGlobalBlacklistPolicy,
  getMockTargetBlacklistPolicy,
  patchMockGlobalBlacklistPolicy,
  patchMockTargetBlacklistPolicy,
  resetMockBlacklistPolicies,
} from './data/blacklist-policy'
