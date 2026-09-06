import { useEffect, useCallback, useRef } from 'react'
import { useRouter } from 'next/navigation'

const BASE_CRITICAL_ROUTES = ['/overview/'] as const
const BASE_SECONDARY_ROUTES = [] as const
const BASE_LOW_PRIORITY_ROUTES = [] as const

const DETAIL_SUB_ROUTES = [
  'subdomains',
  'endpoints',
  'websites',
  'vulnerabilities',
  'directories',
  'ip-addresses',
] as const

const TARGET_DETAIL_ROUTE_SEGMENTS = [
  'overview',
  ...DETAIL_SUB_ROUTES,
  'screenshots',
  'settings',
] as const

const SCAN_DETAIL_ROUTE_SEGMENTS = [
  'overview',
  ...DETAIL_SUB_ROUTES,
  'screenshots',
] as const

type NetworkInformation = {
  saveData?: boolean
  effectiveType?: string
}

type NavigatorWithConnection = Navigator & {
  connection?: NetworkInformation
}

const getNetworkConnection = (): NetworkInformation | undefined => {
  if (typeof navigator === 'undefined') return undefined
  return (navigator as NavigatorWithConnection).connection
}

const canPrefetchSecondaryRoutes = (): boolean => {
  const connection = getNetworkConnection()
  if (!connection) return true
  if (connection.saveData) return false
  const effectiveType = connection.effectiveType
  return effectiveType !== 'slow-2g' && effectiveType !== '2g'
}

const canPrefetchLowPriorityRoutes = (): boolean => {
  const connection = getNetworkConnection()
  if (!connection) return true
  if (connection.saveData) return false
  return connection.effectiveType === '4g'
}

/**
 * Route preloading Hook
 * After the page is loaded, JS/CSS resources of other pages are preloaded in the background
 * No API requests are sent, only page components are loaded
 * @param currentPath current page path (optional), if provided, relevant dynamic routes will be intelligently preloaded
 */
export function useRoutePrefetch(currentPath?: string) {
  const router = useRouter()
  const prefetchedRoutesRef = useRef<Set<string>>(new Set())

  const normalizePath = useCallback((path: string) => {
    const withLeadingSlash = path.startsWith("/") ? path : `/${path}`
    return withLeadingSlash !== "/" && withLeadingSlash.endsWith("/")
      ? withLeadingSlash.slice(0, -1)
      : withLeadingSlash
  }, [])

  const prefetchOnce = useCallback((path: string) => {
    const normalizedPath = normalizePath(path)
    if (currentPath && normalizePath(currentPath) === normalizedPath) return
    if (prefetchedRoutesRef.current.has(normalizedPath)) return
    prefetchedRoutesRef.current.add(normalizedPath)
    void router.prefetch(normalizedPath)
  }, [currentPath, normalizePath, router])

  useEffect(() => {
    const w = typeof window !== 'undefined'
      ? (window as Window & { __lunafoxRoutePrefetchDone?: boolean })
      : null
    const hasPrefetched = !!w?.__lunafoxRoutePrefetchDone
    const allowSecondaryPrefetch = canPrefetchSecondaryRoutes()
    const allowLowPriorityPrefetch = canPrefetchLowPriorityRoutes()
    const idleTaskIds: number[] = []
    const timeoutIds: Array<ReturnType<typeof setTimeout>> = []

    const prefetchBatch = (routes: readonly string[]) => {
      routes.forEach((route) => {
        prefetchOnce(route)
      })
    }

    // Use requestIdleCallback to preload when the browser is idle without affecting the rendering of the current page.
    const prefetchBaseRoutes = () => {
      prefetchBatch(BASE_CRITICAL_ROUTES)
      if (!allowSecondaryPrefetch) return

      const scheduleSecondary = () => {
        prefetchBatch(BASE_SECONDARY_ROUTES)
      }

      const scheduleLowPriority = () => {
        if (!allowLowPriorityPrefetch) return
        prefetchBatch(BASE_LOW_PRIORITY_ROUTES)
      }

      if (typeof window !== 'undefined') {
        if ('requestIdleCallback' in window) {
          idleTaskIds.push(window.requestIdleCallback(scheduleSecondary, { timeout: 2000 }))
          idleTaskIds.push(window.requestIdleCallback(scheduleLowPriority, { timeout: 4000 }))
          return
        }
      }

      scheduleSecondary()
      timeoutIds.push(setTimeout(scheduleLowPriority, 2000))
    }

    const prefetchDynamicRoutes = () => {
      if (!currentPath || !allowSecondaryPrefetch) return
      const normalizedCurrentPath = normalizePath(currentPath)

      // If it is a target detail route (such as /targets/146/overview/), preload sibling routes.
      const targetIdMatch = normalizedCurrentPath.match(
        new RegExp(`^/targets/(\\d+)/(?:${TARGET_DETAIL_ROUTE_SEGMENTS.join("|")})$`)
      )
      if (targetIdMatch) {
        const targetId = targetIdMatch[1]
        TARGET_DETAIL_ROUTE_SEGMENTS.forEach((subRoute) => {
          prefetchOnce(`/targets/${targetId}/${subRoute}`)
        })
      }

      // If it is a scan-history detail route (such as /scan/history/146/overview/), preload sibling routes.
      const scanIdMatch = normalizedCurrentPath.match(
        new RegExp(`^/scan/history/(\\d+)/(?:${SCAN_DETAIL_ROUTE_SEGMENTS.join("|")})$`)
      )
      if (scanIdMatch) {
        const scanId = scanIdMatch[1]
        SCAN_DETAIL_ROUTE_SEGMENTS.forEach((subRoute) => {
          prefetchOnce(`/scan/history/${scanId}/${subRoute}`)
        })
      }
    }

    const runPrefetch = () => {
      if (!hasPrefetched) {
        prefetchBaseRoutes()
        if (w) {
          w.__lunafoxRoutePrefetchDone = true
          w.dispatchEvent(new Event('lunafox:route-prefetch-done'))
        }
      }
      prefetchDynamicRoutes()
    }

    if (hasPrefetched) {
      runPrefetch()
      return () => {
        idleTaskIds.forEach((id) => {
          if (typeof window !== 'undefined' && 'cancelIdleCallback' in window) {
            window.cancelIdleCallback(id)
          }
        })
        timeoutIds.forEach((id) => clearTimeout(id))
      }
    }

    // Use requestIdleCallback to execute when the browser is idle, or immediately if not supported
    if (typeof window !== 'undefined' && 'requestIdleCallback' in window) {
      const idleId = window.requestIdleCallback(runPrefetch)
      return () => {
        window.cancelIdleCallback(idleId)
        idleTaskIds.forEach((id) => window.cancelIdleCallback(id))
        timeoutIds.forEach((id) => clearTimeout(id))
      }
    }

    runPrefetch()
    return () => {
      timeoutIds.forEach((id) => clearTimeout(id))
    }
  }, [currentPath, normalizePath, prefetchOnce])
}
