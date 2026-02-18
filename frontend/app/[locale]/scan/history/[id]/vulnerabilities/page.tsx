"use client"

import React from "react"
import dynamic from "next/dynamic"
import { useParams } from "next/navigation"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"

const VulnerabilitiesDetailView = dynamic(
  () => import("@/components/vulnerabilities/vulnerabilities-detail-view").then((mod) => mod.VulnerabilitiesDetailView),
  {
    ssr: false,
    loading: () => <DataTableSkeleton toolbarButtonCount={2} rows={6} columns={6} />,
  }
)
export default function ScanHistoryVulnerabilitiesPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="px-4 lg:px-6">
      <VulnerabilitiesDetailView scanId={Number(id)} />
    </div>
  )
}
