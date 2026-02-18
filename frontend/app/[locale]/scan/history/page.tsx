"use client"

import dynamic from "next/dynamic"
import { useTranslations } from "next-intl"
import { Skeleton } from "@/components/ui/skeleton"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { PageHeader } from "@/components/common/page-header"

const ScanHistoryStatCards = dynamic(
  () => import("@/components/scan/history/scan-history-stat-cards").then((mod) => mod.ScanHistoryStatCards),
  {
    ssr: false,
    loading: () => (
      <div className="grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className="h-32" />
        ))}
      </div>
    ),
  }
)

const ScanHistoryList = dynamic(
  () => import("@/components/scan/history/scan-history-list").then((mod) => mod.ScanHistoryList),
  {
    ssr: false,
    loading: () => <DataTableSkeleton rows={6} columns={6} withPadding />,
  }
)

/**
 * Scan history page
 * Displays historical records of all scan tasks
 */
export default function ScanHistoryPage() {
  const t = useTranslations("scan.history")

  return (
    <div className="@container/main flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="SCN-01"
        title={t("title")}
        description={t("description")}
      />

      {/* Statistics cards */}
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
