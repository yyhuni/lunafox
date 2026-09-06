import { createResourceKeys } from "@/hooks/_shared/query-keys"
import type { GetVulnerabilitiesParams, VulnerabilityFilterOptionField } from "@/types/vulnerability.types"
import type { WebsiteAssetScope } from "@/types/website.types"

const vulnerabilityKeyBase = createResourceKeys("vulnerabilities")

export const vulnerabilityKeys = {
  ...vulnerabilityKeyBase,
  list: (params: GetVulnerabilitiesParams) =>
    [...vulnerabilityKeyBase.all, "list", params] as const,
  byScan: (scanId: number, params: GetVulnerabilitiesParams) =>
    [...vulnerabilityKeyBase.all, "scan", scanId, params] as const,
  byTarget: (targetId: number, params: GetVulnerabilitiesParams, websiteScope?: WebsiteAssetScope) =>
    [...vulnerabilityKeyBase.all, "target", targetId, params, websiteScope] as const,
  filterOptions: (scope: "global" | "target" | "scan", id: number | undefined, field: VulnerabilityFilterOptionField) =>
    [...vulnerabilityKeyBase.all, "filterOptions", scope, id ?? 0, field] as const,
  stats: () => [...vulnerabilityKeyBase.all, "stats"] as const,
  statsByTarget: (targetId: number) =>
    [...vulnerabilityKeyBase.all, "stats", "target", targetId] as const,
}
