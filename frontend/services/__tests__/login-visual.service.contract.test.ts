import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({
  api: apiMocks,
  ensureMockInterceptionReady: vi.fn(),
}))

import { api } from "@/lib/api-client"
import { LoginVisualService } from "@/services/login-visual.service"

describe("login visual service discoverability contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("uses the authenticated canonical discoverability custom methods", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { unlocked: false } } as never)
    vi.mocked(api.post).mockResolvedValue({ data: { unlocked: true } } as never)

    await expect(LoginVisualService.checkDiscoverability()).resolves.toEqual({ unlocked: false })
    await expect(LoginVisualService.unlockDiscoverability()).resolves.toEqual({ unlocked: true })

    expect(api.get).toHaveBeenCalledWith("/settings/loginVisual:checkDiscoverability")
    expect(api.post).toHaveBeenCalledWith("/settings/loginVisual:unlockDiscoverability")
  })

  it("loads a private preview through the authenticated API client as a Blob", async () => {
    const preview = new Blob(["private login visual"], { type: "image/png" })
    vi.mocked(api.get).mockResolvedValue({ data: preview } as never)

    await expect(LoginVisualService.getPreview("/v1/settings/loginVisual:preview")).resolves.toBe(preview)

    expect(api.get).toHaveBeenCalledWith("/settings/loginVisual:preview", { responseType: "blob" })
  })
})
