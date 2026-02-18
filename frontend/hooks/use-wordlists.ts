"use client"

import { useQuery } from "@tanstack/react-query"
import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import {
  getWordlists,
  uploadWordlist,
  deleteWordlist,
  getWordlistContent,
  updateWordlistContent,
} from "@/services/wordlist.service"
import type { GetWordlistsResponse, Wordlist } from "@/types/wordlist.types"

// Query Keys
const wordlistKeyBase = createResourceKeys("wordlists", {
  list: (params: { page: number; pageSize: number }) => params,
})

export const wordlistKeys = {
  ...wordlistKeyBase,
  content: (id: number | null) => [...wordlistKeyBase.all, 'content', id] as const,
}

// Get wordlist list
export function useWordlists(params?: { page?: number; pageSize?: number }) {
  const page = params?.page ?? 1
  const pageSize = params?.pageSize ?? 10

  return useQuery<GetWordlistsResponse>({
    queryKey: wordlistKeys.list({ page, pageSize }),
    queryFn: () => getWordlists(page, pageSize),
  })
}

// Upload wordlist
export function useUploadWordlist() {
  return useResourceMutation<Wordlist, { name: string; description?: string; file: File }>({
    mutationFn: (payload) => uploadWordlist(payload),
    loadingToast: {
      key: 'common.status.uploading',
      params: {},
      id: 'upload-wordlist',
    },
    invalidate: [{ queryKey: wordlistKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success('toast.wordlist.upload.success')
    },
    errorFallbackKey: 'toast.wordlist.upload.error',
  })
}

// Delete wordlist
export function useDeleteWordlist() {
  return useResourceMutation<void, number>({
    mutationFn: (id: number) => deleteWordlist(id),
    loadingToast: {
      key: 'common.status.deleting',
      params: {},
      id: (id) => `delete-wordlist-${id}`,
    },
    invalidate: [{ queryKey: wordlistKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success('toast.wordlist.delete.success')
    },
    errorFallbackKey: 'toast.wordlist.delete.error',
  })
}

// Get wordlist content
export function useWordlistContent(id: number | null) {
  return useQuery<string>({
    queryKey: wordlistKeys.content(id),
    queryFn: () => getWordlistContent(id!),
    enabled: id !== null,
  })
}

// Update wordlist content
export function useUpdateWordlistContent() {
  return useResourceMutation<Wordlist, { id: number; content: string }>({
    mutationFn: ({ id, content }) => updateWordlistContent(id, content),
    loadingToast: {
      key: 'common.actions.saving',
      params: {},
      id: 'update-wordlist-content',
    },
    invalidate: [
      { queryKey: wordlistKeys.all },
      ({ data }) => ({ queryKey: wordlistKeys.content(data.id) }),
    ],
    onSuccess: ({ toast }) => {
      toast.success('toast.wordlist.update.success')
    },
    errorFallbackKey: 'toast.wordlist.update.error',
  })
}
