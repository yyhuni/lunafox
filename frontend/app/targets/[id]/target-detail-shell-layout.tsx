import * as React from "react"

import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

export const TARGET_DETAIL_SHELL_SLOTS = {
  header: "target-detail-shell-header",
  primaryTabs: "target-detail-shell-primary-tabs",
  secondaryTabs: "target-detail-shell-secondary-tabs",
  content: "target-detail-shell-content",
} as const

// The outer handoff and both state wrappers must establish the same definite
// viewport workspace before a child route contributes its own scroll height.
export const TARGET_DETAIL_SHELL_HANDOFF_CLASS = "flex h-full min-h-0 flex-1 flex-col"
export const TARGET_DETAIL_SHELL_CLASS = "flex h-full min-h-0 flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6"
export const TARGET_DETAIL_SHELL_HEADER_CLASS = "flex min-w-0 items-center gap-2 overflow-x-auto px-4 text-sm lg:px-6"
export const TARGET_DETAIL_SHELL_PRIMARY_TABS_CLASS = "flex items-center justify-between px-4 lg:px-6"
export const TARGET_DETAIL_SHELL_SECONDARY_TABS_CLASS = "flex items-center px-4 lg:px-6"

export function TargetDetailShellLayout({ className, ...props }: React.ComponentProps<"div">) {
  return <div {...props} className={cn(TARGET_DETAIL_SHELL_CLASS, className)} />
}

export function TargetDetailShellHeader({ className, ...props }: React.ComponentProps<"nav">) {
  return (
    <nav
      {...props}
      {...getLoadingStructureSlotAttributes(TARGET_DETAIL_SHELL_SLOTS.header)}
      className={cn(TARGET_DETAIL_SHELL_HEADER_CLASS, className)}
    />
  )
}

export function TargetDetailShellPrimaryTabs({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      {...getLoadingStructureSlotAttributes(TARGET_DETAIL_SHELL_SLOTS.primaryTabs)}
      className={cn(TARGET_DETAIL_SHELL_PRIMARY_TABS_CLASS, className)}
    />
  )
}

export function TargetDetailShellSecondaryTabs({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      {...getLoadingStructureSlotAttributes(TARGET_DETAIL_SHELL_SLOTS.secondaryTabs)}
      className={cn(TARGET_DETAIL_SHELL_SECONDARY_TABS_CLASS, className)}
    />
  )
}

export function TargetDetailShellContent({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      {...getLoadingStructureSlotAttributes(TARGET_DETAIL_SHELL_SLOTS.content)}
      className={cn("flex min-h-0 flex-1 flex-col", className)}
    />
  )
}

interface TargetDetailShellLoadingStateProps {
  primaryTabLabels: readonly string[]
  activePrimaryTabIndex: number
  secondaryTabLabels: readonly string[]
  activeSecondaryTabIndex: number
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

export function TargetDetailShellLoadingState({
  primaryTabLabels,
  activePrimaryTabIndex,
  secondaryTabLabels,
  activeSecondaryTabIndex,
  showSecondaryNav,
  children,
}: TargetDetailShellLoadingStateProps) {
  if (primaryTabLabels.length === 0) {
    throw new Error("TargetDetailShellLoadingState requires primary tab labels.")
  }

  if (activePrimaryTabIndex < 0 || activePrimaryTabIndex >= primaryTabLabels.length) {
    throw new Error("TargetDetailShellLoadingState activePrimaryTabIndex must reference a rendered tab.")
  }

  if (showSecondaryNav && (
    secondaryTabLabels.length === 0
    || activeSecondaryTabIndex < 0
    || activeSecondaryTabIndex >= secondaryTabLabels.length
  )) {
    throw new Error("TargetDetailShellLoadingState activeSecondaryTabIndex must reference a rendered tab.")
  }

  return (
    <TargetDetailShellLayout aria-hidden="true">
      <TargetDetailShellHeader aria-label="Loading target detail">
        <Skeleton className="h-5 w-16 shrink-0 rounded-full" />
        <Skeleton className="h-5 w-32 shrink-0 rounded-full" />
        <Skeleton className="h-5 w-20 shrink-0 rounded-full" />
      </TargetDetailShellHeader>

      <TargetDetailShellPrimaryTabs>
        <div className="flex items-center gap-3">
          <Tabs value={`tab-${activePrimaryTabIndex}`}>
            <TabsList>
              {primaryTabLabels.map((label, index) => (
                <TabsTrigger key={label} value={`tab-${index}`} disabled>
                  <DetailTabPlaceholder label={label} withIcon />
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </div>
      </TargetDetailShellPrimaryTabs>

      {showSecondaryNav ? (
        <TargetDetailShellSecondaryTabs>
          <Tabs value={`secondary-tab-${activeSecondaryTabIndex}`} className="w-full">
            <TabsList variant="content">
              {secondaryTabLabels.map((label, index) => (
                <TabsTrigger key={label} value={`secondary-tab-${index}`} variant="content" disabled>
                  <DetailTabPlaceholder label={label} />
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </TargetDetailShellSecondaryTabs>
      ) : null}

      <TargetDetailShellContent>{children}</TargetDetailShellContent>
    </TargetDetailShellLayout>
  )
}
