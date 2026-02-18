"use client"

import { useQuery } from "@tanstack/react-query"
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { NucleiGitService } from "@/services/nuclei-git.service"
import type { UpdateNucleiGitSettingsRequest } from "@/types/nuclei-git.types"

export const nucleiGitKeys = {
  settings: ["nuclei", "git", "settings"] as const,
}

export function useNucleiGitSettings() {
  return useQuery({
    queryKey: nucleiGitKeys.settings,
    queryFn: () => NucleiGitService.getSettings(),
  })
}

export function useUpdateNucleiGitSettings() {
  return useResourceMutation({
    mutationFn: (data: UpdateNucleiGitSettingsRequest) => NucleiGitService.updateSettings(data),
    invalidate: [{ queryKey: nucleiGitKeys.settings }],
    onSuccess: ({ toast }) => {
      toast.success('toast.nucleiGit.settings.success')
    },
    errorFallbackKey: 'toast.nucleiGit.settings.error',
  })
}
