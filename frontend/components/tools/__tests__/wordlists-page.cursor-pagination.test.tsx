import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import WordlistsPage, { canEditContent } from "@/components/tools/wordlists-page"

const wordlistHooks = vi.hoisted(() => ({
  useDeleteWordlist: vi.fn(),
  useWordlistTags: vi.fn(),
  useWordlists: vi.fn(),
  useWordlistContent: vi.fn(),
  useUpdateWordlistContent: vi.fn(),
  useUpdateWordlistMetadata: vi.fn(),
}))

vi.mock("@/hooks/use-wordlists", () => wordlistHooks)

vi.mock("@/components/tools/wordlist-upload-dialog", () => ({
  WordlistUploadDialog: () => null,
}))

const wordlist = {
  id: 1,
  name: "wordlists/1",
  fileName: "common.txt",
  description: "Common paths",
  tags: [],
  lineCount: 10,
  fileSize: 1024,
  createdAt: "2026-01-01T00:00:00.000Z",
  updatedAt: "2026-01-01T00:00:00.000Z",
  fileHash: "hash",
}

describe("WordlistsPage cursor pagination", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    wordlistHooks.useDeleteWordlist.mockReturnValue({ mutate: vi.fn(), isPending: false })
    wordlistHooks.useWordlistTags.mockReturnValue({ data: { results: [] } })
    wordlistHooks.useWordlistContent.mockReturnValue({ data: "", isLoading: false })
    wordlistHooks.useUpdateWordlistContent.mockReturnValue({ mutate: vi.fn(), isPending: false })
    wordlistHooks.useUpdateWordlistMetadata.mockReturnValue({ mutate: vi.fn(), isPending: false })
    wordlistHooks.useWordlists.mockImplementation((request: { pageToken?: string }) => ({
      data: request.pageToken
        ? { results: [wordlist], totalSize: 42 }
        : { results: [wordlist], totalSize: 42, nextPageToken: "wordlists-page-two" },
      isLoading: false,
      isPlaceholderData: false,
    }))
  })

  it("requires file-size metadata before enabling online content editing", () => {
    expect(canEditContent({ ...wordlist, fileSize: undefined })).toBe(false)
    expect(canEditContent(wordlist)).toBe(true)
  })

  it("uses the response continuation and cached predecessor without numbered controls", () => {
    render(<WordlistsPage />)

    const previous = screen.getByRole("button", { name: "previous" })
    const next = screen.getByRole("button", { name: "next" })
    expect(screen.getByText('tableSummary:{"count":42}')).toBeInTheDocument()
    expect(previous).toBeDisabled()
    expect(next).toBeEnabled()
    expect(screen.getByRole("button", { name: "first" })).toBeDisabled()
    expect(screen.queryByRole("button", { name: "last" })).not.toBeInTheDocument()

    fireEvent.click(next)
    expect(wordlistHooks.useWordlists).toHaveBeenLastCalledWith(expect.objectContaining({
      pageToken: "wordlists-page-two",
      pageSize: 20,
    }))
    expect(screen.getByRole("button", { name: "previous" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "next" })).toBeDisabled()

    const callsBeforeTerminalAttempt = wordlistHooks.useWordlists.mock.calls.length
    fireEvent.click(screen.getByRole("button", { name: "next" }))
    expect(wordlistHooks.useWordlists).toHaveBeenCalledTimes(callsBeforeTerminalAttempt)

    fireEvent.click(screen.getByRole("button", { name: "previous" }))
    expect(wordlistHooks.useWordlists).toHaveBeenLastCalledWith(expect.objectContaining({ pageToken: undefined }))

    fireEvent.click(screen.getByRole("button", { name: "first" }))
    expect(wordlistHooks.useWordlists).toHaveBeenLastCalledWith(expect.objectContaining({ pageToken: undefined }))
  })

  it("does not load wordlist content until the edit tab is selected", async () => {
    render(<WordlistsPage />)

    expect(wordlistHooks.useWordlistContent).toHaveBeenLastCalledWith(null)

    fireEvent.click(screen.getByRole("button", { name: "detailTitle: common.txt" }))
    expect(wordlistHooks.useWordlistContent).toHaveBeenLastCalledWith(null)

    const editTab = await screen.findByRole("tab", { name: "actions.edit" })
    fireEvent.click(editTab)
    await waitFor(() => expect(wordlistHooks.useWordlistContent).toHaveBeenLastCalledWith(1))
  })

  it("keeps an unsaved content draft when moving between detail and edit tabs", async () => {
    render(<WordlistsPage />)

    fireEvent.click(screen.getByRole("button", { name: "detailTitle: common.txt" }))
    fireEvent.click(await screen.findByRole("tab", { name: "actions.edit" }))

    const editor = await screen.findByRole("textbox", { name: "content" })
    fireEvent.change(editor, { target: { value: "local draft" } })

    fireEvent.click(screen.getByRole("tab", { name: "actions.details" }))
    fireEvent.click(screen.getByRole("tab", { name: "actions.edit" }))

    expect(await screen.findByRole("textbox", { name: "content" })).toHaveValue("local draft")
  })
})
