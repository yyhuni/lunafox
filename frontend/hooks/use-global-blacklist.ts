import { useQuery } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { toast } from 'sonner'
import { useTranslations } from 'next-intl'
import {
  getGlobalBlacklist,
  updateGlobalBlacklist,
  type GlobalBlacklistResponse,
  type UpdateGlobalBlacklistRequest,
} from '@/services/global-blacklist.service'

const QUERY_KEY = ['global-blacklist']

/**
 * Hook to fetch global blacklist
 */
export function useGlobalBlacklist() {
  return useQuery<GlobalBlacklistResponse>({
    queryKey: QUERY_KEY,
    queryFn: getGlobalBlacklist,
  })
}

/**
 * Hook to update global blacklist
 */
export function useUpdateGlobalBlacklist() {
  const t = useTranslations('pages.settings.blacklist')

  return useResourceMutation({
    mutationFn: (data: UpdateGlobalBlacklistRequest) => updateGlobalBlacklist(data),
    invalidate: [{ queryKey: QUERY_KEY }],
    onSuccess: () => {
      toast.success(t('toast.saveSuccess'))
    },
    onError: () => {
      toast.error(t('toast.saveError'))
    },
  })
}
