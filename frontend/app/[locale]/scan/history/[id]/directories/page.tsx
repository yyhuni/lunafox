"use client"

import { lazyPage } from "@/components/common/lazy-page"
import { useParams } from "next/navigation"

const DirectoriesView = lazyPage(
  () => import("@/components/directories/directories-view").then((m) => ({ default: m.DirectoriesView }))
)

export default function ScanDirectoriesPage() {
  const { id } = useParams<{ id: string }>()
  const scanId = Number(id)

  return (
    <div className="px-4 lg:px-6">
      <DirectoriesView scanId={scanId} />
    </div>
  )
}
