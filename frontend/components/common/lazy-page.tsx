"use client"

import dynamic, { type Loader } from "next/dynamic"
import type { ComponentType, ReactNode } from "react"
import { PageSectionSkeleton } from "@/components/shared/loading/page-section-skeleton"

export function lazyPage<P extends object = Record<string, never>>(
  loader: Loader<P>,
  loadingOwner = "lazy-page",
  loadingFallback?: ReactNode | null
): ComponentType<P> {
  const hasCustomFallback = arguments.length >= 3

  return dynamic<P>(loader, {
    ssr: false,
    loading: () => (hasCustomFallback ? loadingFallback : <PageSectionSkeleton owner={loadingOwner} />),
  })
}
