import {
  EXTERNALLY_DELIVERABLE_NOTIFICATION_KINDS,
  type ExternalNotificationKind,
} from "@/types/notification.types"
import {
  NOTIFICATION_DESTINATION_PROVIDERS,
  type NotificationDestination,
  type NotificationDestinationListResponse,
  type NotificationDestinationProvider,
  type NotificationLocale,
  type TestNotificationDestinationResponse,
  type UpdateNotificationDestinationRequest,
} from "@/types/notification-settings.types"

const DEFAULT_NOTIFICATION_DESTINATIONS: Record<NotificationDestinationProvider, NotificationDestination> = {
  discord: {
    provider: "discord",
    credential: "https://discord.com/api/webhooks/mock-id/mock-token",
    enabled: true,
    subscriptions: ["scan-failed", "vulnerability-observed"],
    requiresWebhookUpdate: false,
  },
  wecom: {
    provider: "wecom",
    credential: "",
    enabled: false,
    subscriptions: [],
    requiresWebhookUpdate: false,
  },
  feishu: {
    provider: "feishu",
    credential: "",
    enabled: false,
    subscriptions: [],
    requiresWebhookUpdate: false,
  },
}

function createMockNotificationDestinations(): NotificationDestination[] {
  return NOTIFICATION_DESTINATION_PROVIDERS.map((provider) => ({
    ...DEFAULT_NOTIFICATION_DESTINATIONS[provider],
    subscriptions: [...DEFAULT_NOTIFICATION_DESTINATIONS[provider].subscriptions],
  }))
}

export const mockNotificationDestinations = createMockNotificationDestinations()

let mockNotificationLocale: NotificationLocale = { locale: "en" }

function cloneDestination(destination: NotificationDestination): NotificationDestination {
  return {
    ...destination,
    subscriptions: [...destination.subscriptions],
  }
}

function isOfficialWebhookURL(provider: NotificationDestinationProvider, value: string): boolean {
  if (provider === "feishu" && value.trim() !== value) {
    return false
  }

  try {
    const url = new URL(value)
    if (url.protocol !== "https:" || url.username || url.password || url.port || url.hash) {
      return false
    }

    switch (provider) {
      case "discord": {
        const segments = url.pathname.split("/").filter(Boolean)
        return url.hostname === "discord.com" && !url.search && segments.length === 4 && segments[0] === "api" && segments[1] === "webhooks" && Boolean(segments[2]) && Boolean(segments[3])
      }
      case "wecom": {
        const keys = [...url.searchParams.keys()]
        return url.hostname === "qyapi.weixin.qq.com" &&
          url.pathname === "/cgi-bin/webhook/send" &&
          keys.length === 1 &&
          keys[0] === "key" &&
          Boolean(url.searchParams.get("key")?.trim())
      }
      case "feishu": {
        const segments = url.pathname.split("/")
        return url.hostname === "open.feishu.cn" &&
          !url.search &&
          !value.includes("?") &&
          segments.length === 6 &&
          segments[0] === "" &&
          segments[1] === "open-apis" &&
          segments[2] === "bot" &&
          segments[3] === "v2" &&
          segments[4] === "hook" &&
          Boolean(segments[5]) &&
          !value.includes("%") &&
          !value.endsWith("/")
      }
    }
  } catch {
    return false
  }
}

function assertSupportedKinds(kinds: readonly ExternalNotificationKind[]): void {
  if (kinds.some((kind) => !EXTERNALLY_DELIVERABLE_NOTIFICATION_KINDS.includes(kind))) {
    throw new TypeError("Notification destination subscriptions contain an unsupported kind.")
  }
}

export function getMockNotificationDestinations(): NotificationDestinationListResponse {
  return {
    results: mockNotificationDestinations.map(cloneDestination),
    supportedKinds: [...EXTERNALLY_DELIVERABLE_NOTIFICATION_KINDS],
  }
}

export function getMockNotificationDestination(
  provider: NotificationDestinationProvider
): NotificationDestination | undefined {
  const destination = mockNotificationDestinations.find((item) => item.provider === provider)
  return destination ? cloneDestination(destination) : undefined
}

export function updateMockNotificationDestination(
  provider: NotificationDestinationProvider,
  request: UpdateNotificationDestinationRequest
): NotificationDestination | undefined {
  const destination = mockNotificationDestinations.find((item) => item.provider === provider)
  if (!destination) {
    return undefined
  }

  const credential = provider === "feishu" ? request.credential : request.credential.trim()
  assertSupportedKinds(request.subscriptions)
  if (credential && !isOfficialWebhookURL(provider, credential)) {
    throw new TypeError("Notification destination credential must use the provider's official HTTPS webhook URL.")
  }
  if (request.enabled && !credential) {
    throw new TypeError("An enabled notification destination requires a credential.")
  }
  if (request.enabled && request.subscriptions.length === 0) {
    throw new TypeError("An enabled notification destination requires a subscription.")
  }

  destination.credential = credential
  destination.enabled = request.enabled
  destination.subscriptions = [...new Set(request.subscriptions)]
  destination.requiresWebhookUpdate = false
  return cloneDestination(destination)
}

export function getMockNotificationTestDeliveryUnavailableResult(): TestNotificationDestinationResponse {
  return { result: "unavailable" }
}

export function getMockNotificationLocale(): NotificationLocale {
  return { ...mockNotificationLocale }
}

export function updateMockNotificationLocale(locale: NotificationLocale["locale"]): NotificationLocale {
  if (locale !== "zh" && locale !== "en") {
    throw new TypeError("Notification locale is unsupported.")
  }
  mockNotificationLocale = { locale }
  return getMockNotificationLocale()
}

export function resetMockNotificationSettings(): void {
  mockNotificationDestinations.splice(
    0,
    mockNotificationDestinations.length,
    ...createMockNotificationDestinations(),
  )
  mockNotificationLocale = { locale: "en" }
}
