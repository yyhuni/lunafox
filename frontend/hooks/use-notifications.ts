/**
 * Notification-related React Query hooks
 */

import { useQuery } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { NotificationService } from '@/services/notification.service'
import type {
  GetNotificationsRequest,
} from '@/types/notification.types'

// Query Keys
const notificationKeyBase = createResourceKeys("notifications", {
  list: (params?: GetNotificationsRequest) => params,
})

export const notificationKeys = {
  ...notificationKeyBase,
  unreadCount: () => [...notificationKeyBase.all, 'unread-count'] as const,
}

/**
 * Get notification list
 */
export function useNotifications(params?: GetNotificationsRequest) {
  return useQuery({
    queryKey: notificationKeys.list(params),
    queryFn: () => NotificationService.getNotifications(params),
  })
}

/**
 * Get unread notification count
 */
export function useUnreadCount() {
  return useQuery({
    queryKey: notificationKeys.unreadCount(),
    queryFn: () => NotificationService.getUnreadCount(),
    refetchInterval: 30000, // Auto refresh every 30 seconds
  })
}

/**
 * Mark all notifications as read
 */
export function useMarkAllAsRead() {
  return useResourceMutation<Awaited<ReturnType<typeof NotificationService.markAllAsRead>>, void>({
    mutationFn: () => NotificationService.markAllAsRead(),
    invalidate: [{ queryKey: notificationKeys.all }],
    onError: async () => {
    },
  })
}
