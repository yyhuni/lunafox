"use client"

import { useQuery } from "@tanstack/react-query"
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { getNucleiTemplateTree, getNucleiTemplateContent, refreshNucleiTemplates, saveNucleiTemplate, uploadNucleiTemplate } from "@/services/nuclei.service"
import type { NucleiTemplateTreeNode, NucleiTemplateContent, UploadNucleiTemplatePayload, SaveNucleiTemplatePayload } from "@/types/nuclei.types"

export function useNucleiTemplateTree() {
  return useQuery<NucleiTemplateTreeNode[]>({
    queryKey: ["nuclei", "templates", "tree"],
    queryFn: () => getNucleiTemplateTree(),
  })
}

export function useNucleiTemplateContent(path: string | null) {
  return useQuery<NucleiTemplateContent>({
    queryKey: ["nuclei", "templates", "content", path],
    queryFn: () => getNucleiTemplateContent(path as string),
    enabled: !!path,
  })
}

export function useRefreshNucleiTemplates() {
  return useResourceMutation({
    mutationFn: () => refreshNucleiTemplates(),
    loadingToast: {
      key: 'toast.nucleiTemplate.refresh.loading',
      params: {},
      id: 'refresh-nuclei-templates',
    },
    invalidate: [{ queryKey: ["nuclei", "templates", "tree"] }],
    onSuccess: ({ toast }) => {
      toast.success('toast.nucleiTemplate.refresh.success')
    },
    errorFallbackKey: 'toast.nucleiTemplate.refresh.error',
  })
}

export function useUploadNucleiTemplate() {
  return useResourceMutation<void, UploadNucleiTemplatePayload>({
    mutationFn: (payload) => uploadNucleiTemplate(payload),
    loadingToast: {
      key: 'common.status.uploading',
      params: {},
      id: 'upload-nuclei-template',
    },
    invalidate: [{ queryKey: ["nuclei", "templates", "tree"] }],
    onSuccess: ({ toast }) => {
      toast.success('toast.nucleiTemplate.upload.success')
    },
    errorFallbackKey: 'toast.nucleiTemplate.upload.error',
  })
}

export function useSaveNucleiTemplate() {
  return useResourceMutation<void, SaveNucleiTemplatePayload>({
    mutationFn: (payload) => saveNucleiTemplate(payload),
    loadingToast: {
      key: 'common.actions.saving',
      params: {},
      id: 'save-nuclei-template',
    },
    invalidate: [
      ({ variables }) => ({ queryKey: ["nuclei", "templates", "content", variables.path] }),
    ],
    onSuccess: ({ toast }) => {
      toast.success('toast.nucleiTemplate.save.success')
    },
    errorFallbackKey: 'toast.nucleiTemplate.save.error',
  })
}
