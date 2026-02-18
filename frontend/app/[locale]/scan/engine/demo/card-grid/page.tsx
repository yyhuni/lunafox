"use client"

import { lazyPage } from "@/components/common/lazy-page"

const CardGridDemoContent = lazyPage(
  () => import("./content")
)

export default function CardGridDemo() {
  return <CardGridDemoContent />
}
