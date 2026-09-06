"use client"

import * as React from "react"
import { usePathname, useSearchParams } from "next/navigation"
import NProgress from "nprogress"

const ROUTE_PROGRESS_COLOR = "var(--color-highlight)"
const ROUTE_PROGRESS_HEIGHT = 3
const ROUTE_PROGRESS_Z_INDEX = 99999
const ROUTE_PROGRESS_SHOW_DELAY_MS = 240
const ROUTE_PROGRESS_MIN_VISIBLE_MS = 240
const ROUTE_PROGRESS_TRICKLE_SPEED = 180
const ROUTE_PROGRESS_SPEED_MS = 160
const ROUTE_PROGRESS_SUPERSEDING_LOADING_LAYERS = new Set([
  "auth-shell",
  "app-shell",
  "route",
  "workspace",
  "section",
])
const ROUTE_PROGRESS_LOADING_OWNER_SELECTOR = "[data-loading-owner]"
// `nprogress` locates the progress node through `[role="bar"]`.
// Keep that selector token even though the visual is decorative and aria-hidden.
const ROUTE_PROGRESS_TEMPLATE =
  '<div class="route-progress__track"><div class="bar" role="bar" aria-hidden="true"></div></div>'

let showTimer: number | null = null
let completionTimer: number | null = null
let visibleStartedAt = 0
let lastCommittedRouteKey = ""

function clearTimer(timer: number | null) {
  if (timer === null || typeof window === "undefined") return
  window.clearTimeout(timer)
}

function clearShowTimer() {
  clearTimer(showTimer)
  showTimer = null
}

function clearCompletionTimer() {
  clearTimer(completionTimer)
  completionTimer = null
}

function currentTime() {
  return typeof performance === "undefined" ? Date.now() : performance.now()
}

function routeKey(pathname: string, search: string) {
  return `${pathname}?${search}`
}

function currentRouteKey() {
  if (typeof window === "undefined") return ""
  return routeKey(window.location.pathname, window.location.search.replace(/^\?/, ""))
}

function toRouteUrl(href: string) {
  if (typeof window === "undefined") return null

  try {
    return new URL(href, window.location.href)
  } catch {
    return null
  }
}

function shouldStartRouteProgress(href: string) {
  const nextUrl = toRouteUrl(href)
  if (!nextUrl || typeof window === "undefined") return false

  const currentUrl = new URL(window.location.href)
  if (nextUrl.origin !== currentUrl.origin) return false
  if (nextUrl.protocol !== "http:" && nextUrl.protocol !== "https:") return false
  if (nextUrl.pathname === currentUrl.pathname && nextUrl.search === currentUrl.search) return false

  return true
}

function finishVisibleRouteProgress() {
  NProgress.done()
  visibleStartedAt = 0
  completionTimer = null
}

function isRenderedElementVisible(element: HTMLElement) {
  if (element.hidden) return false

  let current: HTMLElement | null = element
  while (current && current !== document.documentElement) {
    const style = window.getComputedStyle(current)
    if (style.display === "none" || style.visibility === "hidden") return false
    current = current.parentElement
  }

  const rect = element.getBoundingClientRect()
  return rect.width > 0 && rect.height > 0
}

function hasVisibleRouteLoadingOwner() {
  if (typeof document === "undefined" || typeof window === "undefined") return false

  return Array.from(document.querySelectorAll<HTMLElement>(ROUTE_PROGRESS_LOADING_OWNER_SELECTOR)).some((element) => {
    const owner = element.getAttribute("data-loading-owner")
    const layer = element.getAttribute("data-loading-layer")
    const phase = element.getAttribute("data-loading-phase")

    if (!owner || owner === "initial-boot") return false
    if (!layer || !ROUTE_PROGRESS_SUPERSEDING_LOADING_LAYERS.has(layer)) return false
    if (phase === "content") return false

    return isRenderedElementVisible(element)
  })
}

function cancelRouteProgressForVisibleOwner() {
  if (!hasVisibleRouteLoadingOwner()) return false

  // Route progress only covers the navigation gap before a page-level owner is visible.
  clearShowTimer()
  clearCompletionTimer()
  visibleStartedAt = 0

  if (NProgress.isStarted()) {
    NProgress.status = null
  }
  NProgress.remove()

  return true
}

function scheduleRouteProgressStart() {
  if (typeof window === "undefined") return

  clearCompletionTimer()
  if (cancelRouteProgressForVisibleOwner()) return
  if (NProgress.isStarted() || showTimer !== null) return

  showTimer = window.setTimeout(() => {
    showTimer = null
    if (cancelRouteProgressForVisibleOwner()) return
    visibleStartedAt = currentTime()
    NProgress.start()
  }, ROUTE_PROGRESS_SHOW_DELAY_MS)
}

function scheduleRouteProgressCompletion() {
  if (typeof window === "undefined") return

  if (cancelRouteProgressForVisibleOwner()) return

  if (showTimer !== null) {
    clearShowTimer()
    return
  }

  if (!NProgress.isStarted()) return

  clearCompletionTimer()
  const elapsed = visibleStartedAt === 0 ? ROUTE_PROGRESS_MIN_VISIBLE_MS : currentTime() - visibleStartedAt
  const delay = Math.max(0, ROUTE_PROGRESS_MIN_VISIBLE_MS - elapsed)
  completionTimer = window.setTimeout(finishVisibleRouteProgress, delay)
}

function findClosestAnchor(element: Element | null) {
  while (element && element.tagName.toLowerCase() !== "a") {
    element = element.parentElement
  }

  return element instanceof HTMLAnchorElement ? element : null
}

function shouldHandleNavigationClick(event: MouseEvent) {
  if (event.defaultPrevented || event.button !== 0) return false
  if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return false

  const anchor = findClosestAnchor(event.target instanceof Element ? event.target : null)
  if (!anchor) return false
  if (anchor.target && anchor.target !== "_self") return false
  if (anchor.hasAttribute("download")) return false

  return shouldStartRouteProgress(anchor.href)
}

export function pushWithRouteProgress<TOptions>(
  router: { push: (href: string, options?: TOptions) => void },
  href: string,
  options?: TOptions
) {
  if (shouldStartRouteProgress(href)) scheduleRouteProgressStart()
  if (options === undefined) {
    router.push(href)
    return
  }
  router.push(href, options)
}

export function replaceWithRouteProgress<TOptions>(
  router: { replace: (href: string, options?: TOptions) => void },
  href: string,
  options?: TOptions
) {
  if (shouldStartRouteProgress(href)) scheduleRouteProgressStart()
  if (options === undefined) {
    router.replace(href)
    return
  }
  router.replace(href, options)
}

function usePrefersReducedMotion() {
  const [prefersReducedMotion, setPrefersReducedMotion] = React.useState<boolean | null>(null)

  React.useEffect(() => {
    const query = window.matchMedia("(prefers-reduced-motion: reduce)")
    const update = () => setPrefersReducedMotion(query.matches)

    update()
    query.addEventListener("change", update)
    return () => query.removeEventListener("change", update)
  }, [])

  return prefersReducedMotion
}

function RouteProgressCoordinator({ prefersReducedMotion }: { prefersReducedMotion: boolean }) {
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const search = searchParams.toString()

  React.useEffect(() => {
    NProgress.configure({
      showSpinner: false,
      trickle: true,
      trickleSpeed: ROUTE_PROGRESS_TRICKLE_SPEED,
      minimum: 0.08,
      easing: "ease-out",
      speed: prefersReducedMotion ? 0 : ROUTE_PROGRESS_SPEED_MS,
      template: ROUTE_PROGRESS_TEMPLATE,
    })
  }, [prefersReducedMotion])

  React.useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      if (shouldHandleNavigationClick(event)) scheduleRouteProgressStart()
    }

    const handlePageHide = () => {
      clearShowTimer()
      clearCompletionTimer()
      NProgress.done()
    }

    const handlePopState = () => {
      if (currentRouteKey() !== lastCommittedRouteKey) scheduleRouteProgressStart()
    }

    document.addEventListener("click", handleClick, true)
    window.addEventListener("pagehide", handlePageHide)
    window.addEventListener("popstate", handlePopState)

    return () => {
      document.removeEventListener("click", handleClick, true)
      window.removeEventListener("pagehide", handlePageHide)
      window.removeEventListener("popstate", handlePopState)
      clearShowTimer()
      clearCompletionTimer()
      NProgress.done()
    }
  }, [])

  React.useEffect(() => {
    const syncRouteProgressOwner = () => {
      if (showTimer !== null || NProgress.isStarted() || NProgress.isRendered()) {
        cancelRouteProgressForVisibleOwner()
      }
    }

    syncRouteProgressOwner()

    if (typeof MutationObserver === "undefined") {
      return
    }

    const observer = new MutationObserver(syncRouteProgressOwner)
    observer.observe(document.body, {
      attributes: true,
      attributeFilter: [
        "class",
        "data-loading-layer",
        "data-loading-owner",
        "data-loading-phase",
        "hidden",
        "style",
      ],
      childList: true,
      subtree: true,
    })

    return () => observer.disconnect()
  }, [])

  React.useEffect(() => {
    lastCommittedRouteKey = routeKey(pathname, search)
    scheduleRouteProgressCompletion()
  }, [pathname, search])

  return null
}

/**
 * LunaFox-owned adapter for route/navigation progress.
 *
 * The adapter owns nprogress directly so visible route progress follows the
 * app router commit signal instead of a package-level history fallback.
 */
export function RouteProgress() {
  const prefersReducedMotion = usePrefersReducedMotion()

  if (prefersReducedMotion === null) return null

  return (
    <>
      <RouteProgressCoordinator prefersReducedMotion={prefersReducedMotion} />
      <style jsx global>{`
        #nprogress {
          height: ${ROUTE_PROGRESS_HEIGHT}px;
          inset: 0 0 auto 0;
          isolation: isolate;
          overflow: hidden;
          pointer-events: none;
          position: fixed;
          z-index: ${ROUTE_PROGRESS_Z_INDEX};
        }

        #nprogress .route-progress__track {
          height: 100%;
          inset: 0;
          overflow: hidden;
          position: absolute;
        }

        #nprogress .bar {
          background: ${ROUTE_PROGRESS_COLOR};
          height: 100%;
          left: 0;
          position: absolute;
          top: 0;
          width: 100%;
          will-change: transform;
        }

        @media (prefers-reduced-motion: reduce) {
          #nprogress,
          #nprogress .bar,
          #nprogress .peg {
            animation: none !important;
            transition: none !important;
          }
        }
      `}</style>
    </>
  )
}
