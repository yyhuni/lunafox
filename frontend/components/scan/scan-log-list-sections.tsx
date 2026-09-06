"use client"

import { useTranslations } from "next-intl"

import { RawLogViewer } from "@/components/shared/visualization/raw-log-viewer"
import { TerminalLogCopyAllButton } from "@/components/shared/visualization/terminal-log-copy-all-button"

export function ScanLogListLoadingState() {
  const t = useTranslations("scan.history.overview")

  return (
    <div className="bg-card flex h-full items-center justify-center text-muted-foreground">
      {t("loadingLogs")}
    </div>
  )
}

export function ScanLogListEmptyState() {
  const t = useTranslations("scan.history.overview")

  return (
    <div className="bg-card flex h-full items-center justify-center text-muted-foreground">
      {t("noLogs")}
    </div>
  )
}

export function ScanLogListContent({ content }: { content: string }) {
  const t = useTranslations("scan.history.overview")

  return (
    <div className="h-full">
      <RawLogViewer
        content={content}
        topRightAction={(
          <TerminalLogCopyAllButton
            value={content}
            copyLabel={t("copyAllLogs")}
            copiedLabel={t("copied")}
            toastId="scan-log-copy-all"
          />
        )}
      />
    </div>
  )
}
