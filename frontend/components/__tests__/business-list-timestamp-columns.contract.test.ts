import { describe, expect, it } from "vitest"
import {
  getColumnWidthContract,
  readComponentSource,
} from "@/test/utils/business-list-width-contract"

const timestampColumnCases = [
  ["components/directories/directories-columns.tsx", "createdAt"],
  ["components/endpoints/endpoints-columns.tsx", "createdAt"],
  ["components/fingerprints/fingerprinthub-fingerprint-columns.tsx", "createdAt"],
  ["components/ip-addresses/ip-addresses-columns.tsx", "createdAt"],
  ["components/organization/organization-columns.tsx", "createdAt"],
  ["components/scan/history/scan-history-columns.tsx", "createdAt"],
  ["components/scan/scheduled/scheduled-scan-columns.tsx", "nextRunTime"],
  ["components/scan/scheduled/scheduled-scan-columns.tsx", "lastRunTime"],
  ["components/subdomains/subdomains-columns.tsx", "createdAt"],
  ["components/target/all-targets-columns.tsx", "createdAt"],
  ["components/target/all-targets-columns.tsx", "lastScannedAt"],
  ["components/tools/commands/commands-columns.tsx", "updatedAt"],
  ["components/vulnerabilities/vulnerabilities-columns.tsx", "createdAt"],
  ["components/websites/websites-columns.tsx", "createdAt"],
] as const

describe("business-list timestamp column contracts", () => {
  it.each(timestampColumnCases)(
    "keeps %s %s on a stable single-line timestamp width",
    (relativePath, accessorKey) => {
      const source = readComponentSource(relativePath)
      const column = getColumnWidthContract(source, accessorKey)

      expect(column.size).toBeGreaterThanOrEqual(176)
      expect(column.minSize).toBeGreaterThanOrEqual(176)
      expect(column.maxSize).toBeLessThanOrEqual(220)
      expect(source).toContain("TimestampCell")
    }
  )
})
