export const GITHUB_REPO = "yyhuni/lunafox"
export const GITHUB_REPO_URL = `https://github.com/${GITHUB_REPO}`

const GITHUB_API_BASE = "https://api.github.com"
const GITHUB_OPEN_ISSUES_QUERY = `repo:${GITHUB_REPO} type:issue state:open`

export const GITHUB_REPO_API_URL = `${GITHUB_API_BASE}/repos/${GITHUB_REPO}`

// The repository endpoint folds open pull requests into `open_issues_count`, so the
// issues-only total has to come from search with an explicit `type:issue` filter.
export const GITHUB_REPO_OPEN_ISSUES_API_URL = `${GITHUB_API_BASE}/search/issues?q=${encodeURIComponent(GITHUB_OPEN_ISSUES_QUERY)}&per_page=1`

export type GithubRepoSnapshot = {
  name: string
  fullName: string
  htmlUrl: string
  description: string | null
  stars: number
  forks: number
  watchers: number
  issues: number
}

/** Shape returned by `/api/github/repo` and persisted in session storage. */
export function parseGithubRepoSnapshot(data: unknown): GithubRepoSnapshot {
  if (!data || typeof data !== "object") {
    throw new Error("Invalid GitHub repository response")
  }

  const repo = data as Record<string, unknown>

  if (
    typeof repo.name === "string" &&
    typeof repo.fullName === "string" &&
    typeof repo.htmlUrl === "string" &&
    (typeof repo.description === "string" || repo.description === null) &&
    typeof repo.stars === "number" &&
    typeof repo.forks === "number" &&
    typeof repo.watchers === "number" &&
    typeof repo.issues === "number"
  ) {
    return repo as GithubRepoSnapshot
  }

  throw new Error("Invalid GitHub repository response")
}

/** Combines the raw repository payload with its issues-only search total. */
export function parseGithubRepoResponse(repoData: unknown, openIssuesData: unknown): GithubRepoSnapshot {
  if (!repoData || typeof repoData !== "object") {
    throw new Error("Invalid GitHub repository response")
  }

  const repo = repoData as Record<string, unknown>

  if (
    typeof repo.name !== "string" ||
    typeof repo.full_name !== "string" ||
    typeof repo.html_url !== "string" ||
    typeof repo.stargazers_count !== "number" ||
    typeof repo.forks_count !== "number" ||
    typeof repo.subscribers_count !== "number"
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
    issues: parseGithubOpenIssueCount(openIssuesData),
  }
}

function parseGithubOpenIssueCount(data: unknown): number {
  if (!data || typeof data !== "object") {
    throw new Error("Invalid GitHub issues search response")
  }

  const search = data as Record<string, unknown>

  if (typeof search.total_count !== "number") {
    throw new Error("Invalid GitHub issues search response")
  }

  return search.total_count
}
