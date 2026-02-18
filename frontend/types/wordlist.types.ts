// Wordlist related types

import type { PaginationInfo } from "@/types/common.types"

// Wordlist basic info
export interface Wordlist {
  id: number
  name: string
  description?: string
  // File size (bytes), optional, returned by backend
  fileSize?: number
  // Line count, for estimating duration, optional, returned by backend
  lineCount?: number
  // File SHA-256 hash, for cache validation
  fileHash?: string
  createdAt: string
  updatedAt: string
}

// Get wordlists list response (follows unified pagination structure)
export interface GetWordlistsResponse extends PaginationInfo {
  results: Wordlist[]
}
