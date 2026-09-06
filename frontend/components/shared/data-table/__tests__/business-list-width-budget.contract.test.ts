import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

import {
  BUSINESS_LIST_DESKTOP_WIDTH_BUDGET_PX,
  getDefaultColumnSizeTotal,
  readComponentSource,
} from "@/test/utils/business-list-width-contract"

const auditedBusinessListColumnFiles = [
  { name: "targets", file: "components/target/all-targets-columns.tsx" },
  { name: "organization targets", file: "components/organization/targets/targets-columns.tsx" },
  { name: "organizations", file: "components/organization/organization-columns.tsx" },
  { name: "subdomains", file: "components/subdomains/subdomains-columns.tsx" },
  { name: "directories", file: "components/directories/directories-columns.tsx" },
  { name: "websites", file: "components/websites/websites-columns.tsx" },
  { name: "endpoints", file: "components/endpoints/endpoints-columns.tsx" },
  { name: "ip addresses", file: "components/ip-addresses/ip-addresses-columns.tsx" },
  { name: "vulnerabilities", file: "components/vulnerabilities/vulnerabilities-columns.tsx" },
  { name: "scan history", file: "components/scan/history/scan-history-columns.tsx" },
  { name: "scheduled scan", file: "components/scan/scheduled/scheduled-scan-columns.tsx" },
  { name: "commands", file: "components/tools/commands/commands-columns.tsx" },
] as const

const readmeSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/data-table/README.md"),
  "utf8"
)

describe("business-list width budget contract", () => {
  it.each(auditedBusinessListColumnFiles)(
    "%s keeps the default visible columns within the shared desktop width budget",
    ({ file }) => {
      const source = readComponentSource(file)

      expect(getDefaultColumnSizeTotal(source)).toBeLessThanOrEqual(
        BUSINESS_LIST_DESKTOP_WIDTH_BUDGET_PX
      )
    }
  )

  it("documents the semantic width-role taxonomy and meaningful resize expectations near code", () => {
    expect(readmeSource).toContain("fixed support")
    expect(readmeSource).toContain("stable metric")
    expect(readmeSource).toContain("flexible summary")
    expect(readmeSource).toContain("primary flexible text")
    expect(readmeSource).toContain("default compact")
    expect(readmeSource).toContain("user-driven width pressure")
  })

  it("documents backend-paginated query control behavior near code", () => {
    expect(readmeSource).toContain("300ms debounce")
    expect(readmeSource).toContain("Enter")
    expect(readmeSource).toContain("选择即生效")
    expect(readmeSource).toContain("应用筛选")
  })
})
