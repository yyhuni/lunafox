import { describe, expect, it } from "vitest"
import { existsSync } from "node:fs"
import path from "node:path"

function appFile(relativePath: string) {
  return path.resolve(process.cwd(), "app", relativePath)
}

describe("route canonicalization contract", () => {
  it("removes legacy redirect-only and deprecated route files", () => {
    expect(existsSync(appFile("targets/[id]/page.tsx"))).toBe(false)
    expect(existsSync(appFile("targets/[id]/details/page.tsx"))).toBe(false)
    expect(existsSync(appFile("scan/history/[id]/page.tsx"))).toBe(false)
    expect(existsSync(appFile("tools/fingerprints/page.tsx"))).toBe(false)
    expect(existsSync(appFile("tools/config/page.tsx"))).toBe(false)
    expect(existsSync(appFile("tools/config/custom/page.tsx"))).toBe(false)
    expect(existsSync(appFile("tools/config/opensource/page.tsx"))).toBe(false)
  })

  it("keeps only pluralized subdomains child routes", () => {
    expect(existsSync(appFile("targets/[id]/subdomains/page.tsx"))).toBe(true)
    expect(existsSync(appFile("scan/history/[id]/subdomains/page.tsx"))).toBe(true)
    expect(existsSync(appFile("targets/[id]/subdomain/page.tsx"))).toBe(false)
    expect(existsSync(appFile("scan/history/[id]/subdomain/page.tsx"))).toBe(false)
  })
})
