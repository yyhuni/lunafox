import { useInfiniteQuery, useQuery } from "@tanstack/react-query"

import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { NotificationService } from "@/services/notification.service"
import type {
  ListNotificationsRequest,
  NotificationInboxItem,
} from "@/types/notification.types"

const notificationKeyBase = createResourceKeys("notification-inbox", {
  list: (params: ListNotificationsRequest) => params,
})

export const notificationKeys = {
  ...notificationKeyBase,
  unreadCount: () => [...notificationKeyBase.all, "unreadCount"] as const,
}

export function useNotifications(
  params: ListNotificationsRequest = {},
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: notificationKeys.list(params),
    queryFn: () => NotificationService.list(params),
    enabled: options?.enabled ?? true,
  })
}

export function useUnreadCount(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: notificationKeys.unreadCount(),
    queryFn: () => NotificationService.getUnreadCount(),
    enabled: options?.enabled ?? true,
  })
}

export function useNotificationPages(pageSize = 50) {
  const params = { pageSize }
  return useInfiniteQuery({
    queryKey: notificationKeys.list(params),
    initialPageParam: "",
    queryFn: ({ pageParam }) => NotificationService.list({
      pageSize,
      ...(pageParam ? { pageToken: pageParam } : {}),
    }),
    getNextPageParam: (page) => page.nextPageToken || undefined,
  })
}

export function useMarkNotificationRead() {
  return useResourceMutation<NotificationInboxItem, string>({
    mutationFn: (name) => NotificationService.markRead(name),
    retry: false,
    loadingToast: {
      key: "toast.notification.markRead.loading",
      id: (name) => `notification-mark-read-${name}`,
    },
    invalidate: [{ queryKey: notificationKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success("toast.notification.markRead.success")
    },
    errorFallbackKey: "toast.notification.markRead.error",
  })
}

export function useMarkAllNotificationsRead() {
  return useResourceMutation<void, void>({
    mutationFn: () => NotificationService.markAllRead(),
    retry: false,
    loadingToast: {
      key: "toast.notification.markAll.loading",
      id: "notification-mark-all-read",
    },
    invalidate: [{ queryKey: notificationKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success("toast.notification.markAll.success")
    },
    // Command result can be unknown after a transport failure. Refetch the
    // durable inbox rather than replaying mark-all and crossing its high-water boundary.
    onError: async ({ queryClient, toast }) => {
      await queryClient.refetchQueries({ queryKey: notificationKeys.all, type: "active" })
      toast.error("toast.notification.markAll.error")
    },
  })
}
