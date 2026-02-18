import { createResourceKeys } from "@/hooks/_shared/query-keys"
import type { GetScansParams } from "@/types/scan.types"

const scanKeyBase = createResourceKeys("scans", {
  list: (params: GetScansParams) => params,
  detail: (id: number) => id,
})

export const scanKeys = {
  ...scanKeyBase,
  target: (targetId: number, pageSize: number) => [...scanKeyBase.all, "target", targetId, pageSize] as const,
  statistics: () => [...scanKeyBase.all, "statistics"] as const,
}
