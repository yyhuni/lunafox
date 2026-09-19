import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const routeSource = readFileSync(path.resolve(process.cwd(), "app/api/github/repo/route.ts"), "utf8")
const snapshotSource = readFileSync(path.resolve(process.cwd(), "lib/github-repo-snapshot.ts"), "utf8")

describe("github repo api route contract", () => {
  it("proxies the public GitHub repository API with server-side caching headers", () => {
    expect(routeSource).toContain('export async function GET()')
    expect(routeSource).toContain('"Accept": "application/vnd.github+json"')
    expect(routeSource).toContain('"User-Agent": "LunaFox"')
    expect(routeSource).toContain("NextResponse.json")
    expect(routeSource).toContain('"Cache-Control"')
    expect(routeSource).toContain("s-maxage")
    expect(routeSource).toContain("stale-while-revalidate")
    expect(routeSource).toContain("revalidate: GITHUB_REVALIDATE_SECONDS")
  })

  it("collects the issues-only count from search instead of the pull-request-inclusive repository field", () => {
    expect(snapshotSource).toContain('const GITHUB_API_BASE = "https://api.github.com"')
    expect(snapshotSource).toContain("/repos/${GITHUB_REPO}`")
    expect(snapshotSource).toContain("/search/issues?q=")
    expect(snapshotSource).toContain("type:issue")
    expect(snapshotSource).toContain("state:open")
    expect(snapshotSource).not.toContain("repo.open_issues_count")
    expect(routeSource).toContain("GITHUB_REPO_API_URL")
    expect(routeSource).toContain("GITHUB_REPO_OPEN_ISSUES_API_URL")
  })

  it("returns only the normalized fields consumed by the client card", () => {
    expect(routeSource).toContain("parseGithubRepoResponse")
    expect(snapshotSource).toContain("export function parseGithubRepoSnapshot")
    expect(snapshotSource).toContain("export function parseGithubRepoResponse")
    expect(snapshotSource).toContain("fullName")
    expect(snapshotSource).toContain("htmlUrl")
    expect(snapshotSource).toContain("description")
    expect(snapshotSource).toContain("stars")
    expect(snapshotSource).toContain("forks")
    expect(snapshotSource).toContain("watchers")
    expect(snapshotSource).toContain("issues")
    expect(routeSource).not.toContain("process.env.NEXT_PUBLIC_GITHUB")
  })
})
