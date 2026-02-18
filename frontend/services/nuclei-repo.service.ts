/**
 * Nuclei template repository API
 */

import { api } from "@/lib/api-client"

const BASE_URL = "/nuclei/repos/"

export interface NucleiRepoResponse {
  id: number
  name: string
  repoUrl: string
  localPath: string
  commitHash: string | null
  lastSyncedAt: string | null
  createdAt: string
  updatedAt: string
}

export interface CreateRepoPayload {
  name: string
  repoUrl: string
}

export interface UpdateRepoPayload {
  repoUrl?: string
}

export interface TemplateTreeResponse {
  roots: Array<{
    type: "folder" | "file"
    name: string
    path: string
    children?: Array<{
      type: "folder" | "file"
      name: string
      path: string
      children?: unknown[]
    }>
  }>
}

export interface TemplateContentResponse {
  path: string
  name: string
  content: string
}

/** Paginated response format */
interface PaginatedResponse<T> {
  results: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

export const nucleiRepoApi = {
  /** Get repository list */
  listRepos: async (): Promise<NucleiRepoResponse[]> => {
    // Repositories are usually not many, get all
    const response = await api.get<PaginatedResponse<NucleiRepoResponse>>(BASE_URL, {
      params: { pageSize: 1000 }
    })
    // Backend returns paginated format, take results array
    return response.data.results
  },

  /** Get single repository */
  getRepo: async (repoId: number): Promise<NucleiRepoResponse> => {
    const response = await api.get<NucleiRepoResponse>(`${BASE_URL}${repoId}/`)
    return response.data
  },

  /** Create repository */
  createRepo: async (payload: CreateRepoPayload): Promise<NucleiRepoResponse> => {
    const response = await api.post<NucleiRepoResponse>(BASE_URL, payload)
    return response.data
  },

  /** Update repository (partial update) */
  updateRepo: async (repoId: number, payload: UpdateRepoPayload): Promise<NucleiRepoResponse> => {
    const response = await api.patch<NucleiRepoResponse>(`${BASE_URL}${repoId}/`, payload)
    return response.data
  },

  /** Delete repository */
  deleteRepo: async (repoId: number): Promise<void> => {
    await api.delete(`${BASE_URL}${repoId}/`)
  },

  /** Refresh repository (Git clone/pull) */
  refreshRepo: async (repoId: number): Promise<{ message: string; result: unknown }> => {
    const response = await api.post<{ message: string; result: unknown }>(
      `${BASE_URL}${repoId}/refresh/`
    )
    return response.data
  },

  /** Get template directory tree */
  getTemplateTree: async (repoId: number): Promise<TemplateTreeResponse> => {
    const response = await api.get<TemplateTreeResponse>(
      `${BASE_URL}${repoId}/templates/tree/`
    )
    return response.data
  },

  /** Get template content */
  getTemplateContent: async (repoId: number, path: string): Promise<TemplateContentResponse> => {
    const response = await api.get<TemplateContentResponse>(
      `${BASE_URL}${repoId}/templates/content/`,
      { params: { path } }
    )
    return response.data
  },
}
