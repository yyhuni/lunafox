import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/boot-layer-controller.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "app/layout.tsx"), "utf8")

describe("boot layer controller contract", () => {
  it("starts a timed CSS exit once a real app layer is visible", () => {
    expect(source).toContain('"use client"')
    expect(source).toContain('document.getElementById("lunafox-boot-layer")')
    expect(source).toContain('data-loading-owner')
    expect(source).toContain('initial-boot')
    expect(source).toContain("hasPendingBootHandoff")
    expect(source).toContain('data-boot-handoff-pending="true"')
    expect(source).toContain("MutationObserver")
    expect(source).toContain("BOOT_LAYER_READY_FALLBACK_MS")
    expect(source).not.toContain("BOOT_LAYER_ANIMATED_CLASS")
    expect(source).not.toContain("requestAnimationFrame")
    expect(source).toContain('lunafox-boot-layer--leaving')
    expect(source).toContain("hideBootLayer")
    expect(source).toContain("BOOT_LAYER_COMPLETE_ATTRIBUTE")
    expect(source).toContain("data-boot-layer-complete")
    expect(source).toContain("immediate: hasLoadingOwner")
    expect(source).toContain("startExit({ immediate:")
    expect(source).toContain("BOOT_LAYER_EXIT_MS")
    expect(source).toContain("exitTimer")
    expect(source).toContain("window.setTimeout")
    expect(source).toContain("window.clearTimeout")
    expect(source).toContain('bootLayer.hidden = true')
    expect(source).toContain("document.documentElement.setAttribute(BOOT_LAYER_COMPLETE_ATTRIBUTE, \"true\")")
    expect(source).toContain('bootLayer.setAttribute("aria-hidden", "true")')
  })

  it("does not use mainContent.textContent as a handoff signal", () => {
    // Strip comments before checking — comments explaining the rationale are acceptable.
    const codeOnly = source.replace(/\/\/.*$/gm, "").replace(/\/\*[\s\S]*?\*\//g, "")
    expect(codeOnly).not.toMatch(/mainContent\.textContent/)
    expect(codeOnly).not.toMatch(/getElementById\("main-content"\)/)
  })

  it("inline handoff script does not use mainContent.textContent as a handoff signal", () => {
    // Extract the bootHandoffScript template literal and verify it doesn't read textContent
    const handoffScriptMatch = layoutSource.match(/const bootHandoffScript = `([\s\S]*?)`/)
    expect(handoffScriptMatch).not.toBeNull()
    const handoffScript = handoffScriptMatch![1]
    // Strip comments before checking
    const codeOnly = handoffScript.replace(/\/\/.*$/gm, "")
    expect(codeOnly).not.toMatch(/mainContent\.textContent/)
    expect(codeOnly).not.toMatch(/getElementById\("main-content"\)/)
  })

  it("hands boot only to the temporary app-shell warmup owner, not its inner chrome", () => {
    expect(source).toContain("APP_SHELL_WARMUP_SELECTOR")
    expect(source).toContain("isInsideAppShellWarmup")
    expect(source).toContain("isAppShellWarmupOwner")
    expect(source).toContain("element.closest(APP_SHELL_WARMUP_SELECTOR)")
    expect(source).toContain("if (isInsideAppShellWarmup(element) && !isAppShellWarmupOwner(element)) return false")
    expect(source).not.toContain("[data-slot='app-shell-warmup'], [data-slot='sidebar-wrapper']")

    const handoffScriptMatch = layoutSource.match(/const bootHandoffScript = `([\s\S]*?)`/)
    expect(handoffScriptMatch).not.toBeNull()
    const handoffScript = handoffScriptMatch![1]
    expect(handoffScript).toContain("APP_SHELL_WARMUP_SELECTOR")
    expect(handoffScript).toContain("isInsideAppShellWarmup")
    expect(handoffScript).toContain("isAppShellWarmupOwner")
    expect(handoffScript).toContain("element.closest(APP_SHELL_WARMUP_SELECTOR)")
    expect(handoffScript).toContain("if (isInsideAppShellWarmup(element) && !isAppShellWarmupOwner(element)) return false")
    expect(handoffScript).not.toContain("[data-slot='app-shell-warmup'], [data-slot='sidebar-wrapper']")
  })

  it("hides boot immediately when handing to a route or workspace loading owner", () => {
    expect(source).toContain("getHandoffReadiness")
    expect(source).toContain("immediate: hasLoadingOwner")
    expect(source).toContain("startExit({ immediate: readiness.immediate })")

    const handoffScriptMatch = layoutSource.match(/const bootHandoffScript = `([\s\S]*?)`/)
    expect(handoffScriptMatch).not.toBeNull()
    const handoffScript = handoffScriptMatch![1]
    expect(handoffScript).toContain("getHandoffReadiness")
    expect(handoffScript).toContain("immediate: hasLoadingOwner")
    expect(handoffScript).toContain("startExit({ immediate: readiness.immediate })")
    expect(handoffScript).toContain('document.documentElement.setAttribute("data-boot-layer-complete", "true")')
  })

  it("is wired directly after the boot layer without animation runtimes", () => {
    const bootLayer = layoutSource.indexOf('id="lunafox-boot-layer"')
    const handoffScript = layoutSource.indexOf("dangerouslySetInnerHTML={{ __html: bootHandoffScript }}")
    const controller = layoutSource.indexOf("<BootLayerController />")
    const intlProvider = layoutSource.indexOf("<NextIntlClientProvider")

    expect(layoutSource).toContain('import { BootLayerController } from "@/components/boot-layer-controller"')
    expect(layoutSource).toContain("const bootHandoffScript =")
    expect(layoutSource).toContain("dangerouslySetInnerHTML={{ __html: bootHandoffScript }}")
    expect(layoutSource).toContain("data-boot-exit-started")
    expect(handoffScript).toBeGreaterThan(bootLayer)
    expect(handoffScript).toBeLessThan(controller)
    expect(controller).toBeGreaterThan(bootLayer)
    expect(controller).toBeLessThan(intlProvider)
    expect(source).not.toContain("lottie")
    expect(source).not.toContain("framer-motion")
    expect(source).not.toContain("motion/react")
    expect(source).not.toContain("gsap")
  })

  it("allows the boot layer element to differ after pre-hydration handoff script mutation", () => {
    const bootLayerMarkup = layoutSource.match(
      /<div\s+id="lunafox-boot-layer"[\s\S]*?>/,
    )

    expect(bootLayerMarkup).not.toBeNull()
    expect(bootLayerMarkup![0]).toContain("suppressHydrationWarning")
  })

  it("keeps boot motion in CSS with reduced-motion support", () => {
    expect(layoutSource).toContain(".lunafox-boot-layer")
    expect(layoutSource).toContain(".lunafox-boot-loader")
    expect(layoutSource).toContain(".lunafox-boot-loader__container")
    expect(layoutSource).toContain("@media (prefers-reduced-motion: reduce)")
    expect(layoutSource).toContain(".lunafox-boot-layer--leaving")
    expect(layoutSource).toContain('html[data-boot-layer-complete="true"] .lunafox-boot-layer')
    expect(layoutSource).toContain("transition:")
    expect(layoutSource).toContain("opacity 220ms")
  })

  it("keeps boot rose motion phase continuous across document reloads", () => {
    expect(layoutSource).toContain("const bootTimelineOrigin = Date.now() - performance.now();")
    expect(layoutSource).toContain("function getBootTimelineTime(now = performance.now())")
    expect(layoutSource).toContain("draw(getBootTimelineTime());")
    expect(layoutSource).toContain("draw(getBootTimelineTime(now));")
    expect(layoutSource).not.toContain("const startedAt = performance.now();")
    expect(layoutSource).not.toContain("draw(0);")
    expect(layoutSource).not.toContain("draw(now - startedAt);")
  })
})
