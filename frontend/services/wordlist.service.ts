import apiClient from "@/lib/api-client"
import type {
  GetWordlistTagsParams,
  GetWordlistTagsResponse,
  GetWordlistsParams,
  GetWordlistsResponse,
  UpdateWordlistMetadataPayload,
  Wordlist,
  WordlistText,
} from "@/types/wordlist.types"

// Dictionary (Wordlist) API service

export async function getWordlists(params: GetWordlistsParams = {}): Promise<GetWordlistsResponse> {
  const response = await apiClient.get<GetWordlistsResponse>("/wordlists", { params })
  return response.data
}

export async function getWordlistTags(params: GetWordlistTagsParams = {}): Promise<GetWordlistTagsResponse> {
  const response = await apiClient.get<GetWordlistTagsResponse>("/wordlistTags", { params })
  return response.data
}

// Upload wordlist file
export async function uploadWordlist(payload: {
  description?: string
  tags?: string[]
  file: File
}): Promise<Wordlist> {
  const formData = new FormData()
  if (payload.description) {
    formData.append("description", payload.description)
  }
  if (payload.tags?.length) {
    formData.append("tags", payload.tags.join(","))
  }
  formData.append("file", payload.file)

  const response = await apiClient.post<Wordlist>("/wordlists", formData, {
    headers: {
      "Content-Type": "multipart/form-data",
    },
  })

  return response.data
}

// Delete wordlist
export async function deleteWordlist(id: number): Promise<void> {
  await apiClient.delete(`/wordlists/${id}`)
}

export async function updateWordlistMetadata(payload: UpdateWordlistMetadataPayload): Promise<Wordlist> {
  const { id, ...body } = payload
  const response = await apiClient.patch<Wordlist>(`/wordlists/${id}`, body)
  return response.data
}

// Get wordlist content
export async function getWordlistContent(id: number): Promise<string> {
  const response = await apiClient.get<WordlistText>(`/wordlists/${id}/text`)
  return response.data.content
}

// Update wordlist content
export async function updateWordlistContent(id: number, content: string): Promise<WordlistText> {
  const response = await apiClient.patch<WordlistText>(`/wordlists/${id}/text`, {
    name: `wordlists/${id}/text`,
    content,
  }, {
    params: { updateMask: "content" },
  })
  return response.data
}
