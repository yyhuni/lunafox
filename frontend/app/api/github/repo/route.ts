import { NextResponse } from "next/server"
import {
  GITHUB_REPO_API_URL,
  GITHUB_REPO_OPEN_ISSUES_API_URL,
  parseGithubRepoResponse,
} from "@/lib/github-repo-snapshot"

const CACHE_CONTROL = "public, s-maxage=300, stale-while-revalidate=1800"
const GITHUB_REVALIDATE_SECONDS = 300

export async function GET() {
  try {
    const headers: Record<string, string> = {
      "Accept": "application/vnd.github+json",
      "User-Agent": "LunaFox",
    }

    if (process.env.GITHUB_TOKEN) {
      headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN}`
    }

    const [repoResponse, openIssuesResponse] = await Promise.all([
      fetch(GITHUB_REPO_API_URL, { headers, next: { revalidate: GITHUB_REVALIDATE_SECONDS } }),
      fetch(GITHUB_REPO_OPEN_ISSUES_API_URL, { headers, next: { revalidate: GITHUB_REVALIDATE_SECONDS } }),
    ])

    if (!repoResponse.ok) {
      return githubFetchFailed(repoResponse.status)
    }

    if (!openIssuesResponse.ok) {
      return githubFetchFailed(openIssuesResponse.status)
    }

    return NextResponse.json(parseGithubRepoResponse(await repoResponse.json(), await openIssuesResponse.json()), {
      headers: { "Cache-Control": CACHE_CONTROL },
    })
  } catch {
    return githubFetchFailed(502)
  }
}

function githubFetchFailed(status: number) {
  return NextResponse.json(
    { error: "Failed to fetch GitHub repository" },
    { status, headers: { "Cache-Control": "no-store" } }
  )
}
