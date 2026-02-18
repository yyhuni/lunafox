import { api } from '@/lib/api-client'
import type { ApiKeySettings } from '@/types/api-key-settings.types'

export class ApiKeySettingsService {
  static async getSettings(): Promise<ApiKeySettings> {
    const res = await api.get<ApiKeySettings>('/settings/api-keys/')
    return res.data
  }

  static async updateSettings(data: Partial<ApiKeySettings>): Promise<ApiKeySettings> {
    const res = await api.put<ApiKeySettings>('/settings/api-keys/', data)
    return res.data
  }
}
