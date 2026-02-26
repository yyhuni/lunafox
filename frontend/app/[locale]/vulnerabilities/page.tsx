"use client"

import React from "react"
import dynamic from "next/dynamic"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"

const VulnerabilitiesVerticalView = dynamic(
  () => import("@/components/vulnerabilities/vulnerabilities-vertical-view").then((mod) => mod.VulnerabilitiesVerticalView),
  {
    ssr: false,
    loading: () => <DataTableSkeleton toolbarButtonCount={2} rows={6} columns={6} />,
  }
)

/**
 * All vulnerabilities page
 * Displays all vulnerabilities with vertical split layout
 */
export default function VulnerabilitiesPage() {
  return (
    <div className="flex h-full flex-col p-4 md:p-6">
      <VulnerabilitiesVerticalView />
    </div>
  )
}
