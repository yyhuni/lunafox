import { describe, expect, it } from "vitest"
import {
  getApiKeyProviderStatus,
  toApiKeyProviderFormState,
  toApiKeyProviderUpdate,
  updateApiKeyProviderFormState,
} from "@/lib/api-key-settings-form"
import type { ApiKeySettings } from "@/types/api-key-settings.types"

const settings: ApiKeySettings = {
  definitions: [
    {
      key: "alienvault",
      sourceName: "alienvault",
      displayName: "alienvault",
      keyRequirement: "required",
      default: true,
      recursive: true,
      fields: [{ name: "apiKey", secret: true, required: true }],
    },
    {
      key: "bevigil",
      sourceName: "bevigil",
      displayName: "bevigil",
      keyRequirement: "required",
      default: true,
      recursive: false,
      fields: [{ name: "apiKey", secret: true, required: true }],
    },
  ],
  providers: {
    alienvault: {
      enabled: true,
      status: "configured",
      values: { apiKey: { configured: true, maskedValue: "********" } },
    },
    bevigil: {
      enabled: false,
      status: "unconfigured",
      values: { apiKey: { configured: false } },
    },
  },
}

describe("api-key settings form state", () => {
  it("keeps configured secret metadata without placing a masked value in editable state", () => {
    const formData = toApiKeyProviderFormState(settings)

    expect(formData.alienvault.values.apiKey).toBe("")
    expect(formData.alienvault.initialValues.apiKey).toEqual({
      configured: true,
      maskedValue: "********",
    })
    expect(getApiKeyProviderStatus(settings.definitions[0], formData, new Set())).toBe("enabled")
  })

  it("omits untouched secrets and preserves an explicit clear operation", () => {
    const formData = toApiKeyProviderFormState(settings)
    const enabledOnly = updateApiKeyProviderFormState(formData, "alienvault", "enabled", false)
    expect(toApiKeyProviderUpdate(settings.definitions[0], enabledOnly.alienvault)).toEqual({
      enabled: false,
      values: {},
    })

    const cleared = updateApiKeyProviderFormState(formData, "alienvault", "apiKey", "")
    expect(toApiKeyProviderUpdate(settings.definitions[0], cleared.alienvault)).toEqual({
      enabled: true,
      values: { apiKey: "" },
    })
  })

  it("serializes a replacement secret for only the edited provider", () => {
    const formData = toApiKeyProviderFormState(settings)
    const updated = updateApiKeyProviderFormState(formData, "bevigil", "apiKey", "bevigil-key")

    expect(toApiKeyProviderUpdate(settings.definitions[0], updated.alienvault)).toEqual({
      enabled: true,
      values: {},
    })
    expect(toApiKeyProviderUpdate(settings.definitions[1], updated.bevigil)).toEqual({
      enabled: false,
      values: { apiKey: "bevigil-key" },
    })
  })
})
