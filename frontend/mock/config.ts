/**
 * Mock data configuration
 * 
 * How to use:
 * 1. Set NEXT_PUBLIC_USE_MOCK=true in .env.local to enable mock data
 * 2. Or directly modify the FORCE_MOCK below to true
 */

// Force the use of mock data (usually kept false, controlled through environment variables)
const FORCE_MOCK = false

// Read mock configuration from environment variables
export const USE_MOCK = FORCE_MOCK || process.env.NEXT_PUBLIC_USE_MOCK === 'true'

// Mock data delay (simulate network request)
export const MOCK_DELAY = 300 // ms

/**
 * Simulate network latency
 */
export function mockDelay(ms: number = MOCK_DELAY): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}
