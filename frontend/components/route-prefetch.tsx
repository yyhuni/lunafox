'use client'

import { usePathname } from 'next/navigation'
import { useRoutePrefetch } from '@/hooks/use-route-prefetch'

/**
 * Route prefetch component
 * Automatically prefetches JS/CSS resources for commonly used pages after app startup
 * This is an invisible component, only used to execute prefetch logic
 */
export function RoutePrefetch() {
  const pathname = usePathname()
  useRoutePrefetch(pathname)
  return null
}
