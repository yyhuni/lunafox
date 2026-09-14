import { getTranslations } from "next-intl/server"
import { PageHeader } from "@/components/common/page-header"
import { ScanHistoryPageRefresh } from "@/components/scan/history/scan-history-page-refresh"
import { ScanHistoryRetentionSummary } from "@/components/scan/history/scan-history-retention-summary"
import { ScanHistoryStatCards } from "@/components/scan/history/scan-history-stat-cards"
import { ScanHistoryList } from "@/components/scan/history/scan-history-list"
import {
  COMPACT_CONTAINER_PAGE_SHELL_CLASS,
  COMPACT_CONTENT_GUTTER_CLASS,
} from "@/components/shared/layout/page-shell-density"

/**
 * Scan history page
 * Displays historical records of all scan tasks
 */
export default async function ScanHistoryPage() {
  const tScan = await getTranslations("scan")

  return (
    <div className={COMPACT_CONTAINER_PAGE_SHELL_CLASS}>
      <PageHeader
        code="SCH-HIS"
        title={tScan("history.title")}
        description={tScan("history.description")}
        descriptionSupplement={<ScanHistoryRetentionSummary />}
        descriptionAction={<ScanHistoryPageRefresh />}
      />

      {/* Statistics strip */}
      <div className={COMPACT_CONTENT_GUTTER_CLASS}>
        <ScanHistoryStatCards />
      </div>

      {/* Scan history list */}
      <div className={COMPACT_CONTENT_GUTTER_CLASS}>
        <ScanHistoryList />
      </div>
    </div>
  )
}
