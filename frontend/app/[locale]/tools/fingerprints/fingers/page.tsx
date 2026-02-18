"use client"

import React from "react"
import dynamic from "next/dynamic"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"

const FingersFingerprintView = dynamic(
  () => import("@/components/fingerprints/fingers-fingerprint-view").then((mod) => mod.FingersFingerprintView),
  {
    ssr: false,
    loading: () => <DataTableSkeleton toolbarButtonCount={3} rows={6} columns={7} />,
  }
)

export default function FingersFingerprintPage() {
  return (
    <div className="px-4 lg:px-6">
      <FingersFingerprintView />
    </div>
  )
}
