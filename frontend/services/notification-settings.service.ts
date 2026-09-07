import { api } from "@/lib/api-client"
import type {
  NotificationDestination,
  NotificationDestinationListResponse,
  NotificationDestinationProvider,
  NotificationLocale,
  TestNotificationDestinationRequest,
  TestNotificationDestinationResponse,
  UpdateNotificationDestinationRequest,
} from "@/types/notification-settings.types"

const localePath = "/users/current/notificationLocale"
const destinationsPath = "/settings/notificationDestinations"

// The drawer sync and language menu can share one browser session. Serializing
// locale writes preserves invocation order so a stale shell sync cannot finish
// after an explicit language choice and overwrite the newer page locale.
let notificationLocaleUpdateQueue = Promise.resolve()
let pendingPageLocaleChange: NotificationLocale["locale"] | null = null

export class NotificationSettingsService {
  static async getDestinations(): Promise<NotificationDestinationListResponse> {
    const res = await api.get<NotificationDestinationListResponse>(destinationsPath)
    return res.data
  }

  static async getDestination(provider: NotificationDestinationProvider): Promise<NotificationDestination> {
    const res = await api.get<NotificationDestination>(`${destinationsPath}/${provider}`)
    return res.data
  }

  static async updateDestination(
    provider: NotificationDestinationProvider,
    data: UpdateNotificationDestinationRequest
  ): Promise<NotificationDestination> {
    const res = await api.patch<NotificationDestination>(`${destinationsPath}/${provider}`, data)
    return res.data
  }

  static async testDestination(
    provider: NotificationDestinationProvider,
    data: TestNotificationDestinationRequest
  ): Promise<TestNotificationDestinationResponse> {
    const res = await api.post<TestNotificationDestinationResponse>(
      `${destinationsPath}/${provider}:testDelivery`,
      data,
    )
    return res.data
  }

  static async getLocale(): Promise<NotificationLocale> {
    const res = await api.get<NotificationLocale>(localePath)
    return res.data
  }

  static async updateLocale(locale: NotificationLocale["locale"]): Promise<NotificationLocale> {
    const request = notificationLocaleUpdateQueue.then(async () => {
      const res = await api.patch<NotificationLocale>(localePath, { locale })
      return res.data
    })
    notificationLocaleUpdateQueue = request.then(() => undefined, () => undefined)
    return request
  }

  static async updateLocaleForPageChange(locale: NotificationLocale["locale"]): Promise<NotificationLocale> {
    pendingPageLocaleChange = locale
    try {
      return await this.updateLocale(locale)
    } catch (error) {
      if (pendingPageLocaleChange === locale) {
        pendingPageLocaleChange = null
      }
      throw error
    }
  }

  static shouldSynchronizePageLocale(locale: NotificationLocale["locale"]): boolean {
    if (pendingPageLocaleChange !== null && pendingPageLocaleChange !== locale) {
      return false
    }
    if (pendingPageLocaleChange === locale) {
      pendingPageLocaleChange = null
    }
    return true
  }
}
