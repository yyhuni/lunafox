import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

function source(relativePath: string) {
  return readFileSync(path.resolve(process.cwd(), relativePath), "utf8")
}

const workspaceStates = [
  source("components/ip-addresses/ip-addresses-view-state.ts"),
  source("components/endpoints/endpoints-detail-view-state.ts"),
  source("components/directories/directories-view-state.ts"),
  source("components/vulnerabilities/vulnerabilities-detail-view-state.ts"),
]

const workspaceSections = [
  source("components/ip-addresses/ip-addresses-view-sections.tsx"),
  source("components/endpoints/endpoints-detail-view-sections.tsx"),
  source("components/directories/directories-view-sections.tsx"),
  source("components/vulnerabilities/vulnerabilities-detail-view-sections.tsx"),
]

describe("Website-scoped asset workspaces", () => {
  it("composes mandatory scope before user controls, includes it in queries, and resets scoped pagination", () => {
    for (const state of workspaceStates) {
      expect(state).toContain("websiteScope?: WebsiteAssetScope")
      expect(state).toContain("const isReadOnly = websiteScope?.readOnly === true")
      expect(state).toContain("const websiteScopeKey = websiteScope ?")
      expect(state).toContain("useCursorPaginationScopeChange")
      expect(state).toContain("const hasCursorScopeChanged")
      expect(state).toContain("hasCursorScopeChanged ? undefined")
      expect(state).toContain("isPlaceholderData || hasCursorScopeChanged")
      expect(state).toMatch(/reset(?:Active)?Paging\(\)/)
      expect(state).toContain("websiteScopeKey")
      expect(state).toContain("websiteScope }")
    }

    expect(workspaceStates[0]).toContain('composeWebsiteScopeFilter(websiteScope, "host", compiledFilter)')
    expect(workspaceStates[0]).toContain('tColumns("ipAddress.resolvedIpAddress")')
    expect(workspaceStates[0]).toContain('tCommon("actions.searchResolvedIPOrHost")')
    expect(workspaceSections[0]).toContain("searchPlaceholder={state.searchPlaceholder}")
    for (const state of workspaceStates.slice(1)) {
      expect(state).toContain('composeWebsiteScopeFilter(websiteScope, "websiteUrl",')
    }
  })

  it("removes mutating affordances and selection only for the Website read-only variant", () => {
    for (const section of workspaceSections.slice(0, 3)) {
      expect(section).toContain("onSelectionChange={state.isReadOnly ? undefined")
      expect(section).toContain("onExportAll={state.isReadOnly ? undefined")
      expect(section).toContain("onBulkDelete={!state.isReadOnly")
    }

    expect(workspaceSections[1]).toContain("onBulkAdd={!state.isReadOnly")
    expect(workspaceSections[2]).toContain("onBulkAdd={!state.isReadOnly")
    expect(workspaceSections[1]).toContain("if (state.isReadOnly) return null")
    expect(workspaceSections[3]).toContain("onBulkMarkAsReviewed={state.isReadOnly ? undefined")
    expect(workspaceSections[3]).toContain("onBulkMarkAsPending={state.isReadOnly ? undefined")
    expect(workspaceSections[3]).toContain("if (state.isReadOnly) return null")
  })

  it("omits the selection column while retaining the ordinary Target workspace default", () => {
    for (const state of workspaceStates) {
      expect(state).toContain("includeSelection: !isReadOnly")
    }
  })
})
