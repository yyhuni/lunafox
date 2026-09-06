"use client"

import * as React from "react"
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query"

import { agentKeys } from "@/hooks/use-agents"
import { agentService } from "@/services/agent.service"
import type { Agent, AgentListQueryParams } from "@/types/agent.types"

export const SCAN_AGENT_PICKER_PAGE_SIZE = 50
export const SCAN_AGENT_PICKER_SEARCH_DEBOUNCE_MS = 250
export const SCAN_AGENT_PICKER_REFRESH_MS = 15_000

const SCAN_AGENT_PICKER_ORDER_BY = "createdAt desc"

export function normalizeScanAgentPickerSearch(search: string) {
  return search.trim().replace(/\s+/g, " ")
}

function quoteFilterValue(value: string) {
  return value.replace(/\\/g, "\\\\").replace(/"/g, '\\"')
}

export function buildScanAgentPickerFilter(normalizedSearch: string) {
  if (!normalizedSearch) return undefined
  const value = quoteFilterValue(normalizedSearch)
  return `(displayName="${value}" || observedHostname="${value}" || connectionIp="${value}")`
}

function flattenCanonicalAgents(pages: Array<{ results: Agent[] }> | undefined) {
  const agentsByName = new Map<string, Agent>()
  for (const page of pages ?? []) {
    for (const agent of page.results) {
      if (!agent.resourceName) {
        throw new Error("Scan Agent picker received an Agent without a canonical resource name")
      }
      if (!agentsByName.has(agent.resourceName)) {
        agentsByName.set(agent.resourceName, agent)
      }
    }
  }
  return Array.from(agentsByName.values())
}

type UseScanAgentPickerOptions = {
  open: boolean
  search: string
}

export function useScanAgentPicker({ open, search }: UseScanAgentPickerOptions) {
  const queryClient = useQueryClient()
  const [normalizedSearch, setNormalizedSearch] = React.useState(() =>
    normalizeScanAgentPickerSearch(search),
  )
  const [failedNextPageGeneration, setFailedNextPageGeneration] = React.useState<string | null>(null)
  const nextPageRequestRef = React.useRef(false)

  React.useEffect(() => {
    const nextSearch = normalizeScanAgentPickerSearch(search)
    if (nextSearch === normalizedSearch) return

    if (!open) {
      setNormalizedSearch(nextSearch)
      return
    }

    const timeout = window.setTimeout(() => {
      setNormalizedSearch(nextSearch)
    }, SCAN_AGENT_PICKER_SEARCH_DEBOUNCE_MS)
    return () => window.clearTimeout(timeout)
  }, [normalizedSearch, open, search])

  React.useEffect(() => {
    nextPageRequestRef.current = false
    setFailedNextPageGeneration(null)
  }, [normalizedSearch])

  const queryKey = React.useMemo(
    () => agentKeys.scanPicker(normalizedSearch),
    [normalizedSearch],
  )
  const filter = React.useMemo(
    () => buildScanAgentPickerFilter(normalizedSearch),
    [normalizedSearch],
  )

  const query = useInfiniteQuery({
    queryKey,
    queryFn: ({ pageParam, signal }) => {
      const params: AgentListQueryParams = {
        pageSize: SCAN_AGENT_PICKER_PAGE_SIZE,
        orderBy: SCAN_AGENT_PICKER_ORDER_BY,
        ...(filter ? { filter } : {}),
        ...(pageParam ? { pageToken: pageParam } : {}),
      }
      return agentService.getAgents(params, signal)
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    enabled: open,
    retry: false,
    refetchInterval: open ? SCAN_AGENT_PICKER_REFRESH_MS : false,
  })
  const {
    fetchNextPage,
    hasNextPage,
    isFetching,
    refetch,
  } = query

  React.useEffect(() => {
    if (!open) {
      void queryClient.cancelQueries({ queryKey, exact: true })
    }
    return () => {
      void queryClient.cancelQueries({ queryKey, exact: true })
    }
  }, [open, queryClient, queryKey])

  const isNextPageError = failedNextPageGeneration === normalizedSearch

  const requestNextPage = React.useCallback(async (explicitRetry: boolean) => {
    if (
      !open ||
      !hasNextPage ||
      isFetching ||
      nextPageRequestRef.current ||
      (isNextPageError && !explicitRetry)
    ) {
      return
    }

    nextPageRequestRef.current = true
    if (explicitRetry) {
      setFailedNextPageGeneration(null)
    }
    try {
      const result = await fetchNextPage({ cancelRefetch: false })
      setFailedNextPageGeneration(result.isFetchNextPageError ? normalizedSearch : null)
    } finally {
      nextPageRequestRef.current = false
    }
  }, [
    isNextPageError,
    normalizedSearch,
    open,
    fetchNextPage,
    hasNextPage,
    isFetching,
  ])

  const loadNextPage = React.useCallback(
    () => requestNextPage(false),
    [requestNextPage],
  )
  const retryNextPage = React.useCallback(
    () => requestNextPage(true),
    [requestNextPage],
  )
  const retryCurrentGeneration = React.useCallback(
    () => refetch({ cancelRefetch: false }),
    [refetch],
  )

  const agents = React.useMemo(
    () => flattenCanonicalAgents(query.data?.pages),
    [query.data?.pages],
  )
  const hasRows = agents.length > 0

  return {
    agents,
    normalizedSearch,
    loadedPageCount: query.data?.pages.length ?? 0,
    hasNextPage,
    canLoadNextPage:
      open && Boolean(hasNextPage) && !isFetching && !isNextPageError,
    isInitialLoading: open && query.isPending,
    isInitialError: query.isError && !hasRows,
    isRefreshing: hasRows && query.isRefetching && !query.isFetchingNextPage,
    isRefreshError: hasRows && query.isRefetchError && !query.isFetchNextPageError,
    isFetchingNextPage: query.isFetchingNextPage,
    isNextPageError,
    loadNextPage,
    retryNextPage,
    retryCurrentGeneration,
  }
}
