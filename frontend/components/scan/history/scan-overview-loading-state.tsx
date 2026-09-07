import { Skeleton } from "@/components/ui/skeleton"
import {
  DetailDrawerTabs,
  DetailDrawerTabsList,
  DetailDrawerTabsTrigger,
} from "@/components/shared/detail-drawer"
import {
  getScanOverviewSummaryItemDividerClass,
  SCAN_OVERVIEW_PRIMARY_COLUMN_CLASS,
  SCAN_OVERVIEW_SIDE_PANEL_CLASS,
  SCAN_OVERVIEW_SUMMARY_GRID_CLASS,
  SCAN_OVERVIEW_SUMMARY_ITEM_CLASS,
  SCAN_OVERVIEW_WORKBENCH_CLASS,
} from "@/components/scan/history/scan-overview-layout"
import { cn } from "@/lib/utils"

export function ScanOverviewLoadingState() {
  return (
    <div className={SCAN_OVERVIEW_WORKBENCH_CLASS}>
      <div className={SCAN_OVERVIEW_PRIMARY_COLUMN_CLASS}>
        <section className="rounded-lg border border-border/60 bg-card px-4 py-4">
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-3">
              <Skeleton className="h-8 w-56" />
              <Skeleton className="h-5 w-16 rounded-full" />
              <Skeleton className="h-5 w-12 rounded-full" />
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-5 w-24 rounded-full" />
              <Skeleton className="h-5 w-32 rounded-full" />
            </div>
          </div>
        </section>

        <section className="space-y-2">
          <div className="overflow-hidden rounded-lg border border-border/60 bg-card">
            <div className={SCAN_OVERVIEW_SUMMARY_GRID_CLASS}>
              {Array.from({ length: 6 }).map((_, index) => (
                <div
                  key={index}
                  className={cn(
                    SCAN_OVERVIEW_SUMMARY_ITEM_CLASS,
                    getScanOverviewSummaryItemDividerClass(index)
                  )}
                >
                  <Skeleton className="h-4 w-12" />
                  <Skeleton className="h-4 w-10" />
                </div>
              ))}
            </div>
          </div>
          <Skeleton className="h-1 w-full" />
        </section>

        <section className="space-y-3">
          <Skeleton className="h-5 w-24" />
          <div className="space-y-2.5">
            {Array.from({ length: 3 }).map((_, index) => (
              <div key={index} className="relative pl-8 sm:pl-10">
                <div className="absolute left-0 top-4 h-5 w-5 rounded-full border border-border/70 bg-background" />
                <div className="overflow-hidden rounded-md border border-border/60 bg-background px-4 py-3">
                  <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div className="min-w-0 space-y-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <Skeleton className="h-5 w-32" />
                        <Skeleton className="h-5 w-10 rounded-full" />
                      </div>
                      <Skeleton className="h-4 w-full" />
                      <Skeleton className="h-4 w-3/4" />
                    </div>
                    <div className="flex shrink-0 items-start gap-4">
                      <div className="space-y-1">
                        <Skeleton className="h-3 w-16" />
                        <Skeleton className="h-4 w-20" />
                      </div>
                      <div className="space-y-1">
                        <Skeleton className="h-3 w-16" />
                        <Skeleton className="h-4 w-20" />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </section>

        <section className="space-y-3">
          <Skeleton className="h-5 w-28" />
          <div className="flex flex-col gap-2 border-b border-border px-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center">
              <DetailDrawerTabs value="logs" aria-hidden="true">
                <DetailDrawerTabsList variant="minimal" size="md" className="min-w-0 max-w-full overflow-x-auto">
                  <DetailDrawerTabsTrigger variant="minimal" activeIndicator="fixed" value="logs" size="md" disabled>
                    <Skeleton className="h-3.5 w-20 rounded-full" />
                  </DetailDrawerTabsTrigger>
                  <DetailDrawerTabsTrigger variant="minimal" activeIndicator="fixed" value="config" size="md" disabled>
                    <Skeleton className="h-3.5 w-16 rounded-full" />
                  </DetailDrawerTabsTrigger>
                </DetailDrawerTabsList>
              </DetailDrawerTabs>
            </div>
          </div>
          <div className="rounded-lg border border-border/60">
            <div className="space-y-2 px-4 py-4">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-5/6" />
              <Skeleton className="h-4 w-2/3" />
              <Skeleton className="h-4 w-3/4" />
            </div>
          </div>
        </section>
      </div>

      <aside className={SCAN_OVERVIEW_SIDE_PANEL_CLASS}>
        <div className="overflow-hidden rounded-lg border border-border/60">
          {Array.from({ length: 4 }).map((_, sectionIndex) => (
            <section
              key={sectionIndex}
              className={cn("space-y-3 p-4", sectionIndex > 0 && "border-t border-border/60")}
            >
              <Skeleton className="h-5 w-24" />
              <div className="space-y-3">
                {Array.from({ length: sectionIndex === 0 || sectionIndex === 3 ? 3 : 4 }).map((__, rowIndex) => (
                  <div key={rowIndex} className="flex items-center justify-between gap-4">
                    <Skeleton className="h-4 w-20" />
                    <Skeleton className="h-4 w-16" />
                  </div>
                ))}
              </div>
            </section>
          ))}
        </div>
      </aside>
    </div>
  )
}
