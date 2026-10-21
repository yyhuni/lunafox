"use client"

import { useState } from "react"
import { useLocale, useTranslations } from "next-intl"
import { semanticIcons } from "@/components/icons"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { textRole } from "@/lib/typography"
import type { AgentLicense } from "@/types/agent-license.types"
import { AgentLicenseDialog } from "./agent-license-dialog"
import { formatAgentLicenseExpiry } from "./agent-license-date"

export function AgentLicensePanel({ license, nodeCount, loading = false, visible = true }: { license: AgentLicense; nodeCount: number; loading?: boolean; visible?: boolean }) {
  const t = useTranslations("settings.agents.activation")
  const locale = useLocale()
  const [open, setOpen] = useState(false)
  const active = license.state === "active"
  const expired = license.state === "expired"
  // Keep the dialog mounted when a successful summary refresh hides the banner.
  return (<>
    {visible && <section aria-label={t("sectionLabel")} aria-hidden={loading || undefined} className="flex flex-col gap-2 border-y py-2 @2xl/main:flex-row @2xl/main:items-center @2xl/main:justify-between">
      <div className="flex min-w-0 items-start gap-3">
        <semanticIcons.concept.apiKey aria-hidden="true" className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
        <div className="min-w-0 space-y-1.5">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className={textRole.sectionTitle}>{t(expired ? "expiredHeading" : active ? "activeHeading" : "basicHeading")}</h2>
            <Badge size="compact" variant={expired ? "warning" : active ? "success" : "outline"}>{t(expired ? "expiredBadge" : active ? "activeBadge" : "basicBadge")}</Badge>
          </div>
          <p className={textRole.bodySubtle}>{expired ? t("expiredDescription", { count: nodeCount, limit: license.agentLimit }) : active
            ? t("activeDescription", { limit: license.agentLimit, months: license.months!, date: formatAgentLicenseExpiry(license.expiresAt!, locale) }) : t("basicDescription", { limit: license.agentLimit })}</p>
        </div>
      </div>
      {loading ? <ActionSkeleton size="sm" className="self-start @2xl/main:self-auto" /> : <Button variant={active ? "outline" : "default"} size="sm" onClick={() => setOpen(true)} className="self-start @2xl/main:self-auto">
        <semanticIcons.concept.apiKey aria-hidden="true" />{t(expired ? "renew" : active ? "manage" : "open")}
      </Button>}
    </section>}
    {open && !loading && <AgentLicenseDialog license={license} onClose={() => setOpen(false)} />}
  </>
  )
}
