"use client"

import { lazyPage } from "@/components/common/lazy-page"

const PageContent = lazyPage(
  () => import("./content")
)

export default function Page() {
  return <PageContent />
}
