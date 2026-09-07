import { describe, expect, it } from "vitest"

import { matchesWebsiteURLScope } from "@/lib/website-scope"

const API_BASE = "http://localhost/v1"

async function getJSON<T>(path: string, filter?: string): Promise<{ response: Response; body: T }> {
  const url = new URL(`${API_BASE}${path}`)
  if (filter) url.searchParams.set("filter", filter)
  const response = await fetch(url)
  return { response, body: await response.json() as T }
}

type WebsiteResponse = {
  id: number
  name: string
  url: string
  screenshot?: {
    id: number
    name: string
    url: string
    image?: string
  }
}

type CollectionResponse<T> = {
  results: T[]
  totalSize: number
  nextPageToken?: string
}

type URLAsset = { id: number; url: string }
type IPAsset = { ip: string; hosts: string[] }

describe("Website detail mock scope contract", () => {
  it("serves canonical Website List/Get summaries and no legacy relation resource", async () => {
    const [{ response: listResponse, body: list }, { response: getResponse, body: website }, legacy] = await Promise.all([
      getJSON<CollectionResponse<WebsiteResponse>>("/targets/1/websites", undefined),
      getJSON<WebsiteResponse>("/websites/1", undefined),
      fetch(`${API_BASE}/targets/1/websiteRelations`),
    ])

    expect(listResponse.status).toBe(200)
    expect(getResponse.status).toBe(200)
    expect(list.results.find((item) => item.id === 1)?.name).toBe("targets/1/websites/1")
    expect(website).toMatchObject({
      id: 1,
      name: "targets/1/websites/1",
      screenshot: {
        name: "targets/1/screenshots/1",
        url: "https://acme.com",
      },
    })
    expect(website.screenshot).not.toHaveProperty("image")
    expect(list.results.find((item) => item.id === 2)?.screenshot).toBeUndefined()
    expect(legacy.status).toBe(501)
  })

  it("uses exact host scope for observed IPs and excludes lookalikes", async () => {
    const { response, body } = await getJSON<CollectionResponse<IPAsset>>(
      "/targets/1/hostPorts",
      'host=="api.acme.com"',
    )

    expect(response.status).toBe(200)
    expect(body.totalSize).toBe(1)
    expect(body.results).toHaveLength(1)
    expect(body.results[0]?.hosts).toContain("api.acme.com")
    expect(body.results.flatMap((item) => item.hosts)).not.toContain("api.acme.com.evil")
  })

  it("applies structured URL scope before total size and pagination in all URL asset handlers", async () => {
    const rootFilter = 'websiteUrl=="https://api.acme.com"'
    const nestedFilter = 'websiteUrl=="https://api.acme.com/a"'
    const [endpoints, directories, vulnerabilities, nested] = await Promise.all([
      getJSON<CollectionResponse<URLAsset>>("/targets/1/endpoints", rootFilter),
      getJSON<CollectionResponse<URLAsset>>("/targets/1/directories", rootFilter),
      getJSON<CollectionResponse<URLAsset>>("/targets/1/vulnerabilities", rootFilter),
      getJSON<CollectionResponse<URLAsset>>("/targets/1/endpoints", nestedFilter),
    ])

    for (const result of [endpoints, directories, vulnerabilities]) {
      expect(result.response.status).toBe(200)
      expect(result.body.totalSize).toBe(result.body.results.length)
      expect(result.body.results.every((item) => matchesWebsiteURLScope("https://api.acme.com", item.url))).toBe(true)
    }

    expect(nested.response.status).toBe(200)
    expect(nested.body.totalSize).toBe(2)
    expect(nested.body.results.map((item) => item.id).sort((left, right) => left - right)).toEqual([16, 17])
    expect(nested.body.results.some((item) => item.url === "https://api.acme.com/ab")).toBe(false)
    expect(endpoints.body.results.map((item) => item.id)).not.toContain(19)
    expect(endpoints.body.results.map((item) => item.id)).not.toContain(20)
    expect(endpoints.body.results.map((item) => item.id)).not.toContain(21)
  })

  it("uses exact raw URL equality outside the Website Scope relation view", async () => {
    const endpointURL = "https://api.acme.com/a"
    const [websites, endpoints, directories, screenshots, vulnerabilities, trailingSpace] = await Promise.all([
      getJSON<CollectionResponse<URLAsset>>("/targets/1/websites", 'url=="https://acme.com"'),
      getJSON<CollectionResponse<URLAsset>>("/targets/1/endpoints", `url=="${endpointURL}"`),
      getJSON<CollectionResponse<URLAsset>>("/targets/1/directories", 'url=="https://acme.com/admin"'),
      getJSON<CollectionResponse<URLAsset>>("/targets/1/screenshots", 'url=="https://acme.com"'),
      getJSON<CollectionResponse<URLAsset>>("/targets/1/vulnerabilities", 'url=="https://acme.com/search?q=test"'),
      getJSON<CollectionResponse<URLAsset>>("/targets/1/endpoints", `url=="${endpointURL} "`),
    ])

    for (const [result, expectedURL] of [
      [websites, "https://acme.com"],
      [endpoints, endpointURL],
      [directories, "https://acme.com/admin"],
      [screenshots, "https://acme.com"],
      [vulnerabilities, "https://acme.com/search?q=test"],
    ] as const) {
      expect(result.response.status).toBe(200)
      expect(result.body.results).toHaveLength(1)
      expect(result.body.results[0]?.url).toBe(expectedURL)
    }
    expect(trailingSpace.response.status).toBe(200)
    expect(trailingSpace.body.totalSize).toBe(0)
  })

  it("rejects URL scope when it would broaden the Website range through OR", async () => {
    const { response } = await getJSON<unknown>(
      "/targets/1/endpoints",
      'websiteUrl=="https://api.acme.com" || statusCode="200"',
    )

    expect(response.status).toBe(400)
  })
})
