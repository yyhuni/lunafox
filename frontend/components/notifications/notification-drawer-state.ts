import React from "react"
import { useLocale, useTranslations } from "next-intl"

import { transformBackendNotification, useNotificationSSE } from "@/hooks/use-notification-sse"
import { useMarkAllAsRead, useNotifications } from "@/hooks/use-notifications"

import type { Notification, NotificationType } from "@/types/notification.types"

function getTimeGroup(dateStr?: string): "today" | "yesterday" | "earlier" {
  if (!dateStr) return "earlier"
  const date = new Date(dateStr)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today.getTime() - 24 * 60 * 60 * 1000)

  if (date >= today) return "today"
  if (date >= yesterday) return "yesterday"
  return "earlier"
}

export function useNotificationDrawerState() {
  const t = useTranslations("notificationDrawer")
  const locale = useLocale()
  const tTime = useTranslations("common.time")
  const [open, setOpen] = React.useState(false)
  const [activeFilter, setActiveFilter] = React.useState<NotificationType | "all">("all")

  const queryParams = React.useMemo(() => ({ pageSize: 100 }), [])
  const { data: notificationResponse, isLoading: isHistoryLoading } = useNotifications(queryParams)
  const { mutate: markAllAsRead, isPending: isMarkingAll } = useMarkAllAsRead()

  const transformOptions = React.useMemo(
    () => ({
      locale,
      timeLabels: {
        justNow: tTime("justNow"),
        minutesAgo: (count: number) => tTime("minutesAgo", { count }),
        hoursAgo: (count: number) => tTime("hoursAgo", { count }),
      },
    }),
    [locale, tTime]
  )

  const { notifications: sseNotifications, markNotificationsAsRead } = useNotificationSSE(transformOptions)

  const [historyNotifications, setHistoryNotifications] = React.useState<Notification[]>([])

  React.useEffect(() => {
    if (!notificationResponse?.results) return
    const backendNotifications = notificationResponse.results ?? []
    setHistoryNotifications(
      backendNotifications.map((notification) => transformBackendNotification(notification, transformOptions))
    )
  }, [notificationResponse, transformOptions])

  const allNotifications = React.useMemo(() => {
    const seen = new Set<number>()
    const merged: Notification[] = []

    for (const notification of sseNotifications) {
      if (!seen.has(notification.id)) {
        merged.push(notification)
        seen.add(notification.id)
      }
    }

    for (const notification of historyNotifications) {
      if (!seen.has(notification.id)) {
        merged.push(notification)
        seen.add(notification.id)
      }
    }

    return merged.toSorted((a, b) => {
      const aTime = a.createdAt ? new Date(a.createdAt).getTime() : 0
      const bTime = b.createdAt ? new Date(b.createdAt).getTime() : 0
      return bTime - aTime
    })
  }, [historyNotifications, sseNotifications])

  const unreadCount = allNotifications.filter((n) => n.unread).length

  const unreadByType = React.useMemo<Record<NotificationType | "all", number>>(() => {
    const counts: Record<NotificationType | "all", number> = {
      all: 0,
      scan: 0,
      vulnerability: 0,
      asset: 0,
      system: 0,
    }

    allNotifications.forEach((notification) => {
      if (!notification.unread) return
      counts.all += 1
      if (counts[notification.type] !== undefined) {
        counts[notification.type] += 1
      }
    })

    return counts
  }, [allNotifications])

  const filteredNotifications = React.useMemo(() => {
    if (activeFilter === "all") return allNotifications
    return allNotifications.filter((n) => n.type === activeFilter)
  }, [allNotifications, activeFilter])

  const handleMarkAll = React.useCallback(() => {
    if (allNotifications.length === 0 || isMarkingAll) return
    markAllAsRead(undefined, {
      onSuccess: () => {
        setHistoryNotifications((prev) =>
          prev.map((notification) => ({ ...notification, unread: false }))
        )
        markNotificationsAsRead()
      },
    })
  }, [allNotifications.length, isMarkingAll, markAllAsRead, markNotificationsAsRead])

  const groupedNotifications = React.useMemo(() => {
    const groups: Record<"today" | "yesterday" | "earlier", Notification[]> = {
      today: [],
      yesterday: [],
      earlier: [],
    }

    filteredNotifications.forEach((notification) => {
      const group = getTimeGroup(notification.createdAt)
      groups[group].push(notification)
    })

    return groups
  }, [filteredNotifications])

  const categoryTitleMap = React.useMemo<Record<NotificationType, string>>(
    () => ({
      scan: t("categories.scan"),
      vulnerability: t("categories.vulnerability"),
      asset: t("categories.asset"),
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

  return {
    t,
    open,
    setOpen,
    activeFilter,
    setActiveFilter,
    isHistoryLoading,
    isMarkingAll,
    allNotifications,
    unreadCount,
    unreadByType,
    filteredNotifications,
    groupedNotifications,
    categoryTitleMap,
    timeGroupLabels,
    handleMarkAll,
  }
}

export type NotificationDrawerState = ReturnType<typeof useNotificationDrawerState>
