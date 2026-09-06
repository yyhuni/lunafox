import { useQuery } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { ApiKeySettingsService } from '@/services/api-key-settings.service'
import type { ApiKeySettingsUpdateRequest } from '@/types/api-key-settings.types'

export const apiKeySettingsKeys = {
  settings: ['api-key-settings'] as const,
}

export function useApiKeySettings() {
  return useQuery({
    queryKey: apiKeySettingsKeys.settings,
    queryFn: () => ApiKeySettingsService.getSettings(),
  })
}

export function useUpdateApiKeySettings() {
  return useResourceMutation({
    mutationFn: (data: ApiKeySettingsUpdateRequest) =>
      ApiKeySettingsService.updateSettings(data),
    loadingToast: {
      key: 'common.status.updating',
      params: {},
      id: 'update-api-key-settings',
    },
    invalidate: [{ queryKey: apiKeySettingsKeys.settings }],
    onSuccess: ({ toast }) => {
      toast.success('toast.apiKeys.settings.success')
    },
    errorFallbackKey: 'toast.apiKeys.settings.error',
  })
}
