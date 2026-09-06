import type {
  BulkDeleteScreenshotResponse,
  Screenshot,
  ScreenshotListQueryParams,
  ScreenshotListResponse,
  ScreenshotSnapshot,
} from '@/types/screenshot.types'

const MOCK_TIMESTAMP = '2026-04-26T00:00:00Z'

export type MockScreenshot = Screenshot & {
  targetId: number
}

export const mockScreenshots: MockScreenshot[] = [
  {
    id: 1,
    targetId: 1,
    url: 'https://acme.com',
    statusCode: 200,
    createdAt: MOCK_TIMESTAMP,
    updatedAt: MOCK_TIMESTAMP,
  },
  {
    id: 2,
    targetId: 1,
    url: 'https://api.acme.com',
    statusCode: 200,
    createdAt: MOCK_TIMESTAMP,
    updatedAt: MOCK_TIMESTAMP,
  },
  {
    id: 3,
    targetId: 1,
    url: 'https://admin.acme.com',
    statusCode: 403,
    createdAt: MOCK_TIMESTAMP,
    updatedAt: MOCK_TIMESTAMP,
  },
]

export const mockScreenshotSnapshots: ScreenshotSnapshot[] = mockScreenshots.map(
  ({ id, url, statusCode, createdAt }) => ({
    id,
    url,
    statusCode,
    createdAt,
  })
)

function paginate<T extends { url: string }>(
  source: T[],
  params?: ScreenshotListQueryParams
): ScreenshotListResponse<T> {
  const page = 1
  const pageSize = params?.pageSize || 12
  const filter = params?.filter?.match(/url(?:==|=)"((?:\\.|[^"\\])*)"/)?.[1]
    ?.replace(/\\"/g, '"')
    .replace(/\\\\/g, "\\")
  const filtered = filter === undefined
    ? source
    : source.filter((item) => item.url === filter)
  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize

  return {
    results: filtered.slice(start, start + pageSize).map((item) => ({ ...item })),
    total,
    page,
    pageSize,
    totalPages,
  }
}

export function getMockScreenshotsByTarget(
  targetId: number,
  params?: ScreenshotListQueryParams
): ScreenshotListResponse<Screenshot> {
  return paginate(
    mockScreenshots
      .filter((screenshot) => screenshot.targetId === targetId)
      .map(({ targetId: _, ...screenshot }) => screenshot),
    params,
  )
}

export function getMockScreenshotsByScan(
  _scanId: number,
  params?: ScreenshotListQueryParams
): ScreenshotListResponse<ScreenshotSnapshot> {
  return paginate(mockScreenshotSnapshots, params)
}

export function getMockScreenshotImageSvg(id: number, _scopeId?: number): string {
  void _scopeId

  const label = `Lunafox mock screenshot ${id}`
  return `<svg xmlns="http://www.w3.org/2000/svg" width="960" height="540" viewBox="0 0 960 540"><rect width="960" height="540" fill="Canvas"/><rect x="80" y="72" width="800" height="396" rx="8" fill="Canvas"/><rect x="112" y="112" width="736" height="40" rx="6" fill="ButtonFace"/><rect x="112" y="184" width="320" height="28" rx="4" fill="GrayText"/><rect x="112" y="236" width="736" height="16" rx="4" fill="ButtonFace"/><rect x="112" y="272" width="680" height="16" rx="4" fill="ButtonFace"/><rect x="112" y="332" width="180" height="48" rx="6" fill="Highlight"/><text x="480" y="438" text-anchor="middle" font-family="Arial, sans-serif" font-size="22" fill="CanvasText">${label}</text></svg>`
}

export function getMockScreenshotImageUrl(id: number, scopeId?: number): string {
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(getMockScreenshotImageSvg(id, scopeId))}`
}

export function bulkDeleteMockScreenshots(ids: number[]): BulkDeleteScreenshotResponse {
  const idsToDelete = new Set(ids)
  const before = mockScreenshots.length

  for (let index = mockScreenshots.length - 1; index >= 0; index -= 1) {
    if (idsToDelete.has(mockScreenshots[index].id)) {
      mockScreenshots.splice(index, 1)
    }
  }

  for (let index = mockScreenshotSnapshots.length - 1; index >= 0; index -= 1) {
    if (idsToDelete.has(mockScreenshotSnapshots[index].id)) {
      mockScreenshotSnapshots.splice(index, 1)
    }
  }

  return {
    deletedCount: before - mockScreenshots.length,
  }
}
