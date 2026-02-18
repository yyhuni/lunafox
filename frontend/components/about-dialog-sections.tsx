"use client"

import {
  IconRefresh,
  IconExternalLink,
  IconBrandGithub,
  IconMessageReport,
  IconBook,
  IconFileText,
  IconCheck,
  IconArrowUp,
} from "@/components/icons"
import { DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { Badge } from "@/components/ui/badge"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface AboutDialogHeaderProps {
  t: TranslationFn
}

export function AboutDialogHeader({ t }: AboutDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{t("title")}</DialogTitle>
    </DialogHeader>
  )
}

interface AboutDialogBrandingProps {
  t: TranslationFn
  logoSrc: string
}

export function AboutDialogBranding({ t, logoSrc }: AboutDialogBrandingProps) {
  return (
    <div className="flex flex-col items-center py-4">
      <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10 mb-3">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img src={logoSrc} alt="LunaFox Logo" className="h-12 w-12" width={48} height={48} />
      </div>
      <h2 className="text-xl font-semibold">{t("productName")}</h2>
      <p className="text-sm text-muted-foreground">{t("description")}</p>
    </div>
  )
}

interface AboutDialogVersionInfoProps {
  t: TranslationFn
  currentVersion: string
  latestVersion?: string
  hasUpdate?: boolean
  checkError: string | null
  isChecking: boolean
  releaseUrl?: string
  showLatest: boolean
  onCheckUpdate: () => void
}

export function AboutDialogVersionInfo({
  t,
  currentVersion,
  latestVersion,
  hasUpdate,
  checkError,
  isChecking,
  releaseUrl,
  showLatest,
  onCheckUpdate,
}: AboutDialogVersionInfoProps) {
  return (
    <div className="rounded-lg border p-4 space-y-3">
      <div className="flex items-center justify-between">
        <span className="text-sm text-muted-foreground">{t("currentVersion")}</span>
        <span className="font-mono text-sm">{currentVersion}</span>
      </div>

      {showLatest && (
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">{t("latestVersion")}</span>
          <div className="flex items-center gap-2">
            <span className="font-mono text-sm">{latestVersion}</span>
            {hasUpdate ? (
              <Badge variant="default" className="gap-1">
                <IconArrowUp className="h-3 w-3" />
                {t("updateAvailable")}
              </Badge>
            ) : (
              <Badge variant="secondary" className="gap-1">
                <IconCheck className="h-3 w-3" />
                {t("upToDate")}
              </Badge>
            )}
          </div>
        </div>
      )}

      {checkError && (
        <p className="text-sm text-destructive">{checkError}</p>
      )}

      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          className="flex-1"
          onClick={onCheckUpdate}
          disabled={isChecking}
        >
          <IconRefresh className={`h-4 w-4 mr-2 ${isChecking ? "animate-spin" : ""}`} />
          {isChecking ? t("checking") : t("checkUpdate")}
        </Button>

        {hasUpdate && releaseUrl && (
          <Button
            variant="default"
            size="sm"
            className="flex-1"
            asChild
          >
            <a href={releaseUrl} target="_blank" rel="noopener noreferrer">
              <IconExternalLink className="h-4 w-4 mr-2" />
              {t("viewRelease")}
            </a>
          </Button>
        )}
      </div>

      {hasUpdate && (
        <div className="rounded-md bg-muted p-3 text-sm text-muted-foreground">
          <p>{t("updateHint")}</p>
          <code className="mt-1 block rounded bg-background px-2 py-1 font-mono text-xs">
            sudo ./update.sh
          </code>
        </div>
      )}
    </div>
  )
}

interface AboutDialogLinksProps {
  t: TranslationFn
}

export function AboutDialogLinks({ t }: AboutDialogLinksProps) {
  return (
    <div className="grid grid-cols-2 gap-2">
      <Button variant="ghost" size="sm" className="justify-start" asChild>
        <a href="https://github.com/yyhuni/xingrin" target="_blank" rel="noopener noreferrer">
          <IconBrandGithub className="h-4 w-4 mr-2" />
          GitHub
        </a>
      </Button>
      <Button variant="ghost" size="sm" className="justify-start" asChild>
        <a href="https://github.com/yyhuni/xingrin/releases" target="_blank" rel="noopener noreferrer">
          <IconFileText className="h-4 w-4 mr-2" />
          {t("changelog")}
        </a>
      </Button>
      <Button variant="ghost" size="sm" className="justify-start" asChild>
        <a href="https://github.com/yyhuni/xingrin/issues" target="_blank" rel="noopener noreferrer">
          <IconMessageReport className="h-4 w-4 mr-2" />
          {t("feedback")}
        </a>
      </Button>
      <Button variant="ghost" size="sm" className="justify-start" asChild>
        <a href="https://github.com/yyhuni/xingrin#readme" target="_blank" rel="noopener noreferrer">
          <IconBook className="h-4 w-4 mr-2" />
          {t("docs")}
        </a>
      </Button>
    </div>
  )
}

interface AboutDialogFooterProps {
  t: TranslationFn
}

export function AboutDialogFooter({ t }: AboutDialogFooterProps) {
  return (
    <p className="text-center text-xs text-muted-foreground">
      © 2026 {t("productName")} · GPL-3.0
    </p>
  )
}

export { Separator }
