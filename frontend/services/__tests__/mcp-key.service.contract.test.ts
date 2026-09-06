import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "services/mcp-key.service.ts"), "utf8")

describe("mcp-key.service contract", () => {
  it("uses the JWT API client and canonical identity actions", () => {
    expect(source).toContain('from "@/lib/api-client"')
    expect(source).toContain('"/users/me/mcpKey"')
    expect(source).toContain('"/users/me:generateMcpKey"')
    expect(source).toContain("api.get<McpKeyStatus>")
    expect(source).toContain("api.post<McpKeyGeneration>")
  })

  it("does not own mock mode or persist a plaintext key", () => {
    expect(source).not.toContain("@/mock")
    expect(source).not.toContain("localStorage")
    expect(source).not.toContain("sessionStorage")
  })
})
