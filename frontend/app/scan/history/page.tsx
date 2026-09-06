import { getTranslations } from "next-intl/server"
import { PageHeader } from "@/components/common/page-header"
import { ScanHistoryPageRefresh } from "@/components/scan/history/scan-history-page-refresh"
import { ScanHistoryRetentionSummary } from "@/components/scan/history/scan-history-retention-summary"
import { ScanHistoryStatCards } from "@/components/scan/history/scan-history-stat-cards"
import { ScanHistoryList } from "@/components/scan/history/scan-history-list"

/**
 * Scan history page
 * Displays historical records of all scan tasks
 */
export default async function ScanHistoryPage() {
  const tScan = await getTranslations("scan")

  return (
    <div className="@container/main flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="SCH-HIS"
        title={tScan("history.title")}
        description={tScan("history.description")}
        descriptionSupplement={<ScanHistoryRetentionSummary />}
        descriptionAction={<ScanHistoryPageRefresh />}
      />

      {/* Statistics strip */}
      <div className="px-4 lg:px-6">
        <ScanHistoryStatCards />
      </div>

      {/* Scan history list */}
      <div className="px-4 lg:px-6">
        <ScanHistoryList />
      </div>
    </div>
  )
}
