"use client"

import type { ReactNode } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import { PageHeader } from "@/components/common/page-header"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"

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
    <div data-slot="scan-configuration-workspace" className="flex h-full min-h-0 min-w-0 flex-1 flex-col gap-4 overflow-auto py-4 md:gap-6 md:py-6">
      <PageHeader code="SCN-02" title={t("title")} description={t("description")} />
      <div className="px-4 lg:px-6">
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
