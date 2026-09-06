import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/add-target-dialog-state.ts"), "utf8")

describe("add-target-dialog-state contract", () => {
  it("preserves original target input line numbers for validation feedback", () => {
    expect(source).toContain("TargetValidator.parseLines(value)")
    expect(source).toContain("lineNumber: r.lineNumber")
  })

  it("uses multi-organization selection and submits organizationIds", () => {
    expect(source).toContain("organizationIds: []")
    expect(source).toContain("handleToggleOrganization")
    expect(source).toContain("payload.organizationIds")
    expect(source).not.toContain("payload.organizationId =")
    expect(source).not.toContain("selectedOrganizations")
  })

  it("keeps selection independent from the result viewport", () => {
    const toggleStart = source.indexOf("const handleToggleOrganization")
    const clearStart = source.indexOf("const handleClearOrganizations")
    const toggleSource = source.slice(toggleStart, clearStart)
    const clearSource = source.slice(clearStart)

    expect(toggleSource).not.toContain("setOrgPage(1)")
    expect(toggleSource).not.toContain("setOrgSearchQuery")
    expect(clearSource).not.toContain("setOrgSearchQuery(\"\")")
    expect(clearSource).not.toContain("setOrgPage(1)")
  })

  it("loads organizations for the visible drawer workspace", () => {
    expect(source).toContain("const [orgPageSize, setOrgPageSizeState] = React.useState(10)")
    expect(source).toContain("const shouldEnableOrgsQuery = Boolean(prefetchEnabled || open)")
    expect(source).toContain("pageSize: orgPageSize")
    expect(source).toContain("pageToken: orgPageTokens[orgPage]")
    expect(source).toContain("filter: orgFilter")
    expect(source).not.toContain("orgPickerOpen")
  })
})
