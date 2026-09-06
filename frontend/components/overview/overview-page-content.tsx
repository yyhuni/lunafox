"use client"

import { OverviewPageHeader } from "@/components/overview/overview-page-header"
import { OverviewSectionsLoader } from "@/components/overview/overview-sections-loader"
import { useOverviewRefresh } from "@/hooks/use-overview-refresh"

export function OverviewPageContent() {
  const refresh = useOverviewRefresh()

  return (
    <>
      <OverviewPageHeader
        isRefreshing={refresh.isRefreshing}
        lastRefreshedAt={refresh.lastRefreshedAt}
        onRefresh={refresh.refresh}
      />
      <OverviewSectionsLoader />
    </>
  )
}
