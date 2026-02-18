"use client"

import { lazyPage } from "@/components/common/lazy-page"
import { useParams } from "next/navigation"

const ScreenshotsGallery = lazyPage(
  () => import("@/components/screenshots/screenshots-gallery").then((m) => ({ default: m.ScreenshotsGallery }))
)

export default function ScreenshotsPage() {
  const { id } = useParams<{ id: string }>()
  const targetId = Number(id)

  return (
    <div className="px-4 lg:px-6">
      <ScreenshotsGallery targetId={targetId} />
    </div>
  )
}
