import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-mcp-key.ts"), "utf8")

describe("use-mcp-key contract", () => {
  it("keeps plaintext out of React Query while caching only key status", () => {
    expect(source).toContain("useMcpKeyStatus")
    expect(source).toContain("useGenerateMcpKey")
    expect(source).toContain("onGenerated(generation)")
    expect(source).toContain("return toStatus(generation)")
    expect(source).toContain("queryClient.setQueryData(mcpKeyKeys.status, status)")
    expect(source).not.toContain("localStorage")
    expect(source).not.toContain("sessionStorage")
  })
})
