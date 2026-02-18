"use client"

import { lazyPage } from "@/components/common/lazy-page"

const SearchPage = lazyPage(
  () => import("@/components/search/search-page").then((m) => ({ default: m.SearchPage }))
)

export default function Search() {
  return <SearchPage />
}
