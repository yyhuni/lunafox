"use client"

import { lazyPage } from "@/components/common/lazy-page"

const TableListDemoContent = lazyPage(
  () => import("./content")
)

export default function TableListDemo() {
  return <TableListDemoContent />
}
