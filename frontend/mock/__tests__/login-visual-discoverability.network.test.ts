import { afterEach, beforeEach, describe, expect, it } from "vitest"

import { resetMockLoginVisualDiscoverability } from "@/mock/data/login-visual"

const API_BASE = "http://localhost/v1"

describe("login visual discoverability mock network boundary", () => {
  beforeEach(() => {
    resetMockLoginVisualDiscoverability()
  })

  afterEach(() => {
    resetMockLoginVisualDiscoverability()
  })

  it("serves the account check and repeats the unlock custom method safely", async () => {
    const initial = await fetch(`${API_BASE}/settings/loginVisual:checkDiscoverability`)
    expect(initial.status).toBe(200)
    await expect(initial.json()).resolves.toEqual({ unlocked: false })

    const firstUnlock = await fetch(`${API_BASE}/settings/loginVisual:unlockDiscoverability`, { method: "POST" })
    const repeatedUnlock = await fetch(`${API_BASE}/settings/loginVisual:unlockDiscoverability`, { method: "POST" })
    const persisted = await fetch(`${API_BASE}/settings/loginVisual:checkDiscoverability`)

    expect(firstUnlock.status).toBe(200)
    await expect(firstUnlock.json()).resolves.toEqual({ unlocked: true })
    expect(repeatedUnlock.status).toBe(200)
    await expect(repeatedUnlock.json()).resolves.toEqual({ unlocked: true })
    await expect(persisted.json()).resolves.toEqual({ unlocked: true })
  })
})
