"use client"

import { useTranslations } from "next-intl"

import { Zap as IconZap } from "@/components/icons"
import { QuickScanDialog } from "@/components/scan/quick-scan-dialog"
import { Button } from "@/components/ui/button"

export function QuickScanHeaderTrigger() {
  const t = useTranslations("navigation")

  return (
    <QuickScanDialog
      trigger={(
        <Button
          data-slot="quick-scan-trigger"
          variant="ghost"
          size="icon-sm"
          aria-label={t("quickScan")}
        >
          <IconZap className="h-4 w-4" />
          <span className="sr-only">{t("quickScan")}</span>
        </Button>
      )}
    />
  )
}
