import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/shared/navigation/route-content-transition.tsx"),
  "utf8"
)
const globals = readFileSync(path.resolve(process.cwd(), "app/globals.css"), "utf8")

describe("route content transition contract", () => {
  it("tracks committed pathname changes inside the protected shell", () => {
    expect(source).toContain('import { usePathname } from "next/navigation"')
    expect(source).toContain("RouteContentTransitionProvider")
    expect(source).toContain("previousPathnameRef")
    expect(source).toContain("useIsomorphicLayoutEffect")
    expect(source).toContain("setNavigationId")
    expect(source).toContain("navigationId")
  })

  it("keys only the shared content-region wrapper from navigation identity", () => {
    expect(source).toContain("export function RouteContentTransition")
    expect(source).toContain('data-slot="route-content-transition"')
    expect(source).toContain("key={navigationId}")
    expect(source).toContain('navigationId > 0 && "route-content-transition"')
    expect(source).not.toContain("startViewTransition")
    expect(source).not.toContain("translate")
    expect(source).not.toContain("scale")
  })

  it("uses the shared reveal tier and reduced-motion fallback", () => {
    expect(globals).toContain(".route-content-transition")
    expect(globals).toContain(
      "animation: route-content-transition var(--motion-duration-reveal) var(--motion-ease-standard) both;"
    )
    expect(globals).toContain("@keyframes route-content-transition")
    expect(globals).toContain("  .route-content-transition,")
    expect(globals).toContain("animation: none !important;")
  })
})
