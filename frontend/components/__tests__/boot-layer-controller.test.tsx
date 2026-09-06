import * as React from "react"
import { act, render } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

function createBootLayer() {
  const bootLayer = document.createElement("div")
  bootLayer.id = "lunafox-boot-layer"
  bootLayer.setAttribute("data-loading-owner", "initial-boot")
  bootLayer.setAttribute("data-slot", "boot-layer")
  bootLayer.setAttribute("role", "status")
  bootLayer.setAttribute("aria-live", "polite")
  document.body.appendChild(bootLayer)
  return bootLayer
}

function createMainContent(text?: string) {
  const mainContent = document.createElement("main")
  mainContent.id = "main-content"
  if (text) mainContent.textContent = text
  document.body.appendChild(mainContent)
  return mainContent
}

function createSidebar() {
  const sidebar = document.createElement("div")
  sidebar.setAttribute("data-slot", "sidebar-wrapper")
  document.body.appendChild(sidebar)
  return sidebar
}

function createAppShellWarmup({ owner = true }: { owner?: boolean } = {}) {
  const warmup = document.createElement("div")
  warmup.setAttribute("data-slot", "app-shell-warmup")
  if (owner) {
    warmup.setAttribute("data-loading-owner", "auth-layout-suspense-fallback")
  }
  const inset = document.createElement("div")
  inset.setAttribute("data-slot", "sidebar-inset")
  warmup.appendChild(inset)
  document.body.appendChild(warmup)
  return warmup
}

function createNonBootOwner(name = "auth-layout-warmup") {
  const owner = document.createElement("div")
  owner.setAttribute("data-loading-owner", name)
  document.body.appendChild(owner)
  return owner
}

describe("BootLayerController", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    document.body.innerHTML = ""
    document.documentElement.removeAttribute("data-boot-layer-complete")
  })

  afterEach(() => {
    vi.useRealTimers()
    document.body.innerHTML = ""
    document.documentElement.removeAttribute("data-boot-layer-complete")
  })

  it("keeps the boot layer visible during the CSS exit window before hiding it", async () => {
    const bootLayer = createBootLayer()
    createSidebar()

    const { BootLayerController } = await import("@/components/boot-layer-controller")
    const removeSpy = vi.spyOn(bootLayer, "remove")

    render(<BootLayerController />)

    expect(bootLayer).toHaveClass("lunafox-boot-layer--leaving")
    expect(bootLayer.getAttribute("aria-hidden")).toBe("true")
    expect(bootLayer.hidden).toBe(false)
    expect(document.body.contains(bootLayer)).toBe(true)

    act(() => {
      vi.advanceTimersByTime(219)
    })

    expect(bootLayer.hidden).toBe(false)

    act(() => {
      vi.advanceTimersByTime(1)
    })

    expect(bootLayer.hidden).toBe(true)
    expect(document.body.contains(bootLayer)).toBe(true)
    expect(removeSpy).not.toHaveBeenCalled()
  })

  it("does not remove the boot layer during cleanup", async () => {
    const bootLayer = createBootLayer()
    const { BootLayerController } = await import("@/components/boot-layer-controller")
    const removeSpy = vi.spyOn(bootLayer, "remove")

    const view = render(<BootLayerController />)
    view.unmount()

    expect(removeSpy).not.toHaveBeenCalled()
    expect(document.body.contains(bootLayer)).toBe(true)
  })

  it("keeps boot hidden after a same-document refresh has already completed handoff", async () => {
    const bootLayer = createBootLayer()
    document.documentElement.setAttribute("data-boot-layer-complete", "true")

    const { BootLayerController } = await import("@/components/boot-layer-controller")

    render(<BootLayerController />)

    expect(bootLayer.hidden).toBe(true)
    expect(bootLayer.getAttribute("aria-hidden")).toBe("true")
    expect(document.documentElement).toHaveAttribute("data-boot-layer-complete", "true")
  })

  it("does not exit boot when only mainContent has text but no visible shell chrome exists", async () => {
    createBootLayer()
    createMainContent("some text content")

    const { BootLayerController } = await import("@/components/boot-layer-controller")

    render(<BootLayerController />)

    // Boot should NOT exit — mainContent.textContent alone is not a valid handoff signal
    const bootLayer = document.getElementById("lunafox-boot-layer")!
    expect(bootLayer).not.toHaveClass("lunafox-boot-layer--leaving")
    expect(bootLayer.hidden).toBe(false)
  })

  it("hides boot immediately when a non-boot loading owner is present", async () => {
    const bootLayer = createBootLayer()
    createNonBootOwner("auth-layout-warmup")

    const { BootLayerController } = await import("@/components/boot-layer-controller")

    render(<BootLayerController />)

    expect(bootLayer).toHaveClass("lunafox-boot-layer--leaving")
    expect(bootLayer.hidden).toBe(true)
  })

  it("exits boot when sidebar-wrapper is present", async () => {
    const bootLayer = createBootLayer()
    createSidebar()

    const { BootLayerController } = await import("@/components/boot-layer-controller")

    render(<BootLayerController />)

    expect(bootLayer).toHaveClass("lunafox-boot-layer--leaving")

    act(() => {
      vi.advanceTimersByTime(220)
    })

    expect(bootLayer.hidden).toBe(true)
  })

  it("hides boot immediately when the temporary app-shell warmup owns loading", async () => {
    const bootLayer = createBootLayer()
    createAppShellWarmup()

    const { BootLayerController } = await import("@/components/boot-layer-controller")

    render(<BootLayerController />)

    expect(bootLayer).toHaveClass("lunafox-boot-layer--leaving")
    expect(bootLayer.hidden).toBe(true)

    act(() => {
      vi.advanceTimersByTime(220)
    })

    expect(bootLayer.hidden).toBe(true)
  })

  it("does not exit boot for warmup chrome without an app-shell loading owner", async () => {
    const bootLayer = createBootLayer()
    createAppShellWarmup({ owner: false })

    const { BootLayerController } = await import("@/components/boot-layer-controller")

    render(<BootLayerController />)

    expect(bootLayer).not.toHaveClass("lunafox-boot-layer--leaving")
    expect(bootLayer.hidden).toBe(false)

    act(() => {
      vi.advanceTimersByTime(220)
    })

    expect(bootLayer.hidden).toBe(false)
  })

  it("exits boot via fallback timer when no owner appears within BOOT_LAYER_READY_FALLBACK_MS", async () => {
    const bootLayer = createBootLayer()

    const { BootLayerController } = await import("@/components/boot-layer-controller")

    render(<BootLayerController />)

    // Boot should still be visible — no owner yet
    expect(bootLayer).not.toHaveClass("lunafox-boot-layer--leaving")

    // Advance past the fallback timer (3200ms)
    act(() => {
      vi.advanceTimersByTime(3200)
    })

    expect(bootLayer).toHaveClass("lunafox-boot-layer--leaving")

    act(() => {
      vi.advanceTimersByTime(220)
    })

    expect(bootLayer.hidden).toBe(true)
  })

  it("does not exit via fallback timer while a boot handoff blocker is pending", async () => {
    const bootLayer = createBootLayer()
    const blocker = document.createElement("div")
    blocker.setAttribute("data-boot-handoff-pending", "true")
    document.body.appendChild(blocker)

    const { BootLayerController } = await import("@/components/boot-layer-controller")

    render(<BootLayerController />)

    act(() => {
      vi.advanceTimersByTime(3200)
    })

    expect(bootLayer).not.toHaveClass("lunafox-boot-layer--leaving")
    expect(bootLayer.hidden).toBe(false)

    createNonBootOwner("login-page-content")
    blocker.removeAttribute("data-boot-handoff-pending")

    await act(async () => {
      await Promise.resolve()
    })

    expect(bootLayer).toHaveClass("lunafox-boot-layer--leaving")
  })
})
