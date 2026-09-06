import { createResourceKeys } from "@/hooks/_shared/query-keys"
import type { WebsiteAssetScope } from "@/types/website.types"

const targetKeyBase = createResourceKeys("targets", {
  list: (params: { pageSize?: number; pageToken?: string; filter?: string; orderBy?: string }) =>
    params,
  detail: (id: number) => id,
})

export const targetKeys = {
  ...targetKeyBase,
  organizations: (targetId: number, page: number, pageSize: number) =>
    [...targetKeyBase.detail(targetId), "organizations", page, pageSize] as const,
  endpoints: (
    targetId: number,
    params: { pageSize?: number; pageToken?: string; filter?: string; orderBy?: string },
    websiteScope?: WebsiteAssetScope,
  ) => [...targetKeyBase.detail(targetId), "endpoints", params, websiteScope] as const,
}
