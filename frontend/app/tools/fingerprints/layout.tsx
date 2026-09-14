"use client"

import type { ReactNode } from "react"
import { useTranslations } from "next-intl"

import { PageHeader } from "@/components/common/page-header"
import { Info } from "@/components/icons"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { COMPACT_PAGE_SHELL_CLASS } from "@/components/shared/layout/page-shell-density"

function FingerprintDescriptionHelp({ label, description }: { label: string; description: string }) {
  return (
    <TooltipProvider delay={100}>
      <Tooltip>
        <TooltipTrigger
          render={(
            <span
              className="inline-flex shrink-0 cursor-help text-muted-foreground"
              role="img"
              tabIndex={0}
              aria-label={label}
            />
          )}
        >
          <Info className="size-4" aria-hidden="true" />
        </TooltipTrigger>
        <TooltipContent side="bottom" sideOffset={8} align="center" className="max-w-sm whitespace-normal text-left">
          {description}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

export default function FingerprintsLayout({ children }: { children: ReactNode }) {
  const t = useTranslations("tools.fingerprints")
  return (
    <div className={COMPACT_PAGE_SHELL_CLASS}>
      <PageHeader
        code="FPR-01"
        title={t("title")}
        description={t("pageDescription")}
        descriptionSupplement={(
          <FingerprintDescriptionHelp
            label={t("fingerprinthubHelpLabel")}
            description={t("fingerprinthubHelpText")}
          />
        )}
      />
      {children}
    </div>
  )
}
