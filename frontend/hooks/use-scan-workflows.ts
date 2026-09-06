import { useCallback } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  createScanWorkflow,
  getScanWorkflow,
  getScanWorkflowProfile,
  listScanWorkflows,
  updateScanWorkflow,
  type ListScanWorkflowsInput,
} from "@/services/scan-workflow.service"
import type { CreateScanWorkflowInput, UpdateScanWorkflowInput } from "@/types/scan-workflow.types"

const scanWorkflowKeys = {
  all: ["scan-workflows"] as const,
  list: (input: ListScanWorkflowsInput) => ["scan-workflows", "list", input] as const,
  detail: (id: string) => ["scan-workflows", "detail", id] as const,
  profile: (id: string) => ["scan-workflows", "profile", id] as const,
}

export function useScanWorkflowList(input: ListScanWorkflowsInput = {}) {
  return useQuery({
    queryKey: scanWorkflowKeys.list(input),
    queryFn: () => listScanWorkflows(input),
  })
}

// Read-only scan launchers need a bounded first page, while management uses
// useScanWorkflowList so pagination remains owned by the server.
export function useScanWorkflows() {
  return useQuery({
    queryKey: scanWorkflowKeys.list({ pageSize: 100 }),
    queryFn: async () => (await listScanWorkflows({ pageSize: 100 })).scanWorkflows,
  })
}

export function useScanWorkflow(id: string) {
  return useQuery({
    queryKey: scanWorkflowKeys.detail(id),
    queryFn: () => getScanWorkflow(id),
    enabled: !!id,
  })
}

export function useScanWorkflowProfile(id: string) {
  return useQuery({
    queryKey: scanWorkflowKeys.profile(id),
    queryFn: () => getScanWorkflowProfile(id),
    enabled: !!id,
  })
}

export function useCreateScanWorkflow() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateScanWorkflowInput) => createScanWorkflow(input),
    onSuccess: () => client.invalidateQueries({ queryKey: scanWorkflowKeys.all }),
  })
}

export function useUpdateScanWorkflow() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateScanWorkflowInput }) => updateScanWorkflow(id, input),
    onSuccess: (workflow) => {
      client.setQueryData(scanWorkflowKeys.detail(workflow.name), workflow)
      client.invalidateQueries({ queryKey: scanWorkflowKeys.all })
    },
  })
}

export function useLoadScanWorkflowProfile() {
  const queryClient = useQueryClient()
  return useCallback((id: string) => queryClient.fetchQuery({
    queryKey: scanWorkflowKeys.profile(id),
    queryFn: () => getScanWorkflowProfile(id),
  }), [queryClient])
}
