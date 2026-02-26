"use client"

import React from "react"
import { useParams } from "next/navigation"
import { VulnerabilitiesDetailView } from "@/components/vulnerabilities/vulnerabilities-detail-view"
export default function ScanHistoryVulnerabilitiesPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="px-4 lg:px-6">
      <VulnerabilitiesDetailView scanId={Number(id)} />
    </div>
  )
}
