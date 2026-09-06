import * as React from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { isLoginVisualUnlocked, unlockLoginVisual, useLoginVisualUnlocked } from "@/lib/login-visual-unlock"
import { LoginVisualService } from "@/services/login-visual.service"
import type { LoginVisualMedia } from "@/types/login-visual.types"

export const loginVisualKeys = {
  all: ["loginVisual"] as const,
  settings: () => [...loginVisualKeys.all, "settings"] as const,
  discoverability: () => [...loginVisualKeys.all, "discoverability"] as const,
  publicCurrent: () => [...loginVisualKeys.all, "publicCurrent"] as const,
  previews: () => [...loginVisualKeys.all, "preview"] as const,
  preview: (media: LoginVisualMedia | undefined, previewURL: string | undefined) => [
    ...loginVisualKeys.previews(),
    previewURL ?? null,
    media?.kind ?? null,
    media?.contentType ?? null,
    media?.sizeBytes ?? null,
    media?.durationMs ?? null,
  ] as const,
}

export function useLoginVisualDiscoverability() {
  const localUnlocked = useLoginVisualUnlocked()
  const query = useQuery({
    queryKey: loginVisualKeys.discoverability(),
    queryFn: LoginVisualService.checkDiscoverability,
    retry: false,
    staleTime: 5 * 60_000,
  })

  React.useEffect(() => {
    if (query.data?.unlocked && !isLoginVisualUnlocked()) {
      unlockLoginVisual()
    }
  }, [query.data?.unlocked])

  const unlocked = query.isPending || query.isError
    ? localUnlocked
    : query.data?.unlocked === true

  return { ...query, unlocked }
}

export function useUnlockLoginVisualDiscoverability() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: LoginVisualService.unlockDiscoverability,
    onSuccess: (result) => {
      queryClient.setQueryData(loginVisualKeys.discoverability(), result)
      if (result.unlocked && !isLoginVisualUnlocked()) {
        unlockLoginVisual()
      }
    },
  })
}

export function useLoginVisualSettings() {
  return useQuery({ queryKey: loginVisualKeys.settings(), queryFn: LoginVisualService.getSettings })
}

export function useLoginVisualPreview(media: LoginVisualMedia | undefined, previewURL: string | undefined) {
  const inlinePreview = previewURL?.startsWith("data:") ? previewURL : undefined
  const query = useQuery({
    queryKey: loginVisualKeys.preview(media, previewURL),
    queryFn: () => LoginVisualService.getPreview(previewURL!),
    enabled: Boolean(media && previewURL && !inlinePreview),
    retry: false,
    staleTime: 60_000,
  })
  const [objectURL, setObjectURL] = React.useState<string | undefined>()

  React.useEffect(() => {
    if (inlinePreview) {
      setObjectURL(undefined)
      return
    }
    if (!query.data) {
      setObjectURL(undefined)
      return
    }

    const nextObjectURL = URL.createObjectURL(query.data)
    setObjectURL(nextObjectURL)
    return () => URL.revokeObjectURL(nextObjectURL)
  }, [inlinePreview, query.data])

  return { ...query, source: inlinePreview ?? objectURL }
}

export function useUploadLoginVisual() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (file: File) => LoginVisualService.upload(file),
    onSuccess: (settings) => {
      queryClient.setQueryData(loginVisualKeys.settings(), settings)
      void queryClient.invalidateQueries({ queryKey: loginVisualKeys.previews() })
    },
  })
}

export function usePublishLoginVisual() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: LoginVisualService.publish,
    onSuccess: (settings) => {
      queryClient.setQueryData(loginVisualKeys.settings(), settings)
      void queryClient.invalidateQueries({ queryKey: loginVisualKeys.publicCurrent() })
    },
  })
}

export function useRestoreLoginVisual() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: LoginVisualService.restoreDefault,
    onSuccess: (settings) => {
      queryClient.setQueryData(loginVisualKeys.settings(), settings)
      void queryClient.invalidateQueries({ queryKey: loginVisualKeys.publicCurrent() })
    },
  })
}

export function usePublicLoginVisual() {
  return useQuery({
    queryKey: loginVisualKeys.publicCurrent(),
    queryFn: LoginVisualService.getPublicCurrent,
    retry: false,
    staleTime: 60_000,
  })
}
