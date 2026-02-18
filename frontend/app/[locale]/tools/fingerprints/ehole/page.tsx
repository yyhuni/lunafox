"use client"

import React from "react"
import dynamic from "next/dynamic"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"

const EholeFingerprintView = dynamic(
  () => import("@/components/fingerprints/ehole-fingerprint-view").then((mod) => mod.EholeFingerprintView),
  {
    ssr: false,
    loading: () => <DataTableSkeleton toolbarButtonCount={3} rows={6} columns={7} />,
  }
)

export default function EholeFingerprintPage() {
  return (
    <div className="px-4 lg:px-6">
      <EholeFingerprintView />
    </div>
  )
}
