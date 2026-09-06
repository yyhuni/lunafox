import { describe, expect, it } from "vitest"
import { buildDirectoryCSV } from "@/components/directories/directory-csv"

describe("buildDirectoryCSV", () => {
  it("exports nullable int64 strings without precision or unit conversion", () => {
    const csv = buildDirectoryCSV([
      {
        id: 1,
        url: "https://example.com/max",
        status: 200,
        contentLength: "9223372036854775807",
        contentType: "text/html",
        duration: "0",
        createdAt: "2026-07-01T00:00:00Z",
      },
      {
        id: 2,
        url: "https://example.com/unknown",
        status: null,
        contentLength: null,
        contentType: "",
        duration: null,
        createdAt: "2026-07-01T00:00:00Z",
      },
    ])

    const [header, maxRow, unknownRow] = csv.replace(/^\ufeff/, "").split("\n")
    expect(header).toBe("url,status,content_length,content_type,duration,created_at")
    expect(maxRow?.split(",").slice(0, 5)).toEqual([
      "https://example.com/max",
      "200",
      "9223372036854775807",
      "text/html",
      "0",
    ])
    expect(unknownRow?.split(",").slice(0, 5)).toEqual([
      "https://example.com/unknown",
      "",
      "",
      "",
      "",
    ])
    expect(csv).not.toContain("words")
    expect(csv).not.toContain("lines")
    expect(csv).not.toContain("websiteUrl")
  })
})
