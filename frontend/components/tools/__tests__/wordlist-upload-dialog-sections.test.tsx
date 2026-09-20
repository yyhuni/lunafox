import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { WordlistUploadDropzone } from "../wordlist-upload-dialog-sections"

const t = (key: string) => key

function renderDropzone(
  file: File | null = null,
  onRemoveFile = vi.fn(),
  onFileSelect = vi.fn(),
) {
  return render(
    <WordlistUploadDropzone
      t={t}
      file={file}
      isDragActive={false}
      onDragOver={vi.fn()}
      onDragLeave={vi.fn()}
      onDrop={vi.fn()}
      onFileSelect={onFileSelect}
      onRemoveFile={onRemoveFile}
      formatFileSize={(bytes) => `${bytes} bytes`}
    />
  )
}

describe("WordlistUploadDropzone", () => {
  it("opens the file picker from the whole empty dropzone", () => {
    renderDropzone()

    const input = screen.getByLabelText("selectFile")
    const onInputClick = vi.fn()
    input.addEventListener("click", onInputClick)
    const dropzoneLabel = screen.getByText("dragHint").closest("label")
    expect(dropzoneLabel).not.toBeNull()

    fireEvent.click(dropzoneLabel!)

    expect(onInputClick).toHaveBeenCalledTimes(1)
  })

  it("keeps file removal separate from replacing the selected file", () => {
    const onRemoveFile = vi.fn()

    renderDropzone(new File(["admin"], "admin.txt", { type: "text/plain" }), onRemoveFile)

    const input = screen.getByLabelText("selectFile")
    const onInputClick = vi.fn()
    input.addEventListener("click", onInputClick)
    fireEvent.click(screen.getByText("admin.txt").closest("label")!)
    fireEvent.click(screen.getByRole("button", { name: "removeFile" }))

    expect(onInputClick).toHaveBeenCalledTimes(1)
    expect(onRemoveFile).toHaveBeenCalledTimes(1)
  })

  it("clears the input after selection so the same file can be selected again", () => {
    const onFileSelect = vi.fn()
    renderDropzone(null, vi.fn(), onFileSelect)

    const input = screen.getByLabelText("selectFile")
    fireEvent.change(input, {
      target: { files: [new File(["admin"], "admin.txt", { type: "text/plain" })] },
    })

    expect(onFileSelect).toHaveBeenCalledTimes(1)
    expect(input).toHaveValue("")
  })
})
