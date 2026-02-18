"use client"

import { lazyPage } from "@/components/common/lazy-page"

const GithubButtonDemoPageContent = lazyPage(
  () => import("./content")
)

export default function GithubButtonDemoPage() {
  return <GithubButtonDemoPageContent />
}
