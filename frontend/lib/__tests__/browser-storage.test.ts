import { beforeEach, describe, expect, it } from "vitest"

import {
  readJsonStorage,
  readSessionValue,
  readStorageValue,
  removeStorageValue,
  writeJsonStorage,
  writeSessionValue,
  writeStorageValue,
} from "@/lib/browser-storage"

describe("browser-storage", () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
  })

  it("reads and writes string values in localStorage", () => {
    writeStorageValue("search-key", "alpha")
    expect(readStorageValue("search-key")).toBe("alpha")

    removeStorageValue("search-key")
    expect(readStorageValue("search-key")).toBeNull()
  })

  it("reads and writes json values in localStorage", () => {
    writeJsonStorage("json-key", { count: 2, items: ["a"] })
    expect(readJsonStorage("json-key", {})).toEqual({ count: 2, items: ["a"] })
    expect(readJsonStorage("missing-key", { fallback: true })).toEqual({ fallback: true })
  })

  it("reads and writes session values", () => {
    writeSessionValue("session-key", "123")
    expect(readSessionValue("session-key")).toBe("123")
  })
})
