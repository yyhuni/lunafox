import type { GetWordlistTagsResponse, GetWordlistsResponse, Wordlist, WordlistTagSummary } from '@/types/wordlist.types'

const MAX_ONLINE_EDIT_BYTES = 5 * 1024 * 1024

export const mockWordlists: Wordlist[] = [
  {
    id: 1,
    name: 'wordlists/1',
    fileName: 'common-dirs.txt',
    description: 'Common directory wordlist',
    tags: ['Directory Scanning', 'fuzz', 'General', 'Admin Paths', 'Authentication Bypass'],
    fileSize: 45678,
    lineCount: 4567,
    fileHash: 'a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6',
    createdAt: '2024-12-20T10:00:00Z',
    updatedAt: '2024-12-28T10:00:00Z',
  },
  {
    id: 2,
    name: 'wordlists/2',
    fileName: 'subdomains-top1million.txt',
    description: 'Top 1 million subdomain wordlist',
    tags: ['Subdomain Scanning', 'fuzz', 'High Frequency', 'Cloud Assets', 'CDN Enumeration'],
    fileSize: 12345678,
    lineCount: 1000000,
    fileHash: 'b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7',
    createdAt: '2024-12-20T10:01:00Z',
    updatedAt: '2024-12-28T10:01:00Z',
  },
  {
    id: 3,
    name: 'wordlists/3',
    fileName: 'api-endpoints.txt',
    description: 'API endpoint wordlist',
    tags: ['API Paths', 'Directory Scanning', 'General', 'GraphQL', 'Mobile APIs'],
    fileSize: 23456,
    lineCount: 2345,
    fileHash: 'c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8',
    createdAt: '2024-12-20T10:02:00Z',
    updatedAt: '2024-12-28T10:02:00Z',
  },
  {
    id: 4,
    name: 'wordlists/4',
    fileName: 'params.txt',
    description: 'Common parameter name wordlist',
    tags: ['Parameter Brute Force', 'API Paths', 'fuzz', 'Debug Interfaces', 'Weak Parameters'],
    fileSize: 8901,
    lineCount: 890,
    fileHash: 'd4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9',
    createdAt: '2024-12-20T10:03:00Z',
    updatedAt: '2024-12-28T10:03:00Z',
  },
  {
    id: 5,
    name: 'wordlists/5',
    fileName: 'sensitive-files.txt',
    description: 'Sensitive file wordlist',
    tags: ['Sensitive Files', 'Directory Scanning', 'Configuration Exposure', 'Credential Traces'],
    fileSize: 5678,
    lineCount: 567,
    fileHash: 'e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0',
    createdAt: '2024-12-20T10:04:00Z',
    updatedAt: '2024-12-28T10:04:00Z',
  },
  {
    id: 6,
    name: 'wordlists/6',
    fileName: 'raft-large-directories.txt',
    description: 'RAFT large directory wordlist',
    tags: ['Directory Scanning', 'External Import', 'Large Wordlist', 'Low Frequency', 'Historical Archive'],
    fileSize: 987654,
    lineCount: 98765,
    fileHash: 'f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1',
    createdAt: '2024-12-20T10:05:00Z',
    updatedAt: '2024-12-28T10:05:00Z',
  },
  {
    id: 7,
    name: 'wordlists/7',
    fileName: 'asset-subdomain-api-mix.txt',
    description: 'Mixed wordlist for asset enumeration and API path probing',
    tags: ['Subdomain Scanning', 'API Paths', 'fuzz', 'Mixed Use', 'Asset Discovery', 'Service Identification'],
    fileSize: 245678,
    lineCount: 16543,
    fileHash: '07f7h8i9j0k1l2m3n4o5p6q7r8s9t0u2',
    createdAt: '2024-12-20T10:06:00Z',
    updatedAt: '2024-12-28T10:06:00Z',
  },
  {
    id: 8,
    name: 'wordlists/8',
    fileName: 'backup-and-debug-paths.txt',
    description: 'Collection of backup, debug, and test paths',
    tags: ['Sensitive Files', 'Directory Scanning', 'Test Environment', 'Backup Files', 'Admin Console'],
    fileSize: 16432,
    lineCount: 1298,
    fileHash: '18g8h9i0j1k2l3m4n5o6p7q8r9s0t1u3',
    createdAt: '2024-12-20T10:07:00Z',
    updatedAt: '2024-12-28T10:07:00Z',
  },
]

export const mockWordlistContent = `admin
api
backup
config
overview
debug
dev
docs
download
files
images
js
login
logs
manager
private
public
static
test
upload
users
v1
v2
wp-admin
wp-content`

export function getMockWordlists(params?: {
  pageSize?: number
  pageToken?: string | null
  filter?: string | null
  orderBy?: string | null
}): GetWordlistsResponse {
  const page = decodeMockPageToken(params?.pageToken) || 1
  const pageSize = params?.pageSize || 10
  const filtered = applyMockWordlistOrder(applyMockWordlistFilter(mockWordlists, params?.filter || ''), params?.orderBy || '')

  const total = filtered.length
  const start = (page - 1) * pageSize
  const results = filtered.slice(start, start + pageSize)
  const nextPageToken = start + pageSize < total ? encodeMockPageToken(page + 1) : undefined

  return {
    results,
    totalSize: total,
    nextPageToken,
  }
}

export function getMockWordlistById(id: number): Wordlist | undefined {
  return mockWordlists.find(w => w.id === id)
}

export function getMockWordlistContent(): string {
  return mockWordlistContent
}

export function getMockWordlistTags(params?: { pageSize?: number; pageToken?: string | null; filter?: string | null }): GetWordlistTagsResponse {
  const counts = new Map<string, number>()
  for (const wordlist of mockWordlists) {
    for (const tag of wordlist.tags) {
      counts.set(tag, (counts.get(tag) ?? 0) + 1)
    }
  }
  const filter = (params?.filter || '').trim().toLowerCase()
  const summaries: WordlistTagSummary[] = Array.from(counts.entries())
    .filter(([tag]) => !filter || tag.toLowerCase().includes(filter))
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([tag, count]) => ({
      name: `wordlistTags/${encodeURIComponent(tag)}`,
      displayName: tag,
      wordlistCount: count,
    }))
  const page = decodeMockPageToken(params?.pageToken) || 1
  const pageSize = params?.pageSize || 20
  const start = (page - 1) * pageSize
  const results = summaries.slice(start, start + pageSize)
  return {
    results,
    totalSize: summaries.length,
    nextPageToken: start + pageSize < summaries.length ? encodeMockPageToken(page + 1) : undefined,
  }
}

export function createMockWordlist(payload: { file: File; description?: string; tags?: string[] }): Wordlist {
  const now = new Date().toISOString()
  const nextId = Math.max(0, ...mockWordlists.map((wordlist) => wordlist.id)) + 1
  const wordlist: Wordlist = {
    id: nextId,
    name: `wordlists/${nextId}`,
    fileName: payload.file.name,
    description: payload.description || '',
    tags: payload.tags ?? [],
    fileSize: payload.file.size,
    lineCount: 0,
    fileHash: `mock-${nextId}`,
    createdAt: now,
    updatedAt: now,
  }
  mockWordlists.unshift(wordlist)
  return wordlist
}

export function updateMockWordlistMetadata(id: number, payload: { description?: string; tags?: string[] }): Wordlist | undefined {
  const wordlist = getMockWordlistById(id)
  if (!wordlist) return undefined
  if (payload.description !== undefined) wordlist.description = payload.description
  if (payload.tags !== undefined) wordlist.tags = payload.tags
  wordlist.updatedAt = new Date().toISOString()
  return wordlist
}

export function isMockWordlistEditable(id: number) {
  const wordlist = getMockWordlistById(id)
  return !!wordlist && (wordlist.fileSize ?? 0) <= MAX_ONLINE_EDIT_BYTES
}

function applyMockWordlistFilter(wordlists: Wordlist[], filter: string) {
  const tagMatches = Array.from(filter.matchAll(/tags=="([^"]*)"/g))
  const fileNameMatch = filter.match(/fileName="([^"]*)"/)
  const tags = tagMatches.map((match) => match[1]).filter(Boolean)
  const search = fileNameMatch?.[1]?.toLowerCase()
  return wordlists.filter((wordlist) => {
    if (tags.length > 0 && !tags.some((tag) => wordlist.tags.includes(tag))) return false
    if (!search) return true
    return [wordlist.fileName, wordlist.description ?? '', wordlist.tags.join(' ')].some((value) => value.toLowerCase().includes(search))
  })
}

function applyMockWordlistOrder(wordlists: Wordlist[], orderBy: string) {
  const normalized = orderBy.trim()
  const [field, direction = 'asc'] = normalized ? normalized.split(/\s+/) : ['updatedAt', 'desc']
  const desc = direction.toLowerCase() === 'desc'
  const sorted = [...wordlists]

  sorted.sort((left, right) => {
    const primary = compareMockWordlistField(left, right, field)
    const result = primary === 0 ? left.id - right.id : primary
    return desc ? -result : result
  })

  return sorted
}

function compareMockWordlistField(left: Wordlist, right: Wordlist, field: string) {
  switch (field) {
    case 'fileName':
      return left.fileName.localeCompare(right.fileName)
    case 'lineCount':
      return (left.lineCount ?? 0) - (right.lineCount ?? 0)
    case 'fileSize':
      return (left.fileSize ?? 0) - (right.fileSize ?? 0)
    case 'updatedAt':
    default:
      return new Date(left.updatedAt).getTime() - new Date(right.updatedAt).getTime()
  }
}

function encodeMockPageToken(page: number) {
  return `mock-page:${page}`
}

function decodeMockPageToken(token?: string | null) {
  if (!token) return 1
  const match = token.match(/^mock-page:(\d+)$/)
  return match ? Number.parseInt(match[1], 10) : 1
}
