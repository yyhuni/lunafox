import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import { defaultLocale } from "@/i18n/config"

const source = readFileSync(path.resolve(process.cwd(), "i18n/config.ts"), "utf8")
const zhMessages = JSON.parse(
  readFileSync(path.resolve(process.cwd(), "messages/zh.json"), "utf8")
)
const enMessages = JSON.parse(
  readFileSync(path.resolve(process.cwd(), "messages/en.json"), "utf8")
)

function flattenMessageKeys(
  value: unknown,
  prefix = "",
  output: string[] = []
): string[] {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    for (const [key, child] of Object.entries(value)) {
      flattenMessageKeys(child, prefix ? `${prefix}.${key}` : key, output)
    }
    return output
  }

  output.push(prefix)
  return output
}

describe("config contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("// Internationalization configuration file")
  })

  it("uses English as the global fallback locale", () => {
    expect(defaultLocale).toBe("en")
  })

  it("keeps zh and en message keys in parity", () => {
    const zhKeys = new Set(flattenMessageKeys(zhMessages))
    const enKeys = new Set(flattenMessageKeys(enMessages))

    expect([...zhKeys].filter((key) => !enKeys.has(key)).sort()).toEqual([])
    expect([...enKeys].filter((key) => !zhKeys.has(key)).sort()).toEqual([])
  })

  it("keeps selected-row export action labels scoped as short verbs", () => {
    expect(zhMessages.common.export.all).toBe("导出所有")
    expect(zhMessages.common.export.selected).toBe("导出选中")
    expect(enMessages.common.export.all).toBe("Export all")
    expect(enMessages.common.export.selected).toBe("Export selected")
  })

  it("does not keep ambiguous bulk-delete labels in common actions", () => {
    expect(zhMessages.common.actions).not.toHaveProperty("bulkDelete")
    expect(enMessages.common.actions).not.toHaveProperty("bulkDelete")
  })

  it("keeps GitHub star encouragement copy action-oriented", () => {
    expect(zhMessages.githubStar.remaining).toBe("还差 {count} 个")
    expect(zhMessages.githubStar.sprintCta).toBe("帮它冲到 {count}")
    expect(enMessages.githubStar.remaining).toBe("{count} to go")
    expect(enMessages.githubStar.sprintCta).toBe("Help it reach {count}")
  })

  it("does not imply default item lists in bulk destructive confirmation copy", () => {
    const bulkConfirmationKeys = [
      "bulkDeleteVulnMessage",
      "bulkDeleteScanMessage",
      "bulkDeleteOrgMessage",
      "bulkDeleteTargetMessage",
      "bulkUnlinkTargetMessage",
    ] as const

    for (const key of bulkConfirmationKeys) {
      expect(zhMessages.common.confirm[key]).not.toContain("以下")
      expect(enMessages.common.confirm[key]).not.toContain("following")
    }

    expect(zhMessages.common.confirm.bulkDeleteTargetMessage).toBe(
      "将永久删除 {count} 个目标及其所有关联数据。此操作无法撤销。"
    )
    expect(enMessages.common.confirm.bulkDeleteTargetMessage).toBe(
      "This will permanently delete {count} targets and all their associated data. This action cannot be undone."
    )
  })

  it("keeps scan bulk feedback copy plural-safe in English", () => {
    expect(enMessages.scan.initiate.bulkTargetsDesc).toBe(
      "{count, plural, one {For 1 selected target} other {For # selected targets}}"
    )
    expect(enMessages.scan.initiate.bulkOrganizationsDesc).toBe(
      "{count, plural, one {For 1 selected organization} other {For # selected organizations}}"
    )
    expect(enMessages.toast.scan.initiate.success).toBe("Scan started")
    expect(enMessages.toast.scan.bulkInitiate.success).toBe(
      "{count, plural, one {Started 1 scan} other {Started # scans}}"
    )
  })
})
