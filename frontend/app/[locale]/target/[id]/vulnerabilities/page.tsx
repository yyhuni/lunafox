"use client"

import React from "react"
import { useParams } from "next/navigation"
import { VulnerabilitiesDetailView } from "@/components/vulnerabilities/vulnerabilities-detail-view"
/**
 * Target vulnerabilities page
 * Displays vulnerability details under the target
 */
export default function TargetVulnerabilitiesPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="px-4 lg:px-6">
      <VulnerabilitiesDetailView targetId={parseInt(id)} />
    </div>
  )
}
