"use client"

import * as React from "react"
import { useTranslations } from "next-intl"

import {
  VisualSplitLogin,
  type VisualSplitLoginPreview,
  type VisualSplitLoginTranslations,
} from "@/components/auth/visual-split-login"
import { PageHeader } from "@/components/common/page-header"
import { RefreshCw, Save, Upload } from "@/components/icons"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { HiddenReadinessRouteBoundary } from "@/components/shared/loading/hidden-readiness-route-boundary"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  useLoginVisualSettings,
  useLoginVisualPreview,
  usePublishLoginVisual,
  useRestoreLoginVisual,
  useUploadLoginVisual,
} from "@/hooks/use-login-visual"
import { normalizeError } from "@/lib/errors/normalize-error"
import { textRole } from "@/lib/typography"
import type { LoginVisualKind, LoginVisualMedia } from "@/types/login-visual.types"

const loginPreviewReferenceWidth = 1440
const loginPreviewReferenceHeight = 900

type PreviewAsset = {
  kind: LoginVisualKind
  source: string
}

function LoginPreviewControl({
  disabled,
  labels,
  onPick,
  scale,
}: {
  disabled: boolean
  labels: LoginPreviewLabels
  onPick: (file: File) => void
  scale: number
}) {
  const inputRef = React.useRef<HTMLInputElement>(null)

  return (
    <>
      <input
        accept="image/jpeg,image/png,image/webp,video/mp4,video/webm"
        disabled={disabled}
        hidden
        id="login-visual-file"
        onChange={(event) => {
          const file = event.target.files?.[0]
          if (file) onPick(file)
          event.target.value = ""
        }}
        ref={inputRef}
        tabIndex={-1}
        type="file"
      />
      <div
        aria-label={labels.replace}
        aria-disabled={disabled}
        className="group absolute inset-0 z-30 flex cursor-pointer items-center justify-center outline-none aria-disabled:cursor-not-allowed"
        onClick={(event) => {
          if (disabled) return
          event.preventDefault()
          inputRef.current?.click()
        }}
        onKeyDown={(event) => {
          if (disabled || (event.key !== "Enter" && event.key !== " ")) return
          event.preventDefault()
          inputRef.current?.click()
        }}
        role="button"
        tabIndex={disabled ? -1 : 0}
      >
        <span aria-hidden="true" className="absolute inset-0 bg-background/0 backdrop-blur-none transition duration-200 group-hover:bg-background/70 group-hover:backdrop-blur-sm group-focus-visible:bg-background/70 group-focus-visible:backdrop-blur-sm" />
        <span
          className="relative flex flex-col items-center gap-3 opacity-0 transition-opacity duration-200 group-hover:opacity-100 group-focus-visible:opacity-100"
          style={{ transform: `scale(${1 / scale})` }}
        >
          <span className="flex size-14 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg ring-4 ring-primary/20"><Upload className="size-6" /></span>
        </span>
      </div>
    </>
  )
}

type LoginPreviewLabels = {
  replace: string
}

function useLoginPreviewScale() {
  const frameRef = React.useRef<HTMLDivElement>(null)
  const [scale, setScale] = React.useState(1)

  React.useLayoutEffect(() => {
    const frame = frameRef.current
    if (!frame) return

    const updateScale = () => setScale(frame.clientWidth / loginPreviewReferenceWidth)
    updateScale()

    const observer = new ResizeObserver(updateScale)
    observer.observe(frame)
    return () => observer.disconnect()
  }, [])

  return { frameRef, scale }
}

function previewVisualFor(asset: PreviewAsset | null): VisualSplitLoginPreview["visual"] {
  if (!asset) return { kind: "builtIn" }
  return { kind: asset.kind, mediaUrl: asset.source }
}

function LoginPreview({
  asset,
  disabled,
  labels,
  onPick,
  translations,
}: {
  asset: PreviewAsset | null
  disabled: boolean
  labels: LoginPreviewLabels
  onPick: (file: File) => void
  translations: VisualSplitLoginTranslations
}) {
  const { frameRef, scale } = useLoginPreviewScale()

  return (
    <section aria-label={translations.ariaLabel} className="w-full">
      <div
        className="relative mx-auto aspect-[8/5] overflow-hidden border bg-background shadow-xs"
        ref={frameRef}
        style={{ width: "min(100%, calc((100svh - 18rem) * 8 / 5))" }}
      >
        {/* Keep the exact desktop page geometry, then scale the complete page into the settings frame. */}
        <div
          className="absolute left-0 top-0 origin-top-left"
          style={{
            height: loginPreviewReferenceHeight,
            transform: `scale(${scale})`,
            width: loginPreviewReferenceWidth,
          }}
        >
          <VisualSplitLogin
            className="h-full"
            onLogin={async () => undefined}
            preview={{
              visual: previewVisualFor(asset),
              visualControl: <LoginPreviewControl disabled={disabled} labels={labels} onPick={onPick} scale={scale} />,
            }}
            translations={translations}
          />
        </div>
      </div>
    </section>
  )
}

function sourceFor(media: LoginVisualMedia | undefined, previewURL: string | undefined): PreviewAsset | null {
  if (!media || !previewURL) return null
  return { kind: media.kind, source: previewURL }
}

function LoginVisualSettingsPageLayout({
  asset,
  disabled,
  labels,
  onPick,
  publication,
  translations,
  mutationError,
  hiddenReadiness = false,
}: {
  asset: PreviewAsset | null
  disabled: boolean
  labels: LoginPreviewLabels
  onPick: (file: File) => void
  publication: React.ReactNode
  translations: VisualSplitLoginTranslations
  mutationError?: React.ReactNode
  hiddenReadiness?: boolean
}) {
  const t = useTranslations("pages.settings.loginVisual")

  return (
    <div data-loading-hidden-readiness={hiddenReadiness ? "true" : undefined} className="flex h-full min-h-0 flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6">
      <header {...getLoadingStructureSlotAttributes("login-visual-header")}>
        <PageHeader code="SET-06" description={t("description")} title={t("title")} />
      </header>
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto px-4 [scrollbar-gutter:stable] lg:px-6">
        <div className="flex w-full flex-1 flex-col justify-center">
          <div {...getLoadingStructureSlotAttributes("login-visual-preview")}>
            <p className={`mb-4 flex items-center gap-1.5 border-b border-border pb-3 ${textRole.helperText}`}>
              <Upload aria-hidden="true" className="size-3.5" />
              {t("replaceHelp")}
            </p>
            <LoginPreview asset={asset} disabled={disabled} labels={labels} onPick={onPick} translations={translations} />
          </div>
          <div {...getLoadingStructureSlotAttributes("login-visual-publication-controls")} className="mt-4 flex flex-wrap items-center justify-between gap-3 border-t pt-4">
            {publication}
          </div>
          {mutationError ? <p className="mt-3 text-sm text-destructive" role="alert">{mutationError}</p> : null}
        </div>
      </div>
    </div>
  )
}

function LoginVisualSettingsLoadingState({
  deferInitialSkeleton = false,
  labels,
  translations,
}: {
  deferInitialSkeleton?: boolean
  labels: LoginPreviewLabels
  translations: VisualSplitLoginTranslations
}) {
  const t = useTranslations("pages.settings.loginVisual")

  return (
    <LoginVisualSettingsPageLayout
      asset={null}
      disabled
      hiddenReadiness={deferInitialSkeleton}
      labels={labels}
      onPick={() => undefined}
      publication={(
        <>
          <div className="flex items-center gap-2">
            <Badge variant="success">{t("published")}</Badge>
            <span className={textRole.helperText}>{t("publishedHint")}</span>
          </div>
          <div className="flex items-center gap-2">
            <Button disabled type="button" variant="ghost"><RefreshCw />{t("restore")}</Button>
            <Button disabled type="button"><Save />{t("publish")}</Button>
          </div>
        </>
      )}
      translations={translations}
    />
  )
}

function LoginVisualSettingsContent({
  deferInitialSkeleton = false,
  labels,
  onReady,
  translations,
}: {
  deferInitialSkeleton?: boolean
  labels: LoginPreviewLabels
  onReady?: () => void
  translations: VisualSplitLoginTranslations
}) {
  const t = useTranslations("pages.settings.loginVisual")
  const settings = useLoginVisualSettings()
  const upload = useUploadLoginVisual()
  const publish = usePublishLoginVisual()
  const restore = useRestoreLoginVisual()
  const [temporary, setTemporary] = React.useState<PreviewAsset | null>(null)
  const temporaryURL = React.useRef<string | null>(null)
  const [awaitingStoredPreview, setAwaitingStoredPreview] = React.useState(false)

  React.useEffect(() => () => { if (temporaryURL.current) URL.revokeObjectURL(temporaryURL.current) }, [])

  const isInitialLoading = settings.isPending && !settings.data
  React.useEffect(() => {
    if (!isInitialLoading) onReady?.()
  }, [isInitialLoading, onReady])

  const acceptedMedia = settings.data?.draft ?? settings.data?.published
  const acceptedPreview = useLoginVisualPreview(acceptedMedia, settings.data?.previewUrl)
  const acceptedAsset = sourceFor(acceptedMedia, acceptedPreview.source)
  const previewAsset = temporary ?? acceptedAsset
  const mutationError = upload.error ?? publish.error ?? restore.error
  const busy = upload.isPending || publish.isPending || restore.isPending

  // Keep the local candidate visible until the authenticated Blob is renderable.
  React.useEffect(() => {
    if (!awaitingStoredPreview || !acceptedPreview.source || acceptedPreview.isFetching || acceptedPreview.isError) return
    if (temporaryURL.current) URL.revokeObjectURL(temporaryURL.current)
    temporaryURL.current = null
    setTemporary(null)
    setAwaitingStoredPreview(false)
  }, [acceptedPreview.isError, acceptedPreview.isFetching, acceptedPreview.source, awaitingStoredPreview])

  const selectFile = (file: File) => {
    if (temporaryURL.current) URL.revokeObjectURL(temporaryURL.current)
    const source = URL.createObjectURL(file)
    temporaryURL.current = source
    setTemporary({ kind: file.type.startsWith("video/") ? "video" : "image", source })
    upload.mutate(file, {
      onSuccess: () => {
        setAwaitingStoredPreview(true)
      },
      onError: () => {
        if (temporaryURL.current) URL.revokeObjectURL(temporaryURL.current)
        temporaryURL.current = null
        setTemporary(null)
        setAwaitingStoredPreview(false)
      },
    })
  }

  if (isInitialLoading) {
    return <LoginVisualSettingsLoadingState deferInitialSkeleton={deferInitialSkeleton} labels={labels} translations={translations} />
  }

  if (settings.error) return <AppErrorState actionHref="/overview/" error={normalizeError(settings.error)} onRetry={() => void settings.refetch()} variant="page" />

  return (
    <LoginVisualSettingsPageLayout
      asset={previewAsset}
      disabled={busy}
      labels={labels}
      mutationError={mutationError ? t("saveError") : undefined}
      onPick={selectFile}
      publication={(
        <>
          <div className="flex items-center gap-2" aria-live="polite">
            <Badge variant={settings.data?.draft ? "warning" : "success"}>{settings.data?.draft ? t("draft") : t("published")}</Badge>
            <span className={textRole.helperText}>{settings.data?.draft ? t("draftHint") : t("publishedHint")}</span>
          </div>
          <div className="flex items-center gap-2">
            <Button disabled={busy} onClick={() => restore.mutate()} type="button" variant="ghost"><RefreshCw />{t("restore")}</Button>
            <Button disabled={busy || !settings.data?.draft} loading={publish.isPending} loadingLabel={t("publish")} onClick={() => publish.mutate()} type="button"><Save />{t("publish")}</Button>
          </div>
        </>
      )}
      translations={translations}
    />
  )
}

export function LoginVisualSettingsWorkspace() {
  const t = useTranslations("pages.settings.loginVisual")
  const tVisualLogin = useTranslations("auth.visualLogin")
  const labels: LoginPreviewLabels = {
    replace: t("replace"),
  }
  const translations: VisualSplitLoginTranslations = {
    ariaLabel: t("preview"),
    consoleLabel: tVisualLogin("consoleLabel"),
    usernameLabel: tVisualLogin("usernameLabel"),
    usernamePlaceholder: tVisualLogin("usernamePlaceholder"),
    passwordLabel: tVisualLogin("passwordLabel"),
    passwordPlaceholder: tVisualLogin("passwordPlaceholder"),
    submit: tVisualLogin("submit"),
    accessHelp: tVisualLogin("accessHelp"),
  }

  return (
    <HiddenReadinessRouteBoundary
      owner="login-visual-settings-page-route"
      layer="workspace"
      intent="data"
      skeleton={<LoginVisualSettingsLoadingState labels={labels} translations={translations} />}
      className="flex min-h-0 flex-1 flex-col"
      skeletonClassName="flex min-h-0 flex-1 flex-col"
      contentClassName="flex min-h-0 flex-1 flex-col"
    >
      {({ onReady, deferInitialSkeleton }) => (
        <LoginVisualSettingsContent
          deferInitialSkeleton={deferInitialSkeleton}
          labels={labels}
          onReady={onReady}
          translations={translations}
        />
      )}
    </HiddenReadinessRouteBoundary>
  )
}
