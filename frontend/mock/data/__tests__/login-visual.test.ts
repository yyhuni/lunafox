import { describe, expect, it } from "vitest"

import {
  getMockLoginVisualDiscoverability,
  getMockLoginVisualSettings,
  getMockPublicLoginVisual,
  publishMockLoginVisual,
  resetMockLoginVisualDiscoverability,
  restoreMockLoginVisual,
  unlockMockLoginVisualDiscoverability,
  uploadMockLoginVisual,
} from "../login-visual"

describe("login visual mock lifecycle", () => {
  it("keeps uploads private until the draft is published", () => {
    restoreMockLoginVisual()
    uploadMockLoginVisual("image/png")

    expect(getMockLoginVisualSettings()).toMatchObject({ draft: { kind: "image" } })
    expect(getMockPublicLoginVisual()).toEqual({ kind: "builtIn" })

    publishMockLoginVisual()

    expect(getMockLoginVisualSettings()).toMatchObject({ published: { kind: "image" } })
    expect(getMockPublicLoginVisual()).toMatchObject({ kind: "image", mediaUrl: expect.any(String) })
  })

  it("restores the built-in visual for public login", () => {
    restoreMockLoginVisual()
    uploadMockLoginVisual("video/webm")
    publishMockLoginVisual()

    expect(getMockPublicLoginVisual()).toMatchObject({ kind: "video", posterUrl: expect.any(String) })

    restoreMockLoginVisual()

    expect(getMockLoginVisualSettings()).toEqual({})
    expect(getMockPublicLoginVisual()).toEqual({ kind: "builtIn" })
  })

  it("mirrors the account discoverability check and idempotent unlock methods", () => {
    resetMockLoginVisualDiscoverability()

    expect(getMockLoginVisualDiscoverability()).toEqual({ unlocked: false })
    expect(unlockMockLoginVisualDiscoverability()).toEqual({ unlocked: true })
    expect(unlockMockLoginVisualDiscoverability()).toEqual({ unlocked: true })
  })
})
