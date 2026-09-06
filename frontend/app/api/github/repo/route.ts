import { NextResponse } from "next/server"

type GithubRepoSnapshot = {
  name: string
  fullName: string
  htmlUrl: string
  description: string | null
  stars: number
  forks: number
  watchers: number
  issues: number
}

const GITHUB_REPO = "yyhuni/xingrin"
const CACHE_CONTROL = "public, s-maxage=300, stale-while-revalidate=1800"

export async function GET() {
  try {
    const headers: Record<string, string> = {
      "Accept": "application/vnd.github+json",
      "User-Agent": "LunaFox",
    }

    if (process.env.GITHUB_TOKEN) {
      headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN}`
    }

    const response = await fetch(`https://api.github.com/repos/${GITHUB_REPO}`, {
      headers,
      next: { revalidate: 300 },
    })

    if (!response.ok) {
      return NextResponse.json(
        { error: "Failed to fetch GitHub repository" },
        { status: response.status, headers: { "Cache-Control": "no-store" } }
      )
    }

    return NextResponse.json(parseGithubRepoSnapshot(await response.json()), {
      headers: { "Cache-Control": CACHE_CONTROL },
    })
  } catch {
    return NextResponse.json(
      { error: "Failed to fetch GitHub repository" },
      { status: 502, headers: { "Cache-Control": "no-store" } }
    )
  }
}

function parseGithubRepoSnapshot(data: unknown): GithubRepoSnapshot {
  if (!data || typeof data !== "object") {
    throw new Error("Invalid GitHub repository response")
  }

  const repo = data as Record<string, unknown>

  if (
    typeof repo.name !== "string" ||
    typeof repo.full_name !== "string" ||
    typeof repo.html_url !== "string" ||
    typeof repo.stargazers_count !== "number" ||
    typeof repo.forks_count !== "number" ||
    typeof repo.subscribers_count !== "number" ||
    typeof repo.open_issues_count !== "number"
  ) {
    throw new Error("Invalid GitHub repository response")
  }

  return {
    name: repo.name,
    fullName: repo.full_name,
    htmlUrl: repo.html_url,
    description: typeof repo.description === "string" && repo.description.trim().length > 0 ? repo.description : null,
    stars: repo.stargazers_count,
    forks: repo.forks_count,
    watchers: repo.subscribers_count,
    issues: repo.open_issues_count,
  }
}
