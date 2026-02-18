"use client"

import { lazyPage } from "@/components/common/lazy-page"
import { useParams } from "next/navigation"

const ScanOverview = lazyPage(
  () => import("@/components/scan/history/scan-overview").then((m) => ({ default: m.ScanOverview }))
)

/**
 * Scan overview page
 * Displays scan statistics and summary information
 */
export default function ScanOverviewPage() {
  const { id } = useParams<{ id: string }>()
  const scanId = Number(id)

  return (
    <div className="flex-1 flex flex-col min-h-0 px-4 lg:px-6">
      <ScanOverview scanId={scanId} />
    </div>
  )
}
