import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/mcp/mcp-connection-flow.tsx"), "utf8")

describe("MCP connection flow contract", () => {
  it("keeps the connection animation automatic without a pause control", () => {
    expect(source).toContain("data-slot=\"mcp-flow-packets\"")
    expect(source).not.toContain("PauseIcon")
    expect(source).not.toContain("ResumeIcon")
    expect(source).not.toContain("pauseFlow")
    expect(source).not.toContain("resumeFlow")
    expect(source).not.toContain("<Button")
    expect(source).not.toContain("<figcaption")
  })
})
