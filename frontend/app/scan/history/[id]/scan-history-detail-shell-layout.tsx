import * as React from "react"

import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

export const SCAN_HISTORY_DETAIL_SHELL_SLOTS = {
  header: "scan-history-detail-shell-header",
  primaryTabs: "scan-history-detail-shell-primary-tabs",
  secondaryTabs: "scan-history-detail-shell-secondary-tabs",
  content: "scan-history-detail-shell-content",
} as const

// The outer handoff and both state wrappers must establish the same definite
// viewport workspace before a child route contributes its own scroll height.
export const SCAN_HISTORY_DETAIL_SHELL_HANDOFF_CLASS = "flex h-full min-h-0 flex-1 flex-col"
export const SCAN_HISTORY_DETAIL_SHELL_CLASS =
  "flex h-full min-h-0 min-w-0 flex-1 flex-col gap-4 overflow-x-hidden py-4 md:gap-6 md:py-6"
export const SCAN_HISTORY_DETAIL_SHELL_HEADER_CLASS = "flex items-center gap-2 px-4 text-sm lg:px-6"
export const SCAN_HISTORY_DETAIL_SHELL_PRIMARY_TABS_CLASS = "overflow-x-auto px-4 lg:px-6"
export const SCAN_HISTORY_DETAIL_SHELL_SECONDARY_TABS_CLASS = "flex min-w-0 items-center overflow-x-auto px-4 lg:px-6"

export function ScanHistoryDetailShellLayout({ className, ...props }: React.ComponentProps<"div">) {
  return <div {...props} className={cn(SCAN_HISTORY_DETAIL_SHELL_CLASS, className)} />
}

export function ScanHistoryDetailShellHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      {...getLoadingStructureSlotAttributes(SCAN_HISTORY_DETAIL_SHELL_SLOTS.header)}
      className={cn(SCAN_HISTORY_DETAIL_SHELL_HEADER_CLASS, className)}
    />
  )
}

export function ScanHistoryDetailShellPrimaryTabs({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      {...getLoadingStructureSlotAttributes(SCAN_HISTORY_DETAIL_SHELL_SLOTS.primaryTabs)}
      className={cn(SCAN_HISTORY_DETAIL_SHELL_PRIMARY_TABS_CLASS, className)}
    />
  )
}

export function ScanHistoryDetailShellSecondaryTabs({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      {...getLoadingStructureSlotAttributes(SCAN_HISTORY_DETAIL_SHELL_SLOTS.secondaryTabs)}
      className={cn(SCAN_HISTORY_DETAIL_SHELL_SECONDARY_TABS_CLASS, className)}
    />
  )
}

export function ScanHistoryDetailShellContent({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      {...getLoadingStructureSlotAttributes(SCAN_HISTORY_DETAIL_SHELL_SLOTS.content)}
      className={cn("flex min-h-0 flex-1 flex-col", className)}
    />
  )
}

interface ScanHistoryDetailShellLoadingStateProps {
  primaryTabLabels: readonly string[]
  activePrimaryTabIndex: number
  secondaryTabLabels: readonly string[]
  showSecondaryNav: boolean
  children: React.ReactNode
}

function DetailTabPlaceholder({ label, withIcon }: { label: string; withIcon?: boolean }) {
  return (
    <span aria-hidden="true" className="grid">
      <span className="invisible col-start-1 row-start-1 inline-flex items-center gap-1.5 whitespace-nowrap">
        {withIcon ? <span className="size-4 shrink-0" /> : null}
        <span>{label}</span>
      </span>
      <Skeleton className="col-start-1 row-start-1 h-4 w-full self-center rounded-full" />
    </span>
  )
}

export function ScanHistoryDetailShellLoadingState({
  primaryTabLabels,
  activePrimaryTabIndex,
  secondaryTabLabels,
  showSecondaryNav,
  children,
}: ScanHistoryDetailShellLoadingStateProps) {
  if (primaryTabLabels.length === 0) {
    throw new Error("ScanHistoryDetailShellLoadingState requires primary tab labels.")
  }

  if (activePrimaryTabIndex < 0 || activePrimaryTabIndex >= primaryTabLabels.length) {
    throw new Error("ScanHistoryDetailShellLoadingState activePrimaryTabIndex must reference a rendered tab.")
  }

  return (
    <ScanHistoryDetailShellLayout aria-hidden="true">
      <ScanHistoryDetailShellHeader>
        <Skeleton className="h-5 w-20 shrink-0 rounded-full" />
        <Skeleton className="h-5 w-32 shrink-0 rounded-full" />
      </ScanHistoryDetailShellHeader>

      <ScanHistoryDetailShellPrimaryTabs>
        <Tabs value={`tab-${activePrimaryTabIndex}`} className="w-max min-w-full">
          <TabsList className="max-w-none min-w-max">
            {primaryTabLabels.map((label, index) => (
              <TabsTrigger key={label} value={`tab-${index}`} disabled>
                <DetailTabPlaceholder label={label} withIcon />
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </ScanHistoryDetailShellPrimaryTabs>

      {showSecondaryNav ? (
        <ScanHistoryDetailShellSecondaryTabs>
          <Tabs value="secondary-tab-0" className="w-full">
            <TabsList variant="content" className="min-w-max">
              {secondaryTabLabels.map((label, index) => (
                <TabsTrigger key={label} value={`secondary-tab-${index}`} variant="content" disabled>
                  <DetailTabPlaceholder label={label} />
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </ScanHistoryDetailShellSecondaryTabs>
      ) : null}

      <ScanHistoryDetailShellContent>{children}</ScanHistoryDetailShellContent>
    </ScanHistoryDetailShellLayout>
  )
}
