import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/loading/content-reveal.tsx"), "utf8")
const globals = readFileSync(path.resolve(process.cwd(), "app/globals.css"), "utf8")

describe("content-reveal contract", () => {
  it("owns shared subtle reveal with explicit owner metadata", () => {
    expect(source).toContain("export function ContentReveal")
    expect(source).toContain("owner: string")
    expect(source).toContain("layer?: LoadingLayer")
    expect(source).toContain("intent?: LoadingIntent")
    expect(source).toContain("getLoadingOwnerAttributes")
    expect(source).toContain('layer = "route"')
    expect(source).toContain('intent = "route"')
    expect(source).toContain("getLoadingOwnerAttributes({ owner, layer, intent })")
    expect(source).toContain("data-slot=\"content-reveal\"")
    expect(source).toContain("data-loading-phase=\"content\"")
    expect(source).toContain("loading-content-reveal")
    expect(source).toContain("ContentReveal requires a non-empty owner.")
  })

  it("uses a short opacity-first reveal with reduced-motion fallback", () => {
    expect(globals).toContain("--motion-ease-standard: cubic-bezier(0.22, 1, 0.36, 1);")
    expect(globals).toContain("--motion-duration-reveal: 180ms;")
    expect(globals).toContain(".loading-content-reveal")
    expect(globals).toContain(
      "animation: loading-content-reveal var(--motion-duration-reveal) var(--motion-ease-standard) both;"
    )
    expect(globals).toContain("opacity: 0.68")
    expect(globals).not.toContain("loading-content-reveal 280ms")
    expect(globals).not.toContain("loading-content-reveal 180ms ease-out")
    expect(globals).not.toContain("translateY(2px)")
    expect(globals).toContain(".loading-content-reveal,")
    expect(globals).toContain("animation: none !important;")
  })

  it("keeps route content non-visible while a descendant blocks the boot handoff", () => {
    expect(globals).toContain('.loading-content-reveal:has([data-boot-handoff-pending="true"])')
    expect(globals).toContain("visibility: hidden;")
  })
})
