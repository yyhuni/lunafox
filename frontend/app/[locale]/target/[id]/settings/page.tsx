"use client"

import { lazyPage } from "@/components/common/lazy-page"
import { useParams } from "next/navigation"

const TargetSettings = lazyPage(
  () => import("@/components/target/target-settings").then((m) => ({ default: m.TargetSettings }))
)

/**
 * Target settings page
 * Contains blacklist configuration and other settings
 */
export default function TargetSettingsPage() {
  const { id } = useParams<{ id: string }>()
  const targetId = Number(id)

  return (
    <div className="px-4 lg:px-6">
      <TargetSettings targetId={targetId} />
    </div>
  )
}
