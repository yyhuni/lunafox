import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-log-list-state.ts"), "utf8")
const taskProgressSource = readFileSync(
  path.resolve(process.cwd(), "components/scan/task-progress-log-list.tsx"),
  "utf8"
)

describe("scan-log-list-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScanLogListState")
    expect(source).toContain("export function formatScanLogLine")
    expect(source).toContain("export function buildScanLogContent")
    expect(source).toContain("from \"react\"")
    expect(source).toContain("formatLocalTimestampSeconds(log.createdAt)")
    expect(source).not.toContain("formatLogTimestamp(log.createdAt)")
    expect(source).toContain(".map(formatScanLogLine)")
    expect(taskProgressSource).toContain('from "@/components/scan/scan-log-list-state"')
    expect(taskProgressSource).not.toContain("task-progress-log-list-state")
  })
})
