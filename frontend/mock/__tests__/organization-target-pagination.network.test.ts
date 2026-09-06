import { describe, expect, it } from "vitest"

const API_BASE = "http://localhost/v1"

type OrganizationTargetPage = {
  results: Array<{
    id: number
    name: string
  }>
  totalSize: number
  nextPageToken?: string
}

async function getOrganizationTargets(pageToken?: string) {
  const url = new URL(`${API_BASE}/organizations/1/targets`)
  url.searchParams.set("pageSize", "1")
  if (pageToken) {
    url.searchParams.set("pageToken", pageToken)
  }

  const response = await fetch(url)
  return {
    response,
    body: await response.json() as OrganizationTargetPage,
  }
}

describe("organization target mock cursor pagination", () => {
  it("follows a continuation token without exposing page-number metadata", async () => {
    const first = await getOrganizationTargets()

    expect(first.response.status).toBe(200)
    expect(first.body).toMatchObject({
      results: [{ id: 1, name: "targets/1" }],
      totalSize: 3,
      nextPageToken: "mock-organization-target-page-2",
    })
    expect(first.body).not.toHaveProperty("page")
    expect(first.body).not.toHaveProperty("pageSize")
    expect(first.body).not.toHaveProperty("totalPages")

    const second = await getOrganizationTargets(first.body.nextPageToken)

    expect(second.response.status).toBe(200)
    expect(second.body).toMatchObject({
      results: [{ id: 2, name: "targets/2" }],
      totalSize: first.body.totalSize,
      nextPageToken: "mock-organization-target-page-3",
    })

    const terminal = await getOrganizationTargets(second.body.nextPageToken)

    expect(terminal.response.status).toBe(200)
    expect(terminal.body).toMatchObject({
      results: [{ id: 12, name: "targets/12" }],
      totalSize: first.body.totalSize,
    })
    expect(terminal.body.nextPageToken).toBeUndefined()
  })
})
