"use client"

import { lazyPage } from "@/components/common/lazy-page"

const WordlistsPageContent = lazyPage(
  () => import("@/components/tools/wordlists-page")
)

export default function WordlistsPage() {
  return <WordlistsPageContent />
}
