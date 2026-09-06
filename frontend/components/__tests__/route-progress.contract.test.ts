import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/route-progress.tsx"), "utf8")

describe("route-progress contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function RouteProgress")
    expect(source).toContain("from \"react\"")
  })

  it("owns nprogress configuration through the LunaFox adapter", () => {
    expect(source).toContain("from \"nprogress\"")
    expect(source).not.toContain("from \"nextjs-toploader\"")
    expect(source).not.toContain("nextjs-toploader/app")
    expect(source).toContain("const ROUTE_PROGRESS_COLOR = \"var(--color-highlight)\"")
    expect(source).toContain("const ROUTE_PROGRESS_HEIGHT = 3")
    expect(source).toContain("showSpinner: false")
    expect(source).toContain("trickle: true")
    expect(source).toContain("trickleSpeed: ROUTE_PROGRESS_TRICKLE_SPEED")
    expect(source).toContain("template: ROUTE_PROGRESS_TEMPLATE")
    expect(source).toContain("z-index: ${ROUTE_PROGRESS_Z_INDEX}")
    expect(source).toContain("ROUTE_PROGRESS_TEMPLATE")
    expect(source).toContain("route-progress__track")
    expect(source).toContain("aria-hidden=\"true\"")
    expect(source).toContain("prefers-reduced-motion: reduce")
    expect(source).not.toContain("setProgress")
  })

  it("preserves the nprogress structural selector contract for the custom template", () => {
    expect(source).toContain('role="bar"')
  })

  it("keeps navigation progress in one top-level overlay layer", () => {
    expect(source).toContain("#nprogress {")
    expect(source).toContain("position: fixed;")
    expect(source).toContain("inset: 0 0 auto 0;")
    expect(source).toContain("isolation: isolate;")
    expect(source).toContain("#nprogress .route-progress__track")
    expect(source).toContain("#nprogress .bar")
    expect(source).toContain("position: absolute;")
    expect(source).toContain("will-change: transform;")
  })

  it("uses theme tokens instead of raw package defaults", () => {
    expect(source).toContain("var(--color-highlight)")
    expect(source).not.toContain("\"#29d\"")
    expect(source).not.toContain("bg-[var(--highlight)]")
    expect(source).not.toContain("to-[var(--highlight)]")
  })

  it("removes the old event bridge while preserving project navigation helpers", () => {
    expect(source).not.toContain("lunafox:route-progress-start")
    expect(source).toContain("pushWithRouteProgress")
    expect(source).toContain("replaceWithRouteProgress")
  })

  it("uses production-grade display timing instead of fixed compatibility fallbacks", () => {
    expect(source).toContain("const ROUTE_PROGRESS_SHOW_DELAY_MS = 240")
    expect(source).toContain("const ROUTE_PROGRESS_MIN_VISIBLE_MS = 240")
    expect(source).toContain("scheduleRouteProgressCompletion")
    expect(source).toContain("usePathname()")
    expect(source).toContain("useSearchParams()")
    expect(source).not.toContain("1200")
    expect(source).not.toContain("loader.done(true)")
  })

  it("lets route and workspace loading owners supersede the top navigation bar", () => {
    expect(source).toContain("ROUTE_PROGRESS_SUPERSEDING_LOADING_LAYERS")
    expect(source).toContain('"route"')
    expect(source).toContain('"workspace"')
    expect(source).toContain('"section"')
    expect(source).toContain("ROUTE_PROGRESS_LOADING_OWNER_SELECTOR")
    expect(source).toContain("hasVisibleRouteLoadingOwner")
    expect(source).toContain("cancelRouteProgressForVisibleOwner")
    expect(source).toContain("owner === \"initial-boot\"")
    expect(source).toContain('phase === "content"')
    expect(source).toContain("NProgress.remove()")
    expect(source).toContain("new MutationObserver(syncRouteProgressOwner)")
    expect(source).toContain('"data-loading-owner"')
    expect(source).toContain('"data-loading-phase"')
  })
})
