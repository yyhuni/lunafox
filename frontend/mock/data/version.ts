import type { UpdateCheckResult, VersionInfo } from '@/types/version.types'

export const mockVersionInfo: VersionInfo = {
  version: 'mock-2026.04.26',
  githubRepo: 'https://github.com/lunafox/lunafox',
}

export const mockUpdateCheckResult: UpdateCheckResult = {
  currentVersion: mockVersionInfo.version,
  latestVersion: mockVersionInfo.version,
  hasUpdate: false,
  releaseUrl: 'https://github.com/lunafox/lunafox/releases',
  releaseNotes: null,
  publishedAt: null,
}

export function getMockVersionInfo(): VersionInfo {
  return { ...mockVersionInfo }
}

export function getMockUpdateCheckResult(): UpdateCheckResult {
  return { ...mockUpdateCheckResult }
}
