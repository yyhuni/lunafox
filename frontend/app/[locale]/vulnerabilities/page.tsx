"use client"

import React from "react"
import { VulnerabilitiesVerticalView } from "@/components/vulnerabilities/vulnerabilities-vertical-view"

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
