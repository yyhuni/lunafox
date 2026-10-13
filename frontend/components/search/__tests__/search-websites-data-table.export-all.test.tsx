import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const saveBlobAsFileMock = vi.hoisted(() => vi.fn())

vi.mock("@/lib/file-save-utils", () => ({
  saveBlobAsFile: saveBlobAsFileMock,
}))

import { useSearchWebsitesResultModel } from "../search-websites-data-table"
import type { WebsiteSearchResult } from "@/types/search.types"

function createSearchResult(id: number): WebsiteSearchResult {
  return {
    id,
    name: `targets/1/websites/${id}`,
    url: `https://example-${id}.test`,
    host: `example-${id}.test`,
    title: "",
    tech: [],
    statusCode: 200,
    contentLength: null,
    contentType: "text/html",
    webserver: "",
    location: "",
    vhost: null,
    responseHeaders: "",
    responseBody: "",
    createdAt: "2026-10-13T00:00:00Z",
    targetId: 1,
  }
}

describe("global website search export", () => {
  beforeEach(() => vi.clearAllMocks())

  it("starts one full export for the submitted query instead of serializing the current page", async () => {
    const currentPage = Array.from({ length: 100 }, (_, index) => createSearchResult(index + 1))
    const onExportAll = vi.fn(() => Promise.resolve())
    const { result } = renderHook(() => useSearchWebsitesResultModel(currentPage, { onExportAll }))
    const exportAll = result.current.exportOptions.find((option) => option.key === "all")

    expect(exportAll).toBeDefined()
    act(() => {
      exportAll?.onClick()
    })

    expect(onExportAll).toHaveBeenCalledOnce()
    expect(saveBlobAsFileMock).not.toHaveBeenCalled()
  })

  it("preserves the local selected-row export", async () => {
    const currentPage = Array.from({ length: 100 }, (_, index) => createSearchResult(index + 1))
    const onExportAll = vi.fn(() => Promise.resolve())
    const { result } = renderHook(() => useSearchWebsitesResultModel(currentPage, { onExportAll }))
    act(() => {
      result.current.onRowSelectionChange(result.current.websites[1], true)
      result.current.onRowSelectionChange(result.current.websites[4], true)
    })
    const exportSelected = result.current.exportOptions.find((option) => option.key === "selected")

    act(() => {
      exportSelected?.onClick()
    })

    const [blob, filename] = saveBlobAsFileMock.mock.calls[0] as [Blob, string]
    expect(filename).toMatch(/^search-websites-selected-/)
    expect((await blob.text()).trim().split("\n")).toHaveLength(3)
    expect(onExportAll).not.toHaveBeenCalled()
  })
})
