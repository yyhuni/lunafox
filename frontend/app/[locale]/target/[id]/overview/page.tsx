"use client"

import { lazyPage } from "@/components/common/lazy-page"
import { useParams } from "next/navigation"

const TargetOverview = lazyPage(
  () => import("@/components/target/target-overview").then((m) => ({ default: m.TargetOverview }))
)

/**
 * Target overview page
 * Displays target statistics and summary information
 */
export default function TargetOverviewPage() {
  const { id } = useParams<{ id: string }>()
  const targetId = Number(id)

  return (
    <div className="px-4 lg:px-6">
      <TargetOverview targetId={targetId} />
    </div>
  )
}
