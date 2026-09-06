"use client"

import * as React from "react"
import { useTranslations } from "next-intl"

import { PageHeader } from "@/components/common/page-header"
import { PageRefreshStatusButton } from "@/components/common/page-refresh-status-button"
import { semanticIcons } from "@/components/icons"
import { QuickScanDialog } from "@/components/scan/quick-scan-dialog"
import { AddTargetDialog } from "@/components/target/add-target-dialog"
import { Button } from "@/components/ui/button"

const AddIcon = semanticIcons.action.add
const RunIcon = semanticIcons.action.run

interface OverviewPageHeaderProps {
  isRefreshing: boolean
  lastRefreshedAt: Date | null
  onRefresh: () => void | Promise<void>
}

export function OverviewPageHeader({
  isRefreshing,
  lastRefreshedAt,
  onRefresh,
}: OverviewPageHeaderProps) {
  const t = useTranslations("overview.pageHeader")
  const [isAddTargetOpen, setIsAddTargetOpen] = React.useState(false)

  return (
    <>
      <PageHeader
        code="overview"
        title={t("systemOverview")}
        middle={(
          <PageRefreshStatusButton
            isRefreshing={isRefreshing}
            lastRefreshedAt={lastRefreshedAt}
            onRefresh={onRefresh}
            refreshLabel={t("refresh")}
            updatedAtLabel={t("updatedAtLabel")}
            updatedAtUnavailable={t("updatedAtUnavailable")}
          />
        )}
        action={(
          <div className="flex shrink-0 items-center gap-2">
            <PageRefreshStatusButton
              display="icon-only"
              isRefreshing={isRefreshing}
              lastRefreshedAt={lastRefreshedAt}
              onRefresh={onRefresh}
              refreshLabel={t("refresh")}
              updatedAtLabel={t("updatedAtLabel")}
              updatedAtUnavailable={t("updatedAtUnavailable")}
              className="text-muted-foreground hover:text-foreground xl:hidden"
            />
            <Button variant="outline" size="default" onClick={() => setIsAddTargetOpen(true)}>
              <AddIcon aria-hidden="true" />
              <span className="hidden sm:inline">{t("addTarget")}</span>
              <span className="sr-only sm:hidden">{t("addTarget")}</span>
            </Button>
            <QuickScanDialog
              trigger={(
                <Button variant="primary" size="default">
                  <RunIcon aria-hidden="true" />
                  <span className="hidden sm:inline">{t("quickScan")}</span>
                  <span className="sr-only sm:hidden">{t("quickScan")}</span>
                </Button>
              )}
            />
          </div>
        )}
      />
      <AddTargetDialog
        open={isAddTargetOpen}
        onOpenChange={setIsAddTargetOpen}
        prefetchEnabled
      />
    </>
  )
}
