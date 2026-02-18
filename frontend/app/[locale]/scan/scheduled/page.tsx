"use client"

import { lazyPage } from "@/components/common/lazy-page"

const ScheduledScanPageContent = lazyPage(
  () => import("@/components/scan/scheduled/scheduled-scan-page")
)

export default function ScheduledScanPage() {
  return <ScheduledScanPageContent />
}
