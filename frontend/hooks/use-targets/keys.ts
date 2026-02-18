import { createResourceKeys } from "@/hooks/_shared/query-keys"

const targetKeyBase = createResourceKeys("targets", {
  list: (params: { page?: number; pageSize?: number; organizationId?: number; filter?: string; type?: string }) =>
    params,
  detail: (id: number) => id,
})

export const targetKeys = {
  ...targetKeyBase,
  organizations: (targetId: number, page: number, pageSize: number) =>
    [...targetKeyBase.detail(targetId), "organizations", page, pageSize] as const,
  endpoints: (targetId: number, params: { page?: number; pageSize?: number; filter?: string }) =>
    [...targetKeyBase.detail(targetId), "endpoints", params] as const,
  blacklist: (targetId: number) => [...targetKeyBase.detail(targetId), "blacklist"] as const,
}
