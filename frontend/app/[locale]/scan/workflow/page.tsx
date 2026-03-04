"use client"

import { lazyPage } from "@/components/common/lazy-page"

const ScanWorkflowPageContent = lazyPage(
  () => import("@/components/scan/workflow/scan-workflow-page")
)

export default function ScanWorkflowPage() {
  return <ScanWorkflowPageContent />
}
