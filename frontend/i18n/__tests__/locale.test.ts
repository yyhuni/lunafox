import { beforeEach, describe, expect, it, vi } from "vitest"

const { mockCookieGet, mockHeaderGet } = vi.hoisted(() => ({
  mockCookieGet: vi.fn(),
  mockHeaderGet: vi.fn(),
}))

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({ get: mockCookieGet })),
  headers: vi.fn(async () => ({ get: mockHeaderGet })),
}))

import { resolveAcceptLanguageLocale, resolveRequestLocale } from "@/i18n/locale"

describe("locale resolution", () => {
  beforeEach(() => {
    mockCookieGet.mockReset()
    mockHeaderGet.mockReset()
    mockCookieGet.mockReturnValue(undefined)
    mockHeaderGet.mockReturnValue(null)
  })

  it("maps regional Chinese and English tags to supported base locales", () => {
    expect(resolveAcceptLanguageLocale("zh-CN,zh;q=0.9,en;q=0.8")).toBe("zh")
    expect(resolveAcceptLanguageLocale("en-US,en;q=0.9")).toBe("en")
  })

  it("respects quality values before header order", () => {
    expect(resolveAcceptLanguageLocale("zh-CN;q=0.4,en-US;q=0.9")).toBe("en")
  })

  it("ignores unsupported or explicitly unacceptable language ranges", () => {
    expect(resolveAcceptLanguageLocale("ja-JP,ja;q=0.9")).toBeUndefined()
    expect(resolveAcceptLanguageLocale("zh-CN;q=0,en-US;q=0")).toBeUndefined()
  })

  it("prefers the explicit locale Cookie over browser language", async () => {
    mockCookieGet.mockReturnValue({ value: "en" })
    mockHeaderGet.mockReturnValue("zh-CN,zh;q=0.9")

    await expect(resolveRequestLocale()).resolves.toBe("en")
  })

  it("uses browser language when no valid Cookie exists", async () => {
    mockCookieGet.mockReturnValue({ value: "fr" })
    mockHeaderGet.mockReturnValue("zh-CN,zh;q=0.9")

    await expect(resolveRequestLocale()).resolves.toBe("zh")
  })

  it("falls back to English when the browser language is unsupported", async () => {
    mockHeaderGet.mockReturnValue("ja-JP,ja;q=0.9")

    await expect(resolveRequestLocale()).resolves.toBe("en")
  })
})
