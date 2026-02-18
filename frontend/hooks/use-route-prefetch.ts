import { useEffect, useCallback, useRef } from 'react'
import { useRouter } from '@/i18n/navigation'

const BASE_CRITICAL_ROUTES = ['/dashboard/'] as const
const BASE_SECONDARY_ROUTES = ['/organization/', '/target/'] as const
const BASE_LOW_PRIORITY_ROUTES = ['/scan/history/', '/vulnerabilities/'] as const

const DETAIL_SUB_ROUTES = [
  'subdomain',
  'endpoints',
  'websites',
  'vulnerabilities',
  'directories',
  'ip-addresses',
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

  const prefetchOnce = useCallback((path: string) => {
    const normalizedPath = path.startsWith("/") ? path : `/${path}`
    if (prefetchedRoutesRef.current.has(normalizedPath)) return
    prefetchedRoutesRef.current.add(normalizedPath)
    void router.prefetch(normalizedPath)
  }, [router])

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
      // If it is a target details page (such as /target/146), preload the sub-route
      const targetIdMatch = currentPath.match(/\/target\/(\d+)$/)
      if (targetIdMatch) {
        const targetId = targetIdMatch[1]
        DETAIL_SUB_ROUTES.forEach((subRoute) => {
          prefetchOnce(`/target/${targetId}/${subRoute}`)
        })
      }
      
      // If scanning history details page (such as /scan/history/146), preload sub-route
      const scanIdMatch = currentPath.match(/\/scan\/history\/(\d+)$/)
      if (scanIdMatch) {
        const scanId = scanIdMatch[1]
        DETAIL_SUB_ROUTES.forEach((subRoute) => {
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
  }, [currentPath, prefetchOnce])
}

/**
 * Smart route preloading Hook
 * Preload the next page the user may visit based on the current path
 * @param currentPath current page path
 */
export function useSmartRoutePrefetch(currentPath: string) {
  const router = useRouter()
  const prefetchedRoutesRef = useRef<Set<string>>(new Set())

  const prefetchOnce = useCallback((path: string) => {
    const normalizedPath = path.startsWith("/") ? path : `/${path}`
    if (prefetchedRoutesRef.current.has(normalizedPath)) return
    prefetchedRoutesRef.current.add(normalizedPath)
    void router.prefetch(normalizedPath)
  }, [router])

  useEffect(() => {
    const timer = setTimeout(() => {
      if (currentPath.includes('/organization')) {
        // On the organization page, preload the target page
        prefetchOnce('/target/')
      } else if (currentPath.includes('/target')) {
        // On the target page, preload scan and vulnerability pages
        prefetchOnce('/scan/history/')
        prefetchOnce('/vulnerabilities/')

        // If it is a target details page (such as /target/146), preload the sub-route
        const targetIdMatch = currentPath.match(/\/target\/(\d+)$/)
        if (targetIdMatch) {
          const targetId = targetIdMatch[1]
          const subRoutes = ['subdomain', 'endpoints', 'websites', 'vulnerabilities']
          subRoutes.forEach((sub) => {
            prefetchOnce(`/target/${targetId}/${sub}`)
          })
        }
      } else if (currentPath.includes('/scan/history')) {
        // On the scanning history page, preload the target page
        prefetchOnce('/target/')
        prefetchOnce('/vulnerabilities/')
      } else if (currentPath === '/') {
        // On the homepage, preload the main page
        prefetchOnce('/dashboard/')
        prefetchOnce('/organization/')
      }
    }, 1500) // Preload after 1.5 seconds

    return () => clearTimeout(timer)
  }, [currentPath, prefetchOnce])
}
