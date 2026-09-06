import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/fingerprints/fingerprinthub-fingerprint-view.tsx"), "utf8")
const workspaceSource = readFileSync(path.resolve(process.cwd(), "components/fingerprints/fingerprint-library-workspace.tsx"), "utf8")

describe("fingerprinthub-fingerprint-view contract", () => {
  it("delegates the persisted FingerPrintHub library workflow to the shared workspace", () => {
    expect(source).toContain("export function FingerPrintHubFingerprintView")
    expect(source).toContain("FingerprintLibraryWorkspace<FingerPrintHubFingerprint, FingerPrintHubFingerprintDetail>")
    expect(source).toContain('library="fingerprinthub"')
    expect(source).toContain('owner="fingerprinthub-fingerprint-view-content"')
    expect(source).toContain("useList={useFingerPrintHubFingerprints}")
    expect(source).toContain("useDetail={useFingerPrintHubFingerprint}")
    expect(source).toContain("useBulkDelete={useBulkDeleteFingerPrintHubFingerprints}")
    expect(source).toContain("renderTable={(props) => <FingerPrintHubFingerprintDataTable {...props} />}")
    expect(source).toContain('inspection={{ source: "fingerprinthub", fingerprint }}')
  })

  it("keeps canonical name detail, deletion, and loading ownership in the workspace", () => {
    expect(workspaceSource).toContain("const detailQuery = useDetail(selectedName)")
    expect(workspaceSource).toContain("selectedRows.map((row) => row.name)")
    expect(workspaceSource).toContain("onRowClick: (row) => setSelectedName(row.name)")
    expect(workspaceSource).toContain("selectedNames.includes(current) ? null : current")
    expect(workspaceSource).toContain("<ContentHandoff")
    expect(workspaceSource).toContain('layer="workspace"')
    expect(workspaceSource).not.toContain("onAddSingle")
    expect(workspaceSource).not.toContain("useInteractionOpenLoader")
    expect(source).not.toContain("SmartFilter")
    expect(source).not.toContain("FingerprintDialog")
  })
})
