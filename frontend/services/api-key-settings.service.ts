import { api } from '@/lib/api-client'
import type { ApiKeySettings, ApiKeySettingsUpdateRequest } from '@/types/api-key-settings.types'

export class ApiKeySettingsService {
  static async getSettings(): Promise<ApiKeySettings> {
    const res = await api.get<ApiKeySettings>('/settings/apiKeys/')
    return res.data
  }

  static async updateSettings(data: ApiKeySettingsUpdateRequest): Promise<ApiKeySettings> {
    const res = await api.patch<ApiKeySettings>('/settings/apiKeys/', data)
    return res.data
  }
}
