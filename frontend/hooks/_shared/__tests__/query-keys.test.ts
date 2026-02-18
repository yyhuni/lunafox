import { describe, expect, it } from "vitest"
import { createResourceKeys } from "@/hooks/_shared/query-keys"

describe("createResourceKeys", () => {
  it("builds list and detail keys using factories", () => {
    const keys = createResourceKeys("items", {
      list: (page: number, pageSize: number) => ({ page, pageSize }),
      detail: (id: string) => id,
    })

    expect(keys.all).toEqual(["items"])
    expect(keys.lists()).toEqual(["items", "list"])
    expect(keys.list(1, 20)).toEqual(["items", "list", { page: 1, pageSize: 20 }])
    expect(keys.details()).toEqual(["items", "detail"])
    expect(keys.detail("alpha")).toEqual(["items", "detail", "alpha"])
  })

  it("supports list keys without parameters", () => {
    const keys = createResourceKeys("logs")

    expect(keys.list()).toEqual(["logs", "list"])
    expect(keys.detail(42)).toEqual(["logs", "detail", 42])
  })
})
