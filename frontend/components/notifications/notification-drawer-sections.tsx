"use client"

import React from "react"
import { Bell, AlertTriangle, Activity, Info, Server, BellOff, Loader2 } from "@/components/icons"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import { cn } from "@/lib/utils"
import { SEVERITY_CARD_STYLES, SEVERITY_ICON_BG } from "@/lib/severity-config"

import type { NotificationType, NotificationSeverity, Notification } from "@/types/notification.types"
import type { NotificationDrawerState } from "./notification-drawer-state"

/** Notification skeleton screen */
function NotificationSkeleton() {
  return (
    <div className="space-y-2">
      {[1, 2, 3].map((i) => (
        <div key={i} className="rounded-md border p-3">
          <div className="flex items-start gap-2.5">
            <Skeleton className="h-5 w-5 rounded-full" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-3 w-full" />
              <Skeleton className="h-3 w-1/2" />
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}

const SEVERITY_ICON_CLASS_MAP: Record<NotificationSeverity, string> = {
  critical: "text-[var(--error)]",
  high: "text-[var(--warning)]",
  medium: "text-[var(--warning)]",
  low: "text-[var(--muted-foreground)]",
}

const SEVERITY_CARD_CLASS_MAP: Record<NotificationSeverity, string> = {
  critical: SEVERITY_CARD_STYLES.critical,
  high: SEVERITY_CARD_STYLES.high,
  medium: SEVERITY_CARD_STYLES.medium,
  low: SEVERITY_CARD_STYLES.low,
}

function getNotificationIcon(type: NotificationType, severity?: NotificationSeverity) {
  const severityClass = severity ? SEVERITY_ICON_CLASS_MAP[severity] : "text-gray-500"

  if (type === "vulnerability") {
    return <AlertTriangle className={cn("h-5 w-5", severityClass)} />
  }
  if (type === "scan") {
    return <Activity className={cn("h-5 w-5", severityClass)} />
  }
  if (type === "asset") {
    return <Server className={cn("h-5 w-5", severityClass)} />
  }
  return <Info className={cn("h-5 w-5", severityClass)} />
}

function getNotificationCardClasses(severity?: NotificationSeverity) {
  if (!severity) {
    return "border-border bg-card hover:bg-accent/50"
  }
  return cn("border-border", SEVERITY_CARD_CLASS_MAP[severity] ?? "")
}

function NotificationCard({
  notification,
  categoryTitle,
}: {
  notification: Notification
  categoryTitle: string
}) {
  return (
    <div
      key={notification.id}
      className={cn(
        "group relative rounded-lg border p-3 transition-[background-color,border-color,box-shadow,transform] duration-200 overflow-hidden",
        "hover:shadow-sm hover:scale-[1.01]",
        getNotificationCardClasses(notification.severity)
      )}
    >
      {notification.unread ? (
        <span className="absolute right-2 bottom-2 h-2 w-2 rounded-full bg-primary" aria-hidden />
      ) : null}
      <div className="flex items-start gap-3">
        <div
          className={cn(
            "mt-0.5 p-1.5 rounded-full shrink-0",
            notification.severity === "critical" ? SEVERITY_ICON_BG.critical : "",
            notification.severity === "high" ? SEVERITY_ICON_BG.high : "",
            notification.severity === "medium" ? SEVERITY_ICON_BG.medium : "",
            !notification.severity || notification.severity === "low" ? SEVERITY_ICON_BG.info : ""
          )}
        >
          {getNotificationIcon(notification.type, notification.severity)}
        </div>
        <div className="flex-1 min-w-0 overflow-hidden">
          <div className="flex items-center justify-between gap-2 mb-1">
            <span className="text-xs font-medium text-muted-foreground">{categoryTitle}</span>
            <span className="text-xs text-muted-foreground tabular-nums shrink-0">{notification.time}</span>
          </div>
          <p className="text-sm font-semibold leading-snug truncate">{notification.title}</p>
          <p className="text-xs text-muted-foreground mt-1 whitespace-pre-line break-all line-clamp-4">
            {notification.description}
          </p>
        </div>
      </div>
    </div>
  )
}

export function NotificationDrawerLayout({
  state,
}: {
  state: NotificationDrawerState
}) {
  const filterTabs: { value: NotificationType | "all"; label: string; icon?: React.ReactNode }[] = [
    { value: "all", label: state.t("filters.all") },
    { value: "scan", label: state.t("filters.scan"), icon: <Activity className="h-3 w-3" /> },
    {
      value: "vulnerability",
      label: state.t("filters.vulnerability"),
      icon: <AlertTriangle className="h-3 w-3" />,
    },
    { value: "asset", label: state.t("filters.asset"), icon: <Server className="h-3 w-3" /> },
    { value: "system", label: state.t("filters.system"), icon: <Info className="h-3 w-3" /> },
  ]

  return (
    <Sheet open={state.open} onOpenChange={state.setOpen}>
      <SheetTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="relative group"
          aria-label={state.t("title")}
        >
          <Bell className="h-5 w-5" />
          {state.unreadCount > 0 ? (
            <Badge
              variant="destructive"
              className="absolute -top-0.5 -right-0.5 h-4 min-w-4 rounded-full p-0 text-[10px] flex items-center justify-center"
            >
              {state.unreadCount > 99 ? "99+" : state.unreadCount}
            </Badge>
          ) : null}
          <span className="sr-only">{state.t("title")}</span>
        </Button>
      </SheetTrigger>
      <SheetContent className="w-full sm:max-w-[440px] p-0 flex flex-col gap-0">
        <SheetHeader className="border-b px-4 py-1.5">
          <div className="flex items-center justify-between gap-2">
            <SheetTitle className="text-sm font-semibold">{state.t("title")}</SheetTitle>
            <div className="flex items-center gap-2">
              <button type="button"
                onClick={state.handleMarkAll}
                disabled={state.isMarkingAll || state.allNotifications.length === 0}
                className="text-xs text-primary hover:text-primary/80 hover:underline underline-offset-2 disabled:opacity-50 disabled:cursor-not-allowed disabled:no-underline transition-colors"
                title={state.t("markAllAsRead")}
              >
                {state.isMarkingAll ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  state.t("markAllRead")
                )}
              </button>
            </div>
          </div>
        </SheetHeader>

        <div className="flex gap-1 px-3 py-1.5 border-b overflow-x-auto">
          {filterTabs.map((tab) => (
            <button type="button"
              key={tab.value}
              onClick={() => state.setActiveFilter(tab.value)}
              className={cn(
                "inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium transition-[background-color,color,box-shadow] whitespace-nowrap",
                state.activeFilter === tab.value
                  ? "bg-primary text-primary-foreground shadow-sm"
                  : "bg-muted text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              )}
            >
              {tab.icon}
              {tab.label}
              {state.unreadByType[tab.value] > 0 ? (
                <span
                  className={cn(
                    "ml-1 h-1.5 w-1.5 rounded-full",
                    state.activeFilter === tab.value ? "bg-primary-foreground" : "bg-primary"
                  )}
                />
              ) : null}
            </button>
          ))}
        </div>

        <ScrollArea className="flex-1">
          <div className="p-3">
            {state.isHistoryLoading && state.allNotifications.length === 0 ? (
              <NotificationSkeleton />
            ) : state.filteredNotifications.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-40 text-muted-foreground">
                <BellOff className="h-10 w-10 mb-2 opacity-50" />
                <p className="text-sm">{state.t("empty")}</p>
              </div>
            ) : (
              <div className="space-y-4">
                {(["today", "yesterday", "earlier"] as const).map((group) => {
                  const items = state.groupedNotifications[group]
                  if (items.length === 0) return null

                  return (
                    <div key={group}>
                      <h3 className="sticky top-0 z-10 text-xs font-medium text-muted-foreground mb-2 px-1 py-1 backdrop-blur bg-background/90">
                        {state.timeGroupLabels[group]}
                      </h3>
                      <div className="space-y-2">
                        {items.map((notification) => (
                          <NotificationCard
                            key={notification.id}
                            notification={notification}
                            categoryTitle={state.categoryTitleMap[notification.type]}
                          />
                        ))}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  )
}
