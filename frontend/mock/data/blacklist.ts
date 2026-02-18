/**
 * Blacklist Mock Data
 * 
 * Blacklist rule mock data
 * - Global blacklist: applies to all Targets
 * - Target blacklist: only applies to specific Targets
 */

export interface BlacklistResponse {
  patterns: string[]
}

export interface UpdateBlacklistRequest {
  patterns: string[]
}

// Global blacklist mock data
let mockGlobalBlacklistPatterns: string[] = [
  '*.gov',
  '*.edu',
  '*.mil',
  '10.0.0.0/8',
  '172.16.0.0/12',
  '192.168.0.0/16',
]

// Target blacklist mock data (stored by targetId)
const mockTargetBlacklistPatterns: Record<number, string[]> = {
  1: ['*.internal.example.com', '192.168.1.0/24'],
  2: ['cdn.example.com', '*.cdn.*'],
}

/**
 * Get global blacklist
 */
export function getMockGlobalBlacklist(): BlacklistResponse {
  return {
    patterns: [...mockGlobalBlacklistPatterns],
  }
}

/**
 * Update global blacklist (full replacement)
 */
export function updateMockGlobalBlacklist(data: UpdateBlacklistRequest): BlacklistResponse {
  mockGlobalBlacklistPatterns = [...data.patterns]
  return {
    patterns: mockGlobalBlacklistPatterns,
  }
}

/**
 * Get Target blacklist
 */
export function getMockTargetBlacklist(targetId: number): BlacklistResponse {
  return {
    patterns: mockTargetBlacklistPatterns[targetId] ? [...mockTargetBlacklistPatterns[targetId]] : [],
  }
}

/**
 * Update Target blacklist (full replacement)
 */
export function updateMockTargetBlacklist(targetId: number, data: UpdateBlacklistRequest): BlacklistResponse {
  mockTargetBlacklistPatterns[targetId] = [...data.patterns]
  return {
    patterns: mockTargetBlacklistPatterns[targetId],
  }
}
