import { createResourceKeys } from "@/hooks/_shared/query-keys"
import type { GetVulnerabilitiesParams } from "@/types/vulnerability.types"

const vulnerabilityKeyBase = createResourceKeys("vulnerabilities")

export const vulnerabilityKeys = {
  ...vulnerabilityKeyBase,
  list: (params: GetVulnerabilitiesParams, filter?: string) =>
    [...vulnerabilityKeyBase.all, "list", params, filter] as const,
  byScan: (scanId: number, params: GetVulnerabilitiesParams, filter?: string) =>
    [...vulnerabilityKeyBase.all, "scan", scanId, params, filter] as const,
  byTarget: (targetId: number, params: GetVulnerabilitiesParams, filter?: string) =>
    [...vulnerabilityKeyBase.all, "target", targetId, params, filter] as const,
  stats: () => [...vulnerabilityKeyBase.all, "stats"] as const,
  statsByTarget: (targetId: number) =>
    [...vulnerabilityKeyBase.all, "stats", "target", targetId] as const,
}
