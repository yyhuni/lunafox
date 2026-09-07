import { describe, expect, it } from "vitest"
import { getMockApiKeySettings } from "@/mock/data/api-key-settings"

describe("API key settings mock data", () => {
  it("does not expose GitLab as a Subfinder provider", () => {
    const settings = getMockApiKeySettings()

    expect(settings.definitions.map((definition) => definition.key)).not.toContain("gitlab")
    expect(settings.providers).not.toHaveProperty("gitlab")
  })
})
