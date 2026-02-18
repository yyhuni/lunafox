"use client"

import React from "react"
import dynamic from "next/dynamic"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"

const ARLFingerprintView = dynamic(
  () => import("@/components/fingerprints/arl-fingerprint-view").then((mod) => mod.ARLFingerprintView),
  {
    ssr: false,
    loading: () => <DataTableSkeleton toolbarButtonCount={3} rows={6} columns={4} />,
  }
)

export default function ARLFingerprintPage() {
  return (
    <div className="px-4 lg:px-6">
      <ARLFingerprintView />
    </div>
  )
}
