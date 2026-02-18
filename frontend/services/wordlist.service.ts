import apiClient from "@/lib/api-client"
import type { GetWordlistsResponse, Wordlist } from "@/types/wordlist.types"
import {
  USE_MOCK,
  mockDelay,
  getMockWordlists,
  getMockWordlistContent,
} from "@/mock"

// Dictionary (Wordlist) API service

// Get wordlist list
export async function getWordlists(page = 1, pageSize = 10): Promise<GetWordlistsResponse> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockWordlists({ page, pageSize })
  }
  const response = await apiClient.get<GetWordlistsResponse>("/wordlists/", {
    params: {
      page,
      pageSize,
    },
  })
  return response.data
}

// Upload wordlist file
export async function uploadWordlist(payload: {
  name: string
  description?: string
  file: File
}): Promise<Wordlist> {
  const formData = new FormData()
  formData.append("name", payload.name)
  if (payload.description) {
    formData.append("description", payload.description)
  }
  formData.append("file", payload.file)

  const response = await apiClient.post<Wordlist>("/wordlists/", formData, {
    headers: {
      "Content-Type": "multipart/form-data",
    },
  })

  return response.data
}

// Delete wordlist
export async function deleteWordlist(id: number): Promise<void> {
  await apiClient.delete(`/wordlists/${id}/`)
}

// Get wordlist content
export async function getWordlistContent(id: number): Promise<string> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockWordlistContent()
  }
  const response = await apiClient.get<{ content: string }>(`/wordlists/${id}/content/`)
  return response.data.content
}

// Update wordlist content
export async function updateWordlistContent(id: number, content: string): Promise<Wordlist> {
  const response = await apiClient.put<Wordlist>(`/wordlists/${id}/content/`, { content })
  return response.data
}
