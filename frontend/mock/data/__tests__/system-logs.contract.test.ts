import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

import { getMockSystemLogs, mockSystemLogItems } from "@/mock/data/system-logs"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/system-logs.ts"), "utf8")

describe("system-logs contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export const mockSystemLogItems")
    expect(source).toContain("export function getMockSystemLogs")
    expect(source).toContain("MockSystemLogEntriesResponse")
    expect(source).toContain("nextPageToken")
    expect(source).toContain("previousPageToken")
  })

  it("matches the canonical system logEntries transport response", () => {
    const response = getMockSystemLogs({ lines: 2 })

    expect(response.results).toHaveLength(2)
    expect(response.nextPageToken).toBe("mock-follow:5000")
    expect(response.previousPageToken).toBe("mock-older:4998")
    expect(response).not.toHaveProperty("logs")
    expect(response).not.toHaveProperty("nextCursor")
    expect(response).not.toHaveProperty("previousCursor")
  })

  it("provides a deterministic 5000-line cursor history for browser performance checks", () => {
    expect(mockSystemLogItems).toHaveLength(5000)

    const latest = getMockSystemLogs({ lines: 500 })
    const older = getMockSystemLogs({
      lines: 500,
      cursor: latest.previousPageToken,
      direction: "older",
    })
    const follow = getMockSystemLogs({
      lines: 500,
      cursor: latest.nextPageToken,
      direction: "newer",
    })

    expect(latest.results).toHaveLength(500)
    expect(older.results).toHaveLength(500)
    expect(older.results.at(-1)?.id).not.toBe(latest.results[0]?.id)
    expect(follow.results).toEqual([])
    expect(follow.nextPageToken).toBe(latest.nextPageToken)
  })

  it("includes overflow-stress log lines for system log layout checks", () => {
    expect(source).toContain("layout_overflow_probe")
    expect(source).toContain("mock_trace_")
    expect(source).toContain("runtime.configuration.digest")
    expect(source).toContain("stderr")
    expect(source).toContain("truncated: true")
  })
})
