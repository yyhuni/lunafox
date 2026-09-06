import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/fingerprints/import-fingerprint-dialog-state.ts"), "utf8")

describe("import-fingerprint-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useImportFingerprintDialogState")
    expect(source).toContain("from \"react\"")
  })

  it("keeps local validation feedback but delegates remote import toast lifecycle", () => {
    expect(source).toContain('toastFeedback.error(tToast("selectFileFirst"))')
    expect(source).not.toContain('toastFeedback.success(t("import.importSuccessDetail"')
    expect(source).not.toContain('tToast("importFailed")')
  })

  it("does not parse uploaded content in the browser", () => {
    expect(source).not.toContain("file.text()")
    expect(source).not.toContain("JSON.parse")
    expect(source).not.toContain("config.validate")
  })

  it("keeps the selected file and reports the server diagnostic on failure", () => {
    expect(source).toContain("getFingerprintImportFailure(error)")
    expect(source).toContain("setImportFailure(getFingerprintImportFailure(error))")
  })
})
