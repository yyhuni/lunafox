import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { getAssetDeletedCount } from "@/hooks/_shared/asset-mutation-helpers"
import {
  createTarget,
  updateTarget,
  deleteTarget,
  batchDeleteTargets,
  batchCreateTargets,
  linkTargetOrganizations,
  unlinkTargetOrganizations,
} from "@/services/target.service"
import type {
  CreateTargetRequest,
  UpdateTargetRequest,
  BatchDeleteTargetsRequest,
  BatchCreateTargetsRequest,
} from "@/types/target.types"
import { targetKeys } from "./keys"

/**
 * create goals
 */
export function useCreateTarget() {
  return useResourceMutation({
    mutationFn: (data: CreateTargetRequest) => createTarget(data),
    loadingToast: {
      key: "common.status.creating",
      params: {},
      id: "create-target",
    },
    invalidate: [{ queryKey: targetKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success("toast.target.create.success")
    },
    errorFallbackKey: "toast.target.create.error",
  })
}

/**
 * update target
 */
export function useUpdateTarget() {
  return useResourceMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateTargetRequest }) =>
      updateTarget(id, data),
    loadingToast: {
      key: "common.status.updating",
      params: {},
      id: ({ id }) => `update-target-${id}`,
    },
    invalidate: [
      { queryKey: targetKeys.all },
      ({ variables }) => ({ queryKey: targetKeys.detail(variables.id) }),
    ],
    onSuccess: ({ toast }) => {
      toast.success("toast.target.update.success")
    },
    errorFallbackKey: "toast.target.update.error",
  })
}

/**
 * Delete target (RESTful 204 No Content)
 */
export function useDeleteTarget() {
  return useResourceMutation({
    mutationFn: ({ id }: { id: number; name: string }) => deleteTarget(id),
    loadingToast: {
      key: "common.status.deleting",
      params: {},
      id: ({ id }) => `delete-target-${id}`,
    },
    invalidate: [
      { queryKey: targetKeys.all },
      { queryKey: ["organizations"] },
    ],
    onSuccess: ({ variables, toast }) => {
      toast.success("toast.target.delete.success", { name: variables.name })
    },
    errorFallbackKey: "toast.target.delete.error",
  })
}

/**
 * Delete targets in batches
 */
export function useBatchDeleteTargets() {
  return useResourceMutation({
    mutationFn: (data: BatchDeleteTargetsRequest) => batchDeleteTargets(data),
    loadingToast: {
      key: "common.status.batchDeleting",
      params: {},
      id: "batch-delete-targets",
    },
    invalidate: [{ queryKey: targetKeys.all }],
    onSuccess: ({ data, toast }) => {
      toast.success("toast.target.delete.bulkSuccess", {
        count: getAssetDeletedCount(data),
      })
    },
    errorFallbackKey: "toast.target.delete.error",
  })
}

/**
 * Create goals in batches
 */
export function useBatchCreateTargets() {
  return useResourceMutation({
    mutationFn: (data: BatchCreateTargetsRequest) => batchCreateTargets(data),
    loadingToast: {
      key: "common.status.batchCreating",
      params: {},
      id: "batch-create-targets",
    },
    invalidate: [
      { queryKey: targetKeys.all },
      { queryKey: ["organizations"] },
    ],
    onSuccess: ({ data, toast }) => {
      toast.success("toast.target.create.bulkSuccess", { count: data.createdCount || 0 })
    },
    errorFallbackKey: "toast.target.create.error",
  })
}

/**
 * Link goals and organizations
 */
export function useLinkTargetOrganizations() {
  return useResourceMutation({
    mutationFn: ({ targetId, organizationIds }: { targetId: number; organizationIds: number[] }) =>
      linkTargetOrganizations(targetId, organizationIds),
    loadingToast: {
      key: "common.status.updating",
      params: {},
      id: ({ targetId }) => `link-target-organizations-${targetId}`,
    },
    invalidate: [
      ({ variables }) => ({
        queryKey: targetKeys.organizations(variables.targetId, 1, 10),
        exact: false,
      }),
      ({ variables }) => ({ queryKey: targetKeys.detail(variables.targetId) }),
    ],
    onSuccess: ({ toast }) => {
      toast.success("toast.target.link.success")
    },
    errorFallbackKey: "toast.target.link.error",
  })
}

/**
 * Disassociate a target from an organization
 */
export function useUnlinkTargetOrganizations() {
  return useResourceMutation({
    mutationFn: ({ targetId, organizationIds }: { targetId: number; organizationIds: number[] }) =>
      unlinkTargetOrganizations(targetId, organizationIds),
    loadingToast: {
      key: "common.status.unlinking",
      params: {},
      id: ({ targetId }) => `unlink-target-organizations-${targetId}`,
    },
    invalidate: [
      ({ variables }) => ({
        queryKey: targetKeys.organizations(variables.targetId, 1, 10),
        exact: false,
      }),
      ({ variables }) => ({ queryKey: targetKeys.detail(variables.targetId) }),
    ],
    onSuccess: ({ toast }) => {
      toast.success("toast.target.unlink.success")
    },
    errorFallbackKey: "toast.target.unlink.error",
  })
}
