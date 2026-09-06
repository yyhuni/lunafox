import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/faulty-terminal.tsx"), "utf8")

describe("faulty-terminal contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default function FaultyTerminal")
    expect(source).toContain("className")
    expect(source).toContain("from 'ogl'")
    expect(source).toContain("cssColorToRgb")
    expect(source).toContain("getImageData")
    expect(source).toContain("MutationObserver")
    expect(source).toContain("backgroundColor")
    expect(source).toContain("uBackground")
    expect(source).toContain("maxFps?: number")
    expect(source).toContain("1000 / maxFps")
    expect(source).toContain("lastRenderTimeRef")
    expect(source).toContain("renderer.render({ scene: mesh })")
    expect(source).toContain("onFirstFrame?: () => void")
    expect(source).toContain("firstVisibleFrameProgress")
    expect(source).toContain("hasRenderedFirstFrameRef")
    expect(source).toContain("onFirstFrame?.()")
  })
})
