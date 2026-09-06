import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/blacklist/blacklist-settings-workspace.tsx"), "utf8")
const routeSource = readFileSync(path.resolve(process.cwd(), "app/settings/blacklist/content.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/settings/blacklist/blacklist-settings-layout.ts"), "utf8")
const loadingStateSource = readFileSync(
  path.resolve(process.cwd(), "components/settings/blacklist/blacklist-settings-loading-state.tsx"),
  "utf8"
)

describe("blacklist settings workspace contract", () => {
  it("keeps route ownership thin and the shared workspace as the only editor owner", () => {
    expect(source).toContain("export function BlacklistSettingsWorkspace")
    expect(routeSource).toContain("BlacklistSettingsWorkspace")
    expect(source).toContain("BlacklistSettingsLoadingState")
    expect(loadingStateSource).toContain("export function BlacklistSettingsLoadingState")
    expect(source).not.toContain("BlacklistSettingsSkeleton")
    expect(source).not.toContain("blacklist-settings-skeleton")
  })

  it("uses the shared line-validation editor and the restricted blacklist parser", () => {
    expect(source).toContain("BulkLineValidationInput")
    expect(source).toContain("parseBlacklistRules")
    expect(source).toContain("submittedBlacklistPatterns")
    expect(source).toContain("data-testid=\"blacklist-rule-list\"")
    expect(source).toContain("data-testid=\"blacklist-rule-editor\"")
    expect(source).not.toContain('import { LineNumberedTextarea }')
    expect(source).not.toContain("<LineNumberedTextarea")
    expect(source).not.toContain("validateBlacklistRule")
    expect(source).not.toContain("categorizeBlacklistRule")
  })

  it("separates the read-only inherited global view from the Target-local mutation payload", () => {
    expect(source).toContain("const isTargetScope = embedded && targetId !== undefined")
    expect(source).toContain("useGlobalBlacklistPolicy")
    expect(source).toContain("useTargetBlacklistPolicy")
    expect(source).toContain('data-testid={readOnly ? "blacklist-inherited-rules" : undefined}')
    expect(source).toContain("readOnly")
    expect(source).toContain("patterns: submittedBlacklistPatterns(parsedRules)")
    expect(source).toContain("etag: editablePolicy.etag")
    expect(source).toContain("updateTargetPolicy.mutate({ targetId, ...input }, { onSuccess })")
    expect(source).toContain("updateGlobalPolicy.mutate(input, { onSuccess })")
    expect(source).toContain('scope={isTargetScope ? "target" : "global"}')
    expect(source).toContain("inherited: true")
  })

  it("blocks invalid local input and renders the same target scope in its loading state", () => {
    expect(source).toContain("disabled={isSaving || errorCount > 0 || !editablePolicy?.etag}")
    expect(source).toContain("validationResult")
    expect(loadingStateSource).toContain('scope?: "global" | "target"')
    expect(loadingStateSource).toContain('scope === "target" ? t("ruleList.targetTitle")')
    expect(loadingStateSource).toContain('scope === "target" ? t("editor.targetTitle")')
  })

  it("keeps the workbench responsive and pairs resolved/loading geometry through the layout contract", () => {
    const sharedLayoutConstants = [
      "BLACKLIST_PAGE_SHELL_CLASS",
      "BLACKLIST_CONTENT_SHELL_CLASS",
      "BLACKLIST_EMBEDDED_PAGE_SHELL_CLASS",
      "BLACKLIST_EMBEDDED_CONTENT_SHELL_CLASS",
      "BLACKLIST_WORKBENCH_GRID_CLASS",
      "BLACKLIST_RULE_LIST_CARD_CLASS",
      "BLACKLIST_RULE_LIST_HEADER_CLASS",
      "BLACKLIST_RULE_LIST_CONTENT_CLASS",
      "BLACKLIST_RULE_GROUP_TRIGGER_CLASS",
      "BLACKLIST_RULE_GROUP_TABLE_CLASS",
      "BLACKLIST_RULE_ROW_CLASS",
      "BLACKLIST_RULE_ICON_CELL_CLASS",
      "BLACKLIST_RULE_VALUE_CELL_CLASS",
      "BLACKLIST_RULE_STATUS_CELL_CLASS",
      "BLACKLIST_EDITOR_CARD_CLASS",
      "BLACKLIST_EDITOR_HEADER_CLASS",
      "BLACKLIST_EDITOR_HEADER_ROW_CLASS",
      "BLACKLIST_EDITOR_TITLE_GROUP_CLASS",
      "BLACKLIST_EDITOR_EXAMPLES_ROW_CLASS",
      "BLACKLIST_EDITOR_STATUS_SLOT_CLASS",
      "BLACKLIST_EDITOR_CONTENT_CLASS",
      "BLACKLIST_EDITOR_ACTION_ROW_CLASS",
      "BLACKLIST_EDITOR_ACTION_GROUP_CLASS",
    ]

    for (const constant of sharedLayoutConstants) {
      expect(layoutSource).toContain(`export const ${constant}`)
      expect(source).toContain(constant)
      expect(loadingStateSource).toContain(constant)
    }

    expect(layoutSource).toContain(
      'export const BLACKLIST_WORKBENCH_GRID_CLASS = "grid gap-4 lg:min-h-0 lg:flex-1 lg:grid-cols-3"'
    )
    expect(layoutSource).toContain(
      'export const BLACKLIST_RULE_LIST_CONTENT_CLASS = "h-96 min-h-0 flex-none space-y-4 overflow-auto lg:h-auto lg:flex-1"'
    )
    expect(source).toContain("collapsedGroups")
    expect(source).toContain("toggleRuleGroup")
    expect(source).toContain("aria-expanded")
    expect(source).toContain("<Button")
  })

  it("retains shared foundation and loading ownership instead of local control or skeleton variants", () => {
    expect(source).toContain('from "@/components/ui/button"')
    expect(source).toContain('from "@/components/ui/badge"')
    expect(source).toContain('from "@/lib/typography"')
    expect(source).not.toContain("text-blue-")
    expect(source).not.toContain("rounded-[")
    expect(source).not.toContain("shadow-[")
    expect(loadingStateSource).toContain("ActionSkeleton")
    expect(loadingStateSource).toContain("BulkLineValidationInput")
    expect(loadingStateSource).toContain('labelClassName="sr-only"')
    expect(loadingStateSource).toContain("showEmptySummary={false}")
    expect(loadingStateSource).toContain("showSuccessSummary={false}")
    expect(loadingStateSource).not.toContain("<LineNumberedTextarea")
    expect(source).toContain("ContentHandoff")
    expect(source).toContain('"blacklist-page-content"')
    expect(source).toContain("prepareContentBeforeHandoff")
  })
})
