import type { DiskStats } from '@/types/disk.types'

export const mockDiskStats: DiskStats = {
  totalBytes: 512 * 1024 * 1024 * 1024,
  usedBytes: 286 * 1024 * 1024 * 1024,
  freeBytes: 226 * 1024 * 1024 * 1024,
  usedPercent: 55.86,
}

export function getMockDiskStats(): DiskStats {
  return { ...mockDiskStats }
}
