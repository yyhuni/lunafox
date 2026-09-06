"use client"

import * as React from "react"
import {
  keepPreviousData,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"

import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"

import {
  getNucleiPoc,
  getNucleiPocActiveSyncTaskName,
  getNucleiPocErrorBody,
  getNucleiPocErrorReason,
  getNucleiPocFilterOptions,
  getNucleiPocSource,
  getNucleiPocSyncTask,
  getNucleiPocHttpStatus,
  listNucleiPocs,
  setNucleiPocActivation,
  syncNucleiPocSource,
  updateNucleiPoc,
} from "@/services/nuclei-poc.service"
import type {
  NucleiPocDetail,
  NucleiPocFilterOptionField,
  NucleiPocFilterOptionsResponse,
  NucleiPocListItem,
  NucleiPocListQuery,
  NucleiPocListResponse,
  NucleiPocSource,
  NucleiPocSyncTask,
  SyncNucleiPocSourceRequest,
  SetNucleiPocActivationRequest,
  SetNucleiPocActivationResponse,
  UpdateNucleiPocRequest,
} from "@/types/nuclei-poc.types"

export const nucleiPocKeys = {
  all: ["nuclei-pocs"] as const,
  source: () => ["nuclei-pocs", "source"] as const,
  lists: () => ["nuclei-pocs", "list"] as const,
  list: (query: NucleiPocListQuery) => ["nuclei-pocs", "list", query] as const,
  filterOptions: (field: NucleiPocFilterOptionField) => ["nuclei-pocs", "filter-options", field] as const,
  details: () => ["nuclei-pocs", "detail"] as const,
  detail: (name: string | null) => ["nuclei-pocs", "detail", name] as const,
  task: (name: string | null) => ["nuclei-pocs", "task", name] as const,
}

export function useNucleiPocSource() {
  return useQuery<NucleiPocSource | null>({
    queryKey: nucleiPocKeys.source(),
    queryFn: getNucleiPocSource,
    retry: false,
  })
}

export function useNucleiPocs(query: NucleiPocListQuery) {
  const requestQuery = {
    pageSize: query.pageSize ?? 50,
    pageToken: query.pageToken,
    filter: query.filter,
    orderBy: query.orderBy ?? "templateId asc",
  }

  return useQuery<NucleiPocListResponse>({
    queryKey: nucleiPocKeys.list(requestQuery),
    queryFn: () => listNucleiPocs(requestQuery),
    placeholderData: keepPreviousData,
    retry: false,
  })
}

export function useNucleiPocFilterOptions(field: NucleiPocFilterOptionField) {
  return useQuery<NucleiPocFilterOptionsResponse>({
    queryKey: nucleiPocKeys.filterOptions(field),
    queryFn: () => getNucleiPocFilterOptions(field),
    placeholderData: keepPreviousData,
    retry: false,
  })
}

export function useNucleiPocDetail(name: string | null, enabled = true) {
  return useQuery<NucleiPocDetail>({
    queryKey: nucleiPocKeys.detail(name),
    queryFn: () => getNucleiPoc(name!),
    enabled: Boolean(name) && enabled,
    retry: false,
  })
}

export function useSyncNucleiPocSource() {
	const queryClient = useQueryClient()
	return useResourceMutation<NucleiPocSyncTask, SyncNucleiPocSourceRequest>({
		mutationFn: syncNucleiPocSource,
		retry: false,
		skipDefaultErrorHandler: true,
		onSuccess: ({ data: task }) => {
			queryClient.setQueryData(nucleiPocKeys.task(task.name), task)
		},
	})
}

const TERMINAL_TASK_STATES = new Set(["SUCCEEDED", "FAILED"])

export function isNucleiPocSyncTaskTerminal(task: NucleiPocSyncTask | null | undefined) {
  return Boolean(task && TERMINAL_TASK_STATES.has(task.state))
}

export function useNucleiPocSyncTask(taskName: string | null) {
  const queryClient = useQueryClient()
  const lastInvalidated = React.useRef<string | null>(null)
  const query = useQuery<NucleiPocSyncTask>({
    queryKey: nucleiPocKeys.task(taskName),
		queryFn: () => getNucleiPocSyncTask(taskName!),
    enabled: Boolean(taskName),
		retry: false,
		refetchInterval: (current) => {
      if (!taskName) return false
			const task = current.state.data
			const status = getNucleiPocHttpStatus(current.state.error)
			if (status === 404 || status === 410 || isNucleiPocSyncTaskTerminal(task)) return false
			return 1000
    },
  })

  React.useEffect(() => {
    const task = query.data
    if (!task || task.state !== "SUCCEEDED" || lastInvalidated.current === task.name) return
    lastInvalidated.current = task.name
    void Promise.all([
      queryClient.invalidateQueries({ queryKey: nucleiPocKeys.source() }),
      queryClient.invalidateQueries({ queryKey: nucleiPocKeys.lists() }),
      queryClient.invalidateQueries({ queryKey: nucleiPocKeys.filterOptions("tags") }),
      queryClient.invalidateQueries({ queryKey: nucleiPocKeys.details() }),
    ])
  }, [query.data, queryClient])

  React.useEffect(() => {
    if (!taskName) lastInvalidated.current = null
  }, [taskName])

  return {
    ...query,
    isExpired: getNucleiPocHttpStatus(query.error) === 404 || getNucleiPocHttpStatus(query.error) === 410,
  }
}

export function useUpdateNucleiPocEnabled() {
	const queryClient = useQueryClient()
	return useResourceMutation<NucleiPocListItem, UpdateNucleiPocRequest, { snapshots: Array<[readonly unknown[], unknown]> }>({
		mutationFn: updateNucleiPoc,
		retry: false,
		loadingToast: {
			key: "toast.nucleiPoc.update.loading",
			id: (variables) => `nuclei-poc-enabled-${variables.name}`,
		},
		invalidate: [
			{ queryKey: nucleiPocKeys.lists() },
			({ variables }) => ({ queryKey: nucleiPocKeys.detail(variables.name) }),
		],
		onMutate: async (variables) => {
			await queryClient.cancelQueries({ queryKey: nucleiPocKeys.all })
      const snapshots: Array<[readonly unknown[], unknown]> = []
      queryClient.getQueriesData<NucleiPocListResponse>({ queryKey: nucleiPocKeys.lists() }).forEach(([key, data]) => {
        if (!data) return
        snapshots.push([key, data])
        queryClient.setQueryData<NucleiPocListResponse>(key, {
          ...data,
          results: data.results.map((item) => item.name === variables.name ? { ...item, isEnabled: variables.isEnabled } : item),
        })
      })
      const detailKey = nucleiPocKeys.detail(variables.name)
      const detail = queryClient.getQueryData<NucleiPocDetail>(detailKey)
      if (detail) {
        snapshots.push([detailKey, detail])
        queryClient.setQueryData<NucleiPocDetail>(detailKey, { ...detail, isEnabled: variables.isEnabled })
      }
      return { snapshots }
    },
		onError: ({ context, toast }) => {
		context?.snapshots.forEach(([key, value]) => queryClient.setQueryData(key, value))
		toast.error("toast.nucleiPoc.update.error")
		},
		onSuccess: ({ data, variables, toast }) => {
			queryClient.setQueryData<NucleiPocDetail | undefined>(nucleiPocKeys.detail(variables.name), (current) => current ? { ...current, ...data } : current)
			toast.success("toast.nucleiPoc.update.success")
		},
	})
}

export function useSetNucleiPocActivation() {
  return useResourceMutation<SetNucleiPocActivationResponse, SetNucleiPocActivationRequest>({
    mutationFn: setNucleiPocActivation,
    retry: false,
    loadingToast: {
      key: "toast.nucleiPoc.activation.loading",
      id: ({ enabled }) => `nuclei-poc-activation-${enabled ? "enable" : "disable"}`,
    },
    invalidate: [
      { queryKey: nucleiPocKeys.lists() },
      { queryKey: nucleiPocKeys.details() },
    ],
    onSuccess: ({ data, toast }) => {
      toast.success(data.affectedCount > 0 ? "toast.nucleiPoc.activation.success" : "toast.nucleiPoc.activation.noChange", { count: data.affectedCount })
    },
    onError: async ({ error, queryClient: client, toast }) => {
      const status = getNucleiPocHttpStatus(error)
      const reason = getNucleiPocErrorReason(error)
      if (status === undefined || status === 0) {
        await Promise.all([
          client.invalidateQueries({ queryKey: nucleiPocKeys.lists() }),
          client.invalidateQueries({ queryKey: nucleiPocKeys.details() }),
        ])
        toast.warning("toast.nucleiPoc.activation.unknown")
        return
      }
      if (reason === "SYNC_ALREADY_RUNNING") {
        toast.warning("toast.nucleiPoc.activation.conflict")
        return
      }
      toast.error("toast.nucleiPoc.activation.error")
    },
  })
}

export function getNucleiPocQueryErrorBody(error: unknown) {
  return getNucleiPocErrorBody(error)
}

export function getNucleiPocQueryErrorReason(error: unknown) {
  return getNucleiPocErrorReason(error)
}

export function getNucleiPocQueryActiveSyncTaskName(error: unknown) {
  return getNucleiPocActiveSyncTaskName(error)
}

export function getNucleiPocQueryErrorStatus(error: unknown) {
  return getNucleiPocHttpStatus(error)
}
