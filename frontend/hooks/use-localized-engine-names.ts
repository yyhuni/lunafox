import { useMemo } from "react"
import { useLocale } from "next-intl"

import { useEngineCatalog } from "@/hooks/use-engine-catalog"
import { defaultLocale, isLocale } from "@/i18n/config"
import { getLocalizedExecutedEngineDisplays } from "@/lib/engine-catalog"

export function useLocalizedEngineNames(engineIds: readonly string[]) {
  const locale = useLocale()
  const distinctEngineIds = useMemo(
    () => Array.from(new Set(engineIds.map((engineId) => engineId.trim()).filter(Boolean))),
    [engineIds]
  )
  const catalog = useEngineCatalog(distinctEngineIds.length > 0)
  const selectedLocale = isLocale(locale) ? locale : defaultLocale

  const result = useMemo(() => {
    if (distinctEngineIds.length === 0) {
      return { engineNamesById: new Map<string, string>(), engineDescriptionsById: new Map<string, string>(), error: undefined }
    }
    if (!catalog.data) {
      return { engineNamesById: new Map<string, string>(), engineDescriptionsById: new Map<string, string>(), error: undefined }
    }
    try {
      const displays = getLocalizedExecutedEngineDisplays(distinctEngineIds, catalog.data, selectedLocale)
      return {
        engineNamesById: new Map(displays.map((engine) => [engine.engineId, engine.displayName])),
        engineDescriptionsById: new Map(displays.map((engine) => [engine.engineId, engine.description])),
        error: undefined,
      }
    } catch (error) {
      return { engineNamesById: new Map<string, string>(), engineDescriptionsById: new Map<string, string>(), error }
    }
  }, [catalog.data, distinctEngineIds, selectedLocale])

  return {
    engineNamesById: result.engineNamesById,
    engineDescriptionsById: result.engineDescriptionsById,
    isLoading: distinctEngineIds.length > 0 && catalog.isLoading,
    error: catalog.error ?? result.error,
    refetch: catalog.refetch,
  }
}
