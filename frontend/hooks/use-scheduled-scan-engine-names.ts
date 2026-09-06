import React from "react"
import { useLocale } from "next-intl"

import { useScanWorkflows } from "@/hooks/use-scan-workflows"
import { useEngineCatalog } from "@/hooks/use-engine-catalog"
import { isLocale } from "@/i18n/config"
import { getLocalizedEngineSummaryName } from "@/lib/engine-catalog"
import type { ScheduledScan } from "@/types/scheduled-scan.types"

export function useScheduledScanEngineNames(scheduledScans: ScheduledScan[]): ScheduledScan[] {
  const locale = useLocale()
  const { data: scanWorkflows = [], isSuccess: hasScanWorkflows } = useScanWorkflows()
  const { data: engines = [], isSuccess: hasEngineCatalog } = useEngineCatalog()

  const engineNamesByWorkflow = React.useMemo(() => {
    // The catalogs resolve independently and can retry after an error; validate only complete catalog data.
    if (!hasScanWorkflows || !hasEngineCatalog) return new Map<string, string[]>()

    const currentLocale = isLocale(locale) ? locale : "zh"
    const next = new Map<string, string[]>()
    const enginesById = new Map(engines.map((engine) => [engine.engineId, engine]))

    for (const workflow of scanWorkflows) {
      const engineNames = (workflow.steps ?? []).map((step) => {
        const engine = enginesById.get(step.engineId)
        if (!engine) throw new Error(`Workflow ${workflow.name} references unavailable engine ${step.engineId}`)
        return getLocalizedEngineSummaryName(engine, currentLocale)
      })
      const keys = [workflow.name, workflow.displayName]
      keys.forEach((key) => {
        if (key) next.set(key, engineNames)
      })
    }

    return next
  }, [engines, hasEngineCatalog, hasScanWorkflows, locale, scanWorkflows])

  return React.useMemo(
    () =>
      scheduledScans.map((scan) => ({
        ...scan,
        engineNames: engineNamesByWorkflow.get(scan.scanWorkflow) ?? [],
      })),
    [engineNamesByWorkflow, scheduledScans]
  )
}
