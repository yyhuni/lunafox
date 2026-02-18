import type { QueryClient } from '@tanstack/react-query'
import { describe, expect, it, vi } from 'vitest'
import {
  getBulkDeleteSuccessCount,
  getQuickScanSuccessCount,
  getStopScanSuccessCount,
  handleScanMutationSuccess,
  invalidateScanQueries,
} from '@/hooks/_shared/scan-mutation-helpers'

describe('scan-mutation-helpers', () => {
  it('quick scan success count prefers scans length', () => {
    expect(getQuickScanSuccessCount({ scans: [{ id: 1 }, { id: 2 }] })).toBe(2)
    expect(getQuickScanSuccessCount({ targetStats: { created: 3 } })).toBe(3)
    expect(getQuickScanSuccessCount({ count: 4 })).toBe(4)
    expect(getQuickScanSuccessCount({})).toBe(0)
  })

  it('bulk delete success count prefers deletedCount', () => {
    expect(getBulkDeleteSuccessCount({ deletedCount: 5 }, 2)).toBe(5)
    expect(getBulkDeleteSuccessCount({}, 2)).toBe(2)
  })

  it('stop scan success count falls back to 1', () => {
    expect(getStopScanSuccessCount({ revokedTaskCount: 3 })).toBe(3)
    expect(getStopScanSuccessCount({})).toBe(1)
  })

  it('invalidates scan queries with consistent keys', async () => {
    const invalidateQueries = vi.fn().mockResolvedValue(undefined)
    const queryClient = { invalidateQueries } as unknown as QueryClient
    const keys = {
      all: ['scans'] as const,
      statistics: ['scans', 'statistics'] as const,
    }

    await invalidateScanQueries(queryClient, keys)

    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: keys.all })
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: keys.statistics })
  })

  it('handleScanMutationSuccess skips when parse returns null', async () => {
    const invalidateQueries = vi.fn().mockResolvedValue(undefined)
    const queryClient = { invalidateQueries } as unknown as QueryClient
    const onValidData = vi.fn()
    const parse = vi.fn().mockReturnValue(null)
    const keys = {
      all: ['scans'] as const,
      statistics: ['scans', 'statistics'] as const,
    }

    const result = await handleScanMutationSuccess({
      response: { error: { code: 'FAIL' } },
      parse,
      onValidData,
      queryClient,
      invalidateKeys: keys,
    })

    expect(result).toBe(false)
    expect(onValidData).not.toHaveBeenCalled()
    expect(invalidateQueries).not.toHaveBeenCalled()
  })

  it('handleScanMutationSuccess invokes callback and invalidation', async () => {
    const invalidateQueries = vi.fn().mockResolvedValue(undefined)
    const queryClient = { invalidateQueries } as unknown as QueryClient
    const onValidData = vi.fn()
    const parse = vi.fn().mockReturnValue({ ok: true })
    const keys = {
      all: ['scans'] as const,
      statistics: ['scans', 'statistics'] as const,
    }

    const result = await handleScanMutationSuccess({
      response: { ok: true },
      parse,
      onValidData,
      queryClient,
      invalidateKeys: keys,
    })

    expect(result).toBe(true)
    expect(onValidData).toHaveBeenCalledWith({ ok: true })
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: keys.all })
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: keys.statistics })
  })
})
