import { describe, expect, it } from "vitest"
import {
  getMockCommandById,
  getMockCommandDeleteCount,
  getMockCommands,
  mockCommands,
} from "@/mock/data/commands"

describe("mock commands", () => {
  it("returns paginated commands with camelCase pagination fields", () => {
    const result = getMockCommands()

    expect(result.commands).toHaveLength(mockCommands.length)
    expect(result.total).toBe(mockCommands.length)
    expect(result.totalPages).toBe(1)
    expect(result).not.toHaveProperty("page_size")
    expect(result).not.toHaveProperty("total_count")
    expect(result).not.toHaveProperty("total_pages")
  })

  it("filters by toolId when provided", () => {
    const result = getMockCommands({ toolId: 1 })

    expect(result.commands.every(cmd => cmd.toolId === 1)).toBe(true)
    expect(result.total).toBe(2)
  })

  it("finds command by id", () => {
    expect(getMockCommandById(2)?.name).toBe("port_scan")
  })

  it("counts deletable commands", () => {
    expect(getMockCommandDeleteCount([1, 2, 999])).toBe(2)
  })
})
