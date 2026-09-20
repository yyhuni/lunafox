import { render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { WordlistEditFooter } from "../wordlist-edit-dialog-sections"

const t = (key: string) => key

function renderFooter({
  canEditContent = true,
  hasChanges = false,
  hasMetadataChanges = false,
}: {
  canEditContent?: boolean
  hasChanges?: boolean
  hasMetadataChanges?: boolean
} = {}) {
  return render(
    <WordlistEditFooter
      t={t}
      isSaving={false}
      isSavingMetadata={false}
      hasChanges={hasChanges}
      hasMetadataChanges={hasMetadataChanges}
      onSaveDialog={vi.fn()}
      canEditContent={canEditContent}
    />
  )
}

describe("WordlistEditFooter", () => {
  it("keeps a disabled save action in place before content changes", () => {
    renderFooter()

    const save = screen.getByRole("button", { name: "save" })
    expect(save).toBeDisabled()
    expect(screen.queryByText("unsavedChanges")).not.toBeInTheDocument()
  })

  it("pairs unsaved state with the enabled save action after an edit", () => {
    renderFooter({ hasChanges: true })

    expect(screen.getByText("unsavedChanges")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "save" })).toBeEnabled()
  })

  it("does not render an empty footer when neither content nor metadata can be saved", () => {
    renderFooter({ canEditContent: false })

    expect(screen.queryByRole("button", { name: "save" })).not.toBeInTheDocument()
  })
})
