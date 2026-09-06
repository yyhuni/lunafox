import { describe, expect, it } from "vitest"

import { getMockOrganizations } from "../organizations"

describe("organization mock query behavior", () => {
  it("matches the backend displayName filter against organization names", () => {
    const response = getMockOrganizations({ filter: 'displayName="cloud"' })

    expect(response.results.map((organization) => organization.name)).toContain("CloudNine Hosting")
  })

  it("does not expose description matching that the backend does not support", () => {
    const response = getMockOrganizations({ filter: 'displayName="cloud computing"' })

    expect(response.results).toEqual([])
    expect(response.total).toBe(0)
  })
})
