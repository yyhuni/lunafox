"use client"

import dynamic, { type Loader } from "next/dynamic"
import type { ComponentType } from "react"

export function lazyPage<P extends object = Record<string, never>>(
  loader: Loader<P>,
  loadingClassName = "py-6"
): ComponentType<P> {
  return dynamic<P>(loader, {
    ssr: false,
    loading: () => <div className={loadingClassName} />,
  })
}
