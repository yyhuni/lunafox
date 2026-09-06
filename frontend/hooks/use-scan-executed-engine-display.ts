import { useMemo } from "react"

import { useLocalizedEngineNames } from "@/hooks/use-localized-engine-names"
import type { ScanRecord } from "@/types/scan.types"

export function useScanExecutedEngineDisplay(scans: readonly ScanRecord[]) {
  const engineIds = useMemo(
    () => scans.flatMap((scan) => [
      ...scan.plannedEngineIds,
      ...(scan.runtimeTasks ?? []).map((task) => task.engineId),
    ]),
    [scans]
  )
  const localizedEngines = useLocalizedEngineNames(engineIds)
  const engineNamesByScanId = useMemo(
    () => new Map(scans.map((scan) => [
      scan.id,
      scan.plannedEngineIds
        .map((engineId) => localizedEngines.engineNamesById.get(engineId))
        .filter((engineName): engineName is string => Boolean(engineName)),
    ])),
    [localizedEngines.engineNamesById, scans]
  )
  const engineDescriptionsByScanId = useMemo(
    () => new Map(scans.map((scan) => [
      scan.id,
      scan.plannedEngineIds
        .map((engineId) => localizedEngines.engineDescriptionsById.get(engineId))
        .filter((description): description is string => Boolean(description)),
    ])),
    [localizedEngines.engineDescriptionsById, scans]
  )
  return {
    engineNamesByScanId,
    engineDescriptionsByScanId,
    engineNamesById: localizedEngines.engineNamesById,
    engineDescriptionsById: localizedEngines.engineDescriptionsById,
    isLoading: localizedEngines.isLoading,
    error: localizedEngines.error,
    refetch: localizedEngines.refetch,
  }
}
