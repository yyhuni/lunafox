import type { Directory, DirectoryFilterOptionField, DirectoryFilterOptionsResponse, DirectoryListResponse } from '@/types/directory.types'

export const mockDirectories: Directory[] = [
  {
    id: 1,
    url: 'https://acme.com/admin',
    status: 200,
    contentLength: '12345',
    contentType: 'text/html',
    duration: '234000000',
    createdAt: '2024-12-28T10:00:00Z',
  },
  {
    id: 2,
    url: 'https://acme.com/api',
    status: 301,
    contentLength: '0',
    contentType: 'text/html',
    duration: '56000000',
    createdAt: '2024-12-28T10:01:00Z',
  },
  {
    id: 3,
    url: 'https://acme.com/login',
    status: 200,
    contentLength: '8765',
    contentType: 'text/html',
    duration: '189000000',
    createdAt: '2024-12-28T10:02:00Z',
  },
  {
    id: 4,
    url: 'https://acme.com/overview',
    status: 302,
    contentLength: '0',
    contentType: 'text/html',
    duration: '78000000',
    createdAt: '2024-12-28T10:03:00Z',
  },
  {
    id: 5,
    url: 'https://acme.com/static/js/app.js',
    status: 200,
    contentLength: '456789',
    contentType: 'application/javascript',
    duration: '345000000',
    createdAt: '2024-12-28T10:04:00Z',
  },
  {
    id: 6,
    url: 'https://acme.com/.git/config',
    status: 200,
    contentLength: '234',
    contentType: 'text/plain',
    duration: '23000000',
    createdAt: '2024-12-28T10:05:00Z',
  },
  {
    id: 7,
    url: 'https://acme.com/backup.zip',
    status: 200,
    contentLength: '9223372036854775807',
    contentType: 'application/zip',
    duration: '9223372036854775807',
    createdAt: '2024-12-28T10:06:00Z',
  },
  {
    id: 8,
    url: 'https://acme.com/robots.txt',
    status: 200,
    contentLength: '567',
    contentType: 'text/plain',
    duration: '34000000',
    createdAt: '2024-12-28T10:07:00Z',
  },
  {
    id: 9,
    url: 'https://api.acme.com/v1/health',
    status: 200,
    contentLength: '45',
    contentType: 'application/json',
    duration: '12000000',
    createdAt: '2024-12-28T10:08:00Z',
  },
  {
    id: 10,
    url: 'https://api.acme.com/swagger-ui.html',
    status: 200,
    contentLength: '23456',
    contentType: 'text/html',
    duration: '267000000',
    createdAt: '2024-12-28T10:09:00Z',
  },
  {
    id: 11,
    url: 'https://techstart.io/wp-admin',
    status: 302,
    contentLength: '0',
    contentType: 'text/html',
    duration: '89000000',
    createdAt: '2024-12-26T08:45:00Z',
  },
  {
    id: 12,
    url: 'https://techstart.io/wp-login.php',
    status: 200,
    contentLength: null,
    contentType: 'text/html',
    duration: null,
    createdAt: '2024-12-26T08:46:00Z',
  },
]

export function getMockDirectories(params?: {
  page?: number
  pageSize?: number
  filter?: string
  targetId?: number
  scanId?: number
}): DirectoryListResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const filter = params?.filter?.match(/url(?:==|=)"((?:\\.|[^"\\])*)"/)?.[1]
    ?.replace(/\\"/g, '"')
    .replace(/\\\\/g, "\\")

  let filtered = mockDirectories

  if (filter !== undefined) {
    filtered = filtered.filter((directory) => directory.url === filter)
  }

  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const results = filtered.slice(start, start + pageSize)

  return {
    results,
    total,
    page,
    pageSize,
    totalPages,
  }
}

export function getMockDirectoryFilterOptions(field: DirectoryFilterOptionField): DirectoryFilterOptionsResponse {
  const counts = new Map<string, number>()
  for (const directory of mockDirectories) {
    const rawValue = field === "status" ? directory.status : directory.contentType
    if (rawValue === null || rawValue === undefined) continue
    const value = String(rawValue).trim()
    if (!value) continue
    counts.set(value, (counts.get(value) ?? 0) + 1)
  }

  return {
    results: Array.from(counts.entries())
      .sort(([left], [right]) => left.localeCompare(right, undefined, { numeric: true }))
      .map(([value, count]) => ({ value, label: value, count })),
  }
}

export function getMockDirectoryById(id: number): Directory | undefined {
  return mockDirectories.find(d => d.id === id)
}

export function deleteMockDirectory(id: number): Directory | undefined {
  const index = mockDirectories.findIndex(directory => directory.id === id)
  if (index === -1) {
    return undefined
  }

  const [deleted] = mockDirectories.splice(index, 1)
  return deleted
}

export function bulkDeleteMockDirectories(ids: number[]) {
  let deletedCount = 0

  for (const id of ids) {
    if (deleteMockDirectory(id)) {
      deletedCount += 1
    }
  }

  return {
    message: "Deleted directories",
    deletedCount,
    requestedIds: ids,
    cascadeDeleted: {},
  }
}
