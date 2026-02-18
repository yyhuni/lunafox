"use client"

import dynamic from "next/dynamic"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"

const WappalyzerFingerprintView = dynamic(
  () => import("@/components/fingerprints/wappalyzer-fingerprint-view").then((mod) => mod.WappalyzerFingerprintView),
  {
    ssr: false,
    loading: () => <DataTableSkeleton toolbarButtonCount={3} rows={6} columns={7} />,
  }
)

export default function WappalyzerFingerprintsPage() {
  return (
    <div className="px-4 lg:px-6">
      <WappalyzerFingerprintView />
    </div>
  )
}
