import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-history-list-state.ts"), "utf8")
const mutationSource = readFileSync(
  path.resolve(process.cwd(), "hooks/use-scans/mutations.ts"),
  "utf8"
)

describe("scan-history-list-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScanHistoryActions")
    expect(source).toContain("from \"react\"")
  })

  it("keeps delete actions on a stable async toast lifecycle", () => {
    expect(source).toContain("useBatchStopScans")
    expect(source).toContain("const deleteMutation = useDeleteScan()")
    expect(source).toContain("const bulkDeleteMutation = useBulkDeleteScans()")
    expect(source).toContain("const batchStopMutation = useBatchStopScans()")
    expect(source).toContain("activeSelectedScans")
    expect(source).toContain("terminalSelectedCount")
    expect(source).toContain("onSelectionClear()")
    expect(source).toContain("await deleteMutation.mutateAsync(scanToDelete.id)")
    expect(source).toContain("await bulkDeleteMutation.mutateAsync(deletedIds)")
    expect(source).not.toContain('loadingToast: {')
    expect(source).not.toContain("toastFeedback.")
    expect(mutationSource).toContain('loadingToast: {')
    expect(mutationSource).toContain('key: "common.status.deleting"')
    expect(mutationSource).toContain('id: (id) => `delete-scan-${id}`')
    expect(mutationSource).toContain('toast.success("toast.scan.delete.success"')
    expect(mutationSource).toContain('errorFallbackKey: "toast.deleteFailed"')
    expect(mutationSource).toContain('toast.success("toast.scan.delete.bulkSuccess"')
    expect(mutationSource).toContain('errorFallbackKey: "toast.bulkDeleteFailed"')
    expect(mutationSource).toContain('id: "batch-stop-scans"')
    expect(mutationSource).toContain('toast.scan.stop.batchSuccess')
  })
})
