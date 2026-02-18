"use client"

import React from "react"
import dynamic from "next/dynamic"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"

const FingerPrintHubFingerprintView = dynamic(
  () => import("@/components/fingerprints/fingerprinthub-fingerprint-view").then((mod) => mod.FingerPrintHubFingerprintView),
  {
    ssr: false,
    loading: () => <DataTableSkeleton toolbarButtonCount={3} rows={6} columns={8} />,
  }
)

export default function FingerPrintHubFingerprintPage() {
  return (
    <div className="px-4 lg:px-6">
      <FingerPrintHubFingerprintView />
    </div>
  )
}
