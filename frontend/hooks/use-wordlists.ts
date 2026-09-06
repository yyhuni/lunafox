"use client"

import { keepPreviousData, useQuery } from "@tanstack/react-query"
import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { getErrorMessage, getErrorResponseData } from "@/lib/response-parser"
import {
  getWordlists,
  getWordlistTags,
  uploadWordlist,
  deleteWordlist,
  updateWordlistMetadata,
  getWordlistContent,
  updateWordlistContent,
} from "@/services/wordlist.service"
import type {
  GetWordlistTagsResponse,
  GetWordlistsParams,
  GetWordlistsResponse,
  UpdateWordlistMetadataPayload,
  Wordlist,
  WordlistText,
} from "@/types/wordlist.types"

// Query Keys
const wordlistKeyBase = createResourceKeys("wordlists", {
  list: (params: GetWordlistsParams) => params,
})

export const wordlistKeys = {
  ...wordlistKeyBase,
  completeCatalog: () => [...wordlistKeyBase.all, "complete-catalog"] as const,
  content: (id: number | null) => [...wordlistKeyBase.all, 'content', id] as const,
  tags: (params: { pageSize?: number; pageToken?: string; search?: string }) => [...wordlistKeyBase.all, 'tags', params] as const,
}

// Get wordlist list
export function useWordlists(params?: GetWordlistsParams) {
  const pageSize = params?.pageSize ?? 10
  const requestParams: GetWordlistsParams = {
    pageSize,
    pageToken: params?.pageToken,
    filter: params?.filter,
    orderBy: params?.orderBy,
  }

  return useQuery<GetWordlistsResponse>({
    queryKey: wordlistKeys.list(requestParams),
    placeholderData: keepPreviousData,
    queryFn: () => getWordlists(requestParams),
  })
}

const COMPLETE_WORDLIST_CATALOG_PAGE_SIZE = 100

export async function fetchCompleteWordlistCatalog(): Promise<Wordlist[]> {
  const byResourceName = new Map<string, Wordlist>()
  const visitedPageTokens = new Set<string>()
  let pageToken: string | undefined

  do {
    const page = await getWordlists({
      pageSize: COMPLETE_WORDLIST_CATALOG_PAGE_SIZE,
      pageToken,
      filter: undefined,
      orderBy: "fileName",
    })
    for (const wordlist of page.results) {
      if (!byResourceName.has(wordlist.name)) {
        byResourceName.set(wordlist.name, wordlist)
      }
    }

    const nextPageToken = page.nextPageToken?.trim() || undefined
    if (nextPageToken && visitedPageTokens.has(nextPageToken)) {
      throw new Error("Wordlist Catalog pagination returned a repeated page token")
    }
    if (nextPageToken) {
      visitedPageTokens.add(nextPageToken)
    }
    pageToken = nextPageToken
  } while (pageToken)

  return Array.from(byResourceName.values())
}

export function useCompleteWordlistCatalog() {
  return useQuery<Wordlist[]>({
    queryKey: wordlistKeys.completeCatalog(),
    queryFn: fetchCompleteWordlistCatalog,
  })
}

export type CompleteWordlistCatalogState = {
  status: "loading" | "incomplete" | "failed" | "complete"
  wordlists: readonly Wordlist[]
  retry: () => void
}

export function useCompleteWordlistCatalogState(): CompleteWordlistCatalogState {
  const query = useCompleteWordlistCatalog()
  const retry = () => {
    void query.refetch()
  }
  if (query.isError) {
    return { status: "failed", wordlists: query.data ?? [], retry }
  }
  if (query.isPending) {
    return { status: "loading", wordlists: [], retry }
  }
  if (query.isFetching) {
    return { status: "incomplete", wordlists: query.data ?? [], retry }
  }
  return { status: "complete", wordlists: query.data ?? [], retry }
}

export function useWordlistTags(params?: { pageSize?: number; pageToken?: string; search?: string }) {
  const requestParams = {
    pageSize: params?.pageSize ?? 50,
    pageToken: params?.pageToken,
    search: params?.search,
  }

  return useQuery<GetWordlistTagsResponse>({
    queryKey: wordlistKeys.tags(requestParams),
    queryFn: () => getWordlistTags({
      pageSize: requestParams.pageSize,
      pageToken: requestParams.pageToken,
      filter: requestParams.search?.trim() || undefined,
    }),
  })
}

// Upload wordlist
export function useUploadWordlist() {
  return useResourceMutation<Wordlist, { description?: string; tags?: string[]; file: File }>({
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

export function useUpdateWordlistMetadata() {
  return useResourceMutation<Wordlist, UpdateWordlistMetadataPayload>({
    mutationFn: (payload) => updateWordlistMetadata(payload),
    loadingToast: {
      key: 'common.actions.saving',
      params: {},
      id: ({ id }) => `update-wordlist-metadata-${id}`,
    },
    invalidate: [{ queryKey: wordlistKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success('toast.wordlist.update.success')
    },
    errorFallbackKey: 'toast.wordlist.update.error',
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
  return useResourceMutation<WordlistText, { id: number; content: string }>({
    mutationFn: ({ id, content }) => updateWordlistContent(id, content),
    loadingToast: {
      key: 'common.actions.saving',
      params: {},
      id: 'update-wordlist-content',
    },
    invalidate: [
      { queryKey: wordlistKeys.all },
      ({ variables }) => ({ queryKey: wordlistKeys.content(variables.id) }),
    ],
    onSuccess: ({ toast }) => {
      toast.success('toast.wordlist.update.success')
    },
    onError: ({ error, toast }) => {
      const message = getErrorMessage(getErrorResponseData(error))
      if (message?.toLowerCase().includes('too large')) {
        toast.error('toast.wordlist.update.oversized')
        return
      }
      toast.error('toast.wordlist.update.error')
    },
    errorFallbackKey: 'toast.wordlist.update.error',
  })
}
