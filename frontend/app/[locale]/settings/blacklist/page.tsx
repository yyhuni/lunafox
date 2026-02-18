"use client"

import { lazyPage } from "@/components/common/lazy-page"

const GlobalBlacklistPageContent = lazyPage(
  () => import("./content")
)

export default function GlobalBlacklistPage() {
  return <GlobalBlacklistPageContent />
}
