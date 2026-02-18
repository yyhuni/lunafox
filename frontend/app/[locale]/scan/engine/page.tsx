"use client"

import { lazyPage } from "@/components/common/lazy-page"

const ScanEnginePageContent = lazyPage(
  () => import("@/components/scan/engine/scan-engine-page")
)

export default function ScanEnginePage() {
  return <ScanEnginePageContent />
}
