"use client"

import { SearchPageContent } from "./search-page-sections"
import { useSearchPageState } from "./search-page-state"

export function SearchPage() {
  const state = useSearchPageState()
  return <SearchPageContent state={state} />
}
