import { fireEvent, render, screen } from "@testing-library/react"
import { useState } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { WordlistTagPicker } from "../wordlist-tag-picker"

const useWordlistTagsMock = vi.hoisted(() => vi.fn())

vi.mock("@/hooks/use-wordlists", () => ({
  useWordlistTags: useWordlistTagsMock,
}))

const suggestions = [
  { name: "wordlistTags/directory", displayName: "directory", wordlistCount: 1 },
  { name: "wordlistTags/dns", displayName: "dns", wordlistCount: 1 },
]

function t(key: string, params?: Record<string, string | number | Date>) {
  if (key === "removeTag") return `remove ${params?.tag ?? ""}`
  return key
}

function TagPickerHarness() {
  const [tags, setTags] = useState<string[]>([])

  return <WordlistTagPicker t={t} value={tags} onChange={setTags} />
}

describe("WordlistTagPicker", () => {
  beforeEach(() => {
    useWordlistTagsMock.mockImplementation(({ search }: { search?: string }) => ({
      data: {
        results: search
          ? suggestions.filter((tag) => tag.displayName.includes(search))
          : suggestions,
      },
    }))
  })

  it("keeps recommended tags visible while typing and commits the draft on blur", () => {
    const onChange = vi.fn()
    render(<WordlistTagPicker t={t} value={[]} onChange={onChange} />)

    fireEvent.click(screen.getByRole("button", { name: "addTag" }))
    const input = screen.getByRole("textbox", { name: "tagPickerPlaceholder" })
    fireEvent.change(input, { target: { value: "private" } })

    expect(screen.getByRole("button", { name: /directory/ })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /dns/ })).toBeInTheDocument()
    expect(useWordlistTagsMock).toHaveBeenLastCalledWith({ pageSize: 20 })

    fireEvent.blur(input)

    expect(onChange).toHaveBeenCalledWith(["private"])
  })

  it("keeps manual tag creation with selected tags instead of common recommendations", () => {
    render(<WordlistTagPicker t={t} value={[]} onChange={vi.fn()} />)

    const addTag = screen.getByRole("button", { name: "addTag" })
    const selectedTags = screen.getByText("noTagsSelected").parentElement
    const recommendations = screen.getByText("recommendedTags").parentElement

    expect(selectedTags).toContainElement(addTag)
    expect(recommendations).not.toContainElement(addTag)
  })

  it("commits the draft before adding a clicked recommendation", () => {
    render(<TagPickerHarness />)

    fireEvent.click(screen.getByRole("button", { name: "addTag" }))
    const input = screen.getByRole("textbox", { name: "tagPickerPlaceholder" })
    fireEvent.change(input, { target: { value: "private" } })

    const directory = screen.getByRole("button", { name: /directory/ })
    fireEvent.blur(input, { relatedTarget: directory })
    fireEvent.click(directory)

    expect(screen.getByRole("button", { name: "remove private" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "remove directory" })).toBeInTheDocument()
  })
})
