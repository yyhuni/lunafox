"use client"

import dynamic from "next/dynamic"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"

const GobyFingerprintView = dynamic(
  () => import("@/components/fingerprints/goby-fingerprint-view").then((mod) => mod.GobyFingerprintView),
  {
    ssr: false,
    loading: () => <DataTableSkeleton toolbarButtonCount={3} rows={6} columns={6} />,
  }
)

export default function GobyFingerprintsPage() {
  return (
    <div className="px-4 lg:px-6">
      <GobyFingerprintView />
    </div>
  )
}
