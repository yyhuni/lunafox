import { describe, expect, it } from "vitest"
import {
  GITHUB_REPO,
  GITHUB_REPO_API_URL,
  GITHUB_REPO_OPEN_ISSUES_API_URL,
  parseGithubRepoResponse,
  parseGithubRepoSnapshot,
} from "@/lib/github-repo-snapshot"

const repositoryPayload = {
  name: "lunafox",
  full_name: GITHUB_REPO,
  html_url: `https://github.com/${GITHUB_REPO}`,
  description: "Attack surface management platform",
  stargazers_count: 17,
  forks_count: 2,
  subscribers_count: 1,
  open_issues_count: 9,
}

const openIssuesPayload = { total_count: 0, incomplete_results: false, items: [] }

describe("github repo snapshot", () => {
  it("reports the issues-only total instead of the pull-request-inclusive repository count", () => {
    const snapshot = parseGithubRepoResponse(repositoryPayload, openIssuesPayload)

    expect(snapshot.issues).toBe(0)
    expect(snapshot.issues).not.toBe(repositoryPayload.open_issues_count)
  })

  it("searches with an explicit issue type filter so pull requests are excluded", () => {
    expect(GITHUB_REPO_API_URL).toBe(`https://api.github.com/repos/${GITHUB_REPO}`)
    expect(GITHUB_REPO_OPEN_ISSUES_API_URL).toContain("/search/issues")
    expect(decodeURIComponent(GITHUB_REPO_OPEN_ISSUES_API_URL)).toContain("type:issue")
    expect(decodeURIComponent(GITHUB_REPO_OPEN_ISSUES_API_URL)).toContain("state:open")
    expect(GITHUB_REPO_OPEN_ISSUES_API_URL).toContain("per_page=1")
  })

  it("normalizes the fields consumed by the repository card", () => {
    expect(parseGithubRepoResponse(repositoryPayload, openIssuesPayload)).toEqual({
      name: "lunafox",
      fullName: GITHUB_REPO,
      htmlUrl: `https://github.com/${GITHUB_REPO}`,
      description: "Attack surface management platform",
      stars: 17,
      forks: 2,
      watchers: 1,
      issues: 0,
    })
  })

  it("drops blank descriptions and keeps an explicit null", () => {
    const snapshot = parseGithubRepoResponse({ ...repositoryPayload, description: "   " }, openIssuesPayload)

    expect(snapshot.description).toBeNull()
  })

  it("fast-fails when the issues search total is missing", () => {
    expect(() => parseGithubRepoResponse(repositoryPayload, { incomplete_results: false })).toThrow()
    expect(() => parseGithubRepoResponse(repositoryPayload, null)).toThrow()
    expect(() => parseGithubRepoResponse(repositoryPayload, { total_count: "9" })).toThrow()
  })

  it("fast-fails on payloads missing required repository fields", () => {
    expect(() => parseGithubRepoResponse({ ...repositoryPayload, subscribers_count: undefined }, openIssuesPayload)).toThrow()
    expect(() => parseGithubRepoResponse({ ...repositoryPayload, forks_count: undefined }, openIssuesPayload)).toThrow()
    expect(() => parseGithubRepoResponse(null, openIssuesPayload)).toThrow()
  })

  it("accepts the normalized snapshot shape served by the same-origin API", () => {
    const snapshot = {
      name: "lunafox",
      fullName: GITHUB_REPO,
      htmlUrl: `https://github.com/${GITHUB_REPO}`,
      description: null,
      stars: 17,
      forks: 2,
      watchers: 1,
      issues: 0,
    }

    expect(parseGithubRepoSnapshot(snapshot)).toEqual(snapshot)
  })

  it("rejects raw GitHub payloads and incomplete normalized snapshots", () => {
    expect(() => parseGithubRepoSnapshot(repositoryPayload)).toThrow()
    expect(() => parseGithubRepoSnapshot({ name: "lunafox" })).toThrow()
    expect(() => parseGithubRepoSnapshot(null)).toThrow()
  })
})
