import { beforeEach, describe, expect, it } from "vitest"
import {
  getMockApiKeySettings,
  resetMockApiKeySettings,
  updateMockApiKeySettings,
} from "@/mock/data/api-key-settings"
import type { ApiKeySettingsUpdateRequest } from "@/types/api-key-settings.types"

describe("API key settings mock data", () => {
  beforeEach(() => {
    resetMockApiKeySettings()
  })

  it("does not expose GitLab as a Subfinder provider", () => {
    const settings = getMockApiKeySettings()

    expect(settings.definitions.map((definition) => definition.key)).not.toContain("gitlab")
    expect(settings.providers).not.toHaveProperty("gitlab")
  })

  it("preserves a configured provider when a later update submits another provider", () => {
    const firstUpdate: ApiKeySettingsUpdateRequest = {
      providers: { alienvault: { enabled: true, values: { apiKey: "alienvault-key" } } },
    }
    const secondUpdate: ApiKeySettingsUpdateRequest = {
      providers: { bevigil: { enabled: true, values: { apiKey: "bevigil-key" } } },
    }

    updateMockApiKeySettings(firstUpdate)
    updateMockApiKeySettings(secondUpdate)

    const settings = getMockApiKeySettings()
    expect(settings.providers.alienvault.values.apiKey).toMatchObject({ configured: true })
    expect(settings.providers.bevigil.values.apiKey).toMatchObject({ configured: true })
  })
})
