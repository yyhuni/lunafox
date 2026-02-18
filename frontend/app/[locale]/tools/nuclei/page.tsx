"use client"

import { lazyPage } from "@/components/common/lazy-page"

const NucleiReposPageContent = lazyPage(
  () => import("@/components/tools/nuclei-repos-page")
)

export default function NucleiReposPage() {
  return <NucleiReposPageContent />
}
