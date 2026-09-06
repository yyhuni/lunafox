import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({ get: vi.fn() }))

vi.mock("@/lib/api-client", () => ({ api: apiMocks }))

import { api } from "@/lib/api-client"
import { SearchService } from "@/services/search.service"

describe("search.service contract", () => {
  beforeEach(() => vi.clearAllMocks())

  it("uses the sole camelCase global search endpoint without legacy pagination", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { results: [{ id: 7, name: "targets/3/websites/7" }], nextPageToken: "next" },
    } as never)

    await SearchService.search({
      q: 'statusCode=="200"',
      assetType: "website",
      pageSize: 10,
      pageToken: "previous",
    })

    expect(api.get).toHaveBeenCalledWith("/assets:search", {
      params: {
        q: 'statusCode=="200"',
        assetType: "website",
        pageSize: 10,
        pageToken: "previous",
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[0]).not.toContain("/assets/search")
    expect(vi.mocked(api.get).mock.calls[0]?.[1]).not.toMatchObject({
      params: expect.objectContaining({ asset_type: expect.anything(), page: expect.anything() }),
    })
  })
})
