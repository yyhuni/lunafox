import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "types/agent-log.types.ts"), "utf8")

describe("agent-log.types contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export type AgentLogStreamName = 'stdout' | 'stderr' | string")
  })
})
