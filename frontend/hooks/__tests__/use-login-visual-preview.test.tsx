import { waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import type { LoginVisualMedia } from "@/types/login-visual.types"

const serviceMocks = vi.hoisted(() => ({
  getPreview: vi.fn(),
}))

vi.mock("@/services/login-visual.service", () => ({
  LoginVisualService: serviceMocks,
}))

vi.mock("@/lib/login-visual-unlock", () => ({
  isLoginVisualUnlocked: () => false,
  unlockLoginVisual: vi.fn(),
  useLoginVisualUnlocked: () => false,
}))

import { useLoginVisualPreview } from "@/hooks/use-login-visual"

const previewURL = "/v1/settings/loginVisual:preview"
const image: LoginVisualMedia = {
  kind: "image",
  contentType: "image/png",
  sizeBytes: 1024,
}

describe("login visual private preview hook", () => {
  const createObjectURL = vi.fn()
  const revokeObjectURL = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    createObjectURL.mockReturnValue("blob:private-login-preview")
    vi.stubGlobal("URL", { createObjectURL, revokeObjectURL })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("turns the authenticated preview Blob into a renderable media URL and releases it on unmount", async () => {
    const preview = new Blob(["private login visual"], { type: "image/png" })
    serviceMocks.getPreview.mockResolvedValue(preview)

    const { result, unmount } = renderHookWithProviders(() => useLoginVisualPreview(image, previewURL))

    await waitFor(() => expect(result.current.source).toBe("blob:private-login-preview"))
    expect(serviceMocks.getPreview).toHaveBeenCalledWith(previewURL)
    expect(createObjectURL).toHaveBeenCalledWith(preview)

    unmount()

    expect(revokeObjectURL).toHaveBeenCalledWith("blob:private-login-preview")
  })

  it("keeps an inline mock preview out of the authenticated media request", async () => {
    const inlinePreview = "data:image/png;base64,bW9jaw=="

    const { result } = renderHookWithProviders(() => useLoginVisualPreview(image, inlinePreview))

    await waitFor(() => expect(result.current.source).toBe(inlinePreview))
    expect(serviceMocks.getPreview).not.toHaveBeenCalled()
    expect(createObjectURL).not.toHaveBeenCalled()
  })

  it("reloads the protected preview when the accepted media changes", async () => {
    const secondImage: LoginVisualMedia = { ...image, sizeBytes: 2048 }
    serviceMocks.getPreview
      .mockResolvedValueOnce(new Blob(["first"], { type: "image/png" }))
      .mockResolvedValueOnce(new Blob(["second"], { type: "image/png" }))
    createObjectURL
      .mockReturnValueOnce("blob:first-login-preview")
      .mockReturnValueOnce("blob:second-login-preview")

    const { result, rerender } = renderHookWithProviders(
      ({ media }: { media: LoginVisualMedia }) => useLoginVisualPreview(media, previewURL),
      { initialProps: { media: image } }
    )

    await waitFor(() => expect(result.current.source).toBe("blob:first-login-preview"))

    rerender({ media: secondImage })

    await waitFor(() => expect(result.current.source).toBe("blob:second-login-preview"))
    expect(serviceMocks.getPreview).toHaveBeenCalledTimes(2)
  })
})
