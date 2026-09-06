import * as React from "react"
import { useLocale, useTranslations } from "next-intl"

import { useSyncNotificationLocale } from "@/hooks/use-notification-settings"
import type { Locale } from "@/i18n/config"
import { useNotificationSSE } from "@/hooks/use-notification-sse"
import {
  useMarkAllNotificationsRead,
  useNotificationPages,
  useUnreadCount,
} from "@/hooks/use-notifications"
import type {
  NotificationCategory,
  NotificationInboxItem,
} from "@/types/notification.types"

export type NotificationFilter = NotificationCategory | "all"

function getTimeGroup(dateValue: string): "today" | "yesterday" | "earlier" {
  const date = new Date(dateValue)
  if (Number.isNaN(date.getTime())) {
    return "earlier"
  }
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today.getTime() - 24 * 60 * 60 * 1_000)

  if (date >= today) return "today"
  if (date >= yesterday) return "yesterday"
  return "earlier"
}

export function formatNotificationTimestamp(value: string, locale: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date)
}

export function useNotificationDrawerState() {
  const t = useTranslations("notificationDrawer")
  const locale = useLocale() as Locale
  const [open, setOpen] = React.useState(false)
  const [activeFilter, setActiveFilter] = React.useState<NotificationFilter>("all")

  useSyncNotificationLocale(locale)
  useNotificationSSE()

  const inbox = useNotificationPages()
  const unread = useUnreadCount()
  const markAllRead = useMarkAllNotificationsRead()

  const allNotifications = React.useMemo<NotificationInboxItem[]>(
    () => inbox.data?.pages.flatMap((page) => page.results) ?? [],
    [inbox.data]
  )
  const unreadCount = unread.data?.unreadCount ?? 0

  const unreadByFilter = React.useMemo<Record<NotificationFilter, number>>(() => {
    const counts: Record<NotificationFilter, number> = {
      all: 0,
      scan: 0,
      vulnerability: 0,
      system: 0,
    }

    for (const notification of allNotifications) {
      if (notification.readAt) {
        continue
      }
      counts.all += 1
      counts[notification.category] += 1
    }

    return counts
  }, [allNotifications])

  const filteredNotifications = React.useMemo(
    () => activeFilter === "all"
      ? allNotifications
      : allNotifications.filter((notification) => notification.category === activeFilter),
    [activeFilter, allNotifications]
  )

  const groupedNotifications = React.useMemo(() => {
    const groups: Record<"today" | "yesterday" | "earlier", NotificationInboxItem[]> = {
      today: [],
      yesterday: [],
      earlier: [],
    }
    for (const notification of filteredNotifications) {
      groups[getTimeGroup(notification.createdAt)].push(notification)
    }
    return groups
  }, [filteredNotifications])

  const categoryTitleMap = React.useMemo<Record<NotificationCategory, string>>(
    () => ({
      scan: t("categories.scan"),
      vulnerability: t("categories.vulnerability"),
      system: t("categories.system"),
    }),
    [t]
  )
  const timeGroupLabels = React.useMemo(
    () => ({
      today: t("timeGroups.today"),
      yesterday: t("timeGroups.yesterday"),
      earlier: t("timeGroups.earlier"),
    }),
    [t]
  )

  const handleMarkAll = React.useCallback(() => {
    if (unreadCount === 0 || markAllRead.isPending) {
      return
    }
    markAllRead.mutate()
  }, [markAllRead, unreadCount])

  return {
    t,
    locale,
    open,
    setOpen,
    activeFilter,
    setActiveFilter,
    isHistoryLoading: inbox.isLoading,
    isFetchingNextPage: inbox.isFetchingNextPage,
    hasNextPage: inbox.hasNextPage,
    fetchNextPage: inbox.fetchNextPage,
    isMarkingAll: markAllRead.isPending,
    allNotifications,
    unreadCount,
    unreadByFilter,
    filteredNotifications,
    groupedNotifications,
    categoryTitleMap,
    timeGroupLabels,
    formatTimestamp: (value: string) => formatNotificationTimestamp(value, locale),
    handleMarkAll,
  }
}

export type NotificationDrawerState = ReturnType<typeof useNotificationDrawerState>
