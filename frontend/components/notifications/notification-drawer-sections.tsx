"use client"

import React from "react"

import { BellOff, Info, semanticIcons } from "@/components/icons"
import { EdgePanelHeader } from "@/components/shared/edge-panel-header"
import { Spinner } from "@/components/shared/loading/spinner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import { getStatusToneBadgeClass, getStatusToneBgClass, type StatusTone } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { compactFeedbackDrawerContentClassName } from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"
import type {
  NotificationInboxItem,
  NotificationPriority,
} from "@/types/notification.types"

import type { NotificationDrawerState, NotificationFilter } from "./notification-drawer-state"

function notificationPriorityTone(priority: NotificationPriority): StatusTone {
  switch (priority) {
    case "normal":
      return "info"
    case "high":
      return "warning"
    case "critical":
      return "error"
  }
}

function NotificationCard({
  notification,
  categoryTitle,
  priorityTitle,
  formattedTime,
  loading = false,
}: {
  notification?: NotificationInboxItem
  categoryTitle?: string
  priorityTitle?: string
  formattedTime?: string
  loading?: boolean
}) {
  if (loading) {
    return (
      <div className="radius-surface border bg-card p-3">
        <div className="flex items-start gap-3">
          <Skeleton className="size-8 shrink-0 rounded-none" />
          <div className="min-w-0 flex-1 space-y-2">
            <div className="flex items-center justify-between gap-2">
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-3 w-12" />
            </div>
            <Skeleton className="h-4 w-3/4" />
            <div className="space-y-1.5">
              <Skeleton className="h-3 w-full" />
              <Skeleton className="h-3 w-1/2" />
            </div>
          </div>
        </div>
      </div>
    )
  }

  if (!notification || !categoryTitle || !priorityTitle || !formattedTime) {
    throw new Error("NotificationCard requires durable notification display data unless loading is true.")
  }

  const priorityTone = notificationPriorityTone(notification.priority)
  const unread = !notification.readAt

  return (
    <article
      className={cn(
        "radius-surface group relative overflow-hidden border p-3 transition-colors",
        unread ? "bg-muted/30 hover:bg-muted/50" : "bg-card hover:bg-muted/30"
      )}
    >
      <span aria-hidden="true" className={cn("absolute inset-y-0 left-0 w-1", getStatusToneBgClass(priorityTone))} />
      <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-2">
            <Badge variant="outline" className={cn("h-4 shrink-0 px-1.5 py-0", getStatusToneBadgeClass(priorityTone))}>
              {priorityTitle}
            </Badge>
            <span className={textRole.helperText}>{categoryTitle}</span>
            <time className={cn("ml-auto shrink-0 tabular-nums", textRole.helperText)} dateTime={notification.createdAt}>
              {formattedTime}
            </time>
            {unread ? <span aria-hidden="true" className={cn("radius-round size-1.5", getStatusToneBgClass(priorityTone))} /> : null}
          </div>
          <p className={cn("mt-1 truncate", textRole.bodyStrong)}>{notification.title}</p>
          <p className={cn("mt-1 whitespace-pre-line break-words", textRole.helperText)}>{notification.message}</p>
      </div>
    </article>
  )
}

export function NotificationDrawerLayout({ state }: { state: NotificationDrawerState }) {
  const filterTabs: Array<{
    value: NotificationFilter
    label: string
    icon?: React.ReactNode
  }> = [
    { value: "all", label: state.t("filters.all") },
    { value: "scan", label: state.t("filters.scan"), icon: <semanticIcons.concept.scan className="size-3.5" /> },
    {
      value: "vulnerability",
      label: state.t("filters.vulnerability"),
      icon: <semanticIcons.concept.vulnerability className="size-3.5" />,
    },
    { value: "system", label: state.t("filters.system"), icon: <Info className="size-3.5" /> },
  ]
  const notificationGroups = (["today", "yesterday", "earlier"] as const)
    .map((group) => ({ group, items: state.groupedNotifications[group] }))
    .filter(({ items }) => items.length > 0)
  const showGroupHeadings = notificationGroups.length > 1

  return (
    <Sheet open={state.open} onOpenChange={state.setOpen}>
      <SheetTrigger render={<Button variant="ghost" size="icon-sm" className="group relative" aria-label={state.t("title")} />}>
        <semanticIcons.concept.notification className="h-4 w-4" />
        {state.unreadCount > 0 ? (
          <>
            <span
              aria-hidden="true"
              // Keep this behind the count: the animation conveys unread status without moving the number itself.
              className="pointer-events-none absolute -right-0.5 -top-0.5 h-4 w-4 animate-ping radius-round bg-destructive opacity-75 motion-reduce:animate-none dark:bg-destructive/60"
            />
            <Badge
              variant="destructive"
              className="absolute -right-0.5 -top-0.5 radius-round flex h-4 min-w-4 items-center justify-center border-0 bg-destructive p-0 text-[10px] text-destructive-foreground dark:bg-destructive/60"
            >
              {state.unreadCount > 99 ? "99+" : state.unreadCount}
            </Badge>
          </>
        ) : null}
      </SheetTrigger>

      {/* This lightweight feedback drawer intentionally relies on the shared backdrop and Escape dismissal paths. */}
      <SheetContent showCloseButton={false} className={compactFeedbackDrawerContentClassName}>
        <SheetHeader className="border-b px-4 py-1.5">
          <EdgePanelHeader
            variant="compact"
            // Compact leading icons use a muted container; notification status uses the bare category glyph instead.
            title={(
              <SheetTitle className={cn("flex min-w-0 items-center gap-2", textRole.sectionTitle)}>
                <semanticIcons.concept.notification aria-hidden="true" className="size-4 shrink-0" />
                <span className="truncate">{state.t("title")}</span>
              </SheetTitle>
            )}
            actions={(
              <Button
                type="button"
                variant="link"
                size="content"
                className={textRole.compactPrimary}
                onClick={state.handleMarkAll}
                disabled={state.isMarkingAll || state.unreadCount === 0}
                title={state.t("markAllAsRead")}
              >
                {state.isMarkingAll ? <Spinner className="size-3.5" /> : state.t("markAllRead")}
              </Button>
            )}
          />
        </SheetHeader>

        <div className="grid grid-cols-2 gap-2 border-b px-4 py-2 sm:flex sm:overflow-x-auto">
          {filterTabs.map((tab) => (
            <Button
              type="button"
              variant={state.activeFilter === tab.value ? "default" : "secondary"}
              size="sm"
              key={tab.value}
              onClick={() => state.setActiveFilter(tab.value)}
              className={cn(
                // The selected Button variant has a border; reserve it here so changing filters cannot shift neighbors.
                "radius-pill w-full justify-center border border-transparent whitespace-nowrap sm:w-auto sm:shrink-0",
                state.activeFilter !== tab.value && "text-muted-foreground hover:text-foreground"
              )}
            >
              {tab.icon}
              {tab.label}
              {state.unreadByFilter[tab.value] > 0 ? <span aria-hidden="true" className="radius-round size-1.5 bg-current" /> : null}
            </Button>
          ))}
        </div>

        <ScrollArea className="flex-1">
          <div className="p-3">
            {state.isHistoryLoading && state.allNotifications.length === 0 ? (
              <div className="space-y-2">
                {Array.from({ length: 3 }).map((_, index) => <NotificationCard key={index} loading />)}
              </div>
            ) : state.filteredNotifications.length === 0 ? (
              <div className="flex h-40 flex-col items-center justify-center text-muted-foreground">
                <BellOff className="mb-2 size-10 opacity-50" aria-hidden="true" />
                <p className={textRole.bodySubtle}>{state.t("empty")}</p>
              </div>
            ) : (
              <div className="space-y-4">
                {notificationGroups.map(({ group, items }) => (
                  <section key={group}>
                    {showGroupHeadings ? (
                      <h3 className={cn("sticky top-0 z-10 mb-2 bg-card px-1 py-1 text-muted-foreground", textRole.compactSectionTitle)}>
                        {state.timeGroupLabels[group]}
                      </h3>
                    ) : null}
                    <div className="space-y-2">
                      {items.map((notification) => (
                        <NotificationCard
                          key={notification.name}
                          notification={notification}
                          categoryTitle={state.categoryTitleMap[notification.category]}
                          priorityTitle={state.t(`priorities.${notification.priority}`)}
                          formattedTime={state.formatTimestamp(notification.createdAt)}
                        />
                      ))}
                    </div>
                  </section>
                ))}
                {state.hasNextPage ? (
                  <div className="flex justify-center pt-1">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={state.isFetchingNextPage}
                      onClick={() => void state.fetchNextPage()}
                    >
                      {state.isFetchingNextPage ? <Spinner className="size-3.5" /> : null}
                      {state.t("loadMore")}
                    </Button>
                  </div>
                ) : null}
              </div>
            )}
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  )
}
