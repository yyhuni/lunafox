import { useQuery } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { NotificationSettingsService } from '@/services/notification-settings.service'
import type { UpdateNotificationSettingsRequest } from '@/types/notification-settings.types'

export const notificationSettingsKeys = {
  settings: ['notification-settings'] as const,
}

export function useNotificationSettings() {
  return useQuery({
    queryKey: notificationSettingsKeys.settings,
    queryFn: () => NotificationSettingsService.getSettings(),
  })
}

export function useUpdateNotificationSettings() {
  return useResourceMutation({
    mutationFn: (data: UpdateNotificationSettingsRequest) =>
      NotificationSettingsService.updateSettings(data),
    invalidate: [{ queryKey: notificationSettingsKeys.settings }],
    onSuccess: ({ toast }) => {
      toast.success('toast.notification.settings.success')
    },
    errorFallbackKey: 'toast.notification.settings.error',
  })
}
