// Wordlist related types

// Wordlist basic info
export interface Wordlist {
  id: number
  // Canonical resource name, for example wordlists/7.
  name: string
  // Immutable uploaded basename; this is a label, not a resource identity.
  fileName: string
  description?: string
  tags: string[]
  // File size (bytes), optional, returned by backend
  fileSize?: number
  // Line count, for estimating duration, optional, returned by backend
  lineCount?: number
  // File SHA-256 hash, for cache validation
  fileHash?: string
  createdAt: string
  updatedAt: string
}

export interface WordlistText {
  name: string
  content: string
  updateTime?: string
}

export interface WordlistTagSummary {
  name: string
  displayName: string
  wordlistCount: number
}

export interface CanonicalListResponse<T> {
  results: T[]
  nextPageToken?: string
  totalSize: number
}

export interface GetWordlistsResponse extends CanonicalListResponse<Wordlist> {
  results: Wordlist[]
}

export interface GetWordlistTagsResponse extends CanonicalListResponse<WordlistTagSummary> {
  results: WordlistTagSummary[]
}

export interface GetWordlistsParams {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}

export interface GetWordlistTagsParams {
  pageSize?: number
  pageToken?: string
  filter?: string
}

export interface UpdateWordlistMetadataPayload {
  id: number
  name: string
  description?: string
  tags?: string[]
  updateMask: string
}
