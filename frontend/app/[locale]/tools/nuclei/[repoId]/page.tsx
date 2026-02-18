"use client"

import { lazyPage } from "@/components/common/lazy-page"

const NucleiRepoDetailPageContent = lazyPage(
  () => import("@/components/tools/nuclei-repo-detail-page")
)

export default function NucleiRepoDetailPage() {
  return <NucleiRepoDetailPageContent />
}
