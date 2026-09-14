"use client"

import type { ReactNode } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import { PageHeader } from "@/components/common/page-header"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  COMPACT_CONTENT_GUTTER_CLASS,
  COMPACT_FULL_PAGE_SHELL_CLASS,
} from "@/components/shared/layout/page-shell-density"

export type ScanConfigurationTab = "workflows" | "engines"

const scanConfigurationPaths = {
  workflows: "/scan/config/workflows/",
  engines: "/scan/config/engines/",
} as const

export function ScanConfigurationWorkspace({
  activeTab,
  children,
}: {
  activeTab: ScanConfigurationTab
  children: ReactNode
}) {
  const t = useTranslations("scan.configuration")

  return (
    <div data-slot="scan-configuration-workspace" className={`${COMPACT_FULL_PAGE_SHELL_CLASS} min-w-0 overflow-auto`}>
      <PageHeader code="SCN-02" title={t("title")} description={t("description")} />
      <div className={COMPACT_CONTENT_GUTTER_CLASS}>
        <Tabs value={activeTab} aria-label={t("tabsLabel")}>
          <TabsList variant="content">
            <TabsTrigger value="workflows" variant="content" render={<Link href={scanConfigurationPaths.workflows} />}>
              {t("workflows")}
            </TabsTrigger>
            <TabsTrigger value="engines" variant="content" render={<Link href={scanConfigurationPaths.engines} />}>
              {t("engines")}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </div>
      {children}
    </div>
  )
}
