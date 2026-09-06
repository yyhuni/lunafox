"use client"

import * as React from "react"

const BOOT_LAYER_EXIT_CLASS = "lunafox-boot-layer--leaving"
const BOOT_LAYER_EXIT_MS = 220
const BOOT_LAYER_READY_FALLBACK_MS = 3200
const APP_SHELL_WARMUP_SELECTOR = "[data-slot='app-shell-warmup']"
const BOOT_LAYER_COMPLETE_ATTRIBUTE = "data-boot-layer-complete"

function isInsideAppShellWarmup(element: Element) {
  return Boolean(element.closest(APP_SHELL_WARMUP_SELECTOR))
}

function isAppShellWarmupOwner(element: Element) {
  return element.matches(APP_SHELL_WARMUP_SELECTOR) && element.hasAttribute("data-loading-owner")
}

function hasRealLoadingOwner(bootLayer: HTMLElement) {
  return Array.from(document.querySelectorAll<HTMLElement>("[data-loading-owner]")).some((element) => {
    if (element === bootLayer || bootLayer.contains(element)) return false
    if (isInsideAppShellWarmup(element) && !isAppShellWarmupOwner(element)) return false
    return element.getAttribute("data-loading-owner") !== "initial-boot"
  })
}

// Shell chrome presence indicates a visible next owner once it is outside the temporary warmup shell.
// Do NOT rely on mainContent.textContent alone — text in the DOM does not prove a
// visually stable owner occupies the first-screen region.
function hasVisibleAppLayer() {
  return Array.from(
    document.querySelectorAll<HTMLElement>("[data-slot='sidebar-wrapper'], [data-slot='sidebar-inset']")
  ).some((element) => !isInsideAppShellWarmup(element))
}

function hasPendingBootHandoff() {
  return Boolean(document.querySelector('[data-boot-handoff-pending="true"]'))
}

function hideBootLayer(bootLayer: HTMLElement) {
  document.documentElement.setAttribute(BOOT_LAYER_COMPLETE_ATTRIBUTE, "true")
  bootLayer.hidden = true
  bootLayer.setAttribute("aria-hidden", "true")
}

export function BootLayerController() {
  React.useEffect(() => {
    const bootLayer = document.getElementById("lunafox-boot-layer")
    if (!bootLayer) return

    if (document.documentElement.getAttribute(BOOT_LAYER_COMPLETE_ATTRIBUTE) === "true") {
      hideBootLayer(bootLayer)
      return
    }

    let exitTimer: number | undefined
    let fallbackTimer: number | undefined
    let hasStartedExit = false
    let observer: MutationObserver | undefined

    const startExit = ({ immediate = false }: { immediate?: boolean } = {}) => {
      if (bootLayer.getAttribute("data-boot-exit-started") === "true") return
      if (hasStartedExit) return
      if (hasPendingBootHandoff()) return
      hasStartedExit = true
      observer?.disconnect()
      bootLayer.setAttribute("data-boot-exit-started", "true")
      bootLayer.classList.add(BOOT_LAYER_EXIT_CLASS)
      bootLayer.setAttribute("aria-hidden", "true")
      if (immediate) {
        hideBootLayer(bootLayer)
        return
      }
      exitTimer = window.setTimeout(() => {
        hideBootLayer(bootLayer)
      }, BOOT_LAYER_EXIT_MS)
    }

    const getHandoffReadiness = () => {
      if (hasPendingBootHandoff()) return { ready: false, immediate: false }

      const hasLoadingOwner = hasRealLoadingOwner(bootLayer)
      return {
        ready: hasLoadingOwner || hasVisibleAppLayer(),
        // Once a route/workspace skeleton owns the next frame, do not fade the
        // full-screen boot mark over it; that reads as a second loader.
        immediate: hasLoadingOwner,
      }
    }

    const initialReadiness = getHandoffReadiness()
    if (initialReadiness.ready) {
      startExit({ immediate: initialReadiness.immediate })
    } else {
      observer = new MutationObserver(() => {
        const readiness = getHandoffReadiness()
        if (readiness.ready) {
          startExit({ immediate: readiness.immediate })
        }
      })
      observer.observe(document.body, {
        attributes: true,
        attributeFilter: ["data-loading-owner", "data-slot", "data-boot-handoff-pending", "id"],
        childList: true,
        subtree: true,
      })
      fallbackTimer = window.setTimeout(startExit, BOOT_LAYER_READY_FALLBACK_MS)
    }

    return () => {
      observer?.disconnect()
      if (exitTimer) window.clearTimeout(exitTimer)
      if (fallbackTimer) window.clearTimeout(fallbackTimer)
    }
  }, [])

  return null
}
