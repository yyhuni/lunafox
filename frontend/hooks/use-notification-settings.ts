import * as React from "react"
import { useMutation, useQuery } from "@tanstack/react-query"

import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { NotificationSettingsService } from "@/services/notification-settings.service"
import type {
  NotificationDestination,
  NotificationDestinationProvider,
  NotificationLocale,
  TestNotificationDestinationRequest,
  TestNotificationDestinationResponse,
  UpdateNotificationDestinationRequest,
} from "@/types/notification-settings.types"

const notificationSettingsKeyBase = createResourceKeys("notification-settings")

export const notificationSettingsKeys = {
  ...notificationSettingsKeyBase,
  destinations: () => [...notificationSettingsKeyBase.all, "destinations"] as const,
  locale: () => [...notificationSettingsKeyBase.all, "locale"] as const,
}

export function useNotificationDestinations() {
  return useQuery({
    queryKey: notificationSettingsKeys.destinations(),
    queryFn: () => NotificationSettingsService.getDestinations(),
  })
}

export function useUpdateNotificationDestination() {
  return useMutation<
    NotificationDestination,
    unknown,
    { provider: NotificationDestinationProvider; data: UpdateNotificationDestinationRequest }
  >({
    mutationFn: ({ provider, data }) => NotificationSettingsService.updateDestination(provider, data),
    retry: false,
  })
}

export function useTestNotificationDestination() {
  return useMutation<
    TestNotificationDestinationResponse,
    unknown,
    { provider: NotificationDestinationProvider; data: TestNotificationDestinationRequest }
  >({
    mutationFn: ({ provider, data }) => NotificationSettingsService.testDestination(provider, data),
    retry: false,
  })
}

export function useUpdateNotificationLocale() {
  return useMutation({
    mutationFn: (locale: NotificationLocale["locale"]) =>
      NotificationSettingsService.updateLocaleForPageChange(locale),
    retry: false,
  })
}

export function useSyncNotificationLocale(locale: NotificationLocale["locale"]) {
  const mutation = useMutation({
    mutationFn: (pageLocale: NotificationLocale["locale"]) =>
      NotificationSettingsService.updateLocale(pageLocale),
    retry: false,
  })
  const { mutate } = mutation
  const lastSynchronizedLocale = React.useRef<NotificationLocale["locale"] | null>(null)

  React.useEffect(() => {
    if (lastSynchronizedLocale.current === locale) {
      return
    }
    if (!NotificationSettingsService.shouldSynchronizePageLocale(locale)) {
      return
    }
    lastSynchronizedLocale.current = locale
    mutate(locale)
  }, [locale, mutate])

  return mutation
}
