import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/api/github/repo/route.ts"), "utf8")

describe("github repo api route contract", () => {
  it("proxies the public GitHub repository API with server-side caching headers", () => {
    expect(source).toContain('export async function GET()')
    expect(source).toContain('fetch(`https://api.github.com/repos/${GITHUB_REPO}`')
    expect(source).toContain('"Accept": "application/vnd.github+json"')
    expect(source).toContain('"User-Agent": "LunaFox"')
    expect(source).toContain("NextResponse.json")
    expect(source).toContain('"Cache-Control"')
    expect(source).toContain("s-maxage")
    expect(source).toContain("stale-while-revalidate")
  })

  it("returns only the normalized fields consumed by the client card", () => {
    expect(source).toContain("parseGithubRepoSnapshot")
    expect(source).toContain("fullName")
    expect(source).toContain("htmlUrl")
    expect(source).toContain("description")
    expect(source).toContain("stars")
    expect(source).toContain("forks")
    expect(source).toContain("watchers")
    expect(source).toContain("issues")
    expect(source).not.toContain("process.env.NEXT_PUBLIC_GITHUB")
  })
})
