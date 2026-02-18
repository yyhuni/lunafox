import type {
  NotificationSettings,
  GetNotificationSettingsResponse,
  UpdateNotificationSettingsResponse,
} from '@/types/notification-settings.types'

export const mockNotificationSettings: NotificationSettings = {
  discord: {
    enabled: true,
    webhookUrl: 'https://discord.com/api/webhooks/1234567890/abcdefghijklmnop',
  },
  wecom: {
    enabled: false,
    webhookUrl: '',
  },
  categories: {
    scan: true,
    vulnerability: true,
    asset: true,
    system: false,
  },
}

export function getMockNotificationSettings(): GetNotificationSettingsResponse {
  return mockNotificationSettings
}

export function updateMockNotificationSettings(
  settings: NotificationSettings
): UpdateNotificationSettingsResponse {
  // Simulate update settings
  Object.assign(mockNotificationSettings, settings)
  
  return {
    message: 'Notification settings updated successfully',
    discord: mockNotificationSettings.discord,
    wecom: mockNotificationSettings.wecom,
    categories: mockNotificationSettings.categories,
  }
}
